package app

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"testing"
)

func TestReportRequirementReviewEventIDsSelectsOnlyPriorUserTurnsAndPending(t *testing.T) {
	events := []ledger.Event{
		{EventID: "evt_mission", EventType: "mission.created", Producer: ledger.Producer{Type: "user"}},
		{EventID: "evt_user", EventType: "turn.user", Producer: ledger.Producer{Type: "user"}},
		{EventID: "evt_agent", EventType: "turn.agent.response", Producer: ledger.Producer{Type: "agent_session"}},
		{EventID: "evt_pending", EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user"}},
		{EventID: "evt_plan", EventType: "report.plan.created", Producer: ledger.Producer{Type: "agent_session"}},
	}
	ids, err := ReportRequirementReviewEventIDs(events, "evt_pending")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "evt_user" || ids[1] != "evt_pending" {
		t.Fatalf("unexpected review events: %#v", ids)
	}
}

func TestReportRequirementReviewEventIDsRejectsInvalidPending(t *testing.T) {
	for _, events := range [][]ledger.Event{
		{{EventID: "evt_other", EventType: "turn.user", Producer: ledger.Producer{Type: "user"}}},
		{{EventID: "evt_pending", EventType: "report.draft.pending", Producer: ledger.Producer{Type: "agent_session"}}},
	} {
		if _, err := ReportRequirementReviewEventIDs(events, "evt_pending"); err == nil {
			t.Fatal("invalid pending event was accepted")
		}
	}
}
