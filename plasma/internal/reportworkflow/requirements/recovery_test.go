package requirements

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestRecoverStateDoesNotTreatRawCreatedEventsAsValidatedStart(t *testing.T) {
	plan := reporting.SectionalReportPlan{Parts: []reporting.ReportPlanPart{{Title: "Part", Sections: []reporting.ReportPlanSection{{Title: "Section"}}}}}
	events := []ledger.Event{
		requirementPendingEvent(),
		{
			EventID: "evt_section", EventType: "report.section.created", MissionID: "mis_1",
			Payload: mustRequirementJSON(map[string]any{
				"pending_event_id": "evt_pending", "plan_event_id": "evt_plan",
				"part_index": 1, "section_index": 1,
			}),
		},
	}

	recovered, err := recoverState(events, "evt_pending", "evt_plan", plan)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.hasValidatedWorkStart {
		t.Fatal("raw section.created event was treated as a validated downstream start")
	}
}

func TestRecoverStateAcceptsExplicitStartedEventAsValidatedStart(t *testing.T) {
	plan := reporting.SectionalReportPlan{Parts: []reporting.ReportPlanPart{{Title: "Part", Sections: []reporting.ReportPlanSection{{Title: "Section"}}}}}
	events := []ledger.Event{
		requirementPendingEvent(),
		{
			EventID: "evt_section_started", EventType: "report.section.started", MissionID: "mis_1",
			Payload: mustRequirementJSON(map[string]any{
				"pending_event_id": "evt_pending", "plan_event_id": "evt_plan",
				"part_index": 1, "section_index": 1,
			}),
		},
	}

	recovered, err := recoverState(events, "evt_pending", "evt_plan", plan)
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.hasValidatedWorkStart {
		t.Fatal("valid section.started event did not mark downstream work as started")
	}
}

func TestRunSkipsLegacyRequirementMappingOnlyWithValidatedDownstreamSignal(t *testing.T) {
	runner := Runner{Service: requirementListService{events: []ledger.Event{requirementPendingEvent()}}}
	out, err := runner.Run(context.Background(), Input{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan",
		PlanSessionID: "plan-session-1", ValidatedDownstream: true,
		Plan: reporting.SectionalReportPlan{Parts: []reporting.ReportPlanPart{{Title: "Part", Sections: []reporting.ReportPlanSection{{Title: "Section"}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Recovered || len(out.RequirementMap.Requirements) != 0 {
		t.Fatalf("unexpected recovered output: %#v", out)
	}
}

type requirementListService struct {
	events []ledger.Event
}

func (service requirementListService) ListEvents(context.Context, string) ([]ledger.Event, error) {
	return append([]ledger.Event(nil), service.events...), nil
}

func requirementPendingEvent() ledger.Event {
	return ledger.Event{
		EventID: "evt_pending", MissionID: "mis_1", EventType: "report.draft.pending",
		Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
		Payload:  mustRequirementJSON(map[string]any{"origin_pending_event_id": "evt_pending", "retry_strategy": "initial"}),
	}
}

func mustRequirementJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}
