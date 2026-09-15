package app

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
)

func TestMissionActivityFromEventsProjectsActiveWorkAndLatestOutcome(t *testing.T) {
	events := []ledger.Event{
		activityEvent(t, "evt_response", 3, "turn.agent.response", map[string]any{"user_event_id": "evt_user"}),
		activityEvent(t, "evt_report_pending", 4, "report.draft.pending", map[string]any{"title": "Report"}),
		activityEvent(t, "evt_report_failed", 5, "report.draft.failed", map[string]any{"pending_event_id": "evt_report_pending"}),
		activityEvent(t, "evt_turn_pending", 6, "turn.agent.pending", map[string]any{"user_event_id": "evt_next"}),
	}

	summary := MissionActivityFromEvents(events)
	if summary.LastSequence != 6 {
		t.Fatalf("last sequence = %d, want 6", summary.LastSequence)
	}
	if len(summary.ActiveWork.Items) != 1 || summary.ActiveWork.Items[0].Kind != mission.ActiveWorkTurn {
		t.Fatalf("active work = %#v, want open agent turn", summary.ActiveWork)
	}
	if summary.LatestTerminalActivity == nil {
		t.Fatal("latest activity is missing")
	}
	if got := summary.LatestTerminalActivity; got.EventID != "evt_report_failed" || got.Sequence != 5 || got.Kind != mission.ActiveWorkReport || got.Outcome != mission.TerminalActivityFailed {
		t.Fatalf("latest activity = %#v", got)
	}
}

func TestMissionActivityFromEventsRecognizesWorkflowCompletionAndFailure(t *testing.T) {
	completed := MissionActivityFromEvents([]ledger.Event{activityEvent(t, "evt_workflow_done", 4, WorkflowRunCompletedEvent, map[string]any{"workflow_run_id": "wfr_1", "mission_id": "mis_1"})})
	if completed.LatestTerminalActivity == nil || completed.LatestTerminalActivity.Outcome != mission.TerminalActivityCompleted || completed.LatestTerminalActivity.Kind != mission.ActiveWorkWorkflow {
		t.Fatalf("completed workflow activity = %#v", completed.LatestTerminalActivity)
	}

	failed := MissionActivityFromEvents([]ledger.Event{activityEvent(t, "evt_workflow_failed", 5, WorkflowRunFailedEvent, map[string]any{"workflow_run_id": "wfr_1", "mission_id": "mis_1"})})
	if failed.LatestTerminalActivity == nil || failed.LatestTerminalActivity.Outcome != mission.TerminalActivityFailed {
		t.Fatalf("failed workflow activity = %#v", failed.LatestTerminalActivity)
	}
}

func TestTerminalActivityFromEventClassifiesAllCurrentTerminalOutcomes(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		payload   map[string]any
		kind      mission.TerminalActivityKind
		outcome   mission.TerminalActivityOutcome
	}{
		{name: "agent response", eventType: turnAgentResponseEvent, payload: map[string]any{"kind": "agent_response"}, kind: mission.TerminalActivityTurn, outcome: mission.TerminalActivityCompleted},
		{name: "agent error", eventType: turnAgentResponseEvent, payload: map[string]any{"kind": "agent_error"}, kind: mission.TerminalActivityTurn, outcome: mission.TerminalActivityFailed},
		{name: "agent canceled", eventType: turnAgentResponseEvent, payload: map[string]any{"kind": "agent_canceled"}, kind: mission.TerminalActivityTurn, outcome: mission.TerminalActivityCanceled},
		{name: "unavailable agent", eventType: turnAgentResponseEvent, payload: map[string]any{"kind": "placeholder"}, kind: mission.TerminalActivityTurn, outcome: mission.TerminalActivityFailed},
		{name: "legacy response", eventType: turnAgentResponseEvent, payload: map[string]any{}, kind: mission.TerminalActivityTurn, outcome: mission.TerminalActivityCompleted},
		{name: "report complete", eventType: "report.artifact.exported", payload: map[string]any{}, kind: mission.TerminalActivityReport, outcome: mission.TerminalActivityCompleted},
		{name: "report failed", eventType: "report.patch.failed", payload: map[string]any{}, kind: mission.TerminalActivityReport, outcome: mission.TerminalActivityFailed},
		{name: "workflow complete", eventType: WorkflowRunCompletedEvent, payload: map[string]any{}, kind: mission.TerminalActivityWorkflow, outcome: mission.TerminalActivityCompleted},
		{name: "workflow paused", eventType: WorkflowRunPausedEvent, payload: map[string]any{}, kind: mission.TerminalActivityWorkflow, outcome: mission.TerminalActivityPaused},
		{name: "workflow stopped", eventType: WorkflowRunStoppedEvent, payload: map[string]any{}, kind: mission.TerminalActivityWorkflow, outcome: mission.TerminalActivityStopped},
		{name: "workflow failed", eventType: WorkflowRunFailedEvent, payload: map[string]any{}, kind: mission.TerminalActivityWorkflow, outcome: mission.TerminalActivityFailed},
		{name: "workflow interrupted", eventType: WorkflowRunInterruptedEvent, payload: map[string]any{}, kind: mission.TerminalActivityWorkflow, outcome: mission.TerminalActivityFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			activity, ok := terminalActivityFromEvent(activityEvent(t, "evt_terminal", 7, test.eventType, test.payload))
			if !ok || activity.Kind != test.kind || activity.Outcome != test.outcome {
				t.Fatalf("terminal activity = %#v, ok=%v", activity, ok)
			}
		})
	}

	if _, ok := terminalActivityFromEvent(activityEvent(t, "evt_stage", 8, "report.plan.failed", map[string]any{})); ok {
		t.Fatal("report stage companion must not replace a report operation terminal outcome")
	}
	if _, ok := terminalActivityFromEvent(activityEvent(t, "evt_unknown", 9, turnAgentResponseEvent, map[string]any{"kind": "future_kind"})); ok {
		t.Fatal("unknown turn response kind must not be assigned a misleading outcome")
	}
}

