package web

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
)

func TestLongFormFinalizationWebAdapterMatchesRootResultShape(t *testing.T) {
	ctx := context.Background()
	webRun := runLongFormFinalizationParity(t, ctx, true)
	directRun := runLongFormFinalizationParity(t, ctx, false)

	if !slices.Equal(finalizationParityStages(webRun.requests), finalizationParityStages(directRun.requests)) {
		t.Fatalf("stage sequence mismatch web=%#v direct=%#v", finalizationParityStages(webRun.requests), finalizationParityStages(directRun.requests))
	}
	if len(webRun.requests) != len(directRun.requests) {
		t.Fatalf("request count mismatch web=%d direct=%d", len(webRun.requests), len(directRun.requests))
	}
	for index := range webRun.requests {
		assertFinalizationParityRequestShape(t, index, webRun.requests[index], directRun.requests[index])
	}
	if webRun.markdown != directRun.markdown || webRun.artifact.SHA256 != directRun.artifact.SHA256 {
		t.Fatalf("final artifact mismatch web sha=%s direct sha=%s", webRun.artifact.SHA256, directRun.artifact.SHA256)
	}
	assertFinalizationResponseShape(t, "web", webRun)
	assertFinalizationResponseShape(t, "direct", directRun)
	if webRun.humanizedPresent != directRun.humanizedPresent {
		t.Fatalf("humanized response key mismatch web=%t direct=%t", webRun.humanizedPresent, directRun.humanizedPresent)
	}
	if webRun.terminalPayload["final_edit_pipeline"] != directRun.terminalPayload["final_edit_pipeline"] ||
		webRun.terminalPayload["planned_final_artifact_id"] != directRun.terminalPayload["planned_final_artifact_id"] ||
		webRun.terminalPayload["artifact_sha256"] != directRun.terminalPayload["artifact_sha256"] {
		t.Fatalf("stable terminal payload mismatch web=%#v direct=%#v", webRun.terminalPayload, directRun.terminalPayload)
	}
}

type finalizationParityRun struct {
	requests         []AgentRequest
	artifact         app.RawArtifact
	event            app.LedgerEvent
	markdown         string
	terminalPayload  map[string]any
	humanizedPresent bool
}

type finalizationParityIDGenerator struct {
	counts map[string]int
}

func (generator *finalizationParityIDGenerator) next(prefix string) string {
	if generator.counts == nil {
		generator.counts = map[string]int{}
	}
	generator.counts[prefix]++
	return fmt.Sprintf("%s_parity_%02d", prefix, generator.counts[prefix])
}

func runLongFormFinalizationParity(t *testing.T, ctx context.Context, throughWeb bool) finalizationParityRun {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := openW4BRestartService(t, ctx, dbPath)
	defer closeStore()
	req := seedW4BRestartFixtureWithPipeline(t, ctx, svc, reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3, reporting.FinalEditHumanizeEnabled)
	req.started = time.Unix(123, 456).UTC()
	executor := &w4BRestartExecutor{service: svc}
	ids := &finalizationParityIDGenerator{}
	var artifact app.RawArtifact
	var event app.LedgerEvent
	var markdown string
	humanizedPresent := false
	if throughWeb {
		server := NewServer(svc, Options{}).(*Server)
		result, err := server.createSectionalLongFormReportDraft(ctx, req.missionID, req.title, req.directionHint, req.executorName,
			req.agentModel, req.agentReasoningEffort, req.agentSelectionSource, req.mcpMode, req.rigor,
			req.reportSessionPolicy, req.reportSessionPolicySelection, req.postReportHumanize,
			req.generationGuidanceProfile, req.generationGuidanceSHA256, req.pendingEventID, executor)
		if err != nil {
			t.Fatal(err)
		}
		artifact = w4BResultArtifact(t, result)
		event = w4BResultEvent(t, result)
		var ok bool
		markdown, ok = result["markdown"].(string)
		if !ok {
			t.Fatalf("web result markdown missing: %#v", result)
		}
		_, humanizedPresent = result["humanized"]
	} else {
		result, err := reportworkflow.NewRunner(reportworkflow.RunnerConfig{
			Service: svc, Lifecycle: reporting.Runner(NewServer(svc, Options{}).(*Server).reportRunner()),
			Executor: executor, NewID: ids.next,
		}).FinalizeLongFormPrefix(ctx, req.toPrefixOutput())
		if err != nil {
			t.Fatal(err)
		}
		artifact = result.Artifact
		event = result.Event
		markdown = result.Markdown
		humanizedPresent = result.Humanized != nil
	}
	events := w4BEvents(t, ctx, svc, req.missionID)
	canonical := w4BCanonicalEvent(t, events)
	return finalizationParityRun{
		requests:         append([]AgentRequest(nil), executor.requests...),
		artifact:         artifact,
		event:            event,
		markdown:         markdown,
		terminalPayload:  finalizationParityPayload(t, canonical),
		humanizedPresent: humanizedPresent,
	}
}

