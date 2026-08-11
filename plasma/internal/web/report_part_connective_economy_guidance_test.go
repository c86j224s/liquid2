package web

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/legacyfinalize"
)

func TestPartConnectiveEconomyGuidanceStaysWithPartAssembler(t *testing.T) {
	profile, sha, err := reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, "part_connective_economy")
	if err != nil || profile != reportprompt.ProfilePartConnectiveEconomyVoice || strings.TrimSpace(sha) == "" {
		t.Fatalf("long-form rejected Part connective-economy profile: profile=%q sha=%q err=%v", profile, sha, err)
	}
	for _, mode := range []string{reportModeOneTake, reportModePlanned} {
		if _, _, err := reportprompt.SelectReportGenerationGuidanceForMode(mode, "part_connective_economy"); err == nil {
			t.Fatalf("mode %s accepted long-form-only Part connective-economy profile", mode)
		}
	}
	_, sectionSHA, err := reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, reportprompt.ProfileSectionDirectReadingVoice)
	if err != nil || sha == sectionSHA {
		t.Fatalf("Part-only guidance must have a distinct policy hash: part=%q section=%q err=%v", sha, sectionSHA, err)
	}
	if !reportprompt.RequireReportWritingContract(profile) || reportprompt.LongFormCompositionStrategy(profile) != reporting.LongFormCompositionNarrativeEdit {
		t.Fatalf("Part connective-economy profile lost the edited-reading contract or final-edit path")
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
	drafts := []sectionalReportDraft{{Title: "Section", ArtifactID: "art_1", WordCount: 120}}

	planPrompt := agentSectionalReportPlanPrompt("Long", "mis_1", "ses_1", "evt_1", "key_1", reportRigorProfiles["balanced"], profile)
	sectionPrompt := agentSectionDraftPrompt("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, part, section, 0, 0, profile)
	partPrompt := agentPartAssemblyPrompt("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, part, drafts, 0, profile)
	partEditPrompt := agentPartAssemblyEditToolsPrompt(reportPartAssemblyAgentRequest{
		title: "Long", missionID: "mis_1", toolSessionID: "ses_1", rigor: reportRigorProfiles["balanced"],
		plan: plan, part: part, drafts: drafts, partIndex: 0, generationGuidanceProfile: profile,
	}, reporting.PartAssemblyBinding{}, "draft_1")
	binding := reporting.LongFormFinalizeBinding{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_final",
		IdempotencyKey: "final-key", CompositionStrategy: reporting.LongFormCompositionNarrativeEdit,
	}
	finalPrompt := legacyfinalize.PromptWithRequirements(legacyfinalize.Input{
		MissionID: binding.MissionID, Title: "Long", Rigor: reportWorkflowRigor(reportRigorProfiles["balanced"]),
		Plan: plan, GenerationGuidanceProfile: profile,
	}, binding, 1, false, reporting.LongFormFinalizationHint{})

	for name, prompt := range map[string]string{"plan": planPrompt, "section": sectionPrompt, "final": finalPrompt} {
		if strings.Contains(prompt, "Part connective-economy guidance:") {
			t.Fatalf("%s prompt received Part-only guidance", name)
		}
	}
	if !strings.Contains(sectionPrompt, "Section direct-writing guidance:") {
		t.Fatalf("candidate Section prompt lost direct-writing guidance")
	}
	for name, prompt := range map[string]string{"part": partPrompt, "part edit": partEditPrompt} {
		for _, expected := range []string{
			"Part connective-economy guidance:",
			"Default to no transitions",
			"reduces assembly overhead, not source-backed explanation",
		} {
			if !strings.Contains(prompt, expected) {
				t.Fatalf("%s prompt missing %q:\n%s", name, expected, prompt)
			}
		}
	}
}

func TestRichSectionGuidanceIsOnlyTheLongFormDefault(t *testing.T) {
	profile, sha, err := reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, "")
	if err != nil || profile != reportprompt.ProfileSectionBriefClusterNarrativeContract || strings.TrimSpace(sha) == "" {
		t.Fatalf("unexpected long-form default: profile=%q sha=%q err=%v", profile, sha, err)
	}
	explicitProfile, explicitSHA, err := reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, reportprompt.ProfileSectionBriefClusterNarrativeContract)
	if err != nil || explicitProfile != profile || explicitSHA != sha {
		t.Fatalf("long-form default diverged from the tested profile: default=%q/%q explicit=%q/%q err=%v", profile, sha, explicitProfile, explicitSHA, err)
	}
	aliasProfile, aliasSHA, err := reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, "section_brief_cluster_memory_narrative_contract")
	if err != nil || aliasProfile != profile || aliasSHA != sha {
		t.Fatalf("long-form default alias diverged from canonical default: default=%q/%q alias=%q/%q err=%v", profile, sha, aliasProfile, aliasSHA, err)
	}
	legacyProfile, legacySHA, err := reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, reportprompt.ProfilePartConnectiveEconomyVoice)
	if err != nil || legacyProfile != reportprompt.ProfilePartConnectiveEconomyVoice || strings.TrimSpace(legacySHA) == "" {
		t.Fatalf("explicit legacy long-form voice profile was not preserved: profile=%q sha=%q err=%v", legacyProfile, legacySHA, err)
	}
	for _, mode := range []string{reportModeOneTake, reportModePlanned} {
		profile, sha, err := reportprompt.SelectReportGenerationGuidanceForMode(mode, "")
		if err != nil || profile != reportprompt.ProfileNarrativeContract || strings.TrimSpace(sha) == "" {
			t.Fatalf("mode %s default changed: profile=%q sha=%q err=%v", mode, profile, sha, err)
		}
	}
	profile, _, err = reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, reportprompt.ProfileNarrativeContract)
	if err != nil || profile != reportprompt.ProfileNarrativeContract {
		t.Fatalf("explicit long-form narrative profile was not preserved: profile=%q err=%v", profile, err)
	}
}

