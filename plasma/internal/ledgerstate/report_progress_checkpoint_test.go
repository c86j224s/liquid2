package ledgerstate

import "testing"

func TestProjectReportProgressExperimentalLongFormEnablesRetryWithCheckpoint(t *testing.T) {
	pendingID := "evt_checkpoint_pending"
	progress := ProjectReportProgress([]Event{
		{EventID: pendingID, EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{
			"origin_pending_event_id": pendingID, "report_mode": "long_form", "pipeline_family": "report_il_experimental",
		})},
		{EventID: "evt_checkpoint", EventType: "report.il.checkpoint.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": pendingID, "checkpoint": map[string]any{"stage": "il_long_form_final"},
		})},
		{EventID: "evt_failed", EventType: "report.draft.failed", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": pendingID, "failed_stage_kind": "il_reader", "failed_stage_id": "il_reader",
		})},
	})
	if !progress.Retry.ResumeFailed || !progress.Retry.Restart || progress.Retry.ReasonCode != "" {
		t.Fatalf("checkpoint retry capability = %#v", progress.Retry)
	}
}
