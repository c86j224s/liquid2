package reporting_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestFinalEditPipelineFromPlanEventPreservesLegacyAndValidatesLiteral(t *testing.T) {
	legacy, ok, err := reporting.FinalEditPipelineFromPlanEvent(app.LedgerEvent{
		EventID:   "evt_plan",
		EventType: "report.plan.created",
		Payload:   testJSON(map[string]any{"pending_event_id": "evt_pending", "artifact_id": "art_final", "report_mode": reporting.ModeLongForm}),
	})
	if err != nil || !ok || legacy.Pipeline != "" || legacy.PendingEventID != "evt_pending" {
		t.Fatalf("legacy plan state=%#v ok=%t err=%v", legacy, ok, err)
	}

	gate, ok, err := reporting.FinalEditPipelineFromPlanEvent(app.LedgerEvent{
		EventID:   "evt_plan",
		EventType: "report.plan.created",
		Payload: testJSON(map[string]any{
			"pending_event_id": "evt_pending", "artifact_id": "art_final", "report_mode": reporting.ModeLongForm,
			"final_edit_pipeline": reporting.FinalEditPipelineReaderStyleGateV1, "post_report_humanize": reporting.FinalEditHumanizeEnabled,
		}),
	})
	if err != nil || !ok || gate.Pipeline != reporting.FinalEditPipelineReaderStyleGateV1 || gate.PostReportHumanize != reporting.FinalEditHumanizeEnabled {
		t.Fatalf("gate plan state=%#v ok=%t err=%v", gate, ok, err)
	}

	writer, ok, err := reporting.FinalEditPipelineFromPlanEvent(app.LedgerEvent{
		EventID:   "evt_plan",
		EventType: "report.plan.created",
		Payload: testJSON(map[string]any{
			"pending_event_id": "evt_pending", "artifact_id": "art_final", "report_mode": reporting.ModeLongForm,
			"final_edit_pipeline": reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, "post_report_humanize": reporting.FinalEditHumanizeDisabled,
		}),
	})
	if err != nil || !ok || writer.Pipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 || writer.PostReportHumanize != reporting.FinalEditHumanizeDisabled {
		t.Fatalf("writer plan state=%#v ok=%t err=%v", writer, ok, err)
	}

	_, ok, err = reporting.FinalEditPipelineFromPlanEvent(app.LedgerEvent{EventType: "mission.created"})
	if err != nil || ok {
		t.Fatalf("non-plan ok=%t err=%v", ok, err)
	}

	_, _, err = reporting.FinalEditPipelineFromPlanEvent(app.LedgerEvent{
		EventID:   "evt_plan",
		EventType: "report.plan.created",
		Payload:   testJSON(map[string]any{"pending_event_id": "evt_pending", "final_edit_pipeline": "reader_style_gate_v2"}),
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("unsupported pipeline error=%v, want conflict", err)
	}

	_, _, err = reporting.FinalEditPipelineFromPlanEvent(app.LedgerEvent{
		EventID:   "evt_plan",
		EventType: "report.plan.created",
		Payload: testJSON(map[string]any{
			"pending_event_id": "evt_pending", "artifact_id": "art_final", "report_mode": reporting.ModeLongForm,
			"final_edit_pipeline": reporting.FinalEditPipelineReaderStyleGateV1, "post_report_humanize": "h5",
		}),
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("new pipeline h5 humanize error=%v, want conflict", err)
	}
}

func TestBuildMarkdownReportPlanCreatedAppendRequestCarriesFinalEditPipeline(t *testing.T) {
	request := reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
		MarkdownReportEventBase: reporting.MarkdownReportEventBase{
			EventID: "evt_plan", MissionID: "mis_1", PendingEventID: "evt_pending",
			ReportMode: reporting.ModeLongForm, Producer: app.Producer{Type: "agent_session", ID: "provider-plan"},
		},
		ArtifactID:        "art_final",
		FinalEditPipeline: reporting.FinalEditPipelineReaderStyleGateV1,
	})
	payload := map[string]any{}
	if err := json.Unmarshal(request.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["final_edit_pipeline"] != reporting.FinalEditPipelineReaderStyleGateV1 {
		t.Fatalf("pipeline not persisted in plan payload: %#v", payload)
	}

	legacy := reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
		MarkdownReportEventBase: reporting.MarkdownReportEventBase{
			EventID: "evt_plan", MissionID: "mis_1", PendingEventID: "evt_pending",
			ReportMode: reporting.ModeLongForm, Producer: app.Producer{Type: "agent_session", ID: "provider-plan"},
		},
		ArtifactID: "art_final",
	})
	payload = map[string]any{}
	if err := json.Unmarshal(legacy.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if _, exists := payload["final_edit_pipeline"]; exists {
		t.Fatalf("legacy plan gained final_edit_pipeline: %#v", payload)
	}
}
