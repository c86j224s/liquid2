package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestLedgerAppendAndRead(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	mission := app.Mission{MissionID: "mis_1", Title: "Mission"}
	if err := store.CreateMission(ctx, mission); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}

	first, err := store.AppendLedgerEvent(ctx, app.LedgerEvent{
		EventID:   "evt_1",
		MissionID: "mis_1",
		EventType: "mission.created",
		Producer:  app.Producer{Type: "user", ID: "ses_1"},
		Payload:   []byte(`{"title":"Mission"}`),
	})
	if err != nil {
		t.Fatalf("AppendLedgerEvent first returned error: %v", err)
	}
	second, err := store.AppendLedgerEvent(ctx, app.LedgerEvent{
		EventID:   "evt_2",
		MissionID: "mis_1",
		EventType: "mission.steered",
		Producer:  app.Producer{Type: "user", ID: "ses_1"},
		Payload:   []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("AppendLedgerEvent second returned error: %v", err)
	}
	if first.Sequence != 1 || second.Sequence != 2 {
		t.Fatalf("unexpected sequences: %d, %d", first.Sequence, second.Sequence)
	}

	events, err := store.ListLedgerEvents(ctx, "mis_1")
	if err != nil {
		t.Fatalf("ListLedgerEvents returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected two events, got %d", len(events))
	}
	if events[0].EventID != "evt_1" || events[1].EventID != "evt_2" {
		t.Fatalf("unexpected event order: %#v", events)
	}
}

func TestLedgerConditionalAppendReadsAndWritesInOneTransaction(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_1", Title: "Mission"}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	if _, err := store.AppendLedgerEvent(ctx, app.LedgerEvent{
		EventID:   "evt_1",
		MissionID: "mis_1",
		EventType: "mission.created",
		Producer:  app.Producer{Type: "user", ID: "ses_1"},
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatalf("AppendLedgerEvent returned error: %v", err)
	}
	appended, err := store.AppendLedgerEventsConditionally(ctx, "mis_1", func(events []app.LedgerEvent) ([]app.LedgerEvent, error) {
		if len(events) != 1 || events[0].EventID != "evt_1" {
			t.Fatalf("expected conditional builder to see existing event, got %#v", events)
		}
		return []app.LedgerEvent{{
			EventID:   "evt_2",
			MissionID: "mis_1",
			EventType: "mission.steered",
			Producer:  app.Producer{Type: "user", ID: "ses_1"},
			Payload:   []byte(`{}`),
		}, {
			EventID:   "evt_3",
			MissionID: "mis_1",
			EventType: "mission.note",
			Producer:  app.Producer{Type: "user", ID: "ses_1"},
			Payload:   []byte(`{}`),
		}}, nil
	})
	if err != nil {
		t.Fatalf("AppendLedgerEventsConditionally returned error: %v", err)
	}
	if len(appended) != 2 || appended[0].Sequence != 2 || appended[1].Sequence != 3 {
		t.Fatalf("unexpected appended events: %#v", appended)
	}
}

func TestLedgerRejectsUnknownMission(t *testing.T) {
	store := newTestStore(t)
	_, err := store.AppendLedgerEvent(context.Background(), app.LedgerEvent{
		EventID:   "evt_1",
		MissionID: "mis_missing",
		EventType: "mission.created",
		Producer:  app.Producer{Type: "user", ID: "ses_1"},
		Payload:   []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLedgerRejectsDuplicateEventID(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_1", Title: "Mission"}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	event := app.LedgerEvent{
		EventID:   "evt_1",
		MissionID: "mis_1",
		EventType: "mission.created",
		Producer:  app.Producer{Type: "user", ID: "ses_1"},
		Payload:   []byte(`{}`),
	}
	if _, err := store.AppendLedgerEvent(ctx, event); err != nil {
		t.Fatalf("first append returned error: %v", err)
	}
	if _, err := store.AppendLedgerEvent(ctx, event); err == nil {
		t.Fatal("expected duplicate event error")
	}
}

func TestWorkflowRunRejectsActiveRunAcrossServiceInstances(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	store1, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store1.Close()
	store2, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()
	if err := store1.CreateMission(ctx, app.Mission{MissionID: "mis_1", Title: "Mission"}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	svc1 := app.NewService(store1)
	svc2 := app.NewService(store2)
	if _, err := svc1.RequestWorkflowRun(ctx, app.RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_first",
		MissionID:          "mis_1",
		RequestedBySurface: app.WorkflowSurfaceCLI,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "first",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	}); err != nil {
		t.Fatalf("first RequestWorkflowRun returned error: %v", err)
	}
	if _, err := svc2.RequestWorkflowRun(ctx, app.RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_second",
		MissionID:          "mis_1",
		RequestedBySurface: app.WorkflowSurfaceWeb,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "second",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected second service to reject active workflow, got %v", err)
	}
}

func TestActiveAgentWorkRejectsConditionalAppendAcrossServiceInstances(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	store1, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store1.Close()
	store2, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()
	if err := store1.CreateMission(ctx, app.Mission{MissionID: "mis_1", Title: "Mission"}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	svc1 := app.NewService(store1)
	svc2 := app.NewService(store2)
	if _, err := svc1.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []app.AppendEventRequest{{
		EventID:   "evt_report_pending",
		MissionID: "mis_1",
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "user", ID: "web"},
		Payload:   []byte(`{"kind":"markdown_report_artifact_pending"}`),
	}}); err != nil {
		t.Fatalf("first conditional append returned error: %v", err)
	}
	_, err = svc2.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []app.AppendEventRequest{
		{
			EventID:   "evt_user_next",
			MissionID: "mis_1",
			EventType: "turn.user",
			Producer:  app.Producer{Type: "user", ID: "cli"},
			Payload:   []byte(`{"kind":"user_turn","text":"next"}`),
		},
		{
			EventID:   "evt_pending_next",
			MissionID: "mis_1",
			EventType: "turn.agent.pending",
			Producer:  app.Producer{Type: "agent", ID: "codex"},
			Payload:   []byte(`{"kind":"agent_pending","user_event_id":"evt_user_next","agent_executor":"codex"}`),
		},
	})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected second service to reject active report draft, got %v", err)
	}
	events, err := store1.ListLedgerEvents(ctx, "mis_1")
	if err != nil {
		t.Fatalf("ListLedgerEvents returned error: %v", err)
	}
	if len(events) != 1 || events[0].EventType != "report.draft.pending" {
		t.Fatalf("unexpected events after rejected conditional append: %#v", events)
	}
}

func TestWorkflowEventsUseMissionLedger(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_1", Title: "Mission"}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	svc := app.NewService(store)
	view, err := svc.RequestWorkflowRun(ctx, app.RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_sqlite",
		MissionID:          "mis_1",
		RequestedBySurface: app.WorkflowSurfaceCLI,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "Run one bounded workflow step.",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	})
	if err != nil {
		t.Fatalf("RequestWorkflowRun returned error: %v", err)
	}
	if view.Status != app.WorkflowStatusQueued {
		t.Fatalf("expected queued workflow, got %#v", view)
	}
	events, err := store.ListLedgerEvents(ctx, "mis_1")
	if err != nil {
		t.Fatalf("ListLedgerEvents returned error: %v", err)
	}
	if len(events) != 1 || events[0].EventType != app.WorkflowRunRequestedEvent {
		t.Fatalf("unexpected workflow events: %#v", events)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("Close returned error: %v", err)
		}
	})
	return store
}
