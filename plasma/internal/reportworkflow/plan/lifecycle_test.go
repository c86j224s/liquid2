package plan

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func TestRunReportPlanLifecycleUsesLifecycleServiceAndIDFactory(t *testing.T) {
	service := &planLifecycleService{selection: reporting.ReportPlanSubmissionSelection{EventID: "evt_submitted", ArgumentsHash: "args", PlanHash: "hash", Plan: mustLifecycleJSON(reporting.ReportPlan{Summary: "summary"})}}
	service.events = []ledger.Event{}
	runner := Runner{Service: service, NewID: func(prefix string) string { return prefix + "_stage" }, Lifecycle: reporting.Runner(reportexecution.Runner{Service: service, NewID: func(prefix string) string { return prefix + "_lifecycle" }})}
	result, err := runner.RunReportPlanLifecycle(context.Background(), ReportPlanLifecycleRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", ReportMode: reportexecution.ModePlanned, AgentExecutor: "codex",
		Invoke: func(_ context.Context, binding ReportPlanLifecycleBinding) (ReportPlanLifecycleAgentResult, error) {
			if binding.ToolSessionID != "ses_lifecycle" || binding.IdempotencyKey != "rpk_lifecycle" {
				t.Fatalf("unexpected lifecycle binding: %#v", binding)
			}
			return ReportPlanLifecycleAgentResult{Text: ReportPlanSubmittedSentinel}, nil
		},
		BuildCanonical: func(any, reporting.ReportPlanSubmissionSelection, ReportPlanLifecycleBinding) (ledger.AppendRequest, error) {
			return ledger.AppendRequest{EventID: "evt_canonical", MissionID: "mis_1", EventType: "report.plan.created", Payload: json.RawMessage(`{}`)}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Event.EventID != "evt_canonical" || service.query.ToolSessionID != "ses_lifecycle" || service.query.IdempotencyKey != "rpk_lifecycle" {
		t.Fatalf("lifecycle dependency was not used: %#v %#v", result, service.query)
	}
}

func TestRunReportPlanLifecycleUsesExactFallbackIDsWhenLifecycleFactoryNil(t *testing.T) {
	service := &planLifecycleService{selection: reporting.ReportPlanSubmissionSelection{EventID: "evt_submitted", Plan: mustLifecycleJSON(reporting.ReportPlan{Summary: "summary"})}}
	runner := Runner{Service: service, NewID: func(prefix string) string { return prefix + "_stage" }, Lifecycle: reporting.Runner(reportexecution.Runner{Service: service})}
	_, err := runner.RunReportPlanLifecycle(context.Background(), ReportPlanLifecycleRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", ReportMode: reportexecution.ModePlanned, AgentExecutor: "codex",
		Invoke: func(_ context.Context, binding ReportPlanLifecycleBinding) (ReportPlanLifecycleAgentResult, error) {
			if binding.ToolSessionID != "ses_report" || binding.IdempotencyKey != "rpk_report" {
				t.Fatalf("unexpected fallback IDs: %#v", binding)
			}
			return ReportPlanLifecycleAgentResult{Text: ReportPlanSubmittedSentinel}, nil
		},
		BuildCanonical: func(any, reporting.ReportPlanSubmissionSelection, ReportPlanLifecycleBinding) (ledger.AppendRequest, error) {
			return ledger.AppendRequest{EventID: "evt_canonical", MissionID: "mis_1", EventType: "report.plan.created", Payload: json.RawMessage(`{}`)}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

type planLifecycleService struct {
	events    []ledger.Event
	selection reporting.ReportPlanSubmissionSelection
	query     reporting.ReportPlanSubmissionQuery
}

func (service *planLifecycleService) AppendEvent(_ context.Context, req ledger.AppendRequest) (ledger.Event, error) {
	event := ledger.Event{EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Payload: req.Payload}
	service.events = append(service.events, event)
	return event, nil
}

func (service *planLifecycleService) AppendEvents(_ context.Context, _ string, reqs []ledger.AppendRequest) ([]ledger.Event, error) {
	return nil, nil
}

func (service *planLifecycleService) AppendReportTerminalIfOpen(context.Context, string, string, []ledger.AppendRequest) ([]ledger.Event, bool, error) {
	return nil, false, nil
}

func (service *planLifecycleService) AppendEventsIfNoActiveAgentWork(context.Context, string, []ledger.AppendRequest) ([]ledger.Event, error) {
	return nil, nil
}

func (service *planLifecycleService) ListEvents(context.Context, string) ([]ledger.Event, error) {
	return service.events, nil
}

func (service *planLifecycleService) ListSourceSnapshotsWithState(context.Context, source.ListRequest) ([]source.Snapshot, error) {
	return nil, nil
}

func (service *planLifecycleService) SelectReportPlanSubmission(_ context.Context, query reporting.ReportPlanSubmissionQuery) (reporting.ReportPlanSubmissionSelection, error) {
	service.query = query
	return service.selection, nil
}

func (service *planLifecycleService) PromoteReportPlan(_ context.Context, req reporting.PromoteReportPlanRequest) (ledger.Event, error) {
	return ledger.Event{EventID: req.Canonical.EventID, MissionID: req.MissionID, EventType: req.Canonical.EventType}, nil
}

func mustLifecycleJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func TestRunReportPlanLifecyclePromotesOnlyAfterExactSentinel(t *testing.T) {
	planValue := reporting.ReportPlan{Summary: "summary", Sections: []reporting.ReportPlanSection{{Title: "section", Purpose: "purpose"}}}
	_, encoded, err := reporting.ReportPlanHash(planValue)
	if err != nil {
		t.Fatal(err)
	}
	service := &planLifecycleService{selection: reporting.ReportPlanSubmissionSelection{EventID: "evt_submitted", ArgumentsHash: "args", PlanHash: "hash", Plan: encoded}}
	runner := Runner{Service: service, Lifecycle: reporting.Runner(reportexecution.Runner{Service: service, NewID: planTestID})}
	built := false
	result, err := runner.RunReportPlanLifecycle(context.Background(), ReportPlanLifecycleRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", ReportMode: reportexecution.ModePlanned, AgentExecutor: "codex", PreviousProviderSessionID: "ses_provider",
		Invoke: func(context.Context, ReportPlanLifecycleBinding) (ReportPlanLifecycleAgentResult, error) {
			return ReportPlanLifecycleAgentResult{Text: ReportPlanSubmittedSentinel, SessionID: "ses_provider"}, nil
		},
		BuildCanonical: func(value any, selection reporting.ReportPlanSubmissionSelection, binding ReportPlanLifecycleBinding) (ledger.AppendRequest, error) {
			built = true
			if _, ok := value.(reporting.ReportPlan); !ok || selection.EventID != "evt_submitted" || binding.ToolSessionID != "ses_tool" {
				t.Fatalf("unexpected lifecycle input: %#v %#v %#v", value, selection, binding)
			}
			return ledger.AppendRequest{EventID: "evt_created", MissionID: "mis_1", EventType: "report.plan.created", Producer: ledger.Producer{Type: "agent_session", ID: "ses_provider"}, Payload: json.RawMessage(`{"pending_event_id":"evt_pending"}`)}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !built || result.Event.EventID != "evt_created" || service.query.ToolSessionID != "ses_tool" {
		t.Fatalf("lifecycle did not promote exact submission: %#v %#v", result, service.query)
	}
}

func TestRunReportPlanLifecycleRejectsEveryNonExactSentinelBeforeSelection(t *testing.T) {
	for _, text := range []string{"", " PLAN_SUBMITTED ", `{"status":"PLAN_SUBMITTED"}`, "```PLAN_SUBMITTED```", "done PLAN_SUBMITTED", "PLAN_SUBMITTED\nextra"} {
		t.Run(text, func(t *testing.T) {
			service := &planLifecycleService{}
			_, err := (Runner{Service: service, Lifecycle: reporting.Runner(reportexecution.Runner{Service: service, NewID: planTestID})}).RunReportPlanLifecycle(context.Background(), ReportPlanLifecycleRequest{MissionID: "mis_1", PendingEventID: "evt_pending", ReportMode: reportexecution.ModePlanned, AgentExecutor: "codex", PreviousProviderSessionID: "ses_provider", Invoke: func(context.Context, ReportPlanLifecycleBinding) (ReportPlanLifecycleAgentResult, error) {
				return ReportPlanLifecycleAgentResult{Text: text}, nil
			}, BuildCanonical: func(any, reporting.ReportPlanSubmissionSelection, ReportPlanLifecycleBinding) (ledger.AppendRequest, error) {
				t.Fatal("canonical builder must not run")
				return ledger.AppendRequest{}, nil
			}})
			if err == nil || service.query.MissionID != "" {
				t.Fatalf("non-exact sentinel advanced lifecycle: %#v", service.query)
			}
		})
	}
}

func TestRunReportPlanLifecycleDoesNotInventFreshProviderSession(t *testing.T) {
	planValue := reporting.ReportPlan{Summary: "summary"}
	_, encoded, _ := reporting.ReportPlanHash(planValue)
	service := &planLifecycleService{selection: reporting.ReportPlanSubmissionSelection{EventID: "evt_submitted", ArgumentsHash: "args", PlanHash: "hash", Plan: encoded}}
	runner := Runner{Service: service, Lifecycle: reporting.Runner(reportexecution.Runner{Service: service, NewID: planTestID})}
	_, err := runner.RunReportPlanLifecycle(context.Background(), ReportPlanLifecycleRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", ReportMode: reportexecution.ModePlanned, AgentExecutor: "codex",
		Invoke: func(_ context.Context, binding ReportPlanLifecycleBinding) (ReportPlanLifecycleAgentResult, error) {
			if binding.ToolSessionID == "" {
				t.Fatal("tool session was not created")
			}
			return ReportPlanLifecycleAgentResult{Text: ReportPlanSubmittedSentinel, SessionID: "ses_returned_provider"}, nil
		},
		BuildCanonical: func(any, reporting.ReportPlanSubmissionSelection, ReportPlanLifecycleBinding) (ledger.AppendRequest, error) {
			return ledger.AppendRequest{EventID: "evt_created", MissionID: "mis_1", EventType: "report.plan.created", Producer: ledger.Producer{Type: "agent_session", ID: "ses_returned_provider"}, Payload: json.RawMessage(`{"pending_event_id":"evt_pending"}`)}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.query.PreviousProviderSessionID != "" {
		t.Fatalf("fresh provider provenance was invented: %#v", service.query)
	}
}

func planTestID(prefix string) string {
	if prefix == "ses" {
		return "ses_tool"
	}
	return "key_plan"
}
