package web

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

type finalWriterV2RecordingExecutor struct {
	delegate AgentExecutor
	archive  string
	requests []AgentRequest
	forks    []AgentSessionForkResult
}

type finalWriterV2FixtureExecutor struct {
	service  *app.Service
	requests []AgentRequest
	forks    int
}

func (executor *finalWriterV2FixtureExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	executor.requests = append(executor.requests, req)
	if req.FinalEditStage == nil {
		return AgentResult{Text: "OK", SessionID: req.PreviousSessionID}, nil
	}
	binding := *req.FinalEditStage
	if _, _, err := reporting.StartFinalEditStage(ctx, executor.service, fmt.Sprintf("evt_exp55_start_%d", len(executor.requests)), binding); err != nil {
		return AgentResult{}, err
	}
	source, err := executor.service.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return AgentResult{}, err
	}
	markdown := string(source.Content)
	operationCount := 0
	if binding.Stage != reporting.FinalEditStageGate {
		markdown = strings.TrimRight(markdown, "\n") + fmt.Sprintf("\n\n%s fixture pass.\n", binding.Stage)
		operationCount = 1
	}
	if binding.Stage == reporting.FinalEditStageGate {
		if req.LongFormFinalize == nil {
			return AgentResult{}, fmt.Errorf("gate fixture requires final binding")
		}
		_, err := reporting.SubmitFinalEditGate(ctx, executor.service, reporting.FinalEditGateSubmitRequest{
			StageBinding:       binding,
			FinalBinding:       *req.LongFormFinalize,
			StageEventID:       fmt.Sprintf("evt_exp55_submit_%d", len(executor.requests)),
			CanonicalEventID:   fmt.Sprintf("evt_exp55_final_%d", len(executor.requests)),
			ManuscriptMarkdown: markdown,
			OperationCount:     operationCount,
			Findings:           nil,
		})
		if err != nil {
			return AgentResult{}, err
		}
		return AgentResult{Text: finalEditGateSubmittedSentinel, SessionID: binding.ProviderSessionID}, nil
	}
	if binding.Stage == reporting.FinalEditStageStyle {
		_, err = reporting.SubmitFinalEditStyleStage(ctx, executor.service, binding, fmt.Sprintf("evt_exp55_submit_%d", len(executor.requests)), markdown, operationCount, finalEditStyleDiagnosesForWebTest(operationCount))
	} else {
		_, err = reporting.SubmitFinalEditStage(ctx, executor.service, binding, fmt.Sprintf("evt_exp55_submit_%d", len(executor.requests)), markdown, operationCount)
	}
	if err != nil {
		return AgentResult{}, err
	}
	return AgentResult{Text: finalEditStageSubmittedSentinel, SessionID: binding.ProviderSessionID}, nil
}

func (executor *finalWriterV2FixtureExecutor) ForkSession(_ context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	executor.forks++
	sessionID := fmt.Sprintf("provider-exp55-fork-%d", executor.forks)
	return AgentSessionForkResult{SessionID: sessionID, SourceSessionID: strings.TrimSpace(sourceSessionID)}, nil
}

func (executor *finalWriterV2FixtureExecutor) CheckForkSession(context.Context, string) error {
	return nil
}

func (executor *finalWriterV2RecordingExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	index := len(executor.requests) + 1
	executor.requests = append(executor.requests, req)
	if err := writeFinalWriterV2RawFile(executor.archive, filepath.Join("prompts", fmt.Sprintf("%02d_%s.txt", index, finalWriterV2RequestKind(req))), req.Prompt); err != nil {
		return AgentResult{}, err
	}
	result, err := executor.delegate.Run(ctx, req)
	trace := map[string]any{
		"index":               index,
		"kind":                finalWriterV2RequestKind(req),
		"mission_id":          req.MissionID,
		"tool_session_id":     req.ToolSessionID,
		"previous_session_id": req.PreviousSessionID,
		"session_id":          result.SessionID,
		"resumed":             result.Resumed,
		"error":               "",
	}
	if req.FinalEditStage != nil {
		trace["stage"] = req.FinalEditStage.Stage
		trace["source_artifact_id"] = req.FinalEditStage.SourceArtifactID
		trace["edited_artifact_id"] = req.FinalEditStage.EditedArtifactID
		trace["final_edit_pipeline"] = req.FinalEditStage.FinalEditPipeline
		trace["fork_source_agent_session_id"] = req.FinalEditStage.ForkSourceAgentSessionID
		trace["previous_provider_session_id"] = req.FinalEditStage.PreviousProviderSessionID
		trace["provider_session_id"] = req.FinalEditStage.ProviderSessionID
		trace["tools"] = append([]string(nil), req.ExtraMCPTools...)
	}
	if err != nil {
		trace["error"] = err.Error()
	}
	_ = writeFinalWriterV2RawFile(executor.archive, filepath.Join("traces", fmt.Sprintf("%02d_%s.log", index, finalWriterV2RequestKind(req))), result.Log)
	_ = writeFinalWriterV2JSONFile(executor.archive, filepath.Join("traces", fmt.Sprintf("%02d_%s.json", index, finalWriterV2RequestKind(req))), trace)
	return result, err
}

