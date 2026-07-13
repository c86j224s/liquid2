package app

import (
	"encoding/json"
	"testing"
)

func TestActiveWorkFromMissionStateUsesOnlyCurrentMissionDurableState(t *testing.T) {
	events := []LedgerEvent{testActiveWorkEvent(t, "evt_turn", "turn.agent.pending", map[string]any{"user_event_id": "evt_user"})}
	state := ActiveWorkFromMissionState(events, nil)
	if len(state.Items) != 1 || state.Items[0].ReasonCode != BlockingReasonAgentTurn || state.Items[0].Action != "cancel_turn" {
		t.Fatalf("unexpected active turn state: %#v", state)
	}

	events = append(events, testActiveWorkEvent(t, "evt_report", "report.draft.pending", map[string]any{}))
	state = ActiveWorkFromMissionState(events, nil)
	if len(state.Blocks) != 2 || len(state.BlockedControls) != 3 {
		t.Fatalf("concurrent active work must keep every action and blocked control: %#v", state)
	}

	state = ActiveWorkFromMissionState(nil, []WorkflowRunView{{WorkflowRunID: "wfr_1", Status: WorkflowStatusRunning}})
	if len(state.Items) != 1 || state.Items[0].ReasonCode != BlockingReasonWorkflow || state.Items[0].WorkflowRunID != "wfr_1" {
		t.Fatalf("unexpected active workflow state: %#v", state)
	}
}

func TestActiveWorkMatrixPreservesTurnAndQueuedWorkflow(t *testing.T) {
	events := []LedgerEvent{testActiveWorkEvent(t, "evt_turn", "turn.agent.pending", map[string]any{"user_event_id": "evt_user"})}
	state := ActiveWorkFromMissionState(events, []WorkflowRunView{{WorkflowRunID: "wfr_queued", Status: WorkflowStatusQueued}})
	if len(state.Items) != 2 || state.Items[0].Action != "cancel_turn" || state.Items[1].Action != "view_workflow" {
		t.Fatalf("expected turn cancel and workflow view actions: %#v", state.Items)
	}
	for _, control := range state.BlockedControls {
		if len(control.ReasonCodes) != 2 {
			t.Fatalf("control %s lost a concurrent reason: %#v", control.Control, control)
		}
	}
}

func TestActiveWorkExcludesTerminalLedgerAndWorkflowState(t *testing.T) {
	events := []LedgerEvent{
		testActiveWorkEvent(t, "evt_pending", "report.draft.pending", map[string]any{}),
		testActiveWorkEvent(t, "evt_failed", "report.draft.failed", map[string]any{"pending_event_id": "evt_pending"}),
	}
	state := ActiveWorkFromMissionState(events, []WorkflowRunView{{WorkflowRunID: "wfr_1", Status: WorkflowStatusCompleted}})
	if len(state.Items) != 0 || len(state.Blocks) != 0 {
		t.Fatalf("terminal work must not remain active: %#v", state)
	}
}

func testActiveWorkEvent(t *testing.T, id, eventType string, payload any) LedgerEvent {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return LedgerEvent{EventID: id, EventType: eventType, Payload: raw}
}
