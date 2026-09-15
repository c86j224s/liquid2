package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"strings"
	"testing"
)

func TestMissionLifecycleArchiveRestoreAndIdempotency(t *testing.T) {
	store := newLifecycleStore(t, "mis_1")
	svc := NewService(store)
	if _, err := svc.RebuildProjection(context.Background(), "mis_1"); err != nil {
		t.Fatal(err)
	}

	archived, err := svc.ArchiveMission(context.Background(), mission.MissionLifecycleChangeRequest{
		EventID: "evt_archive", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "test"},
		Reason: "done",
	})
	if err != nil {
		t.Fatal(err)
	}
	if archived.Event == nil || archived.Event.EventType != mission.ArchivedEvent || archived.Projection.LifecycleState != mission.LifecycleArchived {
		t.Fatalf("archive result = %#v", archived)
	}
	missions, err := svc.ListMissions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 0 {
		t.Fatalf("archived mission must be hidden by default: %#v", missions)
	}
	missions, err = svc.ListMissionsWithState(context.Background(), mission.ListRequest{IncludeArchived: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 1 || missions[0].LifecycleState != mission.LifecycleArchived {
		t.Fatalf("include archived missions = %#v", missions)
	}

	again, err := svc.ArchiveMission(context.Background(), mission.MissionLifecycleChangeRequest{
		EventID: "evt_archive_again", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !again.Idempotent || again.Event != nil {
		t.Fatalf("idempotent archive = %#v", again)
	}

	restored, err := svc.RestoreMission(context.Background(), mission.MissionLifecycleChangeRequest{
		EventID: "evt_restore", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if restored.Event == nil || restored.Event.EventType != mission.RestoredEvent || restored.Projection.LifecycleState != mission.LifecycleActive {
		t.Fatalf("restore result = %#v", restored)
	}
	missions, err = svc.ListMissions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 1 || missions[0].MissionID != "mis_1" || missions[0].LifecycleState != mission.LifecycleActive {
		t.Fatalf("restored default missions = %#v", missions)
	}
}

func TestArchiveMissionIdempotentBypassesOpenActiveWork(t *testing.T) {
	store := newLifecycleStore(t, "mis_1")
	store.events["mis_1"] = append(store.events["mis_1"], lifecycleEvent(t, "evt_archive", "mis_1", 2, mission.ArchivedEvent, map[string]any{"lifecycle_state": mission.LifecycleArchived, "reason": "done"}))
	store.events["mis_1"] = append(store.events["mis_1"], lifecycleEvent(t, "evt_turn_pending", "mis_1", 3, "turn.agent.pending", map[string]any{"user_event_id": "evt_user"}))
	result, err := NewService(store).ArchiveMission(context.Background(), mission.MissionLifecycleChangeRequest{
		EventID: "evt_archive_again", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Idempotent || result.Event != nil {
		t.Fatalf("idempotent archive with active work = %#v", result)
	}
}

func TestArchiveMissionRejectsOpenActiveWork(t *testing.T) {
	store := newLifecycleStore(t, "mis_1")
	store.events["mis_1"] = append(store.events["mis_1"], lifecycleEvent(t, "evt_turn_pending", "mis_1", 2, "turn.agent.pending", map[string]any{"user_event_id": "evt_user"}))
	svc := NewService(store)
	_, err := svc.ArchiveMission(context.Background(), mission.MissionLifecycleChangeRequest{
		EventID: "evt_archive", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "test"},
	})
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "agent turn") {
		t.Fatalf("expected active work rejection, got %v", err)
	}
}

type lifecycleStore struct {
	fakeStore
	events      map[string][]ledger.Event
	projections map[string]mission.Projection
	missions    map[string]mission.Mission
}

func newLifecycleStore(t *testing.T, missionID string) *lifecycleStore {
	t.Helper()
	return &lifecycleStore{
		events: map[string][]ledger.Event{
			missionID: []ledger.Event{lifecycleEvent(t, "evt_created", missionID, 1, "mission.created", map[string]any{"title": "Mission", "objective": "Mission"})},
		},
		projections: map[string]mission.Projection{},
		missions: map[string]mission.Mission{
			missionID: {MissionID: missionID, Title: "Mission", LifecycleState: mission.LifecycleActive},
		},
	}
}

func (s *lifecycleStore) ListMissions(context.Context) ([]mission.Mission, error) {
	missions := make([]mission.Mission, 0, len(s.missions))
	for _, mission := range s.missions {
		missions = append(missions, mission)
	}
	return missions, nil
}

func (s *lifecycleStore) ListMissionActivityInputs(_ context.Context, missionIDs []string) ([]mission.ActivityInput, error) {
	inputs := make([]mission.ActivityInput, 0, len(missionIDs))
	for _, missionID := range missionIDs {
		events := s.events[missionID]
		var lastSequence int64
		if len(events) > 0 {
			lastSequence = events[len(events)-1].Sequence
		}
		inputs = append(inputs, mission.ActivityInput{MissionID: missionID, LastSequence: lastSequence})
	}
	return inputs, nil
}

func (s *lifecycleStore) AppendLedgerEventsConditionally(_ context.Context, missionID string, build func([]ledger.Event) ([]ledger.Event, error)) ([]ledger.Event, error) {
	current := append([]ledger.Event(nil), s.events[missionID]...)
	toAppend, err := build(current)
	if err != nil {
		return nil, err
	}
	for index := range toAppend {
		toAppend[index].Sequence = int64(len(s.events[missionID]) + 1)
		s.events[missionID] = append(s.events[missionID], toAppend[index])
	}
	return toAppend, nil
}

func (s *lifecycleStore) ListLedgerEvents(_ context.Context, missionID string) ([]ledger.Event, error) {
	return append([]ledger.Event(nil), s.events[missionID]...), nil
}

func (s *lifecycleStore) SaveMissionProjection(_ context.Context, projection mission.Projection) error {
	s.projections[projection.MissionID] = projection
	mission := s.missions[projection.MissionID]
	mission.Title = projection.Title
	mission.LifecycleState = projection.LifecycleState
	s.missions[projection.MissionID] = mission
	return nil
}

func (s *lifecycleStore) GetMissionProjection(_ context.Context, missionID string) (mission.Projection, error) {
	return s.projections[missionID], nil
}

func lifecycleEvent(t *testing.T, id, missionID string, sequence int64, eventType string, payload any) ledger.Event {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return ledger.Event{EventID: id, MissionID: missionID, Sequence: sequence, EventType: eventType, Producer: ledger.Producer{Type: "user", ID: "test"}, Payload: encoded}
}
