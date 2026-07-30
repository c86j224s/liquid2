package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestReportLongFormStageEditToolPartitionAndClosedConfig(t *testing.T) {
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeDisabled)
	reader := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageReader, reporting.FinalEditReaderSourceArtifactID(finalBinding.PlanEventID, finalBinding.PartArtifactIDs), "art_reader_edit", "ses_reader", "provider-reader", "provider-plan", "provider-plan")
	readerTools := []string{ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditRead, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit}
	readerServer := NewServer(service, WithBinding(stageMCPBinding(reader)), WithFinalEditStageBinding(reader), WithEnabledTools(readerTools))
	listed := toolNames(readerServer.ListTools())
	for _, name := range readerTools {
		if !containsString(listed, name) {
			t.Fatalf("reader tool %s was not listed: %#v", name, listed)
		}
	}
	if containsString(listed, ToolReportLongFormEditStart) || containsString(listed, ToolReportLongFormStyleEditStart) {
		t.Fatalf("reader server leaked non-reader final edit tools: %#v", listed)
	}

	closed := NewServer(service,
		WithBinding(stageMCPBinding(reader)),
		WithFinalEditStageBinding(reader),
		WithLongFormFinalizeBinding(finalBinding),
		WithEnabledTools(append(readerTools, ToolReportLongFormEditStart)),
	)
	listed = toolNames(closed.ListTools())
	for _, name := range append(readerTools, ToolReportLongFormEditStart) {
		if containsString(listed, name) {
			t.Fatalf("closed final edit config leaked tool %s: %#v", name, listed)
		}
	}
	result := closed.Call(context.Background(), ToolCall{Name: ToolReportLongFormReaderEditStart, Arguments: mustArgs(t, stageStartArgs(reader, "rfe_closed", "closed-start"))})
	if result.Error == nil || result.Error.ErrorKind != "binding" {
		t.Fatalf("closed config call did not fail as binding: %#v", result)
	}

	partialStage := NewServer(service,
		WithBinding(Binding{MissionID: finalBinding.MissionID, AgentSessionID: finalBinding.ToolSessionID, AgentExecutor: finalBinding.AgentExecutor}),
		WithLongFormFinalizeBinding(finalBinding),
		WithFinalEditStageBinding(reporting.FinalEditStageBinding{Title: "partial"}),
		WithEnabledTools([]string{ToolReportLongFormEditStart}),
	)
	if containsString(toolNames(partialStage.ListTools()), ToolReportLongFormEditStart) {
		t.Fatal("partial stage binding fell back to legacy final edit tools")
	}
	partialFinal := NewServer(service,
		WithBinding(stageMCPBinding(reader)),
		WithFinalEditStageBinding(reader),
		WithLongFormFinalizeBinding(reporting.LongFormFinalizeBinding{Title: "partial"}),
		WithEnabledTools(readerTools),
	)
	if containsString(toolNames(partialFinal.ListTools()), ToolReportLongFormReaderEditStart) {
		t.Fatal("reader stage accepted a partial final binding")
	}

	gate := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageGate, "art_reader_edit", finalBinding.ArtifactID, finalBinding.ToolSessionID, finalBinding.ProviderSessionID, finalBinding.PreviousProviderSessionID, finalBinding.ForkSourceAgentSessionID)
	gateServer := NewServer(service,
		WithBinding(stageMCPBinding(gate)),
		WithFinalEditStageBinding(gate),
		WithLongFormFinalizeBinding(finalBinding),
		WithEnabledTools([]string{ToolReportLongFormEditStart, ToolReportLongFormFinalize}),
	)
	listed = toolNames(gateServer.ListTools())
	if !containsString(listed, ToolReportLongFormEditStart) || containsString(listed, ToolReportLongFormFinalize) {
		t.Fatalf("gate server listed wrong finalization tools: %#v", listed)
	}
}

