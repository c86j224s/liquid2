package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestFinalizeAndRecoverLongFormSectionPlanRepair(t *testing.T) {
	plan := sectionPlanRepairTestPlan()
	store := &sectionPlanRepairTestStore{events: sectionPlanRepairTestEvents(plan, true)}
	replacement := ReportSectionPlanReplacement{
		ReportSectionCoordinate: ReportSectionCoordinate{PartIndex: 1, SectionIndex: 1},
		Section:                 ReportPlanSection{Title: "Replacement", Purpose: "Explain supported facts."},
	}
	req := sectionPlanRepairTestRequest(replacement)

	result, err := FinalizeLongFormSectionPlanRepair(context.Background(), store, plan, req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Event.EventType != LongFormSectionPlanRepairCompletedEventType || result.Plan.Parts[0].Sections[0].Title != "Replacement" {
		t.Fatalf("unexpected finalized repair: %#v", result)
	}
	if len(store.events) != 5 {
		t.Fatalf("event count = %d, want one repair append", len(store.events))
	}
	var payload map[string]any
	if json.Unmarshal(result.Event.Payload, &payload) != nil || payload["repair_round"] != float64(1) || payload["outcome"] != sectionPlanRepairOutcomeApplied || payload["artifact_id"] != nil {
		t.Fatalf("unexpected repair payload: %#v", payload)
	}

	store.events = append(store.events, ledger.Event{
		EventID: "evt_replacement", MissionID: "mis_1", EventType: "report.section.created",
		Payload: mustJSON(map[string]any{
			"pending_event_id": "evt_pending", "plan_event_id": "evt_plan",
			"part_index": 1, "section_index": 1,
		}),
	})
	replayed, err := FinalizeLongFormSectionPlanRepair(context.Background(), store, plan, req)
	if err != nil || replayed.Event.EventID != result.Event.EventID || len(store.events) != 6 {
		t.Fatalf("repair replay was not idempotent: result=%#v err=%v", replayed, err)
	}
	recovered, ok, err := RecoverLongFormSectionPlanRepair(store.events, "mis_1", "evt_pending", "evt_plan", plan)
	if err != nil || !ok || recovered.Plan.Parts[0].Sections[0].Title != "Replacement" {
		t.Fatalf("repair recovery failed: result=%#v ok=%v err=%v", recovered, ok, err)
	}
}

func TestFinalizeLongFormSectionPlanUnrepairableConsumesRepairRound(t *testing.T) {
	plan := sectionPlanRepairTestPlan()
	store := &sectionPlanRepairTestStore{events: sectionPlanRepairTestEvents(plan, true)}
	req := sectionPlanRepairTestRequest(ReportSectionPlanReplacement{})
	req.Coordinates = []ReportSectionCoordinate{{PartIndex: 1, SectionIndex: 1}}
	req.Replacements = nil

	result, err := FinalizeLongFormSectionPlanUnrepairable(context.Background(), store, plan, req)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Unrepairable || result.Plan.Parts[0].Sections[0].Title != "Unsupported" || len(store.events) != 5 {
		t.Fatalf("unexpected unrepairable result: %#v events=%d", result, len(store.events))
	}
	var payload map[string]any
	if json.Unmarshal(result.Event.Payload, &payload) != nil || payload["outcome"] != sectionPlanRepairOutcomeUnrepairable || payload["replacements"] != nil {
		t.Fatalf("unexpected unrepairable payload: %#v", payload)
	}

	lateReplacement := sectionPlanRepairTestRequest(ReportSectionPlanReplacement{
		ReportSectionCoordinate: ReportSectionCoordinate{PartIndex: 1, SectionIndex: 1},
		Section:                 ReportPlanSection{Title: "Late Replacement", Purpose: "Explain supported facts."},
	})
	replayed, err := FinalizeLongFormSectionPlanRepair(context.Background(), store, plan, lateReplacement)
	if err != nil || !replayed.Unrepairable || replayed.Event.EventID != result.Event.EventID || len(store.events) != 5 {
		t.Fatalf("unrepairable outcome did not consume the repair round: result=%#v err=%v", replayed, err)
	}
	recovered, ok, err := RecoverLongFormSectionPlanRepair(store.events, "mis_1", "evt_pending", "evt_plan", plan)
	if err != nil || !ok || !recovered.Unrepairable {
		t.Fatalf("unrepairable outcome did not recover: result=%#v ok=%v err=%v", recovered, ok, err)
	}
}

func TestFinalizeLongFormSectionPlanRepairRequiresTerminalGapHistory(t *testing.T) {
	plan := sectionPlanRepairTestPlan()
	store := &sectionPlanRepairTestStore{events: sectionPlanRepairTestEvents(plan, false)}
	_, err := FinalizeLongFormSectionPlanRepair(context.Background(), store, plan, sectionPlanRepairTestRequest(ReportSectionPlanReplacement{
		ReportSectionCoordinate: ReportSectionCoordinate{PartIndex: 1, SectionIndex: 1},
		Section:                 ReportPlanSection{Title: "Replacement", Purpose: "Explain supported facts."},
	}))
	if err == nil || !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want terminal-gap conflict", err)
	}
}