func TestLongFormDefaultPolicyReachesProductPrompts(t *testing.T) {
	profile, _, err := reportprompt.SelectReportGenerationGuidanceForMode(reportModeLongForm, "section_brief_cluster_memory_narrative_contract")
	if err != nil || profile != reportprompt.ProfileLongFormDefault {
		t.Fatalf("long-form default alias did not normalize to active default: profile=%q err=%v", profile, err)
	}
	plan := agentSectionalReportPlan{Summary: "plan"}
	part := agentReportPart{Title: "Part"}
	section := agentReportSection{Title: "Section", Purpose: "mechanism and consequence"}
	drafts := []sectionalReportDraft{{Title: "Section", ArtifactID: "art_1", WordCount: 120}}

	planPrompt := agentSectionalReportPlanPrompt("Long", "mis_1", "ses_1", "evt_1", "key_1", reportRigorProfiles["balanced"], profile)
	if !strings.Contains(planPrompt, "Default long-form planning guidance:") {
		t.Fatalf("default planning marker did not reach plan prompt:\n%s", planPrompt)
	}

	sectionPrompt := agentSectionDraftPromptWithRequirements("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, part, section, 0, 0, profile, nil)
	if !strings.Contains(sectionPrompt, "Default long-form Section writing guidance:") {
		t.Fatalf("default Section marker did not reach Section prompt:\n%s", sectionPrompt)
	}

	partEditPrompt := agentPartAssemblyEditToolsPrompt(reportPartAssemblyAgentRequest{
		title: "Long", missionID: "mis_1", toolSessionID: "ses_1", rigor: reportRigorProfiles["balanced"],
		plan: plan, part: part, drafts: drafts, partIndex: 0, generationGuidanceProfile: profile,
	}, reporting.PartAssemblyBinding{}, "draft_1")
	if !strings.Contains(partEditPrompt, "Intro, transitions, and closing are optional. Add them only when actual Section relationships justify connective text.") ||
		strings.Contains(partEditPrompt, "Prefer one good intro and one good closing over many filler transitions.") {
		t.Fatalf("default MCP Part assembly prompt did not use optional connective rule:\n%s", partEditPrompt)
	}

	legacyPlanPrompt := agentSectionalReportPlanPrompt("Long", "mis_1", "ses_1", "evt_1", "key_1", reportRigorProfiles["balanced"], reportprompt.ProfileNarrativeContract)
	legacySectionPrompt := agentSectionDraftPromptWithRequirements("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, part, section, 0, 0, reportprompt.ProfileNarrativeContract, nil)
	legacyPartEditPrompt := agentPartAssemblyEditToolsPrompt(reportPartAssemblyAgentRequest{
		title: "Long", missionID: "mis_1", toolSessionID: "ses_1", rigor: reportRigorProfiles["balanced"],
		plan: plan, part: part, drafts: drafts, partIndex: 0, generationGuidanceProfile: reportprompt.ProfileNarrativeContract,
	}, reporting.PartAssemblyBinding{}, "draft_1")
	legacyCombined := strings.Join([]string{legacyPlanPrompt, legacySectionPrompt, legacyPartEditPrompt}, "\n")
	for _, marker := range []string{
		"Default long-form planning guidance:",
		"Default long-form Section writing guidance:",
		"Default long-form adjacent-boundary audit:",
	} {
		if strings.Contains(legacyCombined, marker) {
			t.Fatalf("narrative-contract received default-only marker %q:\n%s", marker, legacyCombined)
		}
	}
	if !strings.Contains(legacyPartEditPrompt, "Prefer one good intro and one good closing over many filler transitions.") ||
		strings.Contains(legacyPartEditPrompt, "Intro, transitions, and closing are optional. Add them only when actual Section relationships justify connective text.") {
		t.Fatalf("narrative-contract MCP Part assembly prompt did not retain old intro/closing preference:\n%s", legacyPartEditPrompt)
	}
}

func TestLongFormDefaultPartAssemblyUsesConnectiveEconomy(t *testing.T) {
	profile := reportprompt.ProfileLongFormDefault
	plan := agentSectionalReportPlan{Summary: "plan"}
	part := agentReportPart{Title: "Part"}
	drafts := []sectionalReportDraft{{Title: "Section", ArtifactID: "art_1", WordCount: 120}}

	partPrompt := agentPartAssemblyPrompt("Long", "mis_1", "ses_1", reportRigorProfiles["balanced"], plan, part, drafts, 0, profile)
	for _, expected := range []string{
		"Part connective-economy guidance:",
		"Default to no transitions",
		"Leave the closing empty unless",
	} {
		if !strings.Contains(partPrompt, expected) {
			t.Fatalf("default Part prompt missing %q:\n%s", expected, partPrompt)
		}
	}
	if strings.Contains(partPrompt, "Prefer one good intro and one good closing over many filler transitions") {
		t.Fatalf("Part prompt still prefers filler intro/closing:\n%s", partPrompt)
	}

	partEditPrompt := agentPartAssemblyEditToolsPrompt(reportPartAssemblyAgentRequest{
		title: "Long", missionID: "mis_1", toolSessionID: "ses_1", rigor: reportRigorProfiles["balanced"],
		plan: plan, part: part, drafts: drafts, partIndex: 0, generationGuidanceProfile: profile,
	}, reporting.PartAssemblyBinding{}, "draft_1")
	if !strings.Contains(partEditPrompt, "Intro, transitions, and closing are optional") ||
		!strings.Contains(partEditPrompt, "actual Section relationships justify connective text") ||
		strings.Contains(partEditPrompt, "Prefer one good intro and one good closing over many filler transitions") {
		t.Fatalf("Part edit-tools prompt did not adopt optional connective economy:\n%s", partEditPrompt)
	}
}