func TestReportLongFormFinalWriteStageUsesDedicatedPartitionAndAssembly(t *testing.T) {
	service, finalBinding, writer := seededFinalEditWriterStageService(t)
	assembly, created, err := reporting.EnsureFinalEditAssembly(context.Background(), service, "evt_writer_assembly", writer)
	if err != nil || !created || assembly.Artifact.ArtifactID != writer.SourceArtifactID {
		t.Fatalf("writer assembly created=%t result=%#v err=%v", created, assembly, err)
	}
	writerTools := []string{ToolReportLongFormFinalWriteStart, ToolReportLongFormFinalWriteRead, ToolReportLongFormFinalWritePatch, ToolReportLongFormFinalWriteSubmit}
	server := NewServer(service, WithBinding(stageMCPBinding(writer)), WithFinalEditStageBinding(writer), WithEnabledTools(writerTools))
	if server.finalEditConfigErr != nil {
		t.Fatalf("valid writer server closed config: %v", server.finalEditConfigErr)
	}
	listed := toolNames(server.ListTools())
	for _, name := range writerTools {
		if !containsString(listed, name) {
			t.Fatalf("writer tool %s was not listed: %#v", name, listed)
		}
	}
	for _, name := range []string{ToolReportLongFormReaderEditStart, ToolReportLongFormStyleEditStart, ToolReportLongFormEditStart, ToolReportLongFormFinalize} {
		if containsString(listed, name) {
			t.Fatalf("writer server leaked non-writer tool %s: %#v", name, listed)
		}
	}

	draftID := "rfe_writer"
	start := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormFinalWriteStart, Arguments: mustArgs(t, stageStartArgs(writer, draftID, "writer-start"))})
	if start.Error != nil {
		t.Fatalf("writer start failed: %#v", start.Error)
	}
	read := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormFinalWriteRead, Arguments: mustArgs(t, map[string]any{
		"mission_id": writer.MissionID, "session_id": writer.ToolSessionID, "draft_id": draftID,
	})})
	if read.Error != nil {
		t.Fatalf("writer read failed: %#v", read.Error)
	}
	readState := read.Content.(map[string]any)
	if readState["stage"] != reporting.FinalEditStageWriter ||
		!strings.Contains(readState["content"].(string), "Alpha body.") ||
		!strings.Contains(readState["content"].(string), "Beta body.") {
		t.Fatalf("writer read did not expose deterministic assembly: %#v", readState)
	}
	patchArgs := stageCommonArgs(writer)
	patchArgs["idempotency_key"] = "writer-patch"
	patchArgs["draft_id"] = draftID
	patchArgs["operation"] = "replace"
	patchArgs["match_text"] = "Alpha body."
	patchArgs["replacement"] = "Alpha body with final connective context."
	patchArgs["summary"] = "writer-level connective edit"
	if patch := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormFinalWritePatch, Arguments: mustArgs(t, patchArgs)}); patch.Error != nil {
		t.Fatalf("writer patch failed: %#v", patch.Error)
	}
	submitArgs := stageCommonArgs(writer)
	submitArgs["idempotency_key"] = "writer-submit"
	submitArgs["draft_id"] = draftID
	submitArgs["pending_event_id"] = writer.PendingEventID
	submitArgs["plan_event_id"] = writer.PlanEventID
	submit := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormFinalWriteSubmit, Arguments: mustArgs(t, submitArgs)})
	if submit.Error != nil {
		t.Fatalf("writer submit failed: %#v", submit.Error)
	}
	submitState := submit.Content.(map[string]any)
	if submitState["stage"] != reporting.FinalEditStageWriter || submitState["artifact_id"] != writer.EditedArtifactID || submitState["operation_count"] != 1 {
		t.Fatalf("writer submit state mismatch: %#v", submitState)
	}
	if canonicalEventCount(service) != 0 ||
		countEventsOfType(service, reporting.FinalEditAssemblyCreatedEventType) != 1 ||
		countEventsOfType(service, reporting.FinalEditWriterStartedEventType) != 1 ||
		countEventsOfType(service, reporting.FinalEditWriterSubmittedEventType) != 1 {
		t.Fatalf("writer partition emitted unexpected events: %#v", service.ledgerEvents)
	}
	payload := finalEditStageTestPayload(t, lastEventOfType(t, service, reporting.FinalEditWriterSubmittedEventType))
	if payload["final_edit_pipeline"] != reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 ||
		payload["stage"] != reporting.FinalEditStageWriter ||
		payload["source_artifact_id"] != writer.SourceArtifactID ||
		payload["artifact_id"] != writer.EditedArtifactID {
		t.Fatalf("writer submitted payload mismatch: %#v", payload)
	}

	mismatchArgs := stageStartArgs(writer, "rfe_writer_mismatch", "writer-mismatch")
	mismatchArgs["session_id"] = "ses_other"
	mismatchArgs["producer"] = map[string]any{"type": "agent_session", "id": "ses_other"}
	mismatch := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormFinalWriteStart, Arguments: mustArgs(t, mismatchArgs)})
	if mismatch.Error == nil {
		t.Fatalf("writer binding mismatch was not rejected: %#v", mismatch)
	}

	closed := NewServer(service,
		WithBinding(stageMCPBinding(writer)),
		WithFinalEditStageBinding(writer),
		WithLongFormFinalizeBinding(finalBinding),
		WithEnabledTools(writerTools),
	)
	if containsString(toolNames(closed.ListTools()), ToolReportLongFormFinalWriteStart) {
		t.Fatalf("writer server with final binding leaked tools: %#v", toolNames(closed.ListTools()))
	}
	closedResult := closed.Call(context.Background(), ToolCall{Name: ToolReportLongFormFinalWriteStart, Arguments: mustArgs(t, stageStartArgs(writer, "rfe_writer_closed", "writer-closed"))})
	if closedResult.Error == nil || closedResult.Error.ErrorKind != "binding" {
		t.Fatalf("closed writer config call did not fail as binding: %#v", closedResult)
	}
}

