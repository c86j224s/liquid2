package web

import (
	"slices"
	"strings"
	"testing"

	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestSectionDirectReadingGuidanceStaysWithSectionWriter(t *testing.T) {
	profile, sha, err := SelectReportGenerationGuidanceForMode(reportModeLongForm, "section_direct")
	if err != nil || profile != reportGenerationGuidanceProfileSectionDirectReadingVoice || strings.TrimSpace(sha) == "" {
		t.Fatalf("long-form rejected section-direct profile: profile=%q sha=%q err=%v", profile, sha, err)
	}
	for _, mode := range []string{reportModeOneTake, reportModePlanned} {
		if _, _, err := SelectReportGenerationGuidanceForMode(mode, "section_direct"); err == nil {
			t.Fatalf("mode %s accepted long-form-only section-direct profile", mode)
		}
	}
	_, editedSHA, err := SelectReportGenerationGuidanceForMode(reportModeLongForm, reportGenerationGuidanceProfileEditedReadingVoice)
	if err != nil || sha == editedSHA {
		t.Fatalf("section-only guidance must have a distinct policy hash: section=%q edited=%q err=%v", sha, editedSHA, err)
	}
	if !requireReportWritingContract(profile) || longFormCompositionStrategy(profile) != reporting.LongFormCompositionNarrativeEdit {
		t.Fatalf("section-direct profile lost the edited-reading contract or final-edit path")
	}

	contract := &reporting.ReportWritingContract{
		CentralQuestion: "question",
		ReaderTakeaway:  "takeaway",
		ReadingPath:     []string{"gap", "resolution"},
		MustKeep:        []string{"detail"},
		VisualRole:      "none needed",
		ToneAndShape:    "direct",
	}
	plan := agentSectionalReportPlan{Summary: "plan", WritingContract: contract}
	part := agentReportPart{Title: "Part"}
	section := agentReportSection{Title: "Section", Purpose: "question, mechanism, payoff"}

	planPrompt := agentSectionalReportPlanPrompt("Long", "mis_1", "ses_1", "evt_1", "key_1", reportRigorProfiles["balanced"], profile)
	sectionPrompt := agentSectionDraftPrompt("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, part, section, 0, 0, profile)
	partPrompt := agentPartAssemblyPrompt("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, part, nil, 0, profile)
	binding := reporting.LongFormFinalizeBinding{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_final",
		IdempotencyKey: "final-key", CompositionStrategy: reporting.LongFormCompositionNarrativeEdit,
	}
	finalPrompt := agentLongFormFinalizePrompt("Long", binding.MissionID, reportRigorProfiles["balanced"], plan, nil, profile, binding, 1, false, reporting.LongFormFinalizationHint{})

	for name, prompt := range map[string]string{"plan": planPrompt, "part": partPrompt, "final": finalPrompt} {
		if strings.Contains(prompt, "Section direct-writing guidance:") {
			t.Fatalf("%s prompt received section-only guidance", name)
		}
		for _, expected := range []string{"Curiosity-led explanation", "Natural curiosity-voice", "Edited reading-voice"} {
			if !strings.Contains(prompt, expected) {
				t.Fatalf("%s prompt lost inherited %q guidance", name, expected)
			}
		}
	}
	for _, expected := range []string{
		"Section direct-writing guidance:",
		"Let the heading carry the Section's structural role",
		"This guidance removes outline narration",
	} {
		if !strings.Contains(sectionPrompt, expected) {
			t.Fatalf("section prompt missing %q:\n%s", expected, sectionPrompt)
		}
	}

	if !slices.Contains(reportPartAssemblyMCPTools(profile), plasmamcp.ToolReportPartSectionRead) ||
		!slices.Contains(reportFinalizeMCPTools(profile), plasmamcp.ToolReportLongFormEditStart) {
		t.Fatalf("section-direct profile lost narrative Part/final editor tools")
	}
}
