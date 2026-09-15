package articleexperiment

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// RealRunConfig identifies one E/A cell from a validated real-pilot protocol.
type RealRunConfig struct {
	ArchiveRoot        string
	RepositoryRoot     string
	ProtocolPath       string
	ProtocolSHA256     string
	RunID              string
	IssueLockReceipt   IssueLockReceipt
	LiveIssueComment   LiveIssueComment
	ExecutableRevision string
	ExecutableModified bool
	ExecutorFactory    func(FixtureBundle, ArmBundle) (Executor, error)
	StartedAt          time.Time
}

// RunRealCell reuses the existing pending/artifact/terminal publication path
// while loading identities from a real protocol rather than the synthetic one.
func RunRealCell(ctx context.Context, config RealRunConfig) (Result, error) {
	if config.ExecutorFactory == nil {
		return Result{}, fmt.Errorf("%w: executor is required", producterror.ErrInvalidInput)
	}
	archiveRoot, repositoryRoot, err := prepareArchiveRoot(config.ArchiveRoot, config.RepositoryRoot)
	if err != nil {
		return Result{}, err
	}
	loaded, err := LoadRealProtocol(archiveRoot, repositoryRoot, config.ProtocolPath, config.ProtocolSHA256)
	if err != nil {
		return Result{}, err
	}
	if err := VerifyIssueLockReceipt(loaded, config.IssueLockReceipt, config.LiveIssueComment); err != nil {
		return Result{}, err
	}
	if err := VerifyExecutionRevision(repositoryRoot, loaded, config.ExecutableRevision, config.ExecutableModified); err != nil {
		return Result{}, err
	}
	cell, ok := realRunCell(loaded.Protocol.Runs, strings.TrimSpace(config.RunID))
	if !ok || cell.ArmID == ArmReport {
		return Result{}, fmt.Errorf("%w: real E/A run ID is not preassigned", producterror.ErrInvalidInput)
	}
	fixture, ok := loaded.Fixture(cell.FixtureID)
	if !ok {
		return Result{}, fmt.Errorf("%w: real fixture is unavailable", producterror.ErrConflict)
	}
	arm, ok := loaded.Arm(cell.ArmID)
	if !ok {
		return Result{}, fmt.Errorf("%w: real arm is unavailable", producterror.ErrConflict)
	}
	fixtureRef := findRealRef(loaded.Protocol.Fixtures, cell.FixtureID)
	armRef := findRealRef(loaded.Protocol.Arms, cell.ArmID)
	fixtureBundle, err := LoadFixtureBundle(archiveRoot, repositoryRoot, config.ProtocolPath, fixtureRef.Path, fixtureRef.SHA256)
	if err != nil {
		return Result{}, err
	}
	armBundle, err := LoadArmBundle(archiveRoot, repositoryRoot, config.ProtocolPath, armRef.Path, armRef.SHA256)
	if err != nil {
		return Result{}, err
	}
	executor, err := config.ExecutorFactory(fixtureBundle, armBundle)
	if err != nil || executor == nil {
		return Result{}, fmt.Errorf("%w: real executor is unavailable", producterror.ErrInvalidInput)
	}
	inputs := []InputArtifact{
		{ID: "source_body", SHA256: bytesSHA256(fixtureBundle.SourceBody), Content: fixtureBundle.SourceBody},
		{ID: "source_catalog", SHA256: bytesSHA256(fixtureBundle.SourceCatalog), Content: fixtureBundle.SourceCatalog},
		{ID: "dossier", SHA256: bytesSHA256(fixtureBundle.Dossier), Content: fixtureBundle.Dossier},
		{ID: "claim_inventory", SHA256: bytesSHA256(fixtureBundle.ClaimInventory), Content: fixtureBundle.ClaimInventory},
	}
	return runRealLoadedCell(ctx, archiveRoot, repositoryRoot, loaded, cell, fixture, fixtureRef.SHA256, arm, armRef.SHA256, armBundle.Treatment, inputs, executor, config.StartedAt)
}