func TestReportLongFormReaderAndStyleStagesSubmitWithoutCanonicalizing(t *testing.T) {
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeEnabled)
	reader := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageReader, reporting.FinalEditReaderSourceArtifactID(finalBinding.PlanEventID, finalBinding.PartArtifactIDs), "art_reader_edit", "ses_reader", "provider-reader", "provider-plan", "provider-plan")
	readerResult := runStageEdit(t, service, reader, []string{ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditRead, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit}, ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit, "rfe_reader", "Alpha body.", "Alpha reviewed.", nil)
	if readerResult.Error != nil {
		t.Fatalf("reader submit failed: %#v", readerResult.Error)
	}
	if canonicalEventCount(service) != 0 {
		t.Fatalf("reader edit unexpectedly canonicalized: %#v", service.ledgerEvents)
	}
	if !strings.Contains(string(service.artifacts["art_reader_edit"].Content), "Alpha reviewed.") {
		t.Fatalf("reader edited artifact missing durable patch: %q", service.artifacts["art_reader_edit"].Content)
	}

	style := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageStyle, "art_reader_edit", "art_style_edit", "ses_style", "provider-style", "provider-reader", "provider-reader")
	styleResult := runStageEdit(t, service, style, []string{ToolReportLongFormStyleEditStart, ToolReportLongFormStyleEditRead, ToolReportLongFormStyleEditPatch, ToolReportLongFormStyleEditSubmit}, ToolReportLongFormStyleEditStart, ToolReportLongFormStyleEditPatch, ToolReportLongFormStyleEditSubmit, "rfe_style", "Alpha reviewed.", "Alpha reviewed!", nil)
	if styleResult.Error != nil {
		t.Fatalf("style submit failed: %#v", styleResult.Error)
	}
	if canonicalEventCount(service) != 0 {
		t.Fatalf("style edit unexpectedly canonicalized: %#v", service.ledgerEvents)
	}
	if !strings.Contains(string(service.artifacts["art_style_edit"].Content), "Alpha reviewed!") {
		t.Fatalf("style edited artifact missing durable patch: %q", service.artifacts["art_style_edit"].Content)
	}
}

func TestReportLongFormStyleSubmitFallsBackToSourceArtifactOnStructuralDrift(t *testing.T) {
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeEnabled)
	readerMarkdown := "# Report\n\nAlpha reviewed.\n\n## Sources\n\n[^1]: Evidence 2026-07-28.\n"
	reader := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageReader, reporting.FinalEditReaderSourceArtifactID(finalBinding.PlanEventID, finalBinding.PartArtifactIDs), "art_reader_edit", "ses_reader", "provider-reader", "provider-plan", "provider-plan")
	if result := runStageEdit(t, service, reader, []string{ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit}, ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit, "rfe_fallback_reader", "Alpha body.", readerMarkdown, nil); result.Error != nil {
		t.Fatalf("reader setup failed: %#v", result.Error)
	}

	style := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageStyle, "art_reader_edit", "art_style_fallback", "ses_style", "provider-style", "provider-reader", "provider-reader")
	sourceContentLength := len(service.artifacts["art_reader_edit"].Content)
	styleServer := NewServer(service,
		WithBinding(stageMCPBinding(style)),
		WithFinalEditStageBinding(style),
		WithEnabledTools([]string{ToolReportLongFormStyleEditStart, ToolReportLongFormStyleEditPatch, ToolReportLongFormStyleEditSubmit}),
	)
	if start := styleServer.Call(context.Background(), ToolCall{Name: ToolReportLongFormStyleEditStart, Arguments: mustArgs(t, stageStartArgs(style, "rfe_style_fallback", "style-fallback-start"))}); start.Error != nil {
		t.Fatalf("style start failed: %#v", start.Error)
	}
	patchArgs := stageCommonArgs(style)
	patchArgs["idempotency_key"] = "style-fallback-patch"
	patchArgs["draft_id"] = "rfe_style_fallback"
	patchArgs["operation"] = "replace"
	patchArgs["match_text"] = "# Report"
	patchArgs["replacement"] = "# Changed"
	patchArgs["summary"] = "unsafe structural drift"
	if patch := styleServer.Call(context.Background(), ToolCall{Name: ToolReportLongFormStyleEditPatch, Arguments: mustArgs(t, patchArgs)}); patch.Error != nil {
		t.Fatalf("style patch failed: %#v", patch.Error)
	}
	submitArgs := stageCommonArgs(style)
	submitArgs["idempotency_key"] = "style-fallback-submit"
	submitArgs["draft_id"] = "rfe_style_fallback"
	submitArgs["pending_event_id"] = style.PendingEventID
	submitArgs["plan_event_id"] = style.PlanEventID
	styleResult := styleServer.Call(context.Background(), ToolCall{Name: ToolReportLongFormStyleEditSubmit, Arguments: mustArgs(t, submitArgs)})
	if styleResult.Error != nil {
		t.Fatalf("style fallback submit failed: %#v", styleResult.Error)
	}
	styleState := styleResult.Content.(map[string]any)
	if styleState["artifact_id"] != "art_reader_edit" || styleState["operation_count"] != 0 || styleState["content_length"] != sourceContentLength {
		t.Fatalf("style fallback did not persist as no-op source artifact: %#v", styleState)
	}
	replayResult := styleServer.Call(context.Background(), ToolCall{Name: ToolReportLongFormStyleEditSubmit, Arguments: mustArgs(t, submitArgs)})
	if replayResult.Error != nil {
		t.Fatalf("style fallback replay submit failed: %#v", replayResult.Error)
	}
	replayState := replayResult.Content.(map[string]any)
	for _, key := range []string{"artifact_id", "event_id", "operation_count", "content_length"} {
		if replayState[key] != styleState[key] {
			t.Fatalf("style fallback replay state differs for %s: first=%#v replay=%#v", key, styleState, replayState)
		}
	}
	if _, ok := service.artifacts["art_style_fallback"]; ok {
		t.Fatal("style fallback created changed artifact instead of reusing source")
	}
	if countEventsOfType(service, reporting.FinalEditStyleSubmittedEventType) != 1 || countEventsOfType(service, "report.final.failed") != 0 || countEventsOfType(service, "report.draft.failed") != 0 {
		t.Fatalf("style fallback emitted unexpected events: %#v", service.ledgerEvents)
	}
	payload := finalEditStageTestPayload(t, lastEventOfType(t, service, reporting.FinalEditStyleSubmittedEventType))
	if payload["artifact_id"] != "art_reader_edit" || payload["edited_artifact_id"] != "art_style_fallback" || payload["changed"] != false || payload["operation_count"] != float64(0) {
		t.Fatalf("style fallback payload mismatch: %#v", payload)
	}

	finalForGate := finalBinding
	finalForGate.ToolSessionID = "ses_gate"
	finalForGate.ProviderSessionID = "provider-gate"
	finalForGate.PreviousProviderSessionID = "provider-style"
	finalForGate.Producer = app.Producer{Type: "agent_session", ID: "provider-gate"}
	gate := testFinalEditStageBinding(finalForGate, reporting.FinalEditStageGate, "art_reader_edit", finalForGate.ArtifactID, finalForGate.ToolSessionID, finalForGate.ProviderSessionID, finalForGate.PreviousProviderSessionID, finalForGate.ForkSourceAgentSessionID)
	gateServer := NewServer(service,
		WithBinding(stageMCPBinding(gate)),
		WithFinalEditStageBinding(gate),
		WithLongFormFinalizeBinding(finalForGate),
		WithEnabledTools([]string{ToolReportLongFormEditStart, ToolReportLongFormEditSubmit}),
	)
	gateResult := submitStageWithoutPatch(t, gateServer, gate, ToolReportLongFormEditStart, ToolReportLongFormEditSubmit, "rfe_gate_after_fallback", "gate-after-fallback", []map[string]any{{
		"statement": "Alpha reviewed.", "classification": reporting.FinalEditGateClassMissionSourceGrounded,
	}})
	if gateResult.Error != nil {
		t.Fatalf("gate submit after style fallback failed: %#v", gateResult.Error)
	}
}

