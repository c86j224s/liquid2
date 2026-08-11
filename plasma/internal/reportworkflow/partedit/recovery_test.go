package partedit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestRecoverValidatesRangeSourceAndOutcome(t *testing.T) {
	ctx := context.Background()
	service, binding := partEditRecoverFixture(t, ctx)
	event := partEditEvent(t, service.events)
	plan := reporting.SectionalReportPlan{Parts: []reporting.ReportPlanPart{{Title: "Part", Sections: []reporting.ReportPlanSection{{Title: "Section"}}}}}
	input := partEditRecoverInput(service, event, plan)

	out, ok, err := Recover(ctx, input)
	if err != nil || !ok {
		t.Fatalf("Recover valid ok=%t err=%v", ok, err)
	}
	if out.PartIndex != 0 || out.Draft.ArtifactID != binding.EditedArtifactID || out.Draft.Markdown == "" || out.Draft.Title != "Part" {
		t.Fatalf("unexpected recovered edit: %#v", out)
	}

	input.Event = editedEventWithPartIndex(event, 2)
	_, ok, err = Recover(ctx, input)
	if err != nil || ok {
		t.Fatalf("out-of-range Recover ok=%t err=%v, want ignored", ok, err)
	}

	input = partEditRecoverInput(service, event, plan)
	input.Sources = map[int]PartDraft{}
	_, ok, err = Recover(ctx, input)
	if err != nil || ok {
		t.Fatalf("missing source Recover ok=%t err=%v, want ignored", ok, err)
	}

	input = partEditRecoverInput(service, event, plan)
	input.Service = &partEditRecoverStore{events: service.events, artifacts: map[string]artifact.Raw{
		binding.SourceArtifactID: service.artifacts[binding.SourceArtifactID],
		binding.EditedArtifactID: {ArtifactID: binding.EditedArtifactID, MissionID: "mis_other", MediaType: "text/markdown; charset=utf-8", Content: []byte("# Edited")},
	}}
	_, ok, err = Recover(ctx, input)
	if err != nil || ok {
		t.Fatalf("foreign artifact Recover ok=%t err=%v, want ignored", ok, err)
	}
}

type partEditRecoverStore struct {
	events    []ledger.Event
	artifacts map[string]artifact.Raw
}

func (store partEditRecoverStore) ListEvents(context.Context, string) ([]ledger.Event, error) {
	return append([]ledger.Event(nil), store.events...), nil
}

func (store partEditRecoverStore) GetRawArtifact(_ context.Context, artifactID string) (artifact.Raw, error) {
	raw, ok := store.artifacts[artifactID]
	if !ok {
		return artifact.Raw{}, errors.New("artifact missing")
	}
	return raw, nil
}

func (store *partEditRecoverStore) AppendEventConditionally(_ context.Context, missionID string, build func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error)) (ledger.Event, bool, error) {
	req, existing, create, err := build(store.events)
	if err != nil || !create {
		return existing, false, err
	}
	event := ledger.Event{
		EventID: req.EventID, MissionID: missionID, EventType: req.EventType, Producer: req.Producer,
		CausationEventID: req.CausationEventID, CorrelationID: req.CorrelationID, Payload: req.Payload,
	}
	store.events = append(store.events, event)
	return event, true, nil
}

func (store *partEditRecoverStore) CreateRawArtifactWithEventConditionally(_ context.Context, req artifact.CreateRequest, build func([]ledger.Event, artifact.Raw) (ledger.AppendRequest, ledger.Event, bool, error)) (artifact.Raw, ledger.Event, bool, error) {
	raw := artifact.Raw{
		ArtifactID: req.ArtifactID, MissionID: req.MissionID, MediaType: req.MediaType,
		Filename: req.Filename, Producer: req.Producer, Content: append([]byte(nil), req.Content...),
	}
	eventReq, existing, create, err := build(store.events, raw)
	if err != nil || !create {
		return raw, existing, false, err
	}
	event := ledger.Event{
		EventID: eventReq.EventID, MissionID: eventReq.MissionID, EventType: eventReq.EventType,
		Producer: eventReq.Producer, CausationEventID: eventReq.CausationEventID,
		CorrelationID: eventReq.CorrelationID, Payload: eventReq.Payload,
	}
	store.artifacts[raw.ArtifactID] = raw
	store.events = append(store.events, event)
	return raw, event, true, nil
}

