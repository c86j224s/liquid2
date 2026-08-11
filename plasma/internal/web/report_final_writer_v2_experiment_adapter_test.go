package web

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentmodels"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	finalWriterV2ExperimentID           = "55-final-writer-v2-2026-07-29"
	finalWriterV2ExperimentRunEnv       = "PLASMA_FINAL_WRITER_V2_EXPERIMENT_RUN"
	finalWriterV2ExperimentArchiveEnv   = "PLASMA_FINAL_WRITER_V2_EXPERIMENT_ARCHIVE"
	finalWriterV2ExperimentCommandEnv   = "PLASMA_FINAL_WRITER_V2_EXPERIMENT_CODEX"
	finalWriterV2ExperimentModelEnv     = "PLASMA_FINAL_WRITER_V2_EXPERIMENT_MODEL"
	finalWriterV2ExperimentEffortEnv    = "PLASMA_FINAL_WRITER_V2_EXPERIMENT_EFFORT"
	finalWriterV2ExperimentHumanizeEnv  = "PLASMA_FINAL_WRITER_V2_EXPERIMENT_HUMANIZE"
	finalWriterV2ExperimentTimeoutEnv   = "PLASMA_FINAL_WRITER_V2_EXPERIMENT_TIMEOUT"
	finalWriterV2ExperimentRunNamespace = "w6-b-product-reviewed-parts"
)

type finalWriterV2ExperimentPair struct {
	PairID     string `json:"pair_id"`
	TopicID    string `json:"topic_id"`
	TopicTitle string `json:"topic_title"`
	Rigor      string `json:"rigor"`
}

type finalWriterV2FrozenPart struct {
	PartIndex   int    `json:"part_index"`
	Title       string `json:"title"`
	Markdown    string `json:"markdown"`
	SHA256      string `json:"sha256"`
	ArtifactID  string `json:"artifact_id,omitempty"`
	PartEventID string `json:"part_event_id,omitempty"`
	WordCount   int    `json:"word_count,omitempty"`
}

type finalWriterV2PrepProvenance struct {
	ProductPath          string   `json:"product_path"`
	MissionID            string   `json:"mission_id"`
	PendingEventID       string   `json:"pending_event_id"`
	PlanEventID          string   `json:"plan_event_id"`
	DBPath               string   `json:"db_path"`
	LedgerEventsPath     string   `json:"ledger_events_path"`
	LedgerEventsSHA256   string   `json:"ledger_events_sha256"`
	SourceSnapshotIDs    []string `json:"source_snapshot_ids"`
	SourceArtifactIDs    []string `json:"source_artifact_ids"`
	SourceEventIDs       []string `json:"source_event_ids"`
	DiscardedFinalReport bool     `json:"discarded_final_report"`
}

type finalWriterV2FrozenManifest struct {
	ExperimentID string                      `json:"experiment_id"`
	PairID       string                      `json:"pair_id"`
	TopicID      string                      `json:"topic_id"`
	TopicTitle   string                      `json:"topic_title"`
	Rigor        string                      `json:"rigor"`
	Source       string                      `json:"source"`
	Prep         finalWriterV2PrepProvenance `json:"prep"`
	Parts        []finalWriterV2FrozenPart   `json:"parts"`
	Receipts     map[string]string           `json:"receipts,omitempty"`
}

type finalWriterV2AdapterConfig struct {
	ArchiveRoot     string
	ExecutorName    string
	CodexCommand    string
	AgentModel      string
	ReasoningEffort string
	PostHumanize    string
	Timeout         time.Duration
	Started         time.Time
}

type finalWriterV2ExperimentRun struct {
	PairID              string           `json:"pair_id"`
	Arm                 string           `json:"arm"`
	Pipeline            string           `json:"pipeline"`
	MissionID           string           `json:"mission_id"`
	PendingEventID      string           `json:"pending_event_id"`
	PlanEventID         string           `json:"plan_event_id"`
	ReportPath          string           `json:"report_markdown"`
	CheckPath           string           `json:"machine_check"`
	InputManifestSHA256 string           `json:"input_manifest_sha256"`
	ReportSHA256        string           `json:"report_sha256"`
	DBPath              string           `json:"db_path"`
	LedgerEventsPath    string           `json:"ledger_events_path"`
	LedgerEventsSHA256  string           `json:"ledger_events_sha256"`
	StageTrace          []map[string]any `json:"stage_trace"`
}

