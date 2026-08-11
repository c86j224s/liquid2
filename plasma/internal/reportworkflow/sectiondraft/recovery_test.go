package sectiondraft

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestRecoverValidatesRangeArtifactAndMission(t *testing.T) {
	plan := reporting.SectionalReportPlan{Parts: []reporting.ReportPlanPart{{Title: "Part", Sections: []reporting.ReportPlanSection{{Title: "Section"}}}}}
	service := sectionRecoverStore{artifacts: map[string]artifact.Raw{
		"art_section": {ArtifactID: "art_section", MissionID: "mis_1", MediaType: "text/markdown; charset=utf-8", Content: []byte("# Section\n\nBody text.")},
		"art_foreign": {ArtifactID: "art_foreign", MissionID: "mis_other", MediaType: "text/markdown; charset=utf-8", Content: []byte("# Foreign")},
	}}

	out, ok, err := Recover(context.Background(), RecoverInput{
		Service: service, Event: sectionEvent("art_section", 1, 1), MissionID: "mis_1",
		PendingEventID: "evt_pending", PlanEventID: "evt_plan", Plan: plan,
	})
	if err != nil || !ok {
		t.Fatalf("Recover valid ok=%t err=%v", ok, err)
	}
	if out.PartIndex != 0 || out.SectionIndex != 0 || out.Draft.ArtifactID != "art_section" || out.Draft.WordCount == 0 {
		t.Fatalf("unexpected recovered section: %#v", out)
	}

	for _, tc := range []struct {
		name  string
		event ledger.Event
	}{
		{name: "part outside plan", event: sectionEvent("art_section", 2, 1)},
		{name: "section outside plan", event: sectionEvent("art_section", 1, 2)},
		{name: "foreign artifact", event: sectionEvent("art_foreign", 1, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok, err := Recover(context.Background(), RecoverInput{
				Service: service, Event: tc.event, MissionID: "mis_1",
				PendingEventID: "evt_pending", PlanEventID: "evt_plan", Plan: plan,
			})
			if err != nil || ok {
				t.Fatalf("Recover ok=%t err=%v, want ignored", ok, err)
			}
		})
	}
}

func TestRecoverEvidenceGapRejectsInvalidProducerAndSessionLineage(t *testing.T) {
	valid := validGapRecoveryInput()
	if out, ok, err := RecoverEvidenceGap(valid); err != nil || !ok || out.Attempt != 1 {
		t.Fatalf("valid gap recovery = %#v ok=%t err=%v", out, ok, err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*RecoverEvidenceGapInput)
	}{
		{name: "producer type", mutate: func(input *RecoverEvidenceGapInput) {
			input.Event.Producer.Type = "system"
		}},
		{name: "producer id", mutate: func(input *RecoverEvidenceGapInput) {
			input.Event.Producer.ID = "different-session"
		}},
		{name: "previous session", mutate: func(input *RecoverEvidenceGapInput) {
			setGapRecoveryPayload(input, "previous_agent_session_id", "different-session")
		}},
		{name: "returned session", mutate: func(input *RecoverEvidenceGapInput) {
			setGapRecoveryPayload(input, "returned_agent_session_id", "different-session")
		}},
		{name: "report session", mutate: func(input *RecoverEvidenceGapInput) {
			setGapRecoveryPayload(input, "report_session_id", "different-session")
		}},
		{name: "executor", mutate: func(input *RecoverEvidenceGapInput) {
			setGapRecoveryPayload(input, "agent_executor", "claude")
		}},
		{name: "session chain", mutate: func(input *RecoverEvidenceGapInput) {
			setGapRecoveryPayload(input, "session_chain_kind", "other_chain")
		}},
		{name: "report plan session", mutate: func(input *RecoverEvidenceGapInput) {
			setGapRecoveryPayload(input, "report_plan_session_id", "different-plan")
		}},
		{name: "tool session", mutate: func(input *RecoverEvidenceGapInput) {
			setGapRecoveryPayload(input, "tool_session_id", "")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := validGapRecoveryInput()
			tc.mutate(&input)
			if out, ok, err := RecoverEvidenceGap(input); err != nil || ok {
				t.Fatalf("RecoverEvidenceGap = %#v ok=%t err=%v, want rejected without error", out, ok, err)
			}
		})
	}
}

func validGapRecoveryInput() RecoverEvidenceGapInput {
	plan := reporting.SectionalReportPlan{Parts: []reporting.ReportPlanPart{{
		Title: "Part", Sections: []reporting.ReportPlanSection{{Title: "Section"}},
	}}}
	return RecoverEvidenceGapInput{
		Event:     validGapRecoveryEvent(),
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", Plan: plan,
		AgentExecutor: "codex", SessionChainKind: "fresh_session_report", ReportPlanSessionID: "provider-plan",
	}
}

func validGapRecoveryEvent() ledger.Event {
	return ledger.Event{
		EventID: "evt_gap", MissionID: "mis_1", EventType: EvidenceGapEventType,
		Producer: ledger.Producer{Type: "agent_session", ID: "provider-section"},
		Payload: mustSectionRecoverJSON(map[string]any{
			"pending_event_id": "evt_pending", "plan_event_id": "evt_plan",
			"part_index": 1, "section_index": 1, "attempt_number": 1,
			"reason_code": EvidenceGapReasonCode, "agent_executor": "codex",
			"agent_session_id": "provider-section", "previous_agent_session_id": "provider-section",
			"returned_agent_session_id": "provider-section", "report_session_id": "provider-section",
			"tool_session_id": "ses_gap", "session_chain_kind": "fresh_session_report",
			"report_plan_session_id": "provider-plan",
		}),
	}
}

func setGapRecoveryPayload(input *RecoverEvidenceGapInput, key string, value any) {
	var payload map[string]any
	if err := json.Unmarshal(input.Event.Payload, &payload); err != nil {
		panic(err)
	}
	payload[key] = value
	input.Event.Payload = mustSectionRecoverJSON(payload)
}

type sectionRecoverStore struct {
	artifacts map[string]artifact.Raw
}

func (store sectionRecoverStore) GetRawArtifact(_ context.Context, artifactID string) (artifact.Raw, error) {
	raw, ok := store.artifacts[artifactID]
	if !ok {
		return artifact.Raw{}, errors.New("artifact missing")
	}
	return raw, nil
}

func sectionEvent(artifactID string, partIndex int, sectionIndex int) ledger.Event {
	return ledger.Event{
		EventID: "evt_section", MissionID: "mis_1", EventType: CreatedEventType,
		Payload: mustSectionRecoverJSON(map[string]any{
			"pending_event_id": "evt_pending", "plan_event_id": "evt_plan",
			"artifact_id": artifactID, "title": "Section", "agent_session_id": "provider-section",
			"part_index": partIndex, "section_index": sectionIndex,
		}),
	}
}

func mustSectionRecoverJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}
