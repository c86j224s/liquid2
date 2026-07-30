package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestReportLongFormGateRejectsMissingAndRawFindingsWithoutTraceLeak(t *testing.T) {
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeDisabled)
	reader := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageReader, reporting.FinalEditReaderSourceArtifactID(finalBinding.PlanEventID, finalBinding.PartArtifactIDs), "art_reader_edit", "ses_reader", "provider-reader", "provider-plan", "provider-plan")
	if result := runStageEdit(t, service, reader, []string{ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit}, ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit, "rfe_reader_privacy", "Alpha body.", "Alpha privacy reviewed.", nil); result.Error != nil {
		t.Fatalf("reader setup failed: %#v", result.Error)
	}

	finalBinding.ToolSessionID = "ses_gate"
	finalBinding.ProviderSessionID = "provider-gate"
	finalBinding.PreviousProviderSessionID = "provider-reader"
	finalBinding.Producer = app.Producer{Type: "agent_session", ID: "provider-gate"}
	gate := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageGate, "art_reader_edit", finalBinding.ArtifactID, "ses_gate", "provider-gate", "provider-reader", "provider-plan")
	server := NewServer(service,
		WithBinding(stageMCPBinding(gate)),
		WithFinalEditStageBinding(gate),
		WithLongFormFinalizeBinding(finalBinding),
		WithEnabledTools([]string{ToolReportLongFormEditStart, ToolReportLongFormEditSubmit}),
	)
	draftID := "rfe_gate_validation"
	if start := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditStart, Arguments: mustArgs(t, stageStartArgs(gate, draftID, "gate-validation-start"))}); start.Error != nil {
		t.Fatalf("gate validation start failed: %#v", start.Error)
	}
	submitArgs := stageCommonArgs(gate)
	submitArgs["idempotency_key"] = "gate-validation-missing"
	submitArgs["draft_id"] = draftID
	submitArgs["pending_event_id"] = gate.PendingEventID
	submitArgs["plan_event_id"] = gate.PlanEventID
	if missing := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditSubmit, Arguments: mustArgs(t, submitArgs)}); missing.Error == nil {
		t.Fatalf("gate submit accepted missing findings: %#v", missing)
	}

	rawStatement := "This raw passage must stay out of the trace."
	submitArgs["idempotency_key"] = "gate-validation-raw"
	submitArgs["gate_findings"] = []map[string]any{{
		"statement": rawStatement, "classification": reporting.FinalEditGateClassRhetoricalConstruction,
		"raw_passage": "forbidden raw passage",
	}}
	if raw := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditSubmit, Arguments: mustArgs(t, submitArgs)}); raw.Error == nil {
		t.Fatalf("gate submit accepted raw passage metadata: %#v", raw)
	}
	for _, event := range service.events {
		if strings.Contains(string(event.Payload), rawStatement) {
			t.Fatalf("raw gate statement leaked into MCP trace payload: %s", event.Payload)
		}
	}
}