func TestFinalWriterV2ExperimentAdapterPreservesFrozenReviewedPartBytes(t *testing.T) {
	ctx := context.Background()
	pair := finalWriterV2ExperimentPairs()[0]
	archive := t.TempDir()
	manifest, digest := writeFinalWriterV2TestFrozenManifest(t, archive, pair)
	loaded, loadedDigest, err := loadFinalWriterV2FrozenManifest(archive, pair)
	if err != nil {
		t.Fatal(err)
	}
	if digest != loadedDigest || !slices.EqualFunc(manifest.Parts, loaded.Parts, func(a, b finalWriterV2FrozenPart) bool {
		return a == b
	}) {
		t.Fatalf("frozen manifest did not replay byte-identically digest=%s/%s manifest=%#v loaded=%#v", digest, loadedDigest, manifest, loaded)
	}

	reqs := map[string]finalizationPrefixFixture{}
	for _, arm := range []string{"A", "B"} {
		dbPath := filepath.Join(t.TempDir(), arm+".db")
		svc, closeStore := openFinalWriterV2ExperimentService(t, ctx, dbPath)
		defer closeStore()
		req, err := seedFinalWriterV2ExperimentTerminalPipeline(ctx, svc, pair, arm, loaded, finalWriterV2AdapterConfig{
			ExecutorName:    "codex",
			AgentModel:      agentmodels.DefaultModel,
			ReasoningEffort: "medium",
			PostHumanize:    reporting.FinalEditHumanizeDisabled,
			Started:         time.Unix(0, 0).UTC(),
		}, "provider-plan-"+arm)
		if err != nil {
			t.Fatal(err)
		}
		reqs[arm] = req
		for index, artifactID := range req.partArtifactIDs {
			artifact, err := svc.GetRawArtifact(ctx, artifactID)
			if err != nil {
				t.Fatal(err)
			}
			if got := sha256Hex(artifact.Content); got != loaded.Parts[index].SHA256 || string(artifact.Content) != loaded.Parts[index].Markdown {
				t.Fatalf("%s part %d byte identity changed sha=%s want=%s", arm, index+1, got, loaded.Parts[index].SHA256)
			}
		}
	}
	if reqs["A"].missionID == reqs["B"].missionID ||
		reqs["A"].pendingEventID == reqs["B"].pendingEventID ||
		reqs["A"].planEvent.EventID == reqs["B"].planEvent.EventID ||
		reqs["A"].reportPlanSessionID == reqs["B"].reportPlanSessionID {
		t.Fatalf("A/B lineages were not independent: A=%#v B=%#v", reqs["A"], reqs["B"])
	}
	pipelineA, _, err := reporting.FinalEditPipelineFromPlanEvent(reqs["A"].planEvent)
	if err != nil {
		t.Fatal(err)
	}
	pipelineB, _, err := reporting.FinalEditPipelineFromPlanEvent(reqs["B"].planEvent)
	if err != nil {
		t.Fatal(err)
	}
	if pipelineA.Pipeline != reporting.FinalEditPipelineReaderStyleGateV1 ||
		pipelineB.Pipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 {
		t.Fatalf("unexpected pipelines A=%q B=%q", pipelineA.Pipeline, pipelineB.Pipeline)
	}
}

