package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestReportLongFormGateReplayRecoversCanonicalWithDurableOperationCount(t *testing.T) {
	service, gate, findings, statement := prepareInterruptedGateSubmission(t)
	server := NewServer(service, WithBinding(stageMCPBinding(gate)), WithFinalEditStageBinding(gate), WithLongFormFinalizeBinding(gateFinalBindingForStage(gate)), WithEnabledTools([]string{ToolReportLongFormEditStart, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit}))
	start := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditStart, Arguments: mustArgs(t, stageStartArgs(gate, "rfe_recover", "recover-start"))})
	if start.Error != nil {
		t.Fatalf("recovery start failed: %#v", start.Error)
	}
	state := start.Content.(map[string]any)
	if state["submitted"] != true || state["operation_count"] != 1 || state["artifact_id"] != gate.EditedArtifactID {
		t.Fatalf("durable gate state was not restored: %#v", state)
	}
	stageEventID := state["event_id"]
	patchArgs := stageCommonArgs(gate)
	patchArgs["idempotency_key"] = "recover-patch"
	patchArgs["draft_id"] = "rfe_recover"
	patchArgs["operation"] = "append"
	patchArgs["replacement"] = "\nblocked"
	patch := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditPatch, Arguments: mustArgs(t, patchArgs)})
	if patch.Error == nil || patch.Error.ErrorKind != "conflict" {
		t.Fatalf("durable gate patch was not forbidden: %#v", patch)
	}
	submit := submitGateDraft(t, server, gate, "rfe_recover", "recover-submit", findings)
	if submit.Error != nil || canonicalEventCount(service) != 1 {
		t.Fatalf("canonical recovery failed: result=%#v canonical=%d", submit, canonicalEventCount(service))
	}
	submittedState := submit.Content.(map[string]any)
	if len(submit.CreatedEventIDs) != 1 || submittedState["event_id"] != submit.CreatedEventIDs[0] || submittedState["event_id"] == stageEventID {
		t.Fatalf("canonical recovery returned stale stage identity: result=%#v stage_event_id=%v", submit, stageEventID)
	}
	repeat := submitGateDraft(t, server, gate, "rfe_recover", "recover-submit-repeat", findings)
	if repeat.Error != nil || canonicalEventCount(service) != 1 {
		t.Fatalf("identical gate replay did not stay exactly once: result=%#v canonical=%d", repeat, canonicalEventCount(service))
	}
	for _, event := range service.ledgerEvents {
		if strings.Contains(string(event.Payload), statement) {
			t.Fatalf("raw statement leaked into event payload for %s", event.EventType)
		}
	}
}

func TestReportLongFormGateReplayRejectsChangedFindingsWithoutCanonical(t *testing.T) {
	service, gate, _, _ := prepareInterruptedGateSubmission(t)
	server := NewServer(service, WithBinding(stageMCPBinding(gate)), WithFinalEditStageBinding(gate), WithLongFormFinalizeBinding(gateFinalBindingForStage(gate)), WithEnabledTools([]string{ToolReportLongFormEditStart, ToolReportLongFormEditSubmit}))
	if start := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditStart, Arguments: mustArgs(t, stageStartArgs(gate, "rfe_changed_findings", "changed-start"))}); start.Error != nil {
		t.Fatalf("recovery start failed: %#v", start.Error)
	}
	changed := []map[string]any{{
		"statement": "A different statement.", "classification": reporting.FinalEditGateClassUnverifiedExternalFact,
		"repair_action": reporting.FinalEditRepairAttachApprovedEvidence, "evidence_ids": []string{"evd_gate"},
	}}
	result := submitGateDraft(t, server, gate, "rfe_changed_findings", "changed-submit", changed)
	if result.Error == nil || canonicalEventCount(service) != 0 {
		t.Fatalf("changed findings were accepted: result=%#v canonical=%d", result, canonicalEventCount(service))
	}
}

func TestReportLongFormGateReplayRejectsRevokedEvidenceWithoutCanonical(t *testing.T) {
	service, gate, findings, _ := prepareInterruptedGateSubmission(t)
	server := NewServer(service, WithBinding(stageMCPBinding(gate)), WithFinalEditStageBinding(gate), WithLongFormFinalizeBinding(gateFinalBindingForStage(gate)), WithEnabledTools([]string{ToolReportLongFormEditStart, ToolReportLongFormEditSubmit}))
	if start := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditStart, Arguments: mustArgs(t, stageStartArgs(gate, "rfe_revoked", "revoked-start"))}); start.Error != nil {
		t.Fatalf("recovery start failed: %#v", start.Error)
	}
	service.evidence = []app.EvidenceRecord{{EvidenceID: "evd_gate", MissionID: gate.MissionID, State: "rejected"}}
	result := submitGateDraft(t, server, gate, "rfe_revoked", "revoked-submit", findings)
	if result.Error == nil || canonicalEventCount(service) != 0 {
		t.Fatalf("revoked evidence was accepted: result=%#v canonical=%d", result, canonicalEventCount(service))
	}
}