func TestListMissionsUsesBulkActivityInputsWhenSupported(t *testing.T) {
	store := &missionActivityListStore{
		missions: []mission.Mission{{MissionID: "mis_1", Title: "Mission"}},
		inputs: []mission.ActivityInput{{
			MissionID:    "mis_1",
			LastSequence: 9,
			Events: []ledger.Event{
				activityEvent(t, "evt_response", 4, turnAgentResponseEvent, map[string]any{"kind": "agent_response"}),
			},
		}},
	}
	missions, err := NewService(store).ListMissions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if store.listLedgerEventsCalls != 0 {
		t.Fatalf("bulk activity store must not fall back to per-mission ledger reads: %d", store.listLedgerEventsCalls)
	}
	if len(store.lastRequestedMissionIDs) != 1 || store.lastRequestedMissionIDs[0] != "mis_1" {
		t.Fatalf("bulk activity request must be scoped to visible missions: %#v", store.lastRequestedMissionIDs)
	}
	if len(missions) != 1 || missions[0].Activity.LastSequence != 9 || missions[0].Activity.LatestTerminalActivity == nil || missions[0].Activity.LatestTerminalActivity.Outcome != mission.TerminalActivityCompleted {
		t.Fatalf("missions = %#v", missions)
	}
	activity, err := NewService(store).MissionActivity(context.Background(), "mis_1")
	if err != nil || activity.LastSequence != 9 || len(store.lastRequestedMissionIDs) != 1 || store.lastRequestedMissionIDs[0] != "mis_1" {
		t.Fatalf("single mission activity = %#v, requested=%#v, err=%v", activity, store.lastRequestedMissionIDs, err)
	}
}

func TestListMissionsFiltersArchivedByDefault(t *testing.T) {
	store := &missionActivityListStore{
		missions: []mission.Mission{
			{MissionID: "mis_active", Title: "Active", LifecycleState: mission.LifecycleActive},
			{MissionID: "mis_archived", Title: "Archived", LifecycleState: mission.LifecycleArchived},
		},
		inputs: []mission.ActivityInput{{MissionID: "mis_active", LastSequence: 2}, {MissionID: "mis_archived", LastSequence: 3}},
	}
	missions, err := NewService(store).ListMissions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 1 || missions[0].MissionID != "mis_active" {
		t.Fatalf("default missions = %#v", missions)
	}
	if len(store.lastRequestedMissionIDs) != 1 || store.lastRequestedMissionIDs[0] != "mis_active" {
		t.Fatalf("default activity scope = %#v", store.lastRequestedMissionIDs)
	}

	missions, err = NewService(store).ListMissionsWithState(context.Background(), mission.ListRequest{IncludeArchived: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 2 || missions[1].MissionID != "mis_archived" || missions[1].LifecycleState != mission.LifecycleArchived {
		t.Fatalf("include archived missions = %#v", missions)
	}
	if len(store.lastRequestedMissionIDs) != 2 || store.lastRequestedMissionIDs[0] != "mis_active" || store.lastRequestedMissionIDs[1] != "mis_archived" {
		t.Fatalf("include archived activity scope = %#v", store.lastRequestedMissionIDs)
	}
}

type missionActivityListStore struct {
	fakeStore
	missions                []mission.Mission
	inputs                  []mission.ActivityInput
	listLedgerEventsCalls   int
	lastRequestedMissionIDs []string
}

func (s *missionActivityListStore) ListLedgerEvents(context.Context, string) ([]ledger.Event, error) {
	s.listLedgerEventsCalls++
	return nil, nil
}

func (s *missionActivityListStore) ListMissions(context.Context) ([]mission.Mission, error) {
	return append([]mission.Mission(nil), s.missions...), nil
}

func (s *missionActivityListStore) ListMissionActivityInputs(_ context.Context, missionIDs []string) ([]mission.ActivityInput, error) {
	s.lastRequestedMissionIDs = append([]string(nil), missionIDs...)
	return append([]mission.ActivityInput(nil), s.inputs...), nil
}

func activityEvent(t *testing.T, id string, sequence int64, eventType string, payload any) ledger.Event {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return ledger.Event{EventID: id, MissionID: "mis_1", Sequence: sequence, EventType: eventType, Payload: raw}
}
