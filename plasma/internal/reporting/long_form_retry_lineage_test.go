package reporting

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestLongFormPendingLineageAcceptsResumeFailedOnlyWhenParentDraftFailed(t *testing.T) {
	events := []app.LedgerEvent{
		longFormRetryLineageEvent(t, "evt_root", "report.draft.pending", map[string]any{
			"origin_pending_event_id": "evt_root",
			"retry_strategy":          "initial",
		}),
		longFormRetryLineageEvent(t, "evt_retry", "report.draft.pending", map[string]any{
			"origin_pending_event_id":   "evt_root",
			"retry_of_pending_event_id": "evt_root",
			"retry_strategy":            "resume_failed",
		}),
		longFormRetryLineageEvent(t, "evt_root_failed", "report.draft.failed", map[string]any{
			"pending_event_id": "evt_root",
			"kind":             "report_draft_failed",
		}),
	}
	accepted, err := longFormPendingLineage(events, "evt_retry")
	if err != nil {
		t.Fatal(err)
	}
	if !accepted["evt_retry"] || !accepted["evt_root"] {
		t.Fatalf("accepted lineage differs: %#v", accepted)
	}
}

func TestLongFormPendingLineageRejectsNonFailedRetryParentTerminals(t *testing.T) {
	for _, tc := range []struct {
		name     string
		terminal []app.LedgerEvent
	}{
		{name: "missing"},
		{name: "final_failed_only", terminal: []app.LedgerEvent{
			longFormRetryLineageEvent(t, "evt_root_final_failed", "report.final.failed", map[string]any{"pending_event_id": "evt_root"}),
		}},
		{name: "canceled", terminal: []app.LedgerEvent{
			longFormRetryLineageEvent(t, "evt_root_canceled", "report.draft.failed", map[string]any{"pending_event_id": "evt_root", "kind": "report_draft_canceled"}),
		}},
		{name: "drafted", terminal: []app.LedgerEvent{
			longFormRetryLineageEvent(t, "evt_root_drafted", "report.drafted", map[string]any{"pending_event_id": "evt_root"}),
		}},
		{name: "artifact_created", terminal: []app.LedgerEvent{
			longFormRetryLineageEvent(t, "evt_root_artifact", "report.artifact.created", map[string]any{"pending_event_id": "evt_root"}),
		}},
		{name: "multiple", terminal: []app.LedgerEvent{
			longFormRetryLineageEvent(t, "evt_root_failed", "report.draft.failed", map[string]any{"pending_event_id": "evt_root"}),
			longFormRetryLineageEvent(t, "evt_root_artifact", "report.artifact.created", map[string]any{"pending_event_id": "evt_root"}),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			events := append([]app.LedgerEvent{
				longFormRetryLineageEvent(t, "evt_root", "report.draft.pending", map[string]any{
					"origin_pending_event_id": "evt_root",
					"retry_strategy":          "initial",
				}),
				longFormRetryLineageEvent(t, "evt_retry", "report.draft.pending", map[string]any{
					"origin_pending_event_id":   "evt_root",
					"retry_of_pending_event_id": "evt_root",
					"retry_strategy":            "resume_failed",
				}),
			}, tc.terminal...)
			if _, err := longFormPendingLineage(events, "evt_retry"); !errors.Is(err, app.ErrConflict) {
				t.Fatalf("err=%v, want conflict", err)
			}
		})
	}
}

func TestLongFormPendingLineageRestartRequiresFailedParentButDoesNotAcceptIt(t *testing.T) {
	events := []app.LedgerEvent{
		longFormRetryLineageEvent(t, "evt_root", "report.draft.pending", map[string]any{
			"origin_pending_event_id": "evt_root",
			"retry_strategy":          "initial",
		}),
		longFormRetryLineageEvent(t, "evt_restart", "report.draft.pending", map[string]any{
			"origin_pending_event_id":   "evt_root",
			"retry_of_pending_event_id": "evt_root",
			"retry_strategy":            "restart",
		}),
		longFormRetryLineageEvent(t, "evt_root_failed", "report.draft.failed", map[string]any{
			"pending_event_id": "evt_root",
		}),
	}
	accepted, err := longFormPendingLineage(events, "evt_restart")
	if err != nil {
		t.Fatal(err)
	}
	if !accepted["evt_restart"] || accepted["evt_root"] {
		t.Fatalf("restart accepted lineage differs: %#v", accepted)
	}
}

func longFormRetryLineageEvent(t *testing.T, eventID, eventType string, payload map[string]any) app.LedgerEvent {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return app.LedgerEvent{EventID: eventID, MissionID: "mis_retry_lineage", EventType: eventType, Payload: encoded}
}