func TestReportLongFormStyleSubmitSourceLoadFailureIsHardError(t *testing.T) {
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeEnabled)
	reader := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageReader, reporting.FinalEditReaderSourceArtifactID(finalBinding.PlanEventID, finalBinding.PartArtifactIDs), "art_reader_edit", "ses_reader", "provider-reader", "provider-plan", "provider-plan")
	if result := runStageEdit(t, service, reader, []string{ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit}, ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit, "rfe_source_reader", "Alpha body.", "# Report\n\nAlpha reviewed.\n", nil); result.Error != nil {
		t.Fatalf("reader setup failed: %#v", result.Error)
	}

	style := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageStyle, "art_reader_edit", "art_style_missing_source", "ses_style", "provider-style", "provider-reader", "provider-reader")
	server := NewServer(service,
		WithBinding(stageMCPBinding(style)),
		WithFinalEditStageBinding(style),
		WithEnabledTools([]string{ToolReportLongFormStyleEditStart, ToolReportLongFormStyleEditPatch, ToolReportLongFormStyleEditSubmit}),
	)
	if start := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormStyleEditStart, Arguments: mustArgs(t, stageStartArgs(style, "rfe_missing_source", "missing-source-start"))}); start.Error != nil {
		t.Fatalf("style start failed: %#v", start.Error)
	}
	patchArgs := stageCommonArgs(style)
	patchArgs["idempotency_key"] = "missing-source-patch"
	patchArgs["draft_id"] = "rfe_missing_source"
	patchArgs["operation"] = "replace"
	patchArgs["match_text"] = "Alpha reviewed."
	patchArgs["replacement"] = "Alpha styled."
	patchArgs["summary"] = "style edit"
	if patch := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormStyleEditPatch, Arguments: mustArgs(t, patchArgs)}); patch.Error != nil {
		t.Fatalf("style patch failed: %#v", patch.Error)
	}
	delete(service.artifacts, "art_reader_edit")
	submitArgs := stageCommonArgs(style)
	submitArgs["idempotency_key"] = "missing-source-submit"
	submitArgs["draft_id"] = "rfe_missing_source"
	submitArgs["pending_event_id"] = style.PendingEventID
	submitArgs["plan_event_id"] = style.PlanEventID
	result := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormStyleEditSubmit, Arguments: mustArgs(t, submitArgs)})
	if result.Error == nil {
		t.Fatal("style submit succeeded despite missing source artifact")
	}
	if countEventsOfType(service, reporting.FinalEditStyleSubmittedEventType) != 0 {
		t.Fatalf("missing source unexpectedly submitted style event: %#v", service.ledgerEvents)
	}
	server.mu.Lock()
	finalizing := server.longFormStageEditDrafts["rfe_missing_source"].Finalizing
	server.mu.Unlock()
	if finalizing {
		t.Fatal("style draft remained finalizing after source load failure")
	}
}