func (executor *finalWriterV2RecordingExecutor) ForkSession(ctx context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	forker, ok := executor.delegate.(AgentSessionForker)
	if !ok {
		return AgentSessionForkResult{}, fmt.Errorf("experiment executor cannot fork sessions")
	}
	result, err := forker.ForkSession(ctx, sourceSessionID)
	if err == nil {
		executor.forks = append(executor.forks, result)
		_ = writeFinalWriterV2JSONFile(executor.archive, filepath.Join("traces", fmt.Sprintf("fork_%02d.json", len(executor.forks))), result)
	}
	return result, err
}

func (executor *finalWriterV2RecordingExecutor) CheckForkSession(ctx context.Context, sourceSessionID string) error {
	readiness, ok := executor.delegate.(AgentSessionForkReadiness)
	if !ok {
		return fmt.Errorf("experiment executor cannot check fork readiness")
	}
	return readiness.CheckForkSession(ctx, sourceSessionID)
}

func runFinalWriterV2ExperimentArm(ctx context.Context, cfg finalWriterV2AdapterConfig, binaryPath string, pair finalWriterV2ExperimentPair, arm string, manifest finalWriterV2FrozenManifest, manifestDigest string) (finalWriterV2ExperimentRun, error) {
	runDir := finalWriterV2RunDir(cfg.ArchiveRoot, pair, arm)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return finalWriterV2ExperimentRun{}, err
	}
	dbPath := filepath.Join(runDir, "plasma.db")
	for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return finalWriterV2ExperimentRun{}, err
		}
	}
	svc, closeStore, err := openFinalWriterV2ExperimentServicePath(ctx, dbPath)
	if err != nil {
		return finalWriterV2ExperimentRun{}, err
	}
	executor := &finalWriterV2RecordingExecutor{archive: runDir}
	executor.delegate = CodexExecutor{
		Command: cfg.CodexCommand, WorkDir: runDir, Timeout: cfg.Timeout, Env: os.Environ(),
		MCPServer: CodexMCPServer{Name: "plasma", Command: binaryPath, Args: []string{"mcp", "-db", dbPath}, Required: true, StartupTimeoutSec: 30, ToolTimeoutSec: 240},
	}
	planSession, err := createFinalWriterV2ExperimentPlanSession(ctx, executor, cfg, pair, arm)
	if err != nil {
		closeStore()
		return finalWriterV2ExperimentRun{}, err
	}
	if err := executor.CheckForkSession(ctx, planSession); err != nil {
		closeStore()
		return finalWriterV2ExperimentRun{}, fmt.Errorf("provider/product preflight failed for plan session fork: %w", err)
	}
	req, err := seedFinalWriterV2ExperimentTerminalPipeline(ctx, svc, pair, arm, manifest, cfg, planSession)
	if err != nil {
		closeStore()
		return finalWriterV2ExperimentRun{}, err
	}
	server := NewServer(svc, Options{}).(*Server)
	result, err := server.runLongFormReaderStyleGatePipeline(ctx, req, executor)
	if err != nil {
		closeStore()
		return finalWriterV2ExperimentRun{}, err
	}
	markdown := strings.TrimSpace(fmt.Sprint(result["markdown"]))
	if markdown == "" {
		closeStore()
		return finalWriterV2ExperimentRun{}, fmt.Errorf("empty final report markdown")
	}
	reportPath := filepath.Join(runDir, "report.md")
	if err := os.WriteFile(reportPath, []byte(markdown+"\n"), 0o600); err != nil {
		closeStore()
		return finalWriterV2ExperimentRun{}, err
	}
	events, err := svc.ListEvents(ctx, req.missionID)
	if err != nil {
		closeStore()
		return finalWriterV2ExperimentRun{}, err
	}
	ledgerPath := filepath.Join(runDir, "ledger", "events.json")
	if err := writeFinalWriterV2JSONFilePath(ledgerPath, events); err != nil {
		closeStore()
		return finalWriterV2ExperimentRun{}, err
	}
	trace, traceErrors := finalWriterV2StageTrace(ctx, svc, req, arm, executor.requests)
	closeStore()

	run := finalWriterV2ExperimentRun{
		PairID: pair.PairID, Arm: arm, Pipeline: finalWriterV2PipelineForArm(arm), MissionID: req.missionID, PendingEventID: req.pendingEventID,
		PlanEventID: req.planEvent.EventID, ReportPath: reportPath, CheckPath: finalWriterV2CheckPath(cfg.ArchiveRoot, pair, arm),
		InputManifestSHA256: manifestDigest, ReportSHA256: sha256Hex([]byte(markdown + "\n")), DBPath: dbPath,
		LedgerEventsPath: ledgerPath, LedgerEventsSHA256: finalWriterV2SHA256FileNoErr(ledgerPath), StageTrace: trace,
	}
	check := finalWriterV2MachineCheck(pair, arm, req, manifest, manifestDigest, run, markdown, trace, traceErrors)
	if err := writeFinalWriterV2JSONFilePath(run.CheckPath, check); err != nil {
		return finalWriterV2ExperimentRun{}, err
	}
	if reasons := finalWriterV2PreReadingHardFailReasons(check); len(reasons) > 0 {
		return finalWriterV2ExperimentRun{}, fmt.Errorf("machine check failed before reading: %s", strings.Join(reasons, "; "))
	}
	if err := validateFinalWriterV2ArchiveRun(ctx, run, pair, arm, manifest, manifestDigest); err != nil {
		return finalWriterV2ExperimentRun{}, err
	}
	return run, nil
}

