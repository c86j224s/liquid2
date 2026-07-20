package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestMissionLifecycleArchiveRestoreAndIdempotency(t *testing.T) {
	store := newLifecycleStore(t, "mis_1")
	svc := NewService(store)
	if _, err := svc.RebuildProjection(context.Background(), "mis_1"); err != nil {
		t.Fatal(err)
	}

	archived, err := svc.ArchiveMission(context.Background(), MissionLifecycleChangeRequest{
		EventID: "evt_archive", MissionID: "mis_1", Producer: Producer{Type: "user", ID: "test"},
		Reason: "done",
	})
	if err != nil {
		t.Fatal(err)
	}
	if archived.Event == nil || archived.Event.EventType != MissionArchivedEvent || archived.Projection.LifecycleState != MissionLifecycleArchived {
		t.Fatalf("archive result = %#v", archived)
	}
	missions, err := svc.ListMissions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 0 {
		t.Fatalf("archived mission must be hidden by default: %#v", missions)
	}
	missions, err = svc.ListMissionsWithState(context.Background(), ListMissionsRequest{IncludeArchived: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 1 || missions[0].LifecycleState != MissionLifecycleArchived {
		t.Fatalf("include archived missions = %#v", missions)
	}

	again, err := svc.ArchiveMission(context.Background(), MissionLifecycleChangeRequest{
		EventID: "evt_archive_again", MissionID: "mis_1", Producer: Producer{Type: "user", ID: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !again.Idempotent || again.Event != nil {
		t.Fatalf("idempotent archive = %#v", again)
	}

	restored, err := svc.RestoreMission(context.Background(), MissionLifecycleChangeRequest{
		EventID: "evt_restore", MissionID: "mis_1", Producer: Producer{Type: "user", ID: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if restored.Event == nil || restored.Event.EventType != MissionRestoredEvent || restored.Projection.LifecycleState != MissionLifecycleActive {
		t.Fatalf("restore result = %#v", restored)
	}
	missions, err = svc.ListMissions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 1 || missions[0].MissionID != "mis_1" || missions[0].LifecycleState != MissionLifecycleActive {
		t.Fatalf("restored default missions = %#v", missions)
	}
}

func TestArchiveMissionRejectsOpenActiveWork(t *testing.T) {
	store := newLifecycleStore(t, "mis_1")
	store.events["mis_1"] = append(store.events["mis_1"], lifecycleEvent(t, "evt_turn_pending", "mis_1", 2, "turn.agent.pending", map[string]any{"user_event_id": "evt_user"}))
	svc := NewService(store)
	_, err := svc.ArchiveMission(context.Background(), MissionLifecycleChangeRequest{
		EventID: "evt_archive", MissionID: "mis_1", Producer: Producer{Type: "user", ID: "test"},
	})
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "agent turn") {
		t.Fatalf("expected active work rejection, got %v", err)
	}
}

type lifecycleStore struct {
	fakeStore
	events      map[string][]LedgerEvent
	projections map[string]MissionProjection
	missions    map[string]Mission
}

func newLifecycleStore(t *testing.T, missionID string) *lifecycleStore {
	t.Helper()
	return &lifecycleStore{
		events: map[string][]LedgerEvent{
			missionID: []LedgerEvent{lifecycleEvent(t, "evt_created", missionID, 1, "mission.created", map[string]any{"title": "Mission", "objective": "Mission"})},
		},
		projections: map[string]MissionProjection{},
		missions: map[string]Mission{
			missionID: {MissionID: missionID, Title: "Mission", LifecycleState: MissionLifecycleActive},
		},
	}
}

func (s *lifecycleStore) ListMissions(context.Context) ([]Mission, error) {
	missions := make([]Mission, 0, len(s.missions))
	for _, mission := range s.missions {
		missions = append(missions, mission)
	}
	return missions, nil
}

func (s *lifecycleStore) ListMissionActivityInputs(_ context.Context, missionIDs []string) ([]MissionActivityInput, error) {
	inputs := make([]MissionActivityInput, 0, len(missionIDs))
	for _, missionID := range missionIDs {
		events := s.events[missionID]
		var lastSequence int64
		if len(events) > 0 {
			lastSequence = events[len(events)-1].Sequence
		}
		inputs = append(inputs, MissionActivityInput{MissionID: missionID, LastSequence: lastSequence})
	}
	return inputs, nil
}

func (s *lifecycleStore) AppendLedgerEventsConditionally(_ context.Context, missionID string, build func([]LedgerEvent) ([]LedgerEvent, error)) ([]LedgerEvent, error) {
	current := append([]LedgerEvent(nil), s.events[missionID]...)
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

func (s *lifecycleStore) ListLedgerEvents(_ context.Context, missionID string) ([]LedgerEvent, error) {
	return append([]LedgerEvent(nil), s.events[missionID]...), nil
}

func (s *lifecycleStore) SaveMissionProjection(_ context.Context, projection MissionProjection) error {
	s.projections[projection.MissionID] = projection
	mission := s.missions[projection.MissionID]
	mission.Title = projection.Title
	mission.LifecycleState = projection.LifecycleState
	s.missions[projection.MissionID] = mission
	return nil
}

func (s *lifecycleStore) GetMissionProjection(_ context.Context, missionID string) (MissionProjection, error) {
	return s.projections[missionID], nil
}

func lifecycleEvent(t *testing.T, id, missionID string, sequence int64, eventType string, payload any) LedgerEvent {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return LedgerEvent{EventID: id, MissionID: missionID, Sequence: sequence, EventType: eventType, Producer: Producer{Type: "user", ID: "test"}, Payload: encoded}
}
