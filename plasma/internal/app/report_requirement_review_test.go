package app

import "testing"

func TestReportRequirementReviewEventIDsSelectsOnlyPriorUserTurnsAndPending(t *testing.T) {
	events := []LedgerEvent{
		{EventID: "evt_mission", EventType: "mission.created", Producer: Producer{Type: "user"}},
		{EventID: "evt_user", EventType: "turn.user", Producer: Producer{Type: "user"}},
		{EventID: "evt_agent", EventType: "turn.agent.response", Producer: Producer{Type: "agent_session"}},
		{EventID: "evt_pending", EventType: "report.draft.pending", Producer: Producer{Type: "user"}},
		{EventID: "evt_plan", EventType: "report.plan.created", Producer: Producer{Type: "agent_session"}},
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
	for _, events := range [][]LedgerEvent{
		{{EventID: "evt_other", EventType: "turn.user", Producer: Producer{Type: "user"}}},
		{{EventID: "evt_pending", EventType: "report.draft.pending", Producer: Producer{Type: "agent_session"}}},
	} {
		if _, err := ReportRequirementReviewEventIDs(events, "evt_pending"); err == nil {
			t.Fatal("invalid pending event was accepted")
		}
	}
}
