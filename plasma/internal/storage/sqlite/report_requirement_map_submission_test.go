package sqlite

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestReportRequirementMapSubmissionIsAtomicReplayableAndOrdered(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_requirements", Title: "Requirements"}); err != nil {
		t.Fatal(err)
	}
	for _, event := range []app.LedgerEvent{
		{EventID: "evt_user", MissionID: "mis_requirements", EventType: "turn.user", Producer: app.Producer{Type: "user", ID: "plasma-ui"}, Payload: []byte(`{"kind":"user_turn","text":"include a table"}`)},
		{EventID: "evt_pending", MissionID: "mis_requirements", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "plasma-ui"}, Payload: []byte(`{"report_mode":"long_form","agent_executor":"codex","direction_hint":"include a table"}`)},
		{EventID: "evt_plan", MissionID: "mis_requirements", EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "ses_plan"}, Payload: []byte(`{"pending_event_id":"evt_pending","plan":{"parts":[{"title":"Part","sections":[{"title":"Section"}]}]}}`)},
	} {
		if _, err := store.AppendLedgerEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	svc := app.NewService(store)
	req := app.ReportRequirementMapSubmissionRequest{
		EventID: "evt_map", MissionID: "mis_requirements", PendingEventID: "evt_pending", PlanEventID: "evt_plan",
		ToolSessionID: "ses_tool", PreviousProviderSessionID: "ses_plan", AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "high",
		IdempotencyKey: "rrk_once", ArgumentsHash: "args", RequirementMapHash: "map",
		RequirementMap:   json.RawMessage(`{"reviewed_event_ids":["evt_user","evt_pending"],"requirements":[{"requirement_id":"req_table","instruction":"include a table","source_event_ids":["evt_user","evt_pending"],"owner":{"part_index":1,"section_index":1}}]}`),
		ReviewedEventIDs: []string{"evt_user", "evt_pending"}, Attempt: 1, ToolProducer: app.Producer{Type: "agent_session", ID: "ses_tool"},
	}
	first, err := svc.SubmitReportRequirementMap(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	req.EventID = "evt_map_replay"
	replay, err := app.NewService(store).SubmitReportRequirementMap(ctx, req)
	if err != nil || !replay.Replay || replay.Event.EventID != first.Event.EventID {
		t.Fatalf("restart replay failed: %#v %v", replay, err)
	}
	if first.Event.EventType != "report.requirements.mapped" || first.Event.CausationEventID != "evt_plan" || first.Event.CorrelationID != "evt_pending" || first.Event.Producer.ID != "ses_tool" {
		t.Fatalf("unexpected durable event: %#v", first.Event)
	}
	selection, err := svc.SelectReportRequirementMap(ctx, app.ReportRequirementMapQuery{MissionID: "mis_requirements", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_tool", PreviousProviderSessionID: "ses_plan", AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "high", IdempotencyKey: "rrk_once"})
	if err != nil || selection.Event.EventID != "evt_map" || selection.RequirementMapHash != "map" {
		t.Fatalf("selection failed: %#v %v", selection, err)
	}
	req.EventID, req.RequirementMapHash = "evt_conflict", "different"
	if _, err := svc.SubmitReportRequirementMap(ctx, req); err == nil {
		t.Fatal("mismatched replay was accepted")
	}
}

func TestReportRequirementMapSubmissionRejectsLateStageAndForeignTrace(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_late_map", Title: "Late"}); err != nil {
		t.Fatal(err)
	}
	for _, event := range []app.LedgerEvent{
		{EventID: "evt_pending", MissionID: "mis_late_map", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "plasma-ui"}, Payload: []byte(`{"report_mode":"long_form","agent_executor":"codex"}`)},
		{EventID: "evt_plan", MissionID: "mis_late_map", EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "ses_plan"}, Payload: []byte(`{"pending_event_id":"evt_pending"}`)},
		{EventID: "evt_section", MissionID: "mis_late_map", EventType: "report.section.started", Producer: app.Producer{Type: "agent_session", ID: "ses_section"}, Payload: []byte(`{"pending_event_id":"evt_pending","plan_event_id":"evt_plan","part_index":1,"section_index":1}`)},
	} {
		if _, err := store.AppendLedgerEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	req := app.ReportRequirementMapSubmissionRequest{EventID: "evt_map", MissionID: "mis_late_map", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_tool", AgentExecutor: "codex", IdempotencyKey: "rrk", ArgumentsHash: "args", RequirementMapHash: "map", RequirementMap: json.RawMessage(`{"reviewed_event_ids":["evt_pending"],"requirements":[]}`), ReviewedEventIDs: []string{"evt_pending"}, Attempt: 1, ToolProducer: app.Producer{Type: "agent_session", ID: "ses_tool"}}
	if _, err := app.NewService(store).SubmitReportRequirementMap(ctx, req); err == nil {
		t.Fatal("late requirement mapping was accepted")
	}
	req.EventID, req.ReviewedEventIDs = "evt_foreign", []string{"evt_section", "evt_pending"}
	if _, err := app.NewService(store).SubmitReportRequirementMap(ctx, req); err == nil {
		t.Fatal("non-user trace was accepted")
	}
}
