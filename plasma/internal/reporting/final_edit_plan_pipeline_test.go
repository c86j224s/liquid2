package reporting

import (
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestLongFormPlanPipelineIgnoresMalformedUnrelatedPlan(t *testing.T) {
	state, ok, err := longFormPlanPipeline([]app.LedgerEvent{
		{EventID: "evt_pending", MissionID: "mis_plan", EventType: "report.draft.pending", Payload: mustJSON(map[string]any{"report_mode": ModeLongForm})},
		{EventID: "evt_other_plan", MissionID: "mis_plan", EventType: "report.plan.created", Payload: []byte(`{`)},
		{EventID: "evt_plan", MissionID: "mis_plan", EventType: "report.plan.created", Payload: mustJSON(map[string]any{
			"pending_event_id":      "evt_pending",
			"artifact_id":           "art_final",
			"report_mode":           ModeLongForm,
			"final_edit_pipeline":   FinalEditPipelineReaderStyleGateV1,
			"post_report_humanize":  FinalEditHumanizeDisabled,
			"part_planning_enabled": false,
		})},
	}, "evt_pending", "evt_plan")
	if err != nil || !ok || state.Pipeline != FinalEditPipelineReaderStyleGateV1 {
		t.Fatalf("state=%#v ok=%t err=%v", state, ok, err)
	}
}

func TestLongFormPlanPipelineRejectsMalformedBoundPlan(t *testing.T) {
	_, ok, err := longFormPlanPipeline([]app.LedgerEvent{
		{EventID: "evt_pending", MissionID: "mis_plan", EventType: "report.draft.pending", Payload: mustJSON(map[string]any{"report_mode": ModeLongForm})},
		{EventID: "evt_plan", MissionID: "mis_plan", EventType: "report.plan.created", Payload: []byte(`{`)},
	}, "evt_pending", "evt_plan")
	if ok || !errors.Is(err, app.ErrConflict) {
		t.Fatalf("ok=%t err=%v, want bound conflict", ok, err)
	}
}

func TestLongFormPlanPipelineAcceptsAcceptedAncestorPlanForResumeFailed(t *testing.T) {
	state, ok, err := longFormPlanPipeline([]app.LedgerEvent{
		{EventID: "evt_root_pending", MissionID: "mis_plan", EventType: "report.draft.pending", Payload: mustJSON(map[string]any{
			"origin_pending_event_id": "evt_root_pending",
			"retry_strategy":          "initial",
			"report_mode":             ModeLongForm,
		})},
		{EventID: "evt_plan", MissionID: "mis_plan", EventType: "report.plan.created", Payload: mustJSON(map[string]any{
			"pending_event_id":      "evt_root_pending",
			"artifact_id":           "art_final",
			"report_mode":           ModeLongForm,
			"final_edit_pipeline":   FinalEditPipelineReaderStyleGateV1,
			"post_report_humanize":  FinalEditHumanizeDisabled,
			"part_planning_enabled": false,
		})},
		{EventID: "evt_root_failed", MissionID: "mis_plan", EventType: "report.draft.failed", Payload: mustJSON(map[string]any{
			"pending_event_id": "evt_root_pending",
			"kind":             "report_draft_failed",
		})},
		{EventID: "evt_retry_pending", MissionID: "mis_plan", EventType: "report.draft.pending", Payload: mustJSON(map[string]any{
			"origin_pending_event_id":   "evt_root_pending",
			"retry_of_pending_event_id": "evt_root_pending",
			"retry_strategy":            "resume_failed",
			"report_mode":               ModeLongForm,
		})},
	}, "evt_retry_pending", "evt_plan")
	if err != nil || !ok || state.PendingEventID != "evt_root_pending" {
		t.Fatalf("state=%#v ok=%t err=%v, want accepted ancestor plan", state, ok, err)
	}
}

func TestLongFormPlanPipelineRejectsForeignBoundPlanPending(t *testing.T) {
	_, ok, err := longFormPlanPipeline([]app.LedgerEvent{
		{EventID: "evt_root_pending", MissionID: "mis_plan", EventType: "report.draft.pending", Payload: mustJSON(map[string]any{
			"origin_pending_event_id": "evt_root_pending",
			"retry_strategy":          "initial",
			"report_mode":             ModeLongForm,
		})},
		{EventID: "evt_root_failed", MissionID: "mis_plan", EventType: "report.draft.failed", Payload: mustJSON(map[string]any{
			"pending_event_id": "evt_root_pending",
			"kind":             "report_draft_failed",
		})},
		{EventID: "evt_retry_pending", MissionID: "mis_plan", EventType: "report.draft.pending", Payload: mustJSON(map[string]any{
			"origin_pending_event_id":   "evt_root_pending",
			"retry_of_pending_event_id": "evt_root_pending",
			"retry_strategy":            "resume_failed",
			"report_mode":               ModeLongForm,
		})},
		{EventID: "evt_plan", MissionID: "mis_plan", EventType: "report.plan.created", Payload: mustJSON(map[string]any{
			"pending_event_id":      "evt_foreign_pending",
			"artifact_id":           "art_final",
			"report_mode":           ModeLongForm,
			"final_edit_pipeline":   FinalEditPipelineReaderStyleGateV1,
			"post_report_humanize":  FinalEditHumanizeDisabled,
			"part_planning_enabled": false,
		})},
	}, "evt_retry_pending", "evt_plan")
	if ok || !errors.Is(err, app.ErrConflict) {
		t.Fatalf("ok=%t err=%v, want foreign lineage conflict", ok, err)
	}
}