func createFinalWriterV2ExperimentPlanSession(ctx context.Context, executor AgentExecutor, cfg finalWriterV2AdapterConfig, pair finalWriterV2ExperimentPair, arm string) (string, error) {
	result, err := executor.Run(ctx, AgentRequest{
		UserText: "prepare fixed-reviewed-Part experiment plan session",
		Prompt:   fmt.Sprintf("You are preparing a Plasma fixed-reviewed-Part terminal-pipeline experiment for %s arm %s. Reply exactly: EXPERIMENT_PLAN_SESSION_READY", pair.PairID, arm),
		Model:    cfg.AgentModel, ReasoningEffort: cfg.ReasoningEffort, AgentExecutor: cfg.ExecutorName, DisableTools: true,
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(result.Text) != "EXPERIMENT_PLAN_SESSION_READY" {
		return "", fmt.Errorf("plan session preflight acknowledgement mismatch: %q", result.Text)
	}
	if strings.TrimSpace(result.SessionID) == "" {
		return "", fmt.Errorf("plan session preflight returned empty session")
	}
	return strings.TrimSpace(result.SessionID), nil
}

func buildFinalWriterV2ExperimentBinary(t *testing.T, archive string) string {
	t.Helper()
	binary := filepath.Join(archive, "bin", "plasma")
	if err := os.MkdirAll(filepath.Dir(binary), 0o700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/plasma")
	cmd.Dir = filepath.Clean(filepath.Join("..", ".."))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build plasma: %v\n%s", err, output)
	}
	return binary
}

func openFinalWriterV2ExperimentService(t *testing.T, ctx context.Context, dbPath string) (*app.Service, func()) {
	t.Helper()
	svc, closeStore, err := openFinalWriterV2ExperimentServicePath(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	return svc, closeStore
}

func openFinalWriterV2ExperimentServicePath(ctx context.Context, dbPath string) (*app.Service, func(), error) {
	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		return nil, nil, err
	}
	return app.NewService(store), func() { _ = store.Close() }, nil
}

func finalWriterV2StageSequence(requests []AgentRequest) []string {
	var stages []string
	for _, request := range requests {
		if request.FinalEditStage != nil {
			stages = append(stages, request.FinalEditStage.Stage)
		}
	}
	return stages
}

func finalWriterV2RequestKind(req AgentRequest) string {
	switch {
	case req.FinalEditStage != nil:
		return "final_edit_" + string(req.FinalEditStage.Stage)
	case req.LongFormFinalize != nil:
		return "long_form_finalize"
	case req.PartEdit != nil:
		return "part_edit"
	case req.PartAssembly != nil:
		return "part_assembly"
	case req.ReportRequirements != nil:
		return "report_requirements"
	case req.ReportPlan != nil:
		return "report_plan"
	default:
		return "plan_session"
	}
}

func finalWriterV2RunDir(archive string, pair finalWriterV2ExperimentPair, arm string) string {
	return filepath.Join(archive, "runs", finalWriterV2ExperimentRunNamespace, pair.PairID, arm)
}

func finalWriterV2CheckPath(archive string, pair finalWriterV2ExperimentPair, arm string) string {
	return filepath.Join(archive, "checks", finalWriterV2ExperimentRunNamespace, pair.PairID, arm+".machine_check.json")
}

func writeFinalWriterV2JSONFilePath(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o600)
}
