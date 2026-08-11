package finalwrite

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

func TestPromptCarriesDirectionAndOnlyWritingContract(t *testing.T) {
	input := promptInput("DIRECTION_SENTINEL")
	got := prompt(input, promptBinding(), "draft_1", 1)
	for _, expected := range []string{"DIRECTION_SENTINEL", "writing_contract", finaledit.StageSubmittedSentinel} {
		if !strings.Contains(got, expected) {
			t.Fatalf("writer prompt missing %q:\n%s", expected, got)
		}
	}
	for _, forbidden := range []string{input.Plan.Summary, `"parts"`} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("writer prompt leaked full plan field %q:\n%s", forbidden, got)
		}
	}
}

func promptInput(direction string) finaledit.Input {
	return finaledit.Input{
		Title: "Long", MissionID: "mis_1", DirectionHint: direction,
		Plan: reporting.SectionalReportPlan{
			Summary: "plan summary",
			WritingContract: &reporting.ReportWritingContract{
				CentralQuestion: "question", ReaderTakeaway: "takeaway",
			},
			Parts: []reporting.ReportPlanPart{{Title: "Part"}},
		},
		PlanEvent: ledger.Event{Payload: []byte(`{"final_edit_pipeline":"assembly_writer_reader_style_validation_evidence_gate_v3","post_report_humanize":"disabled"}`)},
	}
}

func promptBinding() reporting.FinalEditStageBinding {
	return reporting.FinalEditStageBinding{MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_1", ProviderSessionID: "provider_1"}
}
