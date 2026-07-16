package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestReportPlanSubmissionAndPromotionAreAtomicAndReplayable(t *testing.T) {
	store := newTestStore(t)
	store.db.SetMaxOpenConns(1)
	ctx := context.Background()
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_plan", Title: "Plan"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendLedgerEvent(ctx, app.LedgerEvent{EventID: "evt_pending", MissionID: "mis_plan", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "user"}, Payload: []byte(`{"report_mode":"planned"}`)}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendLedgerEvent(ctx, app.LedgerEvent{EventID: "evt_pending_other", MissionID: "mis_plan", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "user"}, Payload: []byte(`{"report_mode":"planned"}`)}); err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	request := func(eventID string) app.ReportPlanSubmissionRequest {
		return app.ReportPlanSubmissionRequest{EventID: eventID, MissionID: "mis_plan", PendingEventID: "evt_pending", ReportMode: "planned", ToolSessionID: "ses_tool_new", PreviousProviderSessionID: "ses_provider", AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "high", IdempotencyKey: "key", ArgumentsHash: "args", PlanHash: "plan", Plan: json.RawMessage(`{"summary":"summary","sections":[]}`), Attempt: 1, ToolProducer: app.Producer{Type: "agent_session", ID: "ses_tool_new"}}
	}
	const callers = 12
	results := make(chan app.ReportPlanSubmission, callers)
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			result, err := svc.SubmitReportPlan(ctx, request(fmt.Sprintf("evt_submit_%d", i)))
			results <- result
			errs <- err
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent submit: %v", err)
		}
	}
	submissionID := ""
	for result := range results {
		if submissionID == "" {
			submissionID = result.Event.EventID
		}
		if result.Event.EventID != submissionID {
			t.Fatalf("multiple submissions: %s and %s", submissionID, result.Event.EventID)
		}
		if result.Event.Producer.Type != "mcp_server" || result.Event.Producer.ID != "plasma.report.plan.submit" {
			t.Fatalf("submission event producer is not server-owned: %#v", result.Event.Producer)
		}
	}

	canonical := app.AppendEventRequest{EventID: "evt_canonical", MissionID: "mis_plan", EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "ses_provider"}, Payload: json.RawMessage(`{"pending_event_id":"evt_pending","report_mode":"planned","plan":{"summary":"summary","sections":[]}}`)}
	promote := app.PromoteReportPlanRequest{MissionID: "mis_plan", PendingEventID: "evt_pending", ReportMode: "planned", ToolSessionID: "ses_tool_new", PreviousProviderSessionID: "ses_provider", AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "high", IdempotencyKey: "key", ArgumentsHash: "args", PlanHash: "plan", SubmissionEventID: submissionID, Canonical: canonical}
	first, err := svc.PromoteReportPlan(ctx, promote)
	if err != nil {
		t.Fatal(err)
	}
	second, err := app.NewService(store).PromoteReportPlan(ctx, promote)
	if err != nil {
		t.Fatal(err)
	}
	if first.EventID != second.EventID {
		t.Fatalf("promotion replay changed event: %s %s", first.EventID, second.EventID)
	}
	mismatches := map[string]func(*app.PromoteReportPlanRequest){
		"executor":                  func(req *app.PromoteReportPlanRequest) { req.AgentExecutor = "claude" },
		"model":                     func(req *app.PromoteReportPlanRequest) { req.AgentModel = "other" },
		"effort":                    func(req *app.PromoteReportPlanRequest) { req.AgentReasoningEffort = "low" },
		"previous provider session": func(req *app.PromoteReportPlanRequest) { req.PreviousProviderSessionID = "ses_other" },
		"idempotency key":           func(req *app.PromoteReportPlanRequest) { req.IdempotencyKey = "other" },
		"arguments hash":            func(req *app.PromoteReportPlanRequest) { req.ArgumentsHash = "other" },
		"plan hash":                 func(req *app.PromoteReportPlanRequest) { req.PlanHash = "other" },
		"tool session":              func(req *app.PromoteReportPlanRequest) { req.ToolSessionID = "ses_other" },
		"pending":                   func(req *app.PromoteReportPlanRequest) { req.PendingEventID = "evt_pending_other" },
		"mode":                      func(req *app.PromoteReportPlanRequest) { req.ReportMode = "long_form" },
	}
	for name, mutate := range mismatches {
		t.Run("canonical replay "+name, func(t *testing.T) {
			request := promote
			mutate(&request)
			if _, err := app.NewService(store).PromoteReportPlan(ctx, request); err == nil {
				t.Fatal("canonical replay accepted mismatched semantic binding")
			}
		})
	}
	var replayWG sync.WaitGroup
	for i := 0; i < callers; i++ {
		replayWG.Add(1)
		go func(mismatch bool) {
			defer replayWG.Done()
			request := promote
			if mismatch {
				request.AgentModel = "other"
			}
			event, replayErr := app.NewService(store).PromoteReportPlan(ctx, request)
			if mismatch && replayErr == nil {
				t.Errorf("concurrent mismatched replay succeeded: %#v", event)
			}
			if !mismatch && (replayErr != nil || event.EventID != first.EventID) {
				t.Errorf("concurrent exact replay failed: %#v %v", event, replayErr)
			}
		}(i%2 == 1)
	}
	replayWG.Wait()
	events, err := store.ListLedgerEvents(ctx, "mis_plan")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, event := range events {
		counts[event.EventType]++
	}
	if counts["report.plan.submitted"] != 1 || counts["report.plan.created"] != 1 {
		t.Fatalf("unexpected event counts: %#v", counts)
	}
}

