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
	if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{{EventID: "evt_failed", MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"title": "retry", "agent_executor": "codex", "mcp_mode": "auto", "report_mode": "long_form", "source_context": map[string]any{"schema_version": "plasma.report_source_context.v1", "captured_at": "2026-07-14T01:02:03Z", "confluence_sources": []any{}}})}, {EventID: "evt_terminal", MissionID: missionID, EventType: "report.draft.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(map[string]any{"pending_event_id": "evt_failed", "kind": "report_draft_failed", "failed_stage_id": "plan"})}}); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"failed_pending_event_id": "evt_failed", "strategy": "resume_failed", "retry_request_id": "request_1"}
	first := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/retry", body)
	second := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/retry", body)
	firstID := nestedString(t, first, "pending_event", "EventID")
	if firstID == "" || firstID != nestedString(t, second, "pending_event", "EventID") {
		t.Fatalf("idempotency mismatch: %#v %#v", first, second)
	}
	if capturedAt := nestedString(t, first, "pending_event", "Payload", "source_context", "captured_at"); capturedAt != "2026-07-14T01:02:03Z" {
		t.Fatalf("retry recaptured report source context: %#v", first)
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

func TestReportRetryResumeFailedReusesLongFormStagesAndFinalizes(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	delegate := &fakeAgentExecutor{responses: []AgentResult{
		{Text: agentReportAnyJSON(agentSectionalReportPlan{Summary: "Plan", Parts: []agentReportPart{{Title: "Part", Sections: []agentReportSection{{Title: "Section"}}}}}), SessionID: "ses-report"},
		{Text: "Section body.", SessionID: "ses-report"},
		{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: "ses-report"},
		{Text: "invalid final frame one", SessionID: "ses-report"},
		{Text: "invalid final frame two", SessionID: "ses-report"},
		{Text: `{"front_matter":"# Recovered report","closing":"## Close"}`, SessionID: "ses-report"},
	}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, delegate)}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "resume failed finalization"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Report", "report_mode": "long_form", "post_report_humanize": "disabled",
	})
	failed := waitForEventType(t, server.URL, missionID, "report.draft.failed")
	var originalPendingID string
	for _, raw := range failed["events"].([]any) {
		event := raw.(map[string]any)
		if event["EventType"] == "report.draft.failed" {
			originalPendingID = nestedString(t, event, "Payload", "pending_event_id")
		}
	}
	if originalPendingID == "" {
		t.Fatalf("failed pending id missing: %#v", failed["events"])
	}
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/retry", map[string]any{
		"failed_pending_event_id": originalPendingID, "strategy": "resume_failed", "retry_request_id": "retry-final-only",
	})
	var detail map[string]any
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		detail = getJSON(t, server.URL+"/api/missions/"+missionID)
		if countEvents(detail, "report.artifact.created") == 1 {
			break
		}
		if countEvents(detail, "report.draft.failed") > 1 {
			t.Fatalf("resume_failed finalization failed: events=%#v", detail["events"])
		}
		time.Sleep(20 * time.Millisecond)
	}
	if countEvents(detail, "report.artifact.created") != 1 {
		t.Fatalf("resume_failed finalization timed out: events=%#v", detail["events"])
	}
	for eventType, want := range map[string]int{
		"report.draft.pending":    2,
		"report.plan.created":     1,
		"report.section.created":  1,
		"report.part.created":     1,
		"report.draft.failed":     1,
		"report.artifact.created": 1,
	} {
		if got := countEvents(detail, eventType); got != want {
			t.Fatalf("%s count=%d, want %d: %#v", eventType, got, want, detail["events"])
		}
	}
	if len(delegate.requests) != 6 || delegate.requests[5].LongFormFinalize == nil {
		t.Fatalf("resume_failed regenerated stages instead of final-only recovery: %#v", delegate.requests)
	}
}