func TestFinalWriterV2ExperimentAdapterRoutesPrivateRunnerThroughProductStages(t *testing.T) {
	ctx := context.Background()
	pair := finalWriterV2ExperimentPairs()[1]
	archive := t.TempDir()
	manifest, _ := writeFinalWriterV2TestFrozenManifest(t, archive, pair)
	for _, tc := range []struct {
		arm        string
		wantStages []string
	}{
		{arm: "A", wantStages: []string{reporting.FinalEditStageReader, reporting.FinalEditStageStyle, reporting.FinalEditStageGate}},
		{arm: "B", wantStages: []string{reporting.FinalEditStageWriter, reporting.FinalEditStageReader, reporting.FinalEditStageStyle, reporting.FinalEditStageGate}},
	} {
		t.Run(tc.arm, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), tc.arm+".db")
			svc, closeStore := openFinalWriterV2ExperimentService(t, ctx, dbPath)
			defer closeStore()
			req, err := seedFinalWriterV2ExperimentTerminalPipeline(ctx, svc, pair, tc.arm, manifest, finalWriterV2AdapterConfig{
				ExecutorName:    "codex",
				AgentModel:      agentmodels.DefaultModel,
				ReasoningEffort: "medium",
				PostHumanize:    reporting.FinalEditHumanizeEnabled,
				Started:         time.Unix(0, 0).UTC(),
			}, "provider-plan-"+tc.arm)
			if err != nil {
				t.Fatal(err)
			}
			executor := &finalWriterV2FixtureExecutor{service: svc}
			result, err := finalizePrefixForWebTest(ctx, svc, newID, req, executor)
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(result.Markdown) == "" {
				t.Fatalf("empty final markdown: %#v", result)
			}
			if got := finalWriterV2StageSequence(executor.requests); !slices.Equal(got, tc.wantStages) {
				t.Fatalf("stages=%#v, want %#v", got, tc.wantStages)
			}
			trace, errors := finalWriterV2StageTrace(ctx, svc, req, tc.arm, executor.requests)
			if len(errors) != 0 {
				t.Fatalf("stage trace errors=%#v trace=%#v", errors, trace)
			}
		})
	}
}

func TestFinalWriterV2ExperimentAdapterRejectsMissingFrozenManifestInsteadOfDefaulting(t *testing.T) {
	_, _, err := loadFinalWriterV2FrozenManifest(t.TempDir(), finalWriterV2ExperimentPairs()[0])
	if err == nil {
		t.Fatal("missing frozen manifest was accepted")
	}
}

func TestFinalWriterV2ExperimentAdapterRun(t *testing.T) {
	if os.Getenv(finalWriterV2ExperimentRunEnv) != "1" {
		t.Skip("set PLASMA_FINAL_WRITER_V2_EXPERIMENT_RUN=1 to execute the external fixed-reviewed-Part experiment")
	}
	ctx := context.Background()
	cfg, err := finalWriterV2ExperimentConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.ArchiveRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ensureFinalWriterV2ArchiveOutsideRepo(cfg.ArchiveRoot); err != nil {
		t.Fatal(err)
	}
	binaryPath := buildFinalWriterV2ExperimentBinary(t, cfg.ArchiveRoot)
	writeFinalWriterV2InvalidW6ANote(t, cfg.ArchiveRoot, cfg.Started)
	writeFinalWriterV2JSON(t, filepath.Join(cfg.ArchiveRoot, "control", "go-adapter-run.json"), map[string]any{
		"experiment_id":        finalWriterV2ExperimentID,
		"evidence_version":     finalWriterV2ExperimentRunNamespace,
		"status":               "running",
		"started_at":           cfg.Started.Format(time.RFC3339Nano),
		"executor":             cfg.ExecutorName,
		"agent_model":          cfg.AgentModel,
		"reasoning_effort":     cfg.ReasoningEffort,
		"post_report_humanize": cfg.PostHumanize,
	})

	runs := []finalWriterV2ExperimentRun{}
	for _, pair := range finalWriterV2ExperimentPairs() {
		manifest, manifestDigest, err := ensureFinalWriterV2FrozenReviewedManifest(ctx, cfg, binaryPath, pair)
		if err != nil {
			t.Fatalf("prepare %s frozen reviewed Parts: %v", pair.PairID, err)
		}
		for _, arm := range []string{"A", "B"} {
			if run, ok := loadFinalWriterV2ExistingExperimentRun(t, ctx, cfg, pair, arm, manifest, manifestDigest); ok {
				t.Logf("adopted existing valid W6-B %s arm %s", pair.PairID, arm)
				runs = append(runs, run)
				continue
			}
			t.Logf("running W6-B %s arm %s", pair.PairID, arm)
			run, err := runFinalWriterV2ExperimentArm(ctx, cfg, binaryPath, pair, arm, manifest, manifestDigest)
			if err != nil {
				t.Fatalf("%s %s failed: %v", pair.PairID, arm, err)
			}
			runs = append(runs, run)
		}
	}
	writeFinalWriterV2JSON(t, filepath.Join(cfg.ArchiveRoot, "control", "go-adapter-results.json"), runs)
	writeFinalWriterV2BlindPacks(t, cfg.ArchiveRoot, runs)
	adjudication, err := loadFinalWriterV2ManualAdjudication(cfg.ArchiveRoot)
	if err != nil {
		t.Fatalf("manual adjudication is required after blind packs are generated: %v", err)
	}
	accepted, err := writeFinalWriterV2ReadingResults(t, cfg.ArchiveRoot, runs, adjudication)
	if err != nil {
		t.Fatal(err)
	}
	if !accepted {
		t.Fatal("W6-B acceptance result rejected the candidate")
	}
	writeFinalWriterV2JSON(t, filepath.Join(cfg.ArchiveRoot, "control", "go-adapter-run.json"), map[string]any{
		"experiment_id":        finalWriterV2ExperimentID,
		"evidence_version":     finalWriterV2ExperimentRunNamespace,
		"status":               "completed",
		"completed_at":         time.Now().UTC().Format(time.RFC3339Nano),
		"executor":             cfg.ExecutorName,
		"agent_model":          cfg.AgentModel,
		"reasoning_effort":     cfg.ReasoningEffort,
		"post_report_humanize": cfg.PostHumanize,
	})
}

