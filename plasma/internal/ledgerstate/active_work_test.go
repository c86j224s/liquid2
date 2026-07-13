package ledgerstate

import (
	"encoding/json"
	"testing"
)

func TestHasOpenAgentPendingTracksTerminalResponseByUserEvent(t *testing.T) {
	events := []Event{
		event(t, "evt_pending_1", "turn.agent.pending", map[string]any{"user_event_id": "evt_user_1"}),
		event(t, "evt_response_1", "turn.agent.response", map[string]any{"user_event_id": "evt_user_1"}),
	}
	if HasOpenAgentPending(events) {
		t.Fatalf("expected completed user turn to close pending state")
	}

	events = append(events, event(t, "evt_pending_2", "turn.agent.pending", map[string]any{"user_event_id": "evt_user_2"}))
	if !HasOpenAgentPending(events) {
		t.Fatalf("expected unmatched pending event to keep agent work open")
	}
}

func TestOpenPendingEventsReturnNewestOpenEvent(t *testing.T) {
	events := []Event{
		event(t, "evt_turn", "turn.agent.pending", map[string]any{"user_event_id": "evt_user"}),
		event(t, "evt_report", "report.draft.pending", map[string]any{}),
	}
	if pending, ok := OpenAgentPendingEvent(events); !ok || pending.EventID != "evt_turn" {
		t.Fatalf("unexpected open agent pending event: %#v, %v", pending, ok)
	}
	if pending, ok := OpenReportPendingEvent(events); !ok || pending.EventID != "evt_report" {
		t.Fatalf("unexpected open report pending event: %#v, %v", pending, ok)
	}
}

func TestValidateWorkflowStartAfterEventRequiresOpenUserTurn(t *testing.T) {
	events := []Event{
		event(t, "evt_user", "turn.user", map[string]any{"kind": "user_turn"}),
		event(t, "evt_pending", "turn.agent.pending", map[string]any{"user_event_id": "evt_user"}),
	}
	if message := ValidateWorkflowStartAfterEvent(events, "evt_user"); message != "" {
		t.Fatalf("expected open user turn to be valid, got %q", message)
	}

	events = append(events, event(t, "evt_response", "turn.agent.response", map[string]any{"user_event_id": "evt_user"}))
	if message := ValidateWorkflowStartAfterEvent(events, "evt_user"); message == "" {
		t.Fatalf("expected terminal user turn to be rejected")
	}
}

func TestHasOpenReportPendingTracksTerminalEvents(t *testing.T) {
	events := []Event{
		event(t, "evt_pending", "report.draft.pending", map[string]any{"kind": "report_draft_pending"}),
	}
	if !HasOpenReportPending(events) {
		t.Fatalf("expected draft pending event to be open")
	}
	events = append(events, event(t, "evt_done", "report.draft.failed", map[string]any{"pending_event_id": "evt_pending"}))
	if HasOpenReportPending(events) {
		t.Fatalf("expected draft failure to close pending event")
	}
}

func event(t *testing.T, eventID string, eventType string, payloadValue any) Event {
	t.Helper()
	payload, err := json.Marshal(payloadValue)
	if err != nil {
		t.Fatalf("marshal event payload: %v", err)
	}
	return Event{EventID: eventID, EventType: eventType, Payload: payload}
}
