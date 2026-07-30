package reporting

import (
	"context"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type reportRequirementLifecycleFakeService struct {
	*fakeRunnerService
	selection app.ReportRequirementMapSelection
	query     app.ReportRequirementMapQuery
}

func (service *reportRequirementLifecycleFakeService) SelectReportRequirementMap(_ context.Context, query app.ReportRequirementMapQuery) (app.ReportRequirementMapSelection, error) {
	service.query = query
	return service.selection, nil
}

func TestRunReportRequirementMapLifecycleAcceptsOnlyDurableExactSubmission(t *testing.T) {
	plan := SectionalReportPlan{Parts: []ReportPlanPart{{Title: "Part", Sections: []ReportPlanSection{{Title: "Section"}}}}}
	value := ReportRequirementMap{ReviewedEventIDs: []string{"evt_pending"}, Requirements: []ReportRequirement{{RequirementID: "req_one", Instruction: "include one", SourceEventIDs: []string{"evt_pending"}, Owner: &ReportRequirementOwner{PartIndex: 1, SectionIndex: 1}}}}
	hash, encoded, err := ReportRequirementMapHash(value)
	if err != nil {
		t.Fatal(err)
	}
	service := &reportRequirementLifecycleFakeService{fakeRunnerService: &fakeRunnerService{}, selection: app.ReportRequirementMapSelection{Event: app.LedgerEvent{EventID: "evt_map"}, RequirementMapHash: hash, RequirementMap: encoded}}
	runner := Runner{Service: service, NewID: func(prefix string) string {
		if prefix == "ses" {
			return "ses_tool"
		}
		return "rrk_once"
	}}
	result, err := runner.RunReportRequirementMapLifecycle(context.Background(), ReportRequirementMapLifecycleRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", AgentExecutor: "codex", PreviousProviderSessionID: "ses_plan", Plan: plan,
		Invoke: func(_ context.Context, binding ReportRequirementMapBinding) (ReportRequirementMapAgentResult, error) {
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
			service := &reportRequirementLifecycleFakeService{fakeRunnerService: &fakeRunnerService{}}
			_, err := (Runner{Service: service, NewID: testRunnerID}).RunReportRequirementMapLifecycle(context.Background(), ReportRequirementMapLifecycleRequest{
				MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", AgentExecutor: "codex",
				Invoke: func(context.Context, ReportRequirementMapBinding) (ReportRequirementMapAgentResult, error) {
					return ReportRequirementMapAgentResult{Text: text}, nil
				},
			})
			if err == nil || service.query.MissionID != "" {
				t.Fatal("non-exact sentinel advanced the lifecycle")
			}
		})
	}
}