func TestFinalizeLongFormSectionPlanRepairRejectsSecondCoordinateSet(t *testing.T) {
	plan := sectionPlanRepairTestPlan()
	store := &sectionPlanRepairTestStore{events: sectionPlanRepairTestEvents(plan, true)}
	first := sectionPlanRepairTestRequest(ReportSectionPlanReplacement{
		ReportSectionCoordinate: ReportSectionCoordinate{PartIndex: 1, SectionIndex: 1},
		Section:                 ReportPlanSection{Title: "First Replacement", Purpose: "Explain supported facts."},
	})
	if _, err := FinalizeLongFormSectionPlanRepair(context.Background(), store, plan, first); err != nil {
		t.Fatal(err)
	}
	store.events = append(store.events,
		sectionPlanRepairGapEvent("evt_gap_2_1", 1, 2, 1),
		sectionPlanRepairGapEvent("evt_gap_2_2", 1, 2, 2),
	)
	second := sectionPlanRepairTestRequest(ReportSectionPlanReplacement{
		ReportSectionCoordinate: ReportSectionCoordinate{PartIndex: 1, SectionIndex: 2},
		Section:                 ReportPlanSection{Title: "Second Replacement", Purpose: "Explain other facts."},
	})
	if _, err := FinalizeLongFormSectionPlanRepair(context.Background(), store, plan, second); err == nil || !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want one-repair conflict", err)
	}
}

type sectionPlanRepairTestStore struct{ events []ledger.Event }

func (store *sectionPlanRepairTestStore) AppendEventConditionally(_ context.Context, missionID string, build func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error)) (ledger.Event, bool, error) {
	req, existing, create, err := build(store.events)
	if err != nil || !create {
		return existing, false, err
	}
	event := ledger.Event{EventID: req.EventID, MissionID: missionID, EventType: req.EventType, Producer: req.Producer, CausationEventID: req.CausationEventID, CorrelationID: req.CorrelationID, Payload: req.Payload}
	store.events = append(store.events, event)
	return event, true, nil
}

func sectionPlanRepairTestPlan() SectionalReportPlan {
	return SectionalReportPlan{Summary: "Plan", Parts: []ReportPlanPart{{Title: "Part", Purpose: "Purpose", Sections: []ReportPlanSection{
		{Title: "Unsupported", Purpose: "Unsupported purpose"},
		{Title: "Other", Purpose: "Other purpose"},
	}}}}
}

func sectionPlanRepairTestEvents(plan SectionalReportPlan, withGaps bool) []ledger.Event {
	base := MarkdownReportEventBase{
		EventID: "evt_plan", MissionID: "mis_1", PendingEventID: "evt_pending", AgentExecutor: "codex",
		AgentModel: "gpt-test", AgentReasoningEffort: "medium", AgentSelectionSource: "request",
		AgentSessionID: "plan-session", ToolSessionID: "ses_plan", ReportMode: ModeLongForm,
		ReportSessionPolicy: "fresh_session", ReportSessionPolicySelection: "auto_fresh_session",
		GenerationGuidanceProfile: "profile", GenerationGuidanceSHA256: "sha",
		SessionChainKind: "fresh_session_report", ReportPlanSessionID: "plan-session",
		Producer: ledger.Producer{Type: "agent_session", ID: "plan-session"},
	}
	planReq := BuildMarkdownReportPlanCreatedAppendRequest(MarkdownReportPlanCreatedEventRequest{
		MarkdownReportEventBase: base, ArtifactID: "art_plan", Plan: plan, AssemblyStrategy: "c4_normalized_section_headings",
	})
	events := []ledger.Event{
		{EventID: "evt_pending", MissionID: "mis_1", EventType: "report.draft.pending", Payload: mustJSON(map[string]any{"origin_pending_event_id": "evt_pending", "retry_strategy": "initial"})},
		{EventID: planReq.EventID, MissionID: planReq.MissionID, EventType: planReq.EventType, Producer: planReq.Producer, Payload: planReq.Payload},
	}
	if withGaps {
		events = append(events, sectionPlanRepairGapEvent("evt_gap_1", 1, 1, 1), sectionPlanRepairGapEvent("evt_gap_2", 1, 1, 2))
	}
	return events
}

func sectionPlanRepairGapEvent(eventID string, part, section, attempt int) ledger.Event {
	return ledger.Event{EventID: eventID, MissionID: "mis_1", EventType: "report.section.evidence_gap", Payload: mustJSON(map[string]any{
		"pending_event_id": "evt_pending", "plan_event_id": "evt_plan", "part_index": part, "section_index": section,
		"attempt_number": attempt, "reason_code": "inadequate_section_evidence",
	})}
}

func sectionPlanRepairTestRequest(replacement ReportSectionPlanReplacement) LongFormSectionPlanRepairEventRequest {
	return LongFormSectionPlanRepairEventRequest{MarkdownReportEventBase: MarkdownReportEventBase{
		EventID: "evt_repair", MissionID: "mis_1", PendingEventID: "evt_pending", AgentExecutor: "codex",
		AgentModel: "gpt-test", AgentReasoningEffort: "medium", AgentSelectionSource: "request",
		AgentSessionID: "plan-session", PreviousAgentSessionID: "plan-session", ReturnedAgentSessionID: "plan-session",
		ToolSessionID: "ses_repair", ReportMode: ModeLongForm, ReportSessionPolicy: "fresh_session",
		ReportSessionPolicySelection: "auto_fresh_session", GenerationGuidanceProfile: "profile", GenerationGuidanceSHA256: "sha",
		SessionChainKind: "fresh_session_report", ReportPlanSessionID: "plan-session", ReportSessionID: "plan-session",
		Producer: ledger.Producer{Type: "agent_session", ID: "plan-session"},
	}, PlanEventID: "evt_plan", Replacements: []ReportSectionPlanReplacement{replacement}}
}
