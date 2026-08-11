package reportexperiment

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentmodels"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
)

// Run은 archive-local fixed Part fixture를 제품 V3 finalization tail로 한 번 실행한다.
//
// 이 함수는 새 run directory와 SQLite DB를 만들고, fixture seed 뒤 실제
// reportworkflow.FinalizeLongFormPrefix만 호출한다. provider 종류, retry 정책,
// prompt, tool allowlist, 제품 schema는 여기서 새로 정의하지 않는다.
func Run(ctx context.Context, config Config) (Result, error) {
	if err := validateRunConfig(config); err != nil {
		return Result{}, err
	}
	if err := ValidateBinaryPair(config.BinaryPair); err != nil {
		return Result{}, err
	}
	model, effort, err := agentmodels.ResolveForSession(config.AgentModel, config.AgentReasoningEffort, "")
	if err != nil {
		return Result{}, fmt.Errorf("%w: invalid Codex model settings: %v", producterror.ErrInvalidInput, err)
	}
	config.AgentModel = model
	config.AgentReasoningEffort = effort
	repoRoot, err := canonicalOptionalExistingPath(config.RepositoryRoot)
	if err != nil {
		return Result{}, err
	}
	archiveRoot, err := prepareArchiveRoot(config.ArchiveRoot, repoRoot)
	if err != nil {
		return Result{}, err
	}
	loaded, err := loadFixture(archiveRoot, config.FixturePath, repoRoot)
	if err != nil {
		return Result{}, err
	}
	prepared, err := preflightFixture(loaded, config.RunID)
	if err != nil {
		return Result{}, err
	}
	runDir, err := prepareRunDir(archiveRoot, config.RunID, repoRoot)
	if err != nil {
		return Result{}, err
	}
	dbPath := filepath.Join(runDir, "plasma.db")
	handle, err := config.ServiceFactory(ctx, dbPath)
	if err != nil {
		return Result{}, err
	}
	if handle.Service == nil {
		return Result{}, fmt.Errorf("%w: service factory returned nil service", producterror.ErrInvalidInput)
	}
	closed := false
	if handle.Close != nil {
		defer func() {
			if !closed {
				_ = handle.Close()
			}
		}()
	}
	svc := handle.Service

	started := config.StartedAt
	if started.IsZero() {
		started = time.Now().UTC()
	}
	executor, err := config.ExecutorFactory(ctx, ExecutorContext{Service: svc, RunDir: runDir, DBPath: dbPath})
	if err != nil {
		return Result{}, err
	}
	if executor == nil {
		return Result{}, fmt.Errorf("%w: Codex executor is required", producterror.ErrInvalidInput)
	}

	nodes := &nodeRecorder{}
	requests := newRequestRecorder(executor)
	bootstrap, err := bootstrapPlanSession(ctx, requests, config.AgentModel, config.AgentReasoningEffort)
	if err != nil {
		return Result{}, err
	}
	seed, err := seedFixedPartPrefix(ctx, svc, loaded, prepared, config.RunID, config.AgentModel, config.AgentReasoningEffort, bootstrap.SessionID, started)
	if err != nil {
		return Result{}, err
	}
	idGenerator := newRunIDGenerator(config.RunID)
	runner := reportworkflow.NewRunner(reportworkflow.RunnerConfig{
		Service: svc,
		Lifecycle: reporting.Runner{
			Service: svc,
			NewID:   idGenerator.NewID,
		},
		Executor: requests,
		NewID:    idGenerator.NewID,
	}).WithObserver(nodes)
	output, err := runner.FinalizeLongFormPrefix(ctx, seed.Prefix)
	if err != nil {
		return Result{}, err
	}
	reportPath := filepath.Join(runDir, "report.md")
	ledgerPath := filepath.Join(runDir, "ledger.json")
	manifestPath := filepath.Join(runDir, "manifest.json")
	if err := os.WriteFile(reportPath, []byte(output.Markdown), 0o600); err != nil {
		return Result{}, err
	}
	events, err := svc.ListEvents(ctx, seed.Prefix.MissionID)
	if err != nil {
		return Result{}, err
	}
	ledgerJSON, err := marshalIndented(events)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(ledgerPath, ledgerJSON, 0o600); err != nil {
		return Result{}, err
	}
	reportSHA, err := fileSHA256(reportPath)
	if err != nil {
		return Result{}, err
	}
	ledgerSHA, err := fileSHA256(ledgerPath)
	if err != nil {
		return Result{}, err
	}
	if handle.Close != nil {
		closeErr := handle.Close()
		closed = true
		if closeErr != nil {
			return Result{}, closeErr
		}
	}
	manifest := buildManifest(config, loaded, seed, bootstrap, output, nodes.Snapshot(), requests.Snapshot(), runDir, dbPath, reportPath, ledgerPath, manifestPath, reportSHA, ledgerSHA, started)
	encodedManifest, err := manifestJSON(manifest)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(manifestPath, encodedManifest, 0o600); err != nil {
		return Result{}, err
	}
	return Result{RunDir: runDir, DBPath: dbPath, ReportPath: reportPath, LedgerPath: ledgerPath, ManifestPath: manifestPath, Manifest: manifest}, nil
}

