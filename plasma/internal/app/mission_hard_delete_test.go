package app

import (
	"context"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"testing"
)

func TestMissionHardDeleteRequiresArchivedMissionAndConfirmation(t *testing.T) {
	ctx := context.Background()
	store := newHardDeleteLifecycleStore(t, "mis_1")
	svc := NewService(store)
	if _, err := svc.RebuildProjection(ctx, "mis_1"); err != nil {
		t.Fatal(err)
	}

	preview, err := svc.PreviewMissionHardDelete(ctx, "mis_1")
	if err != nil {
		t.Fatal(err)
	}
	if preview.Eligible || len(preview.BlockingReasons) != 1 || preview.BlockingReasons[0].ReasonCode != MissionHardDeleteBlockerNotArchived {
		t.Fatalf("active mission preview = %#v", preview)
	}
	if _, err := svc.HardDeleteMission(ctx, MissionHardDeleteRequest{
		MissionID: "mis_1", ConfirmMissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "test"},
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected active mission hard delete conflict, got %v", err)
	}
	if store.deleted {
		t.Fatal("active mission was deleted")
	}

	if _, err := svc.ArchiveMission(ctx, mission.MissionLifecycleChangeRequest{
		EventID: "evt_archive", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "test"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.HardDeleteMission(ctx, MissionHardDeleteRequest{
		MissionID: "mis_1", ConfirmMissionID: "mis_wrong", Producer: ledger.Producer{Type: "user", ID: "test"},
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected mismatched confirmation rejection, got %v", err)
	}
	result, err := svc.HardDeleteMission(ctx, MissionHardDeleteRequest{
		MissionID: "mis_1", ConfirmMissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Deleted || result.Impact.RawArtifacts != 2 || !store.deleted {
		t.Fatalf("hard delete result = %#v deleted=%v", result, store.deleted)
	}
}

func TestMissionHardDeleteRechecksActiveWorkAtDeleteTime(t *testing.T) {
	ctx := context.Background()
	store := newHardDeleteLifecycleStore(t, "mis_1")
	svc := NewService(store)
	if _, err := svc.RebuildProjection(ctx, "mis_1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ArchiveMission(ctx, mission.MissionLifecycleChangeRequest{
		EventID: "evt_archive", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "test"},
	}); err != nil {
		t.Fatal(err)
	}
	events := append([]ledger.Event(nil), store.events["mis_1"]...)
	events = append(events, lifecycleEvent(t, "evt_turn_pending", "mis_1", int64(len(events)+1), "turn.agent.pending", map[string]any{"user_event_id": "evt_user"}))
	store.deleteEvents = events

	_, err := svc.HardDeleteMission(ctx, MissionHardDeleteRequest{
		MissionID: "mis_1", ConfirmMissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "test"},
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected delete-time active work conflict, got %v", err)
	}
	if store.deleted {
		t.Fatal("mission was deleted after delete-time active work appeared")
	}
}

type hardDeleteLifecycleStore struct {
	*lifecycleStore
	impact       MissionHardDeleteImpact
	deleteEvents []ledger.Event
	deleted      bool
}

func newHardDeleteLifecycleStore(t *testing.T, missionID string) *hardDeleteLifecycleStore {
	return &hardDeleteLifecycleStore{
		lifecycleStore: newLifecycleStore(t, missionID),
		impact:         MissionHardDeleteImpact{LedgerEvents: 2, RawArtifacts: 2, RawArtifactBytes: 2048},
	}
}

func (s *hardDeleteLifecycleStore) PreviewMissionHardDelete(context.Context, string) (MissionHardDeleteImpact, error) {
	return s.impact, nil
}

func (s *hardDeleteLifecycleStore) HardDeleteMission(_ context.Context, missionID string, validate func([]ledger.Event) error) (MissionHardDeleteImpact, error) {
	events := append([]ledger.Event(nil), s.events[missionID]...)
	if s.deleteEvents != nil {
		events = append([]ledger.Event(nil), s.deleteEvents...)
	}
	if validate != nil {
		if err := validate(events); err != nil {
			return MissionHardDeleteImpact{}, err
		}
	}
	s.deleted = true
	delete(s.events, missionID)
	delete(s.projections, missionID)
	delete(s.missions, missionID)
	return s.impact, nil
}