func TestReportLongFormNoOpStyleAndGateSubmitUseDurableSourceArtifacts(t *testing.T) {
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeEnabled)
	reader := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageReader, reporting.FinalEditReaderSourceArtifactID(finalBinding.PlanEventID, finalBinding.PartArtifactIDs), "art_reader_edit", "ses_reader", "provider-reader", "provider-plan", "provider-plan")
	if result := runStageEdit(t, service, reader, []string{ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit}, ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit, "rfe_noop_reader", "Alpha body.", "Alpha reviewed.", nil); result.Error != nil {
		t.Fatalf("reader setup failed: %#v", result.Error)
	}

	style := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageStyle, "art_reader_edit", "art_style_noop", "ses_style", "provider-style", "provider-reader", "provider-reader")
	styleServer := NewServer(service,
		WithBinding(stageMCPBinding(style)),
		WithFinalEditStageBinding(style),
		WithEnabledTools([]string{ToolReportLongFormStyleEditStart, ToolReportLongFormStyleEditSubmit}),
	)
	styleResult := submitStageWithoutPatch(t, styleServer, style, ToolReportLongFormStyleEditStart, ToolReportLongFormStyleEditSubmit, "rfe_style_noop", "style-noop", nil)
	if styleResult.Error != nil {
		t.Fatalf("style no-op submit failed: %#v", styleResult.Error)
	}
	styleState := styleResult.Content.(map[string]any)
	if styleState["artifact_id"] != "art_reader_edit" || styleState["operation_count"] != 0 {
		t.Fatalf("style no-op did not adopt reader source artifact: %#v", styleState)
	}
	if _, ok := service.artifacts["art_style_noop"]; ok {
		t.Fatal("style no-op created an alias artifact for the unchanged manuscript")
	}
	if canonicalEventCount(service) != 0 {
		t.Fatalf("style no-op unexpectedly canonicalized: %#v", service.ledgerEvents)
	}

	finalForGate := finalBinding
	finalForGate.ToolSessionID = "ses_gate"
	finalForGate.ProviderSessionID = "provider-gate"
	finalForGate.PreviousProviderSessionID = "provider-style"
	finalForGate.Producer = app.Producer{Type: "agent_session", ID: "provider-gate"}
	gate := testFinalEditStageBinding(finalForGate, reporting.FinalEditStageGate, "art_reader_edit", finalForGate.ArtifactID, finalForGate.ToolSessionID, finalForGate.ProviderSessionID, finalForGate.PreviousProviderSessionID, finalForGate.ForkSourceAgentSessionID)
	gateServer := NewServer(service,
		WithBinding(stageMCPBinding(gate)),
		WithFinalEditStageBinding(gate),
		WithLongFormFinalizeBinding(finalForGate),
		WithEnabledTools([]string{ToolReportLongFormEditStart, ToolReportLongFormEditSubmit}),
	)
	gateResult := submitStageWithoutPatch(t, gateServer, gate, ToolReportLongFormEditStart, ToolReportLongFormEditSubmit, "rfe_gate_noop", "gate-noop", []map[string]any{{
		"statement": "Alpha reviewed.", "classification": reporting.FinalEditGateClassMissionSourceGrounded,
	}})
	if gateResult.Error != nil {
		t.Fatalf("gate no-op submit failed: %#v", gateResult.Error)
	}
	gateState := gateResult.Content.(map[string]any)
	if gateState["artifact_id"] != "art_reader_edit" || gateState["operation_count"] != 0 {
		t.Fatalf("gate no-op did not adopt prior stage artifact: %#v", gateState)
	}
	if _, ok := service.artifacts[finalForGate.ArtifactID]; ok {
		t.Fatal("gate no-op created a planned-final alias artifact")
	}
	if canonicalEventCount(service) != 1 || countEventsOfType(service, reporting.FinalEditGateSubmittedEventType) != 1 {
		t.Fatalf("gate no-op did not canonicalize exactly once: events=%#v", service.ledgerEvents)
	}
	payload := finalEditStageTestPayload(t, onlyCanonicalEvent(t, service))
	if payload["artifact_id"] != "art_reader_edit" ||
		payload["planned_final_artifact_id"] != finalForGate.ArtifactID ||
		payload["final_edit_gate_changed"] != false ||
		payload["final_edit_gate_event_id"] == "" ||
		payload["final_edit_pipeline"] != reporting.FinalEditPipelineReaderStyleGateV1 {
		t.Fatalf("gate no-op canonical payload mismatch: %#v", payload)
	}
}

