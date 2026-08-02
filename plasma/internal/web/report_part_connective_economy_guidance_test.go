package web

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestPartConnectiveEconomyGuidanceStaysWithPartAssembler(t *testing.T) {
	profile, sha, err := SelectReportGenerationGuidanceForMode(reportModeLongForm, "part_connective_economy")
	if err != nil || profile != reportGenerationGuidanceProfilePartConnectiveEconomyVoice || strings.TrimSpace(sha) == "" {
		t.Fatalf("long-form rejected Part connective-economy profile: profile=%q sha=%q err=%v", profile, sha, err)
	}
	for _, mode := range []string{reportModeOneTake, reportModePlanned} {
		if _, _, err := SelectReportGenerationGuidanceForMode(mode, "part_connective_economy"); err == nil {
			t.Fatalf("mode %s accepted long-form-only Part connective-economy profile", mode)
		}
	}
	_, sectionSHA, err := SelectReportGenerationGuidanceForMode(reportModeLongForm, reportGenerationGuidanceProfileSectionDirectReadingVoice)
	if err != nil || sha == sectionSHA {
		t.Fatalf("Part-only guidance must have a distinct policy hash: part=%q section=%q err=%v", sha, sectionSHA, err)
	}
	if !requireReportWritingContract(profile) || longFormCompositionStrategy(profile) != reporting.LongFormCompositionNarrativeEdit {
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
	finalPrompt := agentLongFormFinalizePrompt("Long", binding.MissionID, reportRigorProfiles["balanced"], plan, nil, profile, binding, 1, false, reporting.LongFormFinalizationHint{})

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
	profile, sha, err := SelectReportGenerationGuidanceForMode(reportModeLongForm, "")
	if err != nil || profile != reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract || strings.TrimSpace(sha) == "" {
		t.Fatalf("unexpected long-form default: profile=%q sha=%q err=%v", profile, sha, err)
	}
	explicitProfile, explicitSHA, err := SelectReportGenerationGuidanceForMode(reportModeLongForm, reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract)
	if err != nil || explicitProfile != profile || explicitSHA != sha {
		t.Fatalf("long-form default diverged from the tested profile: default=%q/%q explicit=%q/%q err=%v", profile, sha, explicitProfile, explicitSHA, err)
	}
	legacyProfile, legacySHA, err := SelectReportGenerationGuidanceForMode(reportModeLongForm, reportGenerationGuidanceProfilePartConnectiveEconomyVoice)
	if err != nil || legacyProfile != reportGenerationGuidanceProfilePartConnectiveEconomyVoice || strings.TrimSpace(legacySHA) == "" {
		t.Fatalf("explicit legacy long-form voice profile was not preserved: profile=%q sha=%q err=%v", legacyProfile, legacySHA, err)
	}
	for _, mode := range []string{reportModeOneTake, reportModePlanned} {
		profile, sha, err := SelectReportGenerationGuidanceForMode(mode, "")
		if err != nil || profile != reportGenerationGuidanceProfileNarrativeContract || strings.TrimSpace(sha) == "" {
			t.Fatalf("mode %s default changed: profile=%q sha=%q err=%v", mode, profile, sha, err)
		}
	}
	profile, _, err = SelectReportGenerationGuidanceForMode(reportModeLongForm, reportGenerationGuidanceProfileNarrativeContract)
	if err != nil || profile != reportGenerationGuidanceProfileNarrativeContract {
		t.Fatalf("explicit long-form narrative profile was not preserved: profile=%q err=%v", profile, err)
	}
}
