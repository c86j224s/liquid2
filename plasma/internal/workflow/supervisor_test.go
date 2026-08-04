package workflow

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

type supervisorStore struct {
	mu     sync.Mutex
	events []ledger.Event
	runs   []workflowstate.WorkflowRunView
}

func (store *supervisorStore) ListEvents(context.Context, string) ([]ledger.Event, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]ledger.Event(nil), store.events...), nil
}

func (store *supervisorStore) AppendEvents(_ context.Context, _ string, requests []ledger.AppendRequest) ([]ledger.Event, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	appended := make([]ledger.Event, 0, len(requests))
	for _, req := range requests {
		event := ledger.Event{
			EventID:   req.EventID,
			MissionID: req.MissionID,
			Sequence:  int64(len(store.events) + 1),
			EventType: req.EventType,
			Producer:  req.Producer,
			Payload:   req.Payload,
			CreatedAt: time.Now().UTC(),
		}
		store.events = append(store.events, event)
		appended = append(appended, event)
	}
	return appended, nil
}

func (store *supervisorStore) ListWorkflowRuns(context.Context, string) ([]workflowstate.WorkflowRunView, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]workflowstate.WorkflowRunView(nil), store.runs...), nil
}

type blockingRun struct {
	started chan struct{}
	done    chan struct{}
}

func (run *blockingRun) Run(ctx context.Context, _, _ string) (workflowstate.WorkflowRunView, error) {
	close(run.started)
	<-ctx.Done()
	close(run.done)
	return workflowstate.WorkflowRunView{}, ctx.Err()
}

func TestSupervisorStartsOnceAndCancelsOwnedRun(t *testing.T) {
	run := &blockingRun{started: make(chan struct{}), done: make(chan struct{})}
	supervisor := NewSupervisor(SupervisorOptions{
		Service: &supervisorStore{},
		RunnerFactory: func(context.Context, string, string) (RunExecutor, error) {
			return run, nil
		},
		NewID: func(string) string { return "run-owner" },
	})

	if !supervisor.Start("mis_test", "wfr_test", "codex") {
		t.Fatal("expected first start to claim run")
	}
	select {
	case <-run.started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start")
	}
	if supervisor.Start("mis_test", "wfr_test", "codex") {
		t.Fatal("expected duplicate start to be rejected")
	}
	if !supervisor.Has("wfr_test") || !supervisor.Cancel("wfr_test") {
		t.Fatal("expected owned run to be visible and cancelable")
	}
	select {
	case <-run.done:
	case <-time.After(time.Second):
		t.Fatal("runner did not observe cancellation")
	}
}

func TestSupervisorReconcileStartsDeferredRunAfterBoundTurnCompletes(t *testing.T) {
	run := &blockingRun{started: make(chan struct{}), done: make(chan struct{})}
	store := &supervisorStore{
		runs: []workflowstate.WorkflowRunView{{
			MissionID:         "mis_test",
			WorkflowRunID:     "wfr_test",
			Status:            workflowstate.WorkflowStatusQueued,
			StartAfterEventID: "evt_user",
			AgentExecutor:     "codex",
		}},
		events: []ledger.Event{{
			EventID:   "evt_response",
			MissionID: "mis_test",
			EventType: "turn.agent.response",
			Payload:   mustTestJSON(t, map[string]any{"user_event_id": "evt_user"}),
		}},
	}
	supervisor := NewSupervisor(SupervisorOptions{
		Service: store,
		RunnerFactory: func(context.Context, string, string) (RunExecutor, error) {
			return run, nil
		},
		AgentAvailable: func(name string) bool { return name == "codex" },
		NewID:          func(string) string { return "run-owner" },
	})

	if err := supervisor.Reconcile(context.Background(), "mis_test"); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	select {
	case <-run.started:
	case <-time.After(time.Second):
		t.Fatal("deferred run did not start")
	}
	supervisor.Cancel("wfr_test")
	<-run.done
}

func TestSupervisorStopRunNowClosesPendingAndRunTogether(t *testing.T) {
	store := &supervisorStore{events: []ledger.Event{
		{
			EventID:   "evt_requested",
			MissionID: "mis_test",
			Sequence:  1,
			EventType: workflowstate.WorkflowRunRequestedEvent,
			Payload: mustTestJSON(t, workflowstate.WorkflowRunRequestedPayload{
				WorkflowRunID: "wfr_test",
				MissionID:     "mis_test",
				Instruction:   "continue",
				MaxSteps:      2,
			}),
		},
		{
			EventID:   "evt_pending",
			MissionID: "mis_test",
			Sequence:  2,
			EventType: "turn.agent.pending",
			Payload: mustTestJSON(t, map[string]any{
				"user_event_id":   "evt_user",
				"agent_executor":  "codex",
				"workflow_run_id": "wfr_test",
			}),
		},
	}}
	next := 0
	supervisor := NewSupervisor(SupervisorOptions{
		Service: store,
		NewID: func(string) string {
			next++
			return "evt_new_" + string(rune('0'+next))
		},
	})

	if _, err := supervisor.StopRunNow(context.Background(), "mis_test", "wfr_test", "stop now"); err != nil {
		t.Fatalf("stop run: %v", err)
	}
	events, _ := store.ListEvents(context.Background(), "mis_test")
	if got := events[len(events)-2].EventType; got != "turn.agent.response" {
		t.Fatalf("first terminal event = %q", got)
	}
	if got := events[len(events)-1].EventType; got != workflowstate.WorkflowRunStoppedEvent {
		t.Fatalf("workflow terminal event = %q", got)
	}
}

func mustTestJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