func validateRunConfig(config Config) error {
	if strings.TrimSpace(config.ArchiveRoot) == "" || strings.TrimSpace(config.FixturePath) == "" {
		return fmt.Errorf("%w: archive root and fixture path are required", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(config.RepositoryRoot) == "" {
		return fmt.Errorf("%w: repository root is required", producterror.ErrInvalidInput)
	}
	if err := validateRunID(config.RunID); err != nil {
		return err
	}
	if config.ServiceFactory == nil {
		return fmt.Errorf("%w: service factory is required", producterror.ErrInvalidInput)
	}
	if config.ExecutorFactory == nil {
		return fmt.Errorf("%w: Codex executor factory is required", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(config.AgentModel) == "" || strings.TrimSpace(config.AgentReasoningEffort) == "" {
		return fmt.Errorf("%w: Codex model and reasoning effort are required", producterror.ErrInvalidInput)
	}
	return nil
}

func buildManifest(config Config, loaded LoadedFixture, seed seedResult, bootstrap BootstrapReceipt, output reportworkflow.DraftOutput, nodes []NodeObservation, requests []AgentRequestObservation, runDir, dbPath, reportPath, ledgerPath, manifestPath, reportSHA, ledgerSHA string, started time.Time) Manifest {
	productRevision := strings.TrimSpace(config.BinaryPair.Experiment.VCSRevision)
	if productRevision == "" {
		productRevision = strings.TrimSpace(config.BinaryPair.PlasmaMCP.VCSRevision)
	}
	return Manifest{
		SchemaVersion:    ManifestSchemaVersion,
		RunID:            config.RunID,
		CreatedAt:        started.UTC().Format(timeFormat),
		RunnerScope:      RunnerScope,
		ProductRevision:  productRevision,
		Fixture:          fixtureManifest(loaded),
		SourceProvenance: sourceProvenanceManifest(loaded.Spec.SourceProvenance),
		Binaries:         config.BinaryPair,
		Agent:            ManifestAgent{Executor: executorCodex, Model: config.AgentModel, ReasoningEffort: config.AgentReasoningEffort},
		Bootstrap:        bootstrap,
		Seed:             seed.Plan,
		Observations:     nodes,
		AgentRequests:    requests,
		FinalArtifact: ManifestFinalArtifact{
			ArtifactID: output.Artifact.ArtifactID, EventID: output.Event.EventID, ArtifactSHA256: output.Artifact.SHA256,
			ReportSHA256: reportSHA, LedgerSHA256: ledgerSHA,
		},
		Outputs: ManifestOutputs{
			RunDirectory: runDir, DatabasePath: dbPath, ReportPath: reportPath, LedgerPath: ledgerPath, ManifestPath: manifestPath,
		},
		IntentionalDifferences: []string{
			"tools_disabled_bootstrap_session_is_a_finalization_only_harness_receipt_not_a_replacement_for_product_planning_stage",
		},
	}
}

func marshalIndented(value any) ([]byte, error) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}
