package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func TestProjectionRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.CreateMission(ctx, mission.Mission{MissionID: "mis_1", Title: "Initial"}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}

	projection := mission.Projection{
		MissionID:             "mis_1",
		LastEventID:           "evt_2",
		LastSequence:          2,
		Title:                 "Projected",
		Objective:             "Build projection",
		Scope:                 mission.Scope{Included: []string{"ledger"}, Excluded: []string{"auth"}},
		ActiveSessionIDs:      []string{"ses_1"},
		AcceptedClaimIDs:      []string{"clm_1"},
		OpenQuestionIDs:       []string{"qst_1"},
		ActiveReportVersionID: "rvn_1",
		LifecycleState:        "active",
		NeedsReview:           true,
		NeedsReviewReasons:    []string{"conflict"},
	}
	if err := store.SaveMissionProjection(ctx, projection); err != nil {
		t.Fatalf("SaveMissionProjection returned error: %v", err)
	}
	got, err := store.GetMissionProjection(ctx, "mis_1")
	if err != nil {
		t.Fatalf("GetMissionProjection returned error: %v", err)
	}
	if got.Objective != projection.Objective || got.LastSequence != projection.LastSequence {
		t.Fatalf("unexpected projection: %#v", got)
	}
	if !got.NeedsReview || got.NeedsReviewReasons[0] != "conflict" {
		t.Fatalf("unexpected review state: %#v", got)
	}
}

func TestSaveMissionProjectionMissingMissionReturnsError(t *testing.T) {
	store := newTestStore(t)
	err := store.SaveMissionProjection(context.Background(), mission.Projection{
		MissionID:      "mis_missing",
		LifecycleState: "active",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestSaveMissionProjectionRejectsStaleSequence(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.CreateMission(ctx, mission.Mission{MissionID: "mis_1", Title: "Initial"}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}

	if err := store.SaveMissionProjection(ctx, mission.Projection{
		MissionID:      "mis_1",
		LastEventID:    "evt_2",
		LastSequence:   2,
		Title:          "Fresh",
		LifecycleState: "active",
	}); err != nil {
		t.Fatalf("SaveMissionProjection fresh returned error: %v", err)
	}
	err := store.SaveMissionProjection(ctx, mission.Projection{
		MissionID:      "mis_1",
		LastEventID:    "evt_1",
		LastSequence:   1,
		Title:          "Stale",
		LifecycleState: "active",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for stale projection, got %v", err)
	}

	got, err := store.GetMissionProjection(ctx, "mis_1")
	if err != nil {
		t.Fatalf("GetMissionProjection returned error: %v", err)
	}
	if got.LastSequence != 2 || got.Title != "Fresh" {
		t.Fatalf("stale projection overwrote cache: %#v", got)
	}
}

func TestServiceRebuildProjectionPersistsCache(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := app.NewService(store)
	if _, err := svc.CreateMission(ctx, mission.CreateRequest{MissionID: "mis_1", Title: "Mission"}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, ledger.AppendRequest{
		EventID:   "evt_1",
		MissionID: "mis_1",
		EventType: "mission.created",
		Producer:  ledger.Producer{Type: "user", ID: "ses_1"},
		Payload:   []byte(`{"title":"Mission","objective":"Projection"}`),
	}); err != nil {
		t.Fatalf("AppendEvent returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, ledger.AppendRequest{
		EventID:   "evt_2",
		MissionID: "mis_1",
		EventType: "session.attached",
		Producer:  ledger.Producer{Type: "user", ID: "ses_1"},
		Payload:   []byte(`{"session_id":"ses_1"}`),
	}); err != nil {
		t.Fatalf("AppendEvent returned error: %v", err)
	}

	projection, err := svc.RebuildProjection(ctx, "mis_1")
	if err != nil {
		t.Fatalf("RebuildProjection returned error: %v", err)
	}
	if projection.Objective != "Projection" || projection.LastSequence != 2 {
		t.Fatalf("unexpected rebuilt projection: %#v", projection)
	}

	cached, err := svc.GetProjection(ctx, "mis_1")
	if err != nil {
		t.Fatalf("GetProjection returned error: %v", err)
	}
	if cached.ActiveSessionIDs[0] != "ses_1" {
		t.Fatalf("unexpected cached projection: %#v", cached)
	}
}