func TestReportLongFormGateIncompatibleBindingsExposeNoToolsAndWriteNoStart(t *testing.T) {
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeDisabled)
	gate := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageGate, "art_reader_edit", finalBinding.ArtifactID, finalBinding.ToolSessionID, finalBinding.ProviderSessionID, finalBinding.PreviousProviderSessionID, finalBinding.ForkSourceAgentSessionID)
	gate.AgentModel = "other-model"
	server := NewServer(service, WithBinding(stageMCPBinding(gate)), WithFinalEditStageBinding(gate), WithLongFormFinalizeBinding(finalBinding), WithEnabledTools([]string{ToolReportLongFormEditStart}))
	if containsString(toolNames(server.ListTools()), ToolReportLongFormEditStart) {
		t.Fatal("incompatible gate/final bindings exposed final_edit tools")
	}
	before := countEventsOfType(service, reporting.FinalEditGateStartedEventType)
	result := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditStart, Arguments: mustArgs(t, stageStartArgs(gate, "rfe_bad_binding", "bad-binding-start"))})
	if result.Error == nil || result.Error.ErrorKind != "binding" {
		t.Fatalf("incompatible gate start did not fail as binding: %#v", result)
	}
	if countEventsOfType(service, reporting.FinalEditGateStartedEventType) != before {
		t.Fatal("incompatible binding wrote a gate start event")
	}
}

func prepareInterruptedGateSubmission(t *testing.T) (*fakeMCPService, reporting.FinalEditStageBinding, []map[string]any, string) {
	t.Helper()
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeDisabled)
	reader := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageReader, reporting.FinalEditReaderSourceArtifactID(finalBinding.PlanEventID, finalBinding.PartArtifactIDs), "art_reader_edit", "ses_reader", "provider-reader", "provider-plan", "provider-plan")
	if result := runStageEdit(t, service, reader, []string{ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit}, ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit, "rfe_reader_setup", "Alpha body.", "Alpha reviewed.", nil); result.Error != nil {
		t.Fatalf("reader setup failed: %#v", result.Error)
	}
	gate := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageGate, "art_reader_edit", finalBinding.ArtifactID, finalBinding.ToolSessionID, finalBinding.ProviderSessionID, finalBinding.PreviousProviderSessionID, finalBinding.ForkSourceAgentSessionID)
	service.evidence = []app.EvidenceRecord{{EvidenceID: "evd_gate", MissionID: gate.MissionID, State: "approved"}}
	statement := "This external fact must be checked."
	findings := []map[string]any{{
		"statement": statement, "classification": reporting.FinalEditGateClassUnverifiedExternalFact,
		"repair_action": reporting.FinalEditRepairAttachApprovedEvidence, "evidence_ids": []string{"evd_gate"},
	}}
	result := runStageEdit(t, service, gate, []string{ToolReportLongFormEditStart, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit}, ToolReportLongFormEditStart, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit, "rfe_gate_setup", "Alpha reviewed.", "Alpha corrected.", findings)
	if result.Error != nil || canonicalEventCount(service) != 1 {
		t.Fatalf("gate setup failed: result=%#v canonical=%d", result, canonicalEventCount(service))
	}
	service.ledgerEvents = withoutEventsOfType(service.ledgerEvents, "report.artifact.created")
	return service, gate, findings, statement
}

func submitGateDraft(t *testing.T, server *Server, gate reporting.FinalEditStageBinding, draftID string, key string, findings []map[string]any) ToolResult {
	t.Helper()
	args := stageCommonArgs(gate)
	args["idempotency_key"] = key
	args["draft_id"] = draftID
	args["pending_event_id"] = gate.PendingEventID
	args["plan_event_id"] = gate.PlanEventID
	args["gate_findings"] = findings
	args["semantic_acceptance"] = []map[string]any{}
	return server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditSubmit, Arguments: mustArgs(t, args)})
}

func withoutEventsOfType(events []app.LedgerEvent, eventType string) []app.LedgerEvent {
	out := events[:0]
	for _, event := range events {
		if event.EventType != eventType {
			out = append(out, event)
		}
	}
	return out
}

func countEventsOfType(service *fakeMCPService, eventType string) int {
	count := 0
	for _, event := range service.ledgerEvents {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}