func TestReportLongFormGateStageRequiresFindingsAndDoesNotPersistRawStatement(t *testing.T) {
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeDisabled)
	reader := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageReader, reporting.FinalEditReaderSourceArtifactID(finalBinding.PlanEventID, finalBinding.PartArtifactIDs), "art_reader_edit", "ses_reader", "provider-reader", "provider-plan", "provider-plan")
	if result := runStageEdit(t, service, reader, []string{ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit}, ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit, "rfe_reader", "Alpha body.", "Alpha reviewed.", nil); result.Error != nil {
		t.Fatalf("reader setup failed: %#v", result.Error)
	}

	finalBinding.ToolSessionID = "ses_gate"
	finalBinding.ProviderSessionID = "provider-gate"
	finalBinding.PreviousProviderSessionID = "provider-reader"
	finalBinding.Producer = app.Producer{Type: "agent_session", ID: "provider-gate"}
	gate := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageGate, "art_reader_edit", finalBinding.ArtifactID, "ses_gate", "provider-gate", "provider-reader", "provider-plan")
	service.evidence = []app.EvidenceRecord{{EvidenceID: "evd_gate", MissionID: finalBinding.MissionID, State: "approved"}}
	statement := "This external fact must be checked."
	findings := []map[string]any{{
		"statement": statement, "classification": reporting.FinalEditGateClassUnverifiedExternalFact,
		"repair_action": reporting.FinalEditRepairAttachApprovedEvidence, "evidence_ids": []string{"evd_gate"},
	}}
	result := runStageEdit(t, service, gate, []string{ToolReportLongFormEditStart, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit}, ToolReportLongFormEditStart, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit, "rfe_gate", "Alpha reviewed.", "Alpha corrected.", findings)
	if result.Error != nil || len(result.CreatedEventIDs) != 1 {
		t.Fatalf("gate submit failed: %#v", result)
	}
	if canonicalEventCount(service) != 1 {
		t.Fatalf("gate did not canonicalize exactly once: %#v", service.ledgerEvents)
	}
	for _, event := range service.ledgerEvents {
		if strings.Contains(string(event.Payload), statement) {
			t.Fatalf("raw statement leaked into event payload for %s: %s", event.EventType, event.Payload)
		}
	}
	if strings.Contains(mustMarshalString(t, result), statement) {
		t.Fatalf("raw statement leaked into tool result: %#v", result)
	}
}

func TestReportLongFormStageEditSchemasAreClosed(t *testing.T) {
	for name, schema := range map[string]json.RawMessage{
		ToolReportLongFormFinalWriteStart:  schemaReportLongFormStageEditStart,
		ToolReportLongFormFinalWriteRead:   schemaReportLongFormStageEditRead,
		ToolReportLongFormFinalWritePatch:  schemaReportLongFormStageEditPatch,
		ToolReportLongFormFinalWriteSubmit: schemaReportLongFormStageEditSubmit,
		ToolReportLongFormReaderEditStart:  schemaReportLongFormStageEditStart,
		ToolReportLongFormReaderEditRead:   schemaReportLongFormStageEditRead,
		ToolReportLongFormReaderEditPatch:  schemaReportLongFormStageEditPatch,
		ToolReportLongFormReaderEditSubmit: schemaReportLongFormStageEditSubmit,
		ToolReportLongFormEditSubmit:       schemaReportLongFormGateEditSubmit,
	} {
		var value struct {
			AdditionalProperties bool                       `json:"additionalProperties"`
			Properties           map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(schema, &value); err != nil || value.AdditionalProperties {
			t.Fatalf("%s schema is not closed: value=%#v err=%v", name, value, err)
		}
		if _, ok := value.Properties["canonical_event_id"]; ok {
			t.Fatalf("%s exposes canonical_event_id input", name)
		}
	}
}

func runStageEdit(t *testing.T, service *fakeMCPService, binding reporting.FinalEditStageBinding, tools []string, startTool, patchTool, submitTool, draftID, match, replacement string, gateFindings []map[string]any) ToolResult {
	t.Helper()
	server := NewServer(service, WithBinding(stageMCPBinding(binding)), WithFinalEditStageBinding(binding), WithLongFormFinalizeBinding(gateFinalBindingForStage(binding)), WithEnabledTools(tools))
	if binding.Stage != reporting.FinalEditStageGate {
		server = NewServer(service, WithBinding(stageMCPBinding(binding)), WithFinalEditStageBinding(binding), WithEnabledTools(tools))
	}
	start := server.Call(context.Background(), ToolCall{Name: startTool, Arguments: mustArgs(t, stageStartArgs(binding, draftID, draftID+"-start"))})
	if start.Error != nil {
		t.Fatalf("%s failed: %#v", startTool, start.Error)
	}
	patchArgs := stageCommonArgs(binding)
	patchArgs["idempotency_key"] = draftID + "-patch"
	patchArgs["draft_id"] = draftID
	patchArgs["operation"] = "replace"
	patchArgs["match_text"] = match
	patchArgs["replacement"] = replacement
	patchArgs["summary"] = "focused stage edit"
	if patch := server.Call(context.Background(), ToolCall{Name: patchTool, Arguments: mustArgs(t, patchArgs)}); patch.Error != nil {
		t.Fatalf("%s failed: %#v", patchTool, patch.Error)
	}
	submitArgs := stageCommonArgs(binding)
	submitArgs["idempotency_key"] = draftID + "-submit"
	submitArgs["draft_id"] = draftID
	submitArgs["pending_event_id"] = binding.PendingEventID
	submitArgs["plan_event_id"] = binding.PlanEventID
	if gateFindings != nil {
		submitArgs["gate_findings"] = gateFindings
	}
	return server.Call(context.Background(), ToolCall{Name: submitTool, Arguments: mustArgs(t, submitArgs)})
}

func submitStageWithoutPatch(t *testing.T, server *Server, binding reporting.FinalEditStageBinding, startTool, submitTool, draftID string, key string, gateFindings []map[string]any) ToolResult {
	t.Helper()
	if start := server.Call(context.Background(), ToolCall{Name: startTool, Arguments: mustArgs(t, stageStartArgs(binding, draftID, key+"-start"))}); start.Error != nil {
		t.Fatalf("%s failed: %#v", startTool, start.Error)
	}
	submitArgs := stageCommonArgs(binding)
	submitArgs["idempotency_key"] = key + "-submit"
	submitArgs["draft_id"] = draftID
	submitArgs["pending_event_id"] = binding.PendingEventID
	submitArgs["plan_event_id"] = binding.PlanEventID
	if gateFindings != nil {
		submitArgs["gate_findings"] = gateFindings
	}
	return server.Call(context.Background(), ToolCall{Name: submitTool, Arguments: mustArgs(t, submitArgs)})
}

func gateFinalBindingForStage(binding reporting.FinalEditStageBinding) reporting.LongFormFinalizeBinding {
	if binding.Stage != reporting.FinalEditStageGate {
		return reporting.LongFormFinalizeBinding{}
	}
	final := testFinalEditBaseBinding(reporting.FinalEditHumanizeDisabled)
	final.ToolSessionID = binding.ToolSessionID
	final.ProviderSessionID = binding.ProviderSessionID
	final.PreviousProviderSessionID = binding.PreviousProviderSessionID
	final.Producer = binding.Producer
	return final
}

func seededFinalEditStageService(t *testing.T, humanize string) (*fakeMCPService, reporting.LongFormFinalizeBinding) {
	t.Helper()
	final := testFinalEditBaseBinding(humanize)
	producer := app.Producer{Type: "agent_session", ID: "provider-plan"}
	parts := []app.RawArtifact{
		testRawArtifact("art_part_1", final.MissionID, final.Filename, "# Part 1\n\nAlpha body.\n", producer),
		testRawArtifact("art_part_2", final.MissionID, final.Filename, "# Part 2\n\nBeta body.\n", producer),
	}
	sections := []app.RawArtifact{
		testRawArtifact("art_section_1", final.MissionID, final.Filename, "# Section 1\n\nAlpha source.\n", producer),
		testRawArtifact("art_section_2", final.MissionID, final.Filename, "# Section 2\n\nBeta source.\n", producer),
	}
	artifacts := map[string]app.RawArtifact{}
	for _, artifact := range append(parts, sections...) {
		artifacts[artifact.ArtifactID] = artifact
	}
	return &fakeMCPService{
		artifacts: artifacts,
		ledgerEvents: []app.LedgerEvent{
			{EventID: final.PendingEventID, MissionID: final.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"report_mode": reporting.ModeLongForm})},
			{EventID: final.PlanEventID, MissionID: final.MissionID, EventType: "report.plan.created", Producer: producer, Payload: mustJSON(map[string]any{
				"pending_event_id": final.PendingEventID, "report_mode": reporting.ModeLongForm, "artifact_id": final.ArtifactID,
				"final_edit_pipeline": reporting.FinalEditPipelineReaderStyleGateV1, "post_report_humanize": humanize,
				"plan": map[string]any{"parts": []map[string]any{
					{"title": "Part 1", "sections": []map[string]any{{"title": "Section 1"}}},
					{"title": "Part 2", "sections": []map[string]any{{"title": "Section 2"}}},
				}},
			})},
			{EventID: "evt_part_1", MissionID: final.MissionID, EventType: "report.part.created", Producer: producer, Payload: mustJSON(map[string]any{"pending_event_id": final.PendingEventID, "plan_event_id": final.PlanEventID, "artifact_id": "art_part_1", "part_index": 1})},
			{EventID: "evt_section_1", MissionID: final.MissionID, EventType: "report.section.created", Producer: producer, Payload: mustJSON(map[string]any{"pending_event_id": final.PendingEventID, "plan_event_id": final.PlanEventID, "artifact_id": "art_section_1", "part_index": 1, "section_index": 1})},
			{EventID: "evt_part_2", MissionID: final.MissionID, EventType: "report.part.created", Producer: producer, Payload: mustJSON(map[string]any{"pending_event_id": final.PendingEventID, "plan_event_id": final.PlanEventID, "artifact_id": "art_part_2", "part_index": 2})},
			{EventID: "evt_section_2", MissionID: final.MissionID, EventType: "report.section.created", Producer: producer, Payload: mustJSON(map[string]any{"pending_event_id": final.PendingEventID, "plan_event_id": final.PlanEventID, "artifact_id": "art_section_2", "part_index": 2, "section_index": 1})},
		},
	}, final
}