func assertFinalizationParityRequestShape(t *testing.T, index int, webReq AgentRequest, directReq AgentRequest) {
	t.Helper()
	if !slices.Equal(webReq.ExtraMCPTools, directReq.ExtraMCPTools) || webReq.ReplaceMCPTools != directReq.ReplaceMCPTools {
		t.Fatalf("request %d MCP contract differs web=%#v/%t direct=%#v/%t", index, webReq.ExtraMCPTools, webReq.ReplaceMCPTools, directReq.ExtraMCPTools, directReq.ReplaceMCPTools)
	}
	if webReq.FinalEditStage == nil || directReq.FinalEditStage == nil {
		t.Fatalf("request %d missing final edit binding web=%#v direct=%#v", index, webReq.FinalEditStage, directReq.FinalEditStage)
	}
	if webReq.FinalEditStage.Stage != directReq.FinalEditStage.Stage ||
		webReq.FinalEditStage.FinalEditPipeline != directReq.FinalEditStage.FinalEditPipeline ||
		webReq.FinalEditStage.AgentModel != directReq.FinalEditStage.AgentModel ||
		webReq.FinalEditStage.AgentReasoningEffort != directReq.FinalEditStage.AgentReasoningEffort ||
		webReq.FinalEditStage.GenerationGuidanceProfile != directReq.FinalEditStage.GenerationGuidanceProfile {
		t.Fatalf("request %d stable final edit binding mismatch web=%#v direct=%#v", index, *webReq.FinalEditStage, *directReq.FinalEditStage)
	}
	wantTools := mustFinalEditTools(webReq.FinalEditStage.Stage, webReq.FinalEditStage.PostReportHumanize)
	if !slices.Equal(webReq.ExtraMCPTools, wantTools) {
		t.Fatalf("request %d tools=%#v, want package contract %#v", index, webReq.ExtraMCPTools, wantTools)
	}
	if (webReq.LongFormFinalize == nil) != (directReq.LongFormFinalize == nil) {
		t.Fatalf("request %d LongFormFinalize presence differs web=%#v direct=%#v", index, webReq.LongFormFinalize, directReq.LongFormFinalize)
	}
	if webReq.LongFormFinalize != nil &&
		(webReq.LongFormFinalize.PendingEventID != directReq.LongFormFinalize.PendingEventID ||
			webReq.LongFormFinalize.PlanEventID != directReq.LongFormFinalize.PlanEventID ||
			webReq.LongFormFinalize.CompositionStrategy != directReq.LongFormFinalize.CompositionStrategy ||
			webReq.LongFormFinalize.GenerationGuidanceProfile != directReq.LongFormFinalize.GenerationGuidanceProfile) {
		t.Fatalf("request %d stable LongFormFinalize binding mismatch web=%#v direct=%#v", index, *webReq.LongFormFinalize, *directReq.LongFormFinalize)
	}
}

func assertFinalizationResponseShape(t *testing.T, name string, run finalizationParityRun) {
	t.Helper()
	if run.event.EventType != "report.artifact.created" || run.artifact.ArtifactID == "" || run.markdown == "" {
		t.Fatalf("%s result missing canonical report shape: artifact=%#v event=%#v markdown=%q", name, run.artifact, run.event, run.markdown)
	}
	if string(run.artifact.Content) != run.markdown || run.artifact.SHA256 != sha256Hex([]byte(run.markdown)) ||
		run.terminalPayload["artifact_sha256"] != run.artifact.SHA256 {
		t.Fatalf("%s artifact/event markdown shape changed: artifact=%#v payload=%#v markdown=%q", name, run.artifact, run.terminalPayload, run.markdown)
	}
	if run.humanizedPresent {
		t.Fatalf("%s final-edit pipeline response must not include legacy H5 humanized key", name)
	}
}

func finalizationParityStages(requests []AgentRequest) []string {
	stages := make([]string, 0, len(requests))
	for _, req := range requests {
		if req.FinalEditStage != nil {
			stages = append(stages, req.FinalEditStage.Stage)
		}
	}
	return stages
}

func finalizationParityEventLineage(event app.LedgerEvent) map[string]any {
	return map[string]any{
		"event_id":           event.EventID,
		"mission_id":         event.MissionID,
		"sequence":           event.Sequence,
		"event_type":         event.EventType,
		"producer":           event.Producer,
		"causation_event_id": event.CausationEventID,
		"correlation_id":     event.CorrelationID,
	}
}

func finalizationParityPayload(t *testing.T, event app.LedgerEvent) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func finalizationParityPayloadLineage(payload map[string]any) map[string]any {
	return map[string]any{
		"pending_event_id":             payload["pending_event_id"],
		"plan_event_id":                payload["plan_event_id"],
		"final_edit_pipeline":          payload["final_edit_pipeline"],
		"artifact_id":                  payload["artifact_id"],
		"planned_final_artifact_id":    payload["planned_final_artifact_id"],
		"final_edit_gate_event_id":     payload["final_edit_gate_event_id"],
		"final_edit_gate_changed":      payload["final_edit_gate_changed"],
		"artifact_sha256":              payload["artifact_sha256"],
		"report_session_id":            payload["report_session_id"],
		"previous_agent_session_id":    payload["previous_agent_session_id"],
		"fork_source_agent_session_id": payload["fork_source_agent_session_id"],
		"tool_session_id":              payload["tool_session_id"],
	}
}
