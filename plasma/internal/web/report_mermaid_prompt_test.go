package web

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestMarkdownReportPromptsRequireMermaidValidation(t *testing.T) {
	sectionalPlan := agentSectionalReportPlan{
		Parts: []agentReportPart{{
			Title:    "Part",
			Purpose:  "Purpose",
			Sections: []agentReportSection{{Title: "Section", Purpose: "Purpose"}},
		}},
	}
	section := sectionalPlan.Parts[0].Sections[0]
	binding := reporting.LongFormFinalizeBinding{
		ToolSessionID:  "ses_1",
		PendingEventID: "evt_pending",
		PlanEventID:    "evt_plan",
		IdempotencyKey: "key_1",
	}
	prompts := map[string]string{
		"one-take markdown": agentOneTakeMarkdownReportPrompt("Report", "mis_1", "ses_1", reportRigorProfiles["balanced"], ""),
		"planned markdown":  agentMarkdownReportPrompt("Report", "mis_1", "ses_1", reportRigorProfiles["balanced"], agentReportPlan{}, ""),
		"long section":      agentSectionDraftPrompt("Report", "mis_1", "ses_1", reportRigorProfiles["balanced"], sectionalPlan, sectionalPlan.Parts[0], section, 0, 0, ""),
		"part assembly":     agentPartAssemblyPrompt("Report", "mis_1", "ses_1", reportRigorProfiles["balanced"], sectionalPlan, sectionalPlan.Parts[0], nil, 0, ""),
		"long final":        agentLongFormFinalizePrompt("Report", "mis_1", reportRigorProfiles["balanced"], sectionalPlan, nil, "", binding, 1, false, reporting.LongFormFinalizationHint{}),
	}
	for name, prompt := range prompts {
		if !strings.Contains(prompt, "plasma.mermaid.validate") || !strings.Contains(prompt, "static preflight pass") {
			t.Fatalf("%s prompt missing Mermaid validation rule:\n%s", name, prompt)
		}
	}
}

func TestNonMarkdownReportPromptsDoNotRequireUnavailableMermaidValidation(t *testing.T) {
	prompts := map[string]string{
		"plan":  agentReportPlanPrompt("Report", "mis_1", "ses_1", "evt_pending", "key_1", reportRigorProfiles["balanced"], "visual-plan"),
		"patch": AgentReportPatchPrompt("Report", "mis_1", "ses_1", "evt_pending", "art_1", "edit", reporting.PatchRequest{}),
		"ast":   agentReportPrompt("Report", "mis_1", "ses_1", reportRigorProfiles["balanced"], agentReportPlan{}),
	}
	for name, prompt := range prompts {
		if strings.Contains(prompt, "plasma.mermaid.validate") {
			t.Fatalf("%s prompt should not require Mermaid validation:\n%s", name, prompt)
		}
	}
}