func seededFinalEditWriterStageService(t *testing.T) (*fakeMCPService, reporting.LongFormFinalizeBinding, reporting.FinalEditStageBinding) {
	t.Helper()
	service, final := seededFinalEditStageService(t, reporting.FinalEditHumanizeDisabled)
	for index := range service.ledgerEvents {
		if service.ledgerEvents[index].EventID != final.PlanEventID {
			continue
		}
		payload := finalEditStageTestPayload(t, service.ledgerEvents[index])
		payload["final_edit_pipeline"] = reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2
		payload["post_report_humanize"] = reporting.FinalEditHumanizeDisabled
		service.ledgerEvents[index].Payload = mustJSON(payload)
	}
	assemblyID := reporting.FinalEditAssemblyArtifactID(final.PlanEventID, final.PartArtifactIDs)
	writer := testFinalEditStageBinding(final, reporting.FinalEditStageWriter, assemblyID, "art_writer_edit", "ses_writer", "provider-writer", final.ReportPlanSessionID, final.ReportPlanSessionID)
	writer.FinalEditPipeline = reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2
	return service, final, writer
}

func testFinalEditBaseBinding(humanize string) reporting.LongFormFinalizeBinding {
	return reporting.LongFormFinalizeBinding{
		MissionID: "mis_stage", PendingEventID: "evt_pending_stage", PlanEventID: "evt_plan_stage", ArtifactID: "art_final", Filename: "report.md", Title: "Report",
		ToolSessionID: "ses_gate", IdempotencyKey: "final-key", ProviderSessionID: "provider-gate", PreviousProviderSessionID: "provider-reader",
		PartArtifactIDs: []string{"art_part_1", "art_part_2"}, SectionArtifactIDs: []string{"art_section_1", "art_section_2"}, SectionWordCount: 4,
		CompositionStrategy: reporting.LongFormCompositionNarrativeEdit, AgentExecutor: "codex", AgentModel: "gpt-5", AgentReasoningEffort: "medium",
		AgentSelectionSource: "system", MCPMode: "locked", RigorLevel: "standard", RigorLabel: "Standard",
		ReportSessionPolicy: "reuse", ReportSessionPolicySelection: "auto", PostReportHumanize: humanize,
		GenerationGuidanceProfile: "reader-style-gate", GenerationGuidanceSHA256: strings.Repeat("a", 64), SessionChainKind: "report_final_edit",
		PreReportResearchSessionID: "provider-research", ReportPlanSessionID: "provider-plan", ForkSourceAgentSessionID: "provider-plan",
		Producer: app.Producer{Type: "agent_session", ID: "provider-gate"},
	}
}

