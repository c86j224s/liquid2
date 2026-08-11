package web

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/legacyfinalize"
)

func TestSubjectDirectSynthesisProfileTargetsPlanningAndSectionOnly(t *testing.T) {
	profile, sha, err := reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, reportprompt.ProfilePartConnectiveSubjectDirectSynthesis)
	if err != nil || profile != reportprompt.ProfilePartConnectiveSubjectDirectSynthesis || strings.TrimSpace(sha) == "" {
		t.Fatalf("subject-direct profile selection failed: profile=%q sha=%q err=%v", profile, sha, err)
	}
	for _, mode := range []string{reportModeOneTake, reportModePlanned} {
		if _, _, err := reportprompt.SelectReportGenerationGuidanceForMode(mode, profile); err == nil {
			t.Fatalf("subject-direct profile must reject non-long-form mode %s", mode)
		}
	}
	if !reportprompt.RequireReportWritingContract(profile) || reportprompt.LongFormCompositionStrategy(profile) != reporting.LongFormCompositionNarrativeEdit {
		t.Fatal("subject-direct profile did not inherit the current long-form default lineage")
	}
	if !strings.Contains(reportprompt.SectionDirectWritingGuidance(profile), "Section direct-writing guidance:") {
		t.Fatal("subject-direct profile did not inherit Section direct-writing guidance")
	}
	if !strings.Contains(reportprompt.PartConnectiveEconomyGuidance(profile), "Part connective-economy guidance:") {
		t.Fatal("subject-direct profile did not inherit Part connective-economy guidance")
	}
	if strings.Contains(reportprompt.LongFormReportGenerationGuidance(profile), "Long-form section-brief cluster-memory guidance:") {
		t.Fatal("subject-direct profile must not inherit Section brief cluster-memory behavior")
	}

	contract := &reporting.ReportWritingContract{
		CentralQuestion: "question",
		ReaderTakeaway:  "takeaway",
		ReadingPath:     []string{"subject move"},
		MustKeep:        []string{"citation", "mechanism", "number"},
		ToneAndShape:    "subject-direct",
	}
	plan := agentSectionalReportPlan{Summary: "plan", WritingContract: contract}
	part := agentReportPart{Title: "Part"}
	section := agentReportSection{Title: "Section", Purpose: "explain the mechanism and caveat directly"}
	drafts := []sectionalReportDraft{{Title: "Section", ArtifactID: "art_1", WordCount: 120}}

	planPrompt := agentSectionalReportPlanPrompt("Long", "mis_1", "ses_1", "evt_1", "key_1", reportRigorProfiles["balanced"], profile)
	sectionPrompt := agentSectionDraftPrompt("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, part, section, 0, 0, profile)
	partPrompt := agentPartAssemblyPrompt("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, part, drafts, 0, profile)
	defaultPartPrompt := agentPartAssemblyPrompt("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, part, drafts, 0, reportprompt.ProfilePartConnectiveEconomyVoice)
	binding := reporting.LongFormFinalizeBinding{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_final",
		IdempotencyKey: "final-key", CompositionStrategy: reporting.LongFormCompositionNarrativeEdit,
	}
	finalPrompt := legacyfinalize.PromptWithRequirements(legacyfinalize.Input{
		MissionID: binding.MissionID, Title: "Long", Rigor: reportWorkflowRigor(reportRigorProfiles["balanced"]),
		Plan: plan, GenerationGuidanceProfile: profile,
	}, binding, 1, false, reporting.LongFormFinalizationHint{})
	defaultFinalPrompt := legacyfinalize.PromptWithRequirements(legacyfinalize.Input{
		MissionID: binding.MissionID, Title: "Long", Rigor: reportWorkflowRigor(reportRigorProfiles["balanced"]),
		Plan: plan, GenerationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
	}, binding, 1, false, reporting.LongFormFinalizationHint{})

	for _, expected := range []string{
		"Subject-direct synthesis planning guidance:",
		"writing_contract.tone_and_shape",
		"writing_contract.reading_path",
		"first real subject claim",
	} {
		if !strings.Contains(planPrompt, expected) {
			t.Fatalf("plan prompt missing %q:\n%s", expected, planPrompt)
		}
	}
	for _, expected := range []string{
		"Subject-direct synthesis Section guidance:",
		"actual subject and predicate",
		"version differences, inter-source disagreement",
		"preserving citations, mechanisms, examples, numbers, caveats, comparisons, and unresolved tensions",
	} {
		if !strings.Contains(sectionPrompt, expected) {
			t.Fatalf("section prompt missing %q:\n%s", expected, sectionPrompt)
		}
	}
	for name, prompt := range map[string]string{"part": partPrompt, "final": finalPrompt} {
		if strings.Contains(prompt, "Subject-direct synthesis") || strings.Contains(prompt, "actual subject and predicate") {
			t.Fatalf("%s prompt received subject-direct block:\n%s", name, prompt)
		}
	}
	if partPrompt != defaultPartPrompt {
		t.Fatalf("candidate Part prompt must match current default exactly\ncandidate:\n%s\ncurrent default:\n%s", partPrompt, defaultPartPrompt)
	}
	if finalPrompt != defaultFinalPrompt {
		t.Fatalf("candidate final prompt must match current default exactly\ncandidate:\n%s\ncurrent default:\n%s", finalPrompt, defaultFinalPrompt)
	}
}

func TestSubjectDirectSynthesisDoesNotChangeLongFormDefault(t *testing.T) {
	profile, sha, err := reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, "")
	if err != nil || profile != reportprompt.ProfileSectionBriefClusterNarrativeContract || strings.TrimSpace(sha) == "" {
		t.Fatalf("long-form default changed: profile=%q sha=%q err=%v", profile, sha, err)
	}
}