func TestReportPlanSubmissionRejectsStalePromotionAndConflict(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_stale", Title: "Stale"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendLedgerEvent(ctx, app.LedgerEvent{EventID: "evt_pending", MissionID: "mis_stale", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "user"}, Payload: []byte(`{"report_mode":"planned"}`)}); err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	req := app.ReportPlanSubmissionRequest{EventID: "evt_old", MissionID: "mis_stale", PendingEventID: "evt_pending", ReportMode: "planned", ToolSessionID: "ses_old", PreviousProviderSessionID: "ses_provider_old", AgentExecutor: "codex", IdempotencyKey: "key_old", ArgumentsHash: "args_old", PlanHash: "plan_old", Plan: json.RawMessage(`{"summary":"old","sections":[]}`), Attempt: 1, ToolProducer: app.Producer{Type: "agent_session", ID: "ses_old"}}
	old, err := svc.SubmitReportPlan(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	req.EventID, req.ToolSessionID, req.PreviousProviderSessionID, req.IdempotencyKey, req.ArgumentsHash, req.PlanHash, req.Plan = "evt_new", "ses_new", "ses_provider_new", "key_new", "args_new", "plan_new", json.RawMessage(`{"summary":"new","sections":[]}`)
	req.ToolProducer.ID = req.ToolSessionID
	current, err := svc.SubmitReportPlan(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	canonical := app.AppendEventRequest{EventID: "evt_created", MissionID: "mis_stale", EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "ses_provider_new"}, Payload: json.RawMessage(`{"pending_event_id":"evt_pending","report_mode":"planned","plan":{"summary":"new","sections":[]}}`)}
	promote := app.PromoteReportPlanRequest{MissionID: "mis_stale", PendingEventID: "evt_pending", ReportMode: "planned", ToolSessionID: "ses_new", PreviousProviderSessionID: "ses_provider_new", AgentExecutor: "codex", IdempotencyKey: "key_new", ArgumentsHash: "args_new", PlanHash: "plan_new", SubmissionEventID: old.Event.EventID, Canonical: canonical}
	if _, err := svc.PromoteReportPlan(ctx, promote); err == nil {
		t.Fatal("expected stale submission promotion to fail")
	}
	promote.SubmissionEventID = current.Event.EventID
	if _, err := svc.PromoteReportPlan(ctx, promote); err != nil {
		t.Fatal(err)
	}
	promote.SubmissionEventID, promote.ToolSessionID = old.Event.EventID, "ses_old"
	if _, err := svc.PromoteReportPlan(ctx, promote); err == nil {
		t.Fatal("expected competing promotion to fail")
	}
}

func TestReportPlanSubmissionReplayRejectsEveryBindingMismatchAfterRestart(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_replay", Title: "Replay"}); err != nil {
		t.Fatal(err)
	}
	for _, event := range []app.LedgerEvent{
		{EventID: "evt_pending", MissionID: "mis_replay", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "user"}, Payload: []byte(`{"report_mode":"planned","agent_executor":"codex"}`)},
		{EventID: "evt_pending_other", MissionID: "mis_replay", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "user"}, Payload: []byte(`{"report_mode":"planned","agent_executor":"codex"}`)},
	} {
		if _, err := store.AppendLedgerEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	base := app.ReportPlanSubmissionRequest{
		EventID: "evt_submit", MissionID: "mis_replay", PendingEventID: "evt_pending", ReportMode: "planned",
		ToolSessionID: "ses_tool", PreviousProviderSessionID: "ses_previous", AgentExecutor: "codex",
		AgentModel: "gpt-test", AgentReasoningEffort: "high", IdempotencyKey: "key", ArgumentsHash: "args", PlanHash: "plan",
		Plan: json.RawMessage(`{"summary":"summary"}`), Attempt: 1, ToolProducer: app.Producer{Type: "agent_session", ID: "ses_tool"},
	}
	if _, err := app.NewService(store).SubmitReportPlan(ctx, base); err != nil {
		t.Fatal(err)
	}
	replay := base
	replay.EventID = "evt_replay"
	result, err := app.NewService(store).SubmitReportPlan(ctx, replay)
	if err != nil || !result.Replay || result.Event.EventID != "evt_submit" {
		t.Fatalf("exact restart replay failed: %#v %v", result, err)
	}
	cases := map[string]func(*app.ReportPlanSubmissionRequest){
		"pending": func(req *app.ReportPlanSubmissionRequest) { req.PendingEventID = "evt_pending_other" },
		"mode":    func(req *app.ReportPlanSubmissionRequest) { req.ReportMode = "long_form" },
		"tool session": func(req *app.ReportPlanSubmissionRequest) {
			req.ToolSessionID = "ses_other"
			req.ToolProducer.ID = "ses_other"
		},
		"previous session": func(req *app.ReportPlanSubmissionRequest) { req.PreviousProviderSessionID = "ses_other" },
		"executor":         func(req *app.ReportPlanSubmissionRequest) { req.AgentExecutor = "claude" },
		"model":            func(req *app.ReportPlanSubmissionRequest) { req.AgentModel = "other" },
		"effort":           func(req *app.ReportPlanSubmissionRequest) { req.AgentReasoningEffort = "low" },
		"producer":         func(req *app.ReportPlanSubmissionRequest) { req.ToolProducer.Type = "other" },
		"idempotency key":  func(req *app.ReportPlanSubmissionRequest) { req.IdempotencyKey = "other" },
		"arguments hash":   func(req *app.ReportPlanSubmissionRequest) { req.ArgumentsHash = "other" },
		"plan hash":        func(req *app.ReportPlanSubmissionRequest) { req.PlanHash = "other" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			req := base
			req.EventID = "evt_conflict_" + strings.ReplaceAll(name, " ", "_")
			mutate(&req)
			if _, err := app.NewService(store).SubmitReportPlan(ctx, req); err == nil {
				t.Fatal("binding mismatch was accepted as replay")
			}
		})
	}
}
