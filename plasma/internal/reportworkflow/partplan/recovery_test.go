package partplan

import (
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestRecoverUsesStoredPartPlanContract(t *testing.T) {
	event := partPlanEvent(1)
	out, ok, err := Recover(partPlanRecoverInput(event, 1))
	if err != nil || !ok {
		t.Fatalf("Recover valid ok=%t err=%v", ok, err)
	}
	if out.PartIndex != 0 || out.Brief != "Read this Part as a cause-and-effect arc." || out.ProviderSessionID != "provider-part-plan" || !out.Recovered {
		t.Fatalf("unexpected recovered Part plan: %#v", out)
	}

	_, ok, err = Recover(partPlanRecoverInput(partPlanEvent(2), 1))
	if !errors.Is(err, producterror.ErrConflict) || ok {
		t.Fatalf("Recover out-of-range ok=%t err=%v, want conflict from durable decoder", ok, err)
	}
}

func partPlanRecoverInput(event ledger.Event, partCount int) RecoverInput {
	return RecoverInput{
		Event: event, MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan",
		PartCount: partCount, AgentExecutor: "codex", AgentModel: "gpt-test",
		AgentReasoningEffort: "medium", AgentSelectionSource: "auto",
		ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
		GenerationGuidanceProfile:    "profile", GenerationGuidanceSHA256: "sha",
		SessionChainKind: "section_fanout_report", ReportPlanSessionID: "plan-session",
	}
}

func partPlanEvent(partIndex int) ledger.Event {
	req := reporting.BuildPartPlanCreatedAppendRequest(reporting.PartPlanCreatedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: "evt_part_plan", MissionID: "mis_1", PendingEventID: "evt_pending",
			PlanEventID: "evt_plan", Title: "Part",
			AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
			AgentSelectionSource: "auto", AgentSessionID: "provider-part-plan",
			PreviousAgentSessionID: "provider-part-plan", ReturnedAgentSessionID: "provider-part-plan",
			ToolSessionID: "tool-part-plan", ReportMode: reportexecution.ModeLongForm,
			ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
			ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
			GenerationGuidanceProfile:    "profile", GenerationGuidanceSHA256: "sha",
			SessionChainKind: "section_fanout_report", ReportPlanSessionID: "plan-session",
			ReportSessionID: "provider-part-plan", ForkSourceAgentSessionID: "plan-session",
			Producer: ledger.Producer{Type: "agent_session", ID: "provider-part-plan"},
		},
		PartIndex: partIndex,
		Brief:     "Read this Part as a cause-and-effect arc.",
	})
	return ledger.Event{
		EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType,
		Producer: req.Producer, CausationEventID: req.CausationEventID,
		CorrelationID: req.CorrelationID, Payload: req.Payload,
	}
}
