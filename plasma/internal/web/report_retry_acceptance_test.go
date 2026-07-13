package web

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

type retryBlockingExecutor struct {
	started  chan struct{}
	release  chan struct{}
	canceled chan struct{}
	calls    atomic.Int32
}

func (e *retryBlockingExecutor) Run(ctx context.Context, _ AgentRequest) (AgentResult, error) {
	e.calls.Add(1)
	select {
	case e.started <- struct{}{}:
	default:
	}
	select {
	case <-e.release:
		return AgentResult{Text: `{"summary":"s","parts":[{"title":"p","sections":[{"title":"s"}]}]}`, SessionID: "ses"}, nil
	case <-ctx.Done():
		close(e.canceled)
		return AgentResult{}, ctx.Err()
	}
}

func TestReportRetryHTTPIdempotencyStartsOneDetachedWorker(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	exec := &retryBlockingExecutor{started: make(chan struct{}, 2), release: make(chan struct{}), canceled: make(chan struct{})}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: exec}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "retry"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{{EventID: "evt_failed", MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"title": "retry", "agent_executor": "codex", "mcp_mode": "auto", "report_mode": "long_form"})}, {EventID: "evt_terminal", MissionID: missionID, EventType: "report.draft.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(map[string]any{"pending_event_id": "evt_failed", "kind": "report_draft_failed", "failed_stage_id": "plan"})}}); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"failed_pending_event_id": "evt_failed", "strategy": "resume_failed", "retry_request_id": "request_1"}
	first := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/retry", body)
	second := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/retry", body)
	firstID := nestedString(t, first, "pending_event", "EventID")
	if firstID == "" || firstID != nestedString(t, second, "pending_event", "EventID") {
		t.Fatalf("idempotency mismatch: %#v %#v", first, second)
	}
	select {
	case <-exec.started:
	case <-time.After(time.Second):
		t.Fatal("retry worker did not start")
	}
	time.Sleep(20 * time.Millisecond)
	if exec.calls.Load() != 1 {
		t.Fatalf("expected one worker, got %d", exec.calls.Load())
	}
	select {
	case <-exec.canceled:
		t.Fatal("accepted retry worker was canceled with the completed HTTP request")
	default:
	}
	close(exec.release)
}
