package reporting

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type reportPlanLifecycleFakeService struct {
	*fakeRunnerService
	selection app.ReportPlanSubmissionSelection
	query     app.ReportPlanSubmissionQuery
	promote   app.PromoteReportPlanRequest
}

func (service *reportPlanLifecycleFakeService) SelectReportPlanSubmission(_ context.Context, query app.ReportPlanSubmissionQuery) (app.ReportPlanSubmissionSelection, error) {
	service.query = query
	return service.selection, nil
}

func (service *reportPlanLifecycleFakeService) PromoteReportPlan(_ context.Context, req app.PromoteReportPlanRequest) (app.LedgerEvent, error) {
	service.promote = req
	return app.LedgerEvent{EventID: req.Canonical.EventID, MissionID: req.MissionID, EventType: "report.plan.created", Payload: req.Canonical.Payload}, nil
}

func TestRunReportPlanLifecyclePromotesOnlyAfterExactSentinel(t *testing.T) {
	plan := ReportPlan{Summary: "summary", Sections: []ReportPlanSection{{Title: "section", Purpose: "purpose"}}}
	_, encoded, err := ReportPlanHash(plan)
	if err != nil {
		t.Fatal(err)
	}
	service := &reportPlanLifecycleFakeService{fakeRunnerService: &fakeRunnerService{}, selection: app.ReportPlanSubmissionSelection{EventID: "evt_submitted", ArgumentsHash: "args", PlanHash: "hash", Plan: encoded}}
	runner := Runner{Service: service, NewID: func(prefix string) string {
		if prefix == "ses" {
			return "ses_tool"
		}
		return "key_plan"
	}}
	built := false
	result, err := runner.RunReportPlanLifecycle(context.Background(), ReportPlanLifecycleRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", ReportMode: ModePlanned, AgentExecutor: "codex", PreviousProviderSessionID: "ses_provider",
		Invoke: func(context.Context, ReportPlanLifecycleBinding) (ReportPlanLifecycleAgentResult, error) {
			return ReportPlanLifecycleAgentResult{Text: ReportPlanSubmittedSentinel, SessionID: "ses_provider"}, nil
		},
		BuildCanonical: func(value any, selection app.ReportPlanSubmissionSelection, binding ReportPlanLifecycleBinding) (app.AppendEventRequest, error) {
			built = true
			if _, ok := value.(ReportPlan); !ok || selection.EventID != "evt_submitted" || binding.ToolSessionID != "ses_tool" {
				t.Fatalf("unexpected lifecycle input: %#v %#v %#v", value, selection, binding)
			}
			return app.AppendEventRequest{EventID: "evt_created", MissionID: "mis_1", EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "ses_provider"}, Payload: json.RawMessage(`{"pending_event_id":"evt_pending","report_mode":"planned","plan":{}}`)}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !built || result.Event.EventID != "evt_created" || service.promote.SubmissionEventID != "evt_submitted" || service.promote.ToolSessionID != "ses_tool" {
		t.Fatalf("lifecycle did not promote exact submission: %#v %#v", result, service.promote)
	}
}

func TestRunReportPlanLifecycleRejectsEveryNonExactSentinelBeforeSelection(t *testing.T) {
	for _, text := range []string{"", " PLAN_SUBMITTED ", `{"status":"PLAN_SUBMITTED"}`, "```PLAN_SUBMITTED```", "done PLAN_SUBMITTED", "PLAN_SUBMITTED\nextra"} {
		t.Run(text, func(t *testing.T) {
			service := &reportPlanLifecycleFakeService{fakeRunnerService: &fakeRunnerService{}}
			runner := Runner{Service: service, NewID: testRunnerID}
			_, err := runner.RunReportPlanLifecycle(context.Background(), ReportPlanLifecycleRequest{MissionID: "mis_1", PendingEventID: "evt_pending", ReportMode: ModePlanned, AgentExecutor: "codex", PreviousProviderSessionID: "ses_provider", Invoke: func(context.Context, ReportPlanLifecycleBinding) (ReportPlanLifecycleAgentResult, error) {
				return ReportPlanLifecycleAgentResult{Text: text}, nil
			}, BuildCanonical: func(any, app.ReportPlanSubmissionSelection, ReportPlanLifecycleBinding) (app.AppendEventRequest, error) {
				t.Fatal("canonical builder must not run")
				return app.AppendEventRequest{}, nil
			}})
			if err == nil {
				t.Fatalf("expected sentinel %q to fail", text)
			}
			if service.query.MissionID != "" || service.promote.MissionID != "" {
				t.Fatalf("non-sentinel advanced lifecycle: %#v %#v", service.query, service.promote)
			}
		})
	}
}

func TestRunReportPlanLifecycleDoesNotInventFreshProviderSession(t *testing.T) {
	plan := ReportPlan{Summary: "summary"}
	_, encoded, _ := ReportPlanHash(plan)
	service := &reportPlanLifecycleFakeService{fakeRunnerService: &fakeRunnerService{}, selection: app.ReportPlanSubmissionSelection{EventID: "evt_submitted", ArgumentsHash: "args", PlanHash: "hash", Plan: encoded}}
	runner := Runner{Service: service, NewID: testRunnerID}
	_, err := runner.RunReportPlanLifecycle(context.Background(), ReportPlanLifecycleRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", ReportMode: ModePlanned, AgentExecutor: "codex",
		Invoke: func(_ context.Context, binding ReportPlanLifecycleBinding) (ReportPlanLifecycleAgentResult, error) {
			if binding.ToolSessionID == "" {
				t.Fatal("tool session was not created")
			}
			return ReportPlanLifecycleAgentResult{Text: ReportPlanSubmittedSentinel, SessionID: "ses_returned_provider"}, nil
		},
		BuildCanonical: func(any, app.ReportPlanSubmissionSelection, ReportPlanLifecycleBinding) (app.AppendEventRequest, error) {
			return app.AppendEventRequest{EventID: "evt_created", MissionID: "mis_1", EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "ses_returned_provider"}, Payload: json.RawMessage(`{"pending_event_id":"evt_pending"}`)}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.query.PreviousProviderSessionID != "" || service.promote.PreviousProviderSessionID != "" {
		t.Fatalf("fresh provider provenance was invented: %#v %#v", service.query, service.promote)
	}
}
