package app

import (
	"context"
	"errors"
	"testing"
)

func TestListMissionChangesReturnsMeaningfulChangesOnly(t *testing.T) {
	store := &researchIDEVisibilityStore{events: []LedgerEvent{
		changeTestEvent("evt_mission", 1, "mission.created", Producer{Type: "user", ID: "plasma-ui"}),
		changeTestEvent("evt_trace", 2, "mcp.tool.called", Producer{Type: "mcp", ID: "plasma"}),
		changeTestEvent("evt_step", 3, "workflow.step.started", Producer{Type: "workflow", ID: "wfr_1"}),
		changeTestEvent("evt_steering", 4, "turn.user", Producer{Type: "workflow", ID: "wfr_1"}),
		changeTestEvent("evt_pending", 5, "turn.agent.pending", Producer{Type: "agent", ID: "codex"}),
		changeTestEvent("evt_report", 6, "report.artifact.created", Producer{Type: "agent", ID: "codex"}),
		changeTestEvent("evt_user", 7, "turn.user", Producer{Type: "user", ID: "plasma-ui"}),
		changeTestEvent("evt_source", 8, "source.removed", Producer{Type: "user", ID: "plasma-ui"}),
		changeTestEvent("evt_question", 9, "question.proposed", Producer{Type: "agent_session", ID: "ses_1"}),
	}}

	result, err := NewService(store).ListMissionChanges(context.Background(), ResearchIDEChangesRequest{
		MissionID: "mis_1", AfterSequence: 1, Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListMissionChanges returned error: %v", err)
	}
	if result.CurrentSequence != 9 || result.NextAfterSequence != 9 || result.Truncated || result.ResyncRequired {
		t.Fatalf("unexpected change cursor state: %#v", result)
	}
	if len(result.Items) != 3 || result.Items[0].ObjectID != "evt_user" || result.Items[1].ObjectID != "evt_source" || result.Items[2].ObjectID != "evt_question" {
		t.Fatalf("unexpected meaningful changes: %#v", result.Items)
	}
}

func TestListMissionChangesAdvancesAcrossInternalOnlyEvents(t *testing.T) {
	store := &researchIDEVisibilityStore{events: []LedgerEvent{
		changeTestEvent("evt_user", 7, "turn.user", Producer{Type: "user", ID: "plasma-ui"}),
		changeTestEvent("evt_trace", 8, "mcp.tool.called", Producer{Type: "mcp", ID: "plasma"}),
		changeTestEvent("evt_step", 9, "workflow.step.completed", Producer{Type: "workflow", ID: "wfr_1"}),
	}}
	result, err := NewService(store).ListMissionChanges(context.Background(), ResearchIDEChangesRequest{
		MissionID: "mis_1", AfterSequence: 7,
	})
	if err != nil {
		t.Fatalf("ListMissionChanges returned error: %v", err)
	}
	if len(result.Items) != 0 || result.CurrentSequence != 9 || result.NextAfterSequence != 9 || result.ResyncRequired {
		t.Fatalf("internal events must advance the cursor without becoming changes: %#v", result)
	}
}

func TestListMissionChangesPaginatesByLedgerSequence(t *testing.T) {
	store := &researchIDEVisibilityStore{events: []LedgerEvent{
		changeTestEvent("evt_source", 1, "source.snapshotted", Producer{Type: "connector", ID: "liquid2"}),
		changeTestEvent("evt_trace", 2, "mcp.tool.called", Producer{Type: "mcp", ID: "plasma"}),
		changeTestEvent("evt_mission", 3, "mission.updated", Producer{Type: "user", ID: "plasma-ui"}),
	}}
	svc := NewService(store)
	first, err := svc.ListMissionChanges(context.Background(), ResearchIDEChangesRequest{MissionID: "mis_1", Limit: 1})
	if err != nil {
		t.Fatalf("first ListMissionChanges returned error: %v", err)
	}
	if !first.Truncated || first.NextAfterSequence != 1 || first.CurrentSequence != 3 || len(first.Items) != 1 || first.Items[0].ObjectID != "evt_source" {
		t.Fatalf("unexpected first page: %#v", first)
	}
	second, err := svc.ListMissionChanges(context.Background(), ResearchIDEChangesRequest{MissionID: "mis_1", AfterSequence: first.NextAfterSequence, Limit: 1})
	if err != nil {
		t.Fatalf("second ListMissionChanges returned error: %v", err)
	}
	if second.Truncated || second.NextAfterSequence != 3 || len(second.Items) != 1 || second.Items[0].ObjectID != "evt_mission" {
		t.Fatalf("unexpected second page: %#v", second)
	}
}

func TestListMissionChangesRequiresResyncForFutureCursor(t *testing.T) {
	store := &researchIDEVisibilityStore{events: []LedgerEvent{
		changeTestEvent("evt_mission", 2, "mission.updated", Producer{Type: "user", ID: "plasma-ui"}),
	}}
	result, err := NewService(store).ListMissionChanges(context.Background(), ResearchIDEChangesRequest{
		MissionID: "mis_1", AfterSequence: 4,
	})
	if err != nil {
		t.Fatalf("ListMissionChanges returned error: %v", err)
	}
	if !result.ResyncRequired || result.CurrentSequence != 2 || result.NextAfterSequence != 0 || len(result.Items) != 0 {
		t.Fatalf("unexpected resync response: %#v", result)
	}
}

func TestListMissionChangesRejectsNegativeCursor(t *testing.T) {
	_, err := NewService(&researchIDEVisibilityStore{}).ListMissionChanges(context.Background(), ResearchIDEChangesRequest{
		MissionID: "mis_1", AfterSequence: -1,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func changeTestEvent(eventID string, sequence int64, eventType string, producer Producer) LedgerEvent {
	return LedgerEvent{EventID: eventID, MissionID: "mis_1", Sequence: sequence, EventType: eventType, Producer: producer}
}
