package evidencecheck

import (
	"slices"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

func TestCorrectiveGatePromptAndToolsAreHumanizeAware(t *testing.T) {
	input := promptInput("")
	binding := promptBinding()
	disabled := gatePrompt(input, binding, "draft_1", 1)
	if slices.Contains(gateMCPToolsForHumanize(reporting.FinalEditHumanizeDisabled), mcptools.ToolReportLongFormStyleReviewRead) ||
		strings.Contains(disabled, "style_review") ||
		strings.Contains(disabled, "semantic_acceptance") ||
		!strings.Contains(disabled, "5. Submit with plasma.report.long_form.final_edit.submit and gate_findings") {
		t.Fatalf("disabled corrective gate leaked semantic review:\n%s", disabled)
	}
	input.PostReportHumanize = reporting.FinalEditHumanizeEnabled
	enabled := semanticGatePrompt(input, binding, "draft_1", 1)
	if !slices.Contains(gateMCPToolsForHumanize(reporting.FinalEditHumanizeEnabled), mcptools.ToolReportLongFormStyleReviewRead) ||
		!strings.Contains(enabled, "plasma.report.long_form.style_review.read") ||
		!strings.Contains(enabled, "semantic_acceptance") {
		t.Fatalf("enabled corrective gate omitted semantic review:\n%s", enabled)
	}
}

func TestDirectionDoesNotEnterGatePrompts(t *testing.T) {
	input := promptInput("DIRECTION_SENTINEL")
	binding := promptBinding()
	for name, got := range map[string]string{
		"corrective": gatePrompt(input, binding, "draft_1", 1),
		"semantic":   semanticGatePrompt(input, binding, "draft_1", 1),
		"evidence":   evidencePrompt(input, binding, "draft_1", 1),
	} {
		if strings.Contains(got, input.DirectionHint) {
			t.Fatalf("%s gate prompt leaked direction:\n%s", name, got)
		}
	}
}

func TestEvidencePromptBindsOneDraftSessionAndSequentialRead(t *testing.T) {
	input := promptInput("")
	binding := promptBinding()
	binding.Stage = reporting.FinalEditStageEvidenceGate
	binding.ToolSessionID = "ses_evidence"
	prompt := evidencePrompt(input, binding, "rfe_evidence", 1)
	for _, expected := range []string{
		`draft_id "rfe_evidence"`,
		`session_id "ses_evidence"`,
		"provider session IDs are not MCP session IDs",
		"offset 0",
		"copy the returned next_offset exactly",
		"Do not create or switch drafts",
		"continuation content",
		"exactly once with the same draft_id and session_id",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("evidence prompt missing %q:\n%s", expected, prompt)
		}
	}
}

func promptInput(direction string) finaledit.Input {
	return finaledit.Input{
		Title: "Report", MissionID: "mis_gate", DirectionHint: direction,
		Rigor:     finaledit.Rigor{Level: "balanced", Label: "균형형"},
		PlanEvent: ledger.Event{Payload: []byte(`{"final_edit_pipeline":"assembly_writer_reader_style_validation_evidence_gate_v3","post_report_humanize":"disabled"}`)},
	}
}

func promptBinding() reporting.FinalEditStageBinding {
	return reporting.FinalEditStageBinding{Stage: reporting.FinalEditStageGate, PostReportHumanize: reporting.FinalEditHumanizeDisabled}
}
