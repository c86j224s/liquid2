package requirements

import (
	"context"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func TestRunReportRequirementMapLifecycleUsesLifecycleServiceAndIDFactory(t *testing.T) {
	plan := reporting.SectionalReportPlan{Parts: []reporting.ReportPlanPart{{Title: "Part", Sections: []reporting.ReportPlanSection{{Title: "Section"}}}}}
	value := reporting.ReportRequirementMap{ReviewedEventIDs: []string{"evt_pending"}, Requirements: []reporting.ReportRequirement{{RequirementID: "req_one", Instruction: "include one", SourceEventIDs: []string{"evt_pending"}, Owner: &reporting.ReportRequirementOwner{PartIndex: 1, SectionIndex: 1}}}}
	hash, encoded, err := reporting.ReportRequirementMapHash(value)
	if err != nil {
		t.Fatal(err)
	}
	service := &requirementLifecycleService{selection: reporting.ReportRequirementMapSelection{Event: ledger.Event{EventID: "evt_map"}, RequirementMapHash: hash, RequirementMap: encoded}}
	runner := Runner{Service: service, Lifecycle: reporting.Runner(reportexecution.Runner{Service: service, NewID: func(prefix string) string { return prefix + "_lifecycle" }})}
	result, err := runner.RunReportRequirementMapLifecycle(context.Background(), ReportRequirementMapLifecycleRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", AgentExecutor: "codex", PreviousProviderSessionID: "ses_plan", Plan: plan,
		Invoke: func(_ context.Context, binding reporting.ReportRequirementMapBinding) (ReportRequirementMapAgentResult, error) {
			if binding.ToolSessionID != "ses_lifecycle" || binding.IdempotencyKey != "rrk_lifecycle" || binding.Producer.ID != "ses_lifecycle" {
				t.Fatalf("unexpected lifecycle binding: %#v", binding)
			}
			return ReportRequirementMapAgentResult{Text: ReportRequirementsMappedSentinel, SessionID: "ses_plan"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Event.EventID != "evt_map" || service.query.ToolSessionID != "ses_lifecycle" || service.query.IdempotencyKey != "rrk_lifecycle" {
		t.Fatalf("lifecycle dependency was not used: %#v %#v", result, service.query)
	}
	if len(service.events) != 1 || service.events[0].EventType != reporting.ReportRequirementsStartedEventType {
		t.Fatalf("required start event was not appended before invoke: %#v", service.events)
	}
}

func TestRunReportRequirementMapLifecycleUsesExactFallbackIDsWhenLifecycleFactoryNil(t *testing.T) {
	plan := reporting.SectionalReportPlan{Parts: []reporting.ReportPlanPart{{Title: "Part", Sections: []reporting.ReportPlanSection{{Title: "Section"}}}}}
	value := reporting.ReportRequirementMap{ReviewedEventIDs: []string{"evt_pending"}, Requirements: []reporting.ReportRequirement{{RequirementID: "req_one", Instruction: "include one", SourceEventIDs: []string{"evt_pending"}, Owner: &reporting.ReportRequirementOwner{PartIndex: 1, SectionIndex: 1}}}}
	hash, encoded, err := reporting.ReportRequirementMapHash(value)
	if err != nil {
		t.Fatal(err)
	}
	service := &requirementLifecycleService{selection: reporting.ReportRequirementMapSelection{Event: ledger.Event{EventID: "evt_map"}, RequirementMapHash: hash, RequirementMap: encoded}}
	runner := Runner{Service: service, Lifecycle: reporting.Runner(reportexecution.Runner{Service: service})}
	_, err = runner.RunReportRequirementMapLifecycle(context.Background(), ReportRequirementMapLifecycleRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", AgentExecutor: "codex", Plan: plan,
		Invoke: func(_ context.Context, binding reporting.ReportRequirementMapBinding) (ReportRequirementMapAgentResult, error) {
			if binding.ToolSessionID != "ses_report" || binding.IdempotencyKey != "rrk_report" {
				t.Fatalf("unexpected fallback IDs: %#v", binding)
			}
			return ReportRequirementMapAgentResult{Text: ReportRequirementsMappedSentinel, SessionID: "ses_plan"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

type requirementLifecycleService struct {
	selection reporting.ReportRequirementMapSelection
	query     reporting.ReportRequirementMapQuery
	events    []ledger.Event
}

func (service *requirementLifecycleService) ListEvents(context.Context, string) ([]ledger.Event, error) {
	return service.events, nil
}

func (service *requirementLifecycleService) AppendEvents(context.Context, string, []ledger.AppendRequest) ([]ledger.Event, error) {
	return nil, nil
}

func (service *requirementLifecycleService) AppendReportTerminalIfOpen(context.Context, string, string, []ledger.AppendRequest) ([]ledger.Event, bool, error) {
	return nil, false, nil
}

func (service *requirementLifecycleService) AppendEventsIfNoActiveAgentWork(context.Context, string, []ledger.AppendRequest) ([]ledger.Event, error) {
	return nil, nil
}

func (service *requirementLifecycleService) ListSourceSnapshotsWithState(context.Context, source.ListRequest) ([]source.Snapshot, error) {
	return nil, nil
}

func (service *requirementLifecycleService) SelectReportRequirementMap(_ context.Context, query reporting.ReportRequirementMapQuery) (reporting.ReportRequirementMapSelection, error) {
	service.query = query
	return service.selection, nil
}

func (service *requirementLifecycleService) AppendEvent(_ context.Context, req ledger.AppendRequest) (ledger.Event, error) {
	event := ledger.Event{EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Producer: req.Producer, Payload: req.Payload}
	service.events = append(service.events, event)
	return event, nil
}

func (service *requirementLifecycleService) AppendEventConditionally(_ context.Context, _ string, _ func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error)) (ledger.Event, bool, error) {
	return ledger.Event{}, false, nil
}

func TestRunReportRequirementMapLifecycleAcceptsOnlyDurableExactSubmission(t *testing.T) {
	planValue := reporting.SectionalReportPlan{Parts: []reporting.ReportPlanPart{{Title: "Part", Sections: []reporting.ReportPlanSection{{Title: "Section"}}}}}
	value := reporting.ReportRequirementMap{ReviewedEventIDs: []string{"evt_pending"}, Requirements: []reporting.ReportRequirement{{RequirementID: "req_one", Instruction: "include one", SourceEventIDs: []string{"evt_pending"}, Owner: &reporting.ReportRequirementOwner{PartIndex: 1, SectionIndex: 1}}}}
	hash, encoded, err := reporting.ReportRequirementMapHash(value)
	if err != nil {
		t.Fatal(err)
	}
	service := &requirementLifecycleService{selection: reporting.ReportRequirementMapSelection{Event: ledger.Event{EventID: "evt_map"}, RequirementMapHash: hash, RequirementMap: encoded}}
	runner := Runner{Service: service, Lifecycle: reporting.Runner(reportexecution.Runner{Service: service, NewID: requirementTestID})}
	result, err := runner.RunReportRequirementMapLifecycle(context.Background(), ReportRequirementMapLifecycleRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", AgentExecutor: "codex", PreviousProviderSessionID: "ses_plan", Plan: planValue,
		Invoke: func(_ context.Context, binding reporting.ReportRequirementMapBinding) (ReportRequirementMapAgentResult, error) {
			if binding.ToolSessionID != "ses_tool" || binding.Producer.ID != "ses_tool" {
				t.Fatalf("unexpected binding: %#v", binding)
			}
			return ReportRequirementMapAgentResult{Text: ReportRequirementsMappedSentinel, SessionID: "ses_plan"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Event.EventID != "evt_map" || service.query.PlanEventID != "evt_plan" || service.query.IdempotencyKey != "rrk_once" {
		t.Fatalf("unexpected lifecycle result: %#v %#v", result, service.query)
	}
}

func TestRunReportRequirementMapLifecycleRejectsNonExactSentinelBeforeSelection(t *testing.T) {
	for _, text := range []string{"", " REQUIREMENTS_MAPPED ", "done REQUIREMENTS_MAPPED", "REQUIREMENTS_MAPPED\nextra"} {
		t.Run(text, func(t *testing.T) {
			service := &requirementLifecycleService{}
			_, err := (Runner{Service: service, Lifecycle: reporting.Runner(reportexecution.Runner{Service: service, NewID: requirementTestID})}).RunReportRequirementMapLifecycle(context.Background(), ReportRequirementMapLifecycleRequest{MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", AgentExecutor: "codex", Invoke: func(context.Context, reporting.ReportRequirementMapBinding) (ReportRequirementMapAgentResult, error) {
				return ReportRequirementMapAgentResult{Text: text}, nil
			}})
			if err == nil || service.query.MissionID != "" {
				t.Fatal("non-exact sentinel advanced the lifecycle")
			}
		})
	}
}

func requirementTestID(prefix string) string {
	if prefix == "ses" {
		return "ses_tool"
	}
	return "rrk_once"
}
