package web

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestAgentLongFormReaderEditPromptContractsReaderResponsibilities(t *testing.T) {
	prompt := agentLongFormReaderEditPrompt(
		longFormReaderStyleGatePipelineRequest{title: "Reader Contract", missionID: "mis-test"},
		reporting.FinalEditStageBinding{Stage: reporting.FinalEditStageReader},
		"draft-test",
		1,
	)

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
		if !strings.Contains(prompt, expected) {
			t.Fatalf("reader prompt missing responsibility %q:\n%s", expected, prompt)
		}
	}

	if !strings.Contains(prompt, "Submit unchanged only after a full read finds none of these responsibilities applicable.") {
		t.Fatalf("reader prompt missing narrow no-op rule:\n%s", prompt)
	}
	if strings.Contains(prompt, "Preserve every fact, citation, caveat") {
		t.Fatalf("reader prompt still contains old blanket preservation phrase:\n%s", prompt)
	}
}
