package reportusage

import (
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func TestTargetForEventTable(t *testing.T) {
	tests := []struct{ name, eventType, stage, wantSurface, wantSession, wantPrevious string }{
		{"requirements", "report.requirements.mapped", "", "report_requirements", "ses-prev", "ses-prev"},
		{"part edit", "report.part.edited", "", "report_part_edit", "ses-current", "ses-prev"},
		{"writer", "report.final_edit.writer.submitted", "final_write", "report_final_write", "ses-current", "ses-current"},
		{"reader", "report.final_edit.reader.submitted", "reader_edit", "report_reader_edit", "ses-current", "ses-current"},
		{"style", "report.final_edit.style.submitted", "style_edit", "report_style_edit", "ses-current", "ses-current"},
		{"gate", "report.final_edit.gate.submitted", "corrective_gate", "report_corrective_gate", "ses-current", "ses-current"},
		{"semantic", "report.final_edit.style_semantic_validation.submitted", "style_semantic_validation", "report_style_semantic_validation", "ses-current", "ses-current"},
		{"evidence", "report.final_edit.evidence_gate.submitted", "evidence_gate", "report_evidence_gate", "ses-current", "ses-current"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]any{"provider_session_id": "ses-current", "previous_provider_session_id": "ses-prev", "stage": tc.stage, "fork_source_agent_session_id": "ses-fork", "agent_executor": "codex", "agent_model": "model", "agent_reasoning_effort": "high"}
			data, _ := json.Marshal(payload)
			got, ok, err := TargetForEvent(ledger.Event{EventID: "evt-target", EventType: tc.eventType, Payload: data})
			if err != nil || !ok {
				t.Fatalf("target=%#v ok=%v err=%v", got, ok, err)
			}
			if got.Surface != tc.wantSurface || got.AgentSessionID != tc.wantSession || got.PreviousAgentSessionID != tc.wantPrevious || got.ForkSourceAgentSessionID != "ses-fork" {
				t.Fatalf("unexpected target %#v", got)
			}
		})
	}
}

func TestTargetForEventRejectsMalformedStageAndJSON(t *testing.T) {
	if _, ok, err := TargetForEvent(ledger.Event{EventID: "evt-stage", EventType: "report.final_edit.writer.submitted", Payload: []byte(`{"stage":"reader_edit"}`)}); err == nil || ok {
		t.Fatalf("expected invalid stage error, ok=%v err=%v", ok, err)
	}
	if _, ok, err := TargetForEvent(ledger.Event{EventID: "evt-json", EventType: "report.part.edited", Payload: []byte("{")}); err == nil || ok {
		t.Fatalf("expected malformed JSON error, ok=%v err=%v", ok, err)
	}
}

func TestTargetForEventIgnoresNonTargets(t *testing.T) {
	if _, ok, err := TargetForEvent(ledger.Event{EventType: "report.final_edit.writer.started"}); err != nil || ok {
		t.Fatalf("unexpected non-target result ok=%v err=%v", ok, err)
	}
}