func partEditRecoverFixture(t *testing.T, ctx context.Context) (*partEditRecoverStore, reporting.PartEditBinding) {
	t.Helper()
	binding := reporting.PartEditBinding{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan",
		SourcePartEventID: "evt_part", SourceArtifactID: "art_part",
		EditedArtifactID: "art_edit", Filename: "part-edit-outcome.md",
		ToolSessionID: "tool-edit", ProviderSessionID: "provider-edit",
		PreviousProviderSessionID: "provider-edit",
		IdempotencyKey:            "report-part-edit:evt_pending:evt_plan:1", PartIndex: 1,
		AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
		AgentSelectionSource: "auto", MCPMode: "auto",
		ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
		GenerationGuidanceProfile:    "profile", GenerationGuidanceSHA256: "sha",
		SessionChainKind: "section_fanout_report", ReportPlanSessionID: "plan-session",
		ForkSourceAgentSessionID: "plan-session",
	}
	store := &partEditRecoverStore{
		events: []ledger.Event{
			{EventID: binding.PendingEventID, MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "test"}, Payload: mustPartEditRecoverJSON(map[string]any{"report_mode": "long_form", "origin_pending_event_id": binding.PendingEventID, "retry_strategy": "initial"})},
			{EventID: binding.PlanEventID, MissionID: binding.MissionID, EventType: "report.plan.created", Producer: ledger.Producer{Type: "agent_session", ID: "plan-session"}, Payload: mustPartEditRecoverJSON(map[string]any{"pending_event_id": binding.PendingEventID, "report_mode": "long_form", "artifact_id": "art_plan"})},
			{EventID: binding.SourcePartEventID, MissionID: binding.MissionID, EventType: "report.part.created", Producer: ledger.Producer{Type: "agent_session", ID: "provider-part"}, Payload: mustPartEditRecoverJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.SourceArtifactID, "part_index": 1})},
		},
		artifacts: map[string]artifact.Raw{
			binding.SourceArtifactID: {ArtifactID: binding.SourceArtifactID, MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Content: []byte("# Part\n\nSource.")},
		},
	}
	if _, _, err := reporting.StartPartEdit(ctx, store, "evt_part_edit_started", binding); err != nil {
		t.Fatal(err)
	}
	if _, err := reporting.FinalizePartEdit(ctx, store, binding, "evt_part_edited", "# Part\n\nEdited.", 1); err != nil {
		t.Fatal(err)
	}
	return store, binding
}

func partEditRecoverInput(service *partEditRecoverStore, event ledger.Event, plan reporting.SectionalReportPlan) RecoverInput {
	return RecoverInput{
		Service: service, Event: event, Events: service.events, MissionID: "mis_1",
		PendingEventID: "evt_pending", PlanEventID: "evt_plan", Plan: plan,
		Sources:       map[int]PartDraft{0: {Title: "Part", Markdown: "# Part\n\nSource.", ArtifactID: "art_part", WordCount: 3}},
		AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
		AgentSelectionSource: "auto", MCPMode: "auto",
		ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
		GenerationGuidanceProfile:    "profile", GenerationGuidanceSHA256: "sha",
		SessionChainKind: "section_fanout_report", ReportPlanSessionID: "plan-session",
	}
}

func partEditEvent(t *testing.T, events []ledger.Event) ledger.Event {
	t.Helper()
	for _, event := range events {
		if event.EventType == reporting.PartEditedEventType {
			return event
		}
	}
	t.Fatal("missing Part edit event")
	return ledger.Event{}
}

func editedEventWithPartIndex(event ledger.Event, partIndex int) ledger.Event {
	payload := map[string]any{}
	_ = json.Unmarshal(event.Payload, &payload)
	payload["part_index"] = partIndex
	event.Payload = mustPartEditRecoverJSON(payload)
	return event
}

func mustPartEditRecoverJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}
