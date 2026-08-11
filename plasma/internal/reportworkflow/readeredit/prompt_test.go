package readeredit

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

func TestPromptContractsReaderResponsibilities(t *testing.T) {
	got := prompt(promptInput(""), promptBinding(), "draft_1", 1)
	for _, expected := range []string{
		"Explain the subject as the report's author to a reader who will only see this report.",
		"Digest the material and present the explanation instead of telling the reader how to interpret the sources.",
		"Use source-boundary language only where it changes claim scope or certainty.",
		"Do not optimize for brevity by itself.",
		"Keep or add explanation when it makes a supported concept, causal link, context, condition, example, or technical detail easier to understand",
		"Keep or create a brief report-level opening that states the subject, central question, and main answer or evidence boundary.",
		"Treat this orientation as useful content, not removable meta-signposting.",
		"Let later transitions follow the subject and the reader's next question.",
		"Remove repeated section roadmaps or writing-process narration, but keep transitions that add context, logic, or stakes.",
		"Clean obviously duplicated headings when their intended form is clear.",
		"Preserve every unique fact, citation, caveat meaning, number, code identifier, technical identifier, uncertainty boundary, and assigned requirement.",
		"Consolidate redundant caveats and source-process narration without losing unique information",
		"keep the remaining limit near the claim it qualifies",
		"Judge repetition by function: keep a brief reminder when a long-form reader or a new context needs it",
		"remove adjacent restatements and section-level duplication",
		"keeping the strongest occurrence and merging unique detail into it",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("reader prompt missing responsibility %q:\n%s", expected, got)
		}
	}
	if !strings.Contains(got, "Submit unchanged only after a full read finds none of these responsibilities applicable.") {
		t.Fatalf("reader prompt missing narrow no-op rule:\n%s", got)
	}
	if strings.Contains(got, "Preserve every fact, citation, caveat") {
		t.Fatalf("reader prompt still contains old blanket preservation phrase:\n%s", got)
	}
}

func TestDirectionOnlyEntersReaderPrompts(t *testing.T) {
	hint := "DIRECTION_SENTINEL"
	for name, input := range map[string]finaledit.Input{
		"v1": promptInput(hint),
		"v2": promptInputWithPipeline(hint, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2),
		"v3": promptInputWithPipeline(hint, reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3),
	} {
		got := prompt(input, promptBinding(), "draft_1", 1)
		if !strings.Contains(got, hint) || !strings.Contains(got, "writing_contract") {
			t.Fatalf("%s reader prompt lost direction or writing contract:\n%s", name, got)
		}
		for _, forbidden := range []string{input.Plan.Summary, `"parts"`} {
			if strings.Contains(got, forbidden) {
				t.Fatalf("%s reader prompt leaked full plan field %q:\n%s", name, forbidden, got)
			}
		}
	}
}

func promptInput(direction string) finaledit.Input {
	return promptInputWithPipeline(direction, reporting.FinalEditPipelineReaderStyleGateV1)
}

func promptInputWithPipeline(direction string, pipeline string) finaledit.Input {
	return finaledit.Input{
		Title: "Long", MissionID: "mis_1", DirectionHint: direction,
		Plan: reporting.SectionalReportPlan{
			Summary: "plan summary",
			WritingContract: &reporting.ReportWritingContract{
				CentralQuestion: "question", ReaderTakeaway: "takeaway",
			},
			Parts: []reporting.ReportPlanPart{{Title: "Part"}},
		},
		PlanEvent: ledger.Event{Payload: []byte(`{"final_edit_pipeline":"` + pipeline + `","post_report_humanize":"disabled"}`)},
	}
}

func promptBinding() reporting.FinalEditStageBinding {
	return reporting.FinalEditStageBinding{MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_1", ProviderSessionID: "provider_1"}
}