func finalWriterV2ExperimentPairs() []finalWriterV2ExperimentPair {
	return []finalWriterV2ExperimentPair{
		{PairID: "wang-anshi-northern-song-exploratory", TopicID: "wang-anshi-northern-song", TopicTitle: "Wang Anshi and Northern Song reform memory", Rigor: "exploratory"},
		{PairID: "wang-anshi-northern-song-strict", TopicID: "wang-anshi-northern-song", TopicTitle: "Wang Anshi and Northern Song reform memory", Rigor: "strict"},
		{PairID: "go-raft-implementation-roadmap-exploratory", TopicID: "go-raft-implementation-roadmap", TopicTitle: "Go Raft implementation roadmap", Rigor: "exploratory"},
		{PairID: "go-raft-implementation-roadmap-strict", TopicID: "go-raft-implementation-roadmap", TopicTitle: "Go Raft implementation roadmap", Rigor: "strict"},
	}
}

func finalWriterV2ExperimentConfig() (finalWriterV2AdapterConfig, error) {
	archive := strings.TrimSpace(os.Getenv(finalWriterV2ExperimentArchiveEnv))
	if archive == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return finalWriterV2AdapterConfig{}, err
		}
		archive = filepath.Join(home, "research-artifacts", "liquid2", "plasma", "experiments", finalWriterV2ExperimentID)
	}
	timeout := 20 * time.Minute
	if raw := strings.TrimSpace(os.Getenv(finalWriterV2ExperimentTimeoutEnv)); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return finalWriterV2AdapterConfig{}, err
		}
		timeout = parsed
	}
	model := strings.TrimSpace(os.Getenv(finalWriterV2ExperimentModelEnv))
	if model == "" {
		model = agentmodels.DefaultModel
	}
	effort := strings.TrimSpace(os.Getenv(finalWriterV2ExperimentEffortEnv))
	if effort == "" {
		effort = agentmodels.DefaultReasoningEffort
	}
	if _, _, err := agentmodels.Resolve(model, effort); err != nil {
		return finalWriterV2AdapterConfig{}, err
	}
	humanize := strings.TrimSpace(os.Getenv(finalWriterV2ExperimentHumanizeEnv))
	if humanize == "" {
		humanize = reporting.FinalEditHumanizeDisabled
	}
	if humanize != reporting.FinalEditHumanizeDisabled && humanize != reporting.FinalEditHumanizeEnabled {
		return finalWriterV2AdapterConfig{}, fmt.Errorf("unsupported experiment humanize setting %q", humanize)
	}
	command := strings.TrimSpace(os.Getenv(finalWriterV2ExperimentCommandEnv))
	if command == "" {
		command = "codex"
	}
	return finalWriterV2AdapterConfig{
		ArchiveRoot: filepath.Clean(archive), ExecutorName: "codex", CodexCommand: command, AgentModel: model,
		ReasoningEffort: effort, PostHumanize: humanize, Timeout: timeout, Started: time.Now().UTC(),
	}, nil
}