func testFinalEditStageBinding(final reporting.LongFormFinalizeBinding, stage, sourceID, editedID, toolSessionID, providerSessionID, previousProviderSessionID, forkSource string) reporting.FinalEditStageBinding {
	return reporting.FinalEditStageBinding{
		MissionID: final.MissionID, PendingEventID: final.PendingEventID, PlanEventID: final.PlanEventID, Title: final.Title, Stage: stage,
		SourceArtifactID: sourceID, EditedArtifactID: editedID, Filename: final.Filename, ToolSessionID: toolSessionID,
		ProviderSessionID: providerSessionID, PreviousProviderSessionID: previousProviderSessionID,
		IdempotencyKey: reporting.FinalEditStageIdempotencyKey(stage, final.PendingEventID, final.PlanEventID),
		AgentExecutor:  final.AgentExecutor, AgentModel: final.AgentModel, AgentReasoningEffort: final.AgentReasoningEffort, AgentSelectionSource: final.AgentSelectionSource,
		MCPMode: final.MCPMode, RigorLevel: final.RigorLevel, RigorLabel: final.RigorLabel, ReportSessionPolicy: final.ReportSessionPolicy,
		ReportSessionPolicySelection: final.ReportSessionPolicySelection, PostReportHumanize: final.PostReportHumanize,
		GenerationGuidanceProfile: final.GenerationGuidanceProfile, GenerationGuidanceSHA256: final.GenerationGuidanceSHA256,
		SessionChainKind: final.SessionChainKind, PreReportResearchSessionID: final.PreReportResearchSessionID,
		ReportPlanSessionID: final.ReportPlanSessionID, ForkSourceAgentSessionID: forkSource,
		Producer: app.Producer{Type: "agent_session", ID: providerSessionID},
	}
}

func stageMCPBinding(binding reporting.FinalEditStageBinding) Binding {
	return Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}
}

func stageCommonArgs(binding reporting.FinalEditStageBinding) map[string]any {
	return map[string]any{
		"mission_id": binding.MissionID, "session_id": binding.ToolSessionID,
		"producer": map[string]any{"type": "agent_session", "id": binding.ToolSessionID},
	}
}

func stageStartArgs(binding reporting.FinalEditStageBinding, draftID string, key string) map[string]any {
	args := stageCommonArgs(binding)
	args["idempotency_key"] = key
	args["draft_id"] = draftID
	args["pending_event_id"] = binding.PendingEventID
	args["plan_event_id"] = binding.PlanEventID
	return args
}

func testRawArtifact(artifactID string, missionID string, filename string, content string, producer app.Producer) app.RawArtifact {
	sum := sha256.Sum256([]byte(content))
	return app.RawArtifact{
		ArtifactID: artifactID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8",
		Filename: filename, Producer: producer, Content: []byte(content), SHA256: hex.EncodeToString(sum[:]),
	}
}

func canonicalEventCount(service *fakeMCPService) int {
	count := 0
	for _, event := range service.ledgerEvents {
		if event.EventType == "report.artifact.created" {
			count++
		}
	}
	return count
}

func onlyCanonicalEvent(t *testing.T, service *fakeMCPService) app.LedgerEvent {
	t.Helper()
	var found app.LedgerEvent
	count := 0
	for _, event := range service.ledgerEvents {
		if event.EventType == "report.artifact.created" {
			found = event
			count++
		}
	}
	if count != 1 {
		t.Fatalf("canonical event count=%d, want 1", count)
	}
	return found
}

func finalEditStageTestPayload(t *testing.T, event app.LedgerEvent) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func lastEventOfType(t *testing.T, service *fakeMCPService, eventType string) app.LedgerEvent {
	t.Helper()
	for i := len(service.ledgerEvents) - 1; i >= 0; i-- {
		if service.ledgerEvents[i].EventType == eventType {
			return service.ledgerEvents[i]
		}
	}
	t.Fatalf("event type %s not found", eventType)
	return app.LedgerEvent{}
}

func mustMarshalString(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}