func runRealLoadedCell(ctx context.Context, archiveRoot, repositoryRoot string, loaded LoadedRealProtocol, cell RunCell, fixture RealFixture, fixtureSHA string, arm RealArm, armSHA string, treatment []byte, inputs []InputArtifact, executor Executor, startedAt time.Time) (Result, error) {
	now := time.Now().UTC()
	started := startedAt.UTC()
	if startedAt.IsZero() {
		started = now
	} else if started.After(now) {
		return Result{}, fmt.Errorf("%w: started time is in the future", producterror.ErrInvalidInput)
	}
	runDir, err := prepareRunDir(archiveRoot, repositoryRoot, cell.RunID)
	if err != nil {
		return Result{}, err
	}
	runInfo, err := os.Stat(runDir)
	if err != nil {
		return Result{RunDir: runDir}, err
	}
	pending := PendingManifest{SchemaVersion: PendingSchemaVersion, ProtocolID: loaded.Protocol.ProtocolID, ProtocolSHA256: loaded.ProtocolSHA256, FixtureID: cell.FixtureID, FixtureSHA256: fixtureSHA, ArmID: cell.ArmID, ArmSHA256: armSHA, RunID: cell.RunID, StartedAt: started.Format(terminalTimeFormat)}
	pendingPath := filepath.Join(runDir, "run.pending.json")
	if err := writeJSONExclusive(pendingPath, pending); err != nil {
		return Result{RunDir: runDir}, err
	}
	pendingRaw, err := readRegularFile(pendingPath, maxContractBytes)
	if err != nil {
		return Result{RunDir: runDir, PendingPath: pendingPath}, err
	}
	output, executeErr := executor.Execute(ctx, ExecutionInput{ProtocolID: loaded.Protocol.ProtocolID, ProtocolSHA256: loaded.ProtocolSHA256, Fixture: ExecutionFixture{FixtureID: fixture.FixtureID, ValueType: fixture.ValueType, Audience: fixture.Audience, ReaderPromise: fixture.ReaderPromise, Emphasis: fixture.Emphasis, Language: fixture.Language}, Inputs: cloneInputs(inputs), Arm: ExecutionArm{ArmID: arm.ArmID, Role: arm.Role, ContractSHA256: arm.Treatment.Contract.SHA256, ContextualOnly: arm.ContextualOnly}, ArmContract: InputArtifact{ID: "treatment", SHA256: bytesSHA256(treatment), Content: append([]byte(nil), treatment...)}, Cell: cell})
	if err := validateRunDirectory(archiveRoot, repositoryRoot, runDir, runInfo); err != nil {
		return Result{RunDir: runDir, PendingPath: pendingPath}, err
	}
	return finalizeRunResult(runDir, pendingPath, pendingRaw, pending, arm, output, executeErr)
}

func finalizeRunResult(runDir, pendingPath string, pendingRaw []byte, pending PendingManifest, arm RealArm, output ExecutionOutput, executeErr error) (Result, error) {
	if executeErr == nil {
		if len(output.Artifacts) != len(arm.Artifacts) {
			executeErr = fmt.Errorf("%w: real artifact set is incomplete", producterror.ErrConflict)
		} else {
			for index, contract := range arm.Artifacts {
				artifact := output.Artifacts[index]
				if artifact.Kind != contract.Kind || artifact.MediaType != contract.MediaType || artifact.Filename != contract.Filename || len(artifact.Content) == 0 || int64(len(artifact.Content)) > contract.MaxByteSize {
					executeErr = fmt.Errorf("%w: real artifact contract differs", producterror.ErrConflict)
					break
				}
			}
		}
	}
	failureClass := ""
	if receiptErr := validateAttempts(output.Attempts, executeErr != nil); receiptErr != nil {
		executeErr = receiptErr
		failureClass = "executor_contract_failed"
		output.Attempts = []AttemptReceipt{{Attempt: 1, Kind: "semantic", Outcome: "failed"}}
	} else if executeErr != nil {
		failureClass = "executor_failed"
	}
	terminal := TerminalManifest{SchemaVersion: TerminalSchemaVersion, Pending: pending, PendingSHA256: bytesSHA256(pendingRaw), Attempts: append([]AttemptReceipt(nil), output.Attempts...)}
	if executeErr != nil {
		terminal.Status = "failed"
		terminal.Failure = &FailureReceipt{Class: failureClass, Message: "real pilot executor failed"}
		if failureClass == "executor_failed" && len(output.Artifacts) != 0 {
			receipts, writeErr := writeArtifacts(runDir, output.Artifacts)
			if writeErr != nil {
				terminal.Failure = &FailureReceipt{Class: "artifact_failed", Message: "real pilot artifact storage failed"}
				executeErr = writeErr
			} else {
				terminal.PartialArtifacts = receipts
			}
		}
	} else {
		terminal.Status = "completed"
		receipts, writeErr := writeArtifacts(runDir, output.Artifacts)
		if writeErr != nil {
			terminal.Status = "failed"
			terminal.PartialArtifacts = receipts
			terminal.Failure = &FailureReceipt{Class: "artifact_failed", Message: "real pilot artifact storage failed"}
			executeErr = writeErr
		} else {
			terminal.Artifacts = receipts
		}
	}
	terminal.CompletedAt = time.Now().UTC().Format(terminalTimeFormat)
	terminalPath, terminalErr := writeTerminalAtomic(runDir, terminal)
	result := Result{RunDir: runDir, PendingPath: pendingPath, TerminalPath: terminalPath, Terminal: terminal}
	if terminalErr != nil {
		return result, terminalErr
	}
	validated, err := ValidateTerminal(runDir)
	if err != nil {
		return result, err
	}
	result.Terminal = validated
	if executeErr != nil {
		return result, safeRunFailure(terminal, executeErr)
	}
	return result, nil
}

func realRunCell(cells []RunCell, runID string) (RunCell, bool) {
	for _, cell := range cells {
		if cell.RunID == runID {
			return cell, true
		}
	}
	return RunCell{}, false
}
func findRealRef(refs []FileRef, id string) FileRef {
	for _, ref := range refs {
		if ref.ID == id {
			return ref
		}
	}
	return FileRef{}
}
