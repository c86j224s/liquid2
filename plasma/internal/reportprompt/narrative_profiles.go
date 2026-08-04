package reportprompt

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	reportGenerationGuidanceProfileNarrativeContract                    = "narrative-contract"
	reportGenerationGuidanceProfileReaderParagraphContract              = "reader-paragraph-contract"
	reportGenerationGuidanceProfileCuriosityLedExplanation              = "curiosity-led-explanation"
	reportGenerationGuidanceProfileCuriosityNaturalVoice                = "curiosity-natural-voice"
	reportGenerationGuidanceProfileCuriosityTightVoice                  = "curiosity-tight-voice"
	reportGenerationGuidanceProfileEditedReadingVoice                   = "edited-reading-voice"
	reportGenerationGuidanceProfileSectionBriefNarrativeContract        = "section-brief-narrative-contract"
	reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract = "section-brief-cluster-memory-narrative-contract"
)

func isReportGenerationGuidanceProfileNarrativeContract(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileNarrativeContract, "narrative_contract", "reader-first-editor", "reader_first_editor",
		reportGenerationGuidanceProfileReaderParagraphContract, "reader_paragraph_contract", "direct-explanation-contract", "direct_explanation_contract",
		reportGenerationGuidanceProfileCuriosityLedExplanation, "curiosity_led_explanation", "curiosity-explanation", "curiosity_explanation", "processed-reading-artifact", "processed_reading_artifact",
		reportGenerationGuidanceProfileCuriosityNaturalVoice, "curiosity_natural_voice", "natural-curiosity", "natural_curiosity",
		reportGenerationGuidanceProfileCuriosityTightVoice, "curiosity_tight_voice", "tight-curiosity", "tight_curiosity", "compact-curiosity", "compact_curiosity",
		reportGenerationGuidanceProfileEditedReadingVoice, "edited_reading_voice", "edited-reading", "edited_reading", "reading-editor", "reading_editor",
		reportGenerationGuidanceProfileSectionDirectReadingVoice, "section_direct_reading_voice", "section-direct", "section_direct",
		reportGenerationGuidanceProfilePartConnectiveEconomyVoice, "part_connective_economy_voice", "part-connective-economy", "part_connective_economy",
		reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice,
		reportGenerationGuidanceProfileSectionBriefNarrativeContract, "section_brief_narrative_contract",
		reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract, "section_brief_cluster_memory_narrative_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionBriefNarrativeContract(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionBriefNarrativeContract, "section_brief_narrative_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionBriefClusterNarrativeContract(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract, "section_brief_cluster_memory_narrative_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileReaderParagraphContract(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileReaderParagraphContract, "reader_paragraph_contract", "direct-explanation-contract", "direct_explanation_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileCuriosityLedExplanation(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileCuriosityLedExplanation, "curiosity_led_explanation", "curiosity-explanation", "curiosity_explanation", "processed-reading-artifact", "processed_reading_artifact",
		reportGenerationGuidanceProfileCuriosityNaturalVoice, "curiosity_natural_voice", "natural-curiosity", "natural_curiosity",
		reportGenerationGuidanceProfileCuriosityTightVoice, "curiosity_tight_voice", "tight-curiosity", "tight_curiosity", "compact-curiosity", "compact_curiosity",
		reportGenerationGuidanceProfileEditedReadingVoice, "edited_reading_voice", "edited-reading", "edited_reading", "reading-editor", "reading_editor",
		reportGenerationGuidanceProfileSectionDirectReadingVoice, "section_direct_reading_voice", "section-direct", "section_direct",
		reportGenerationGuidanceProfilePartConnectiveEconomyVoice, "part_connective_economy_voice", "part-connective-economy", "part_connective_economy",
		reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice:
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileCuriosityNaturalVoice(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileCuriosityNaturalVoice, "curiosity_natural_voice", "natural-curiosity", "natural_curiosity",
		reportGenerationGuidanceProfileCuriosityTightVoice, "curiosity_tight_voice", "tight-curiosity", "tight_curiosity", "compact-curiosity", "compact_curiosity",
		reportGenerationGuidanceProfileEditedReadingVoice, "edited_reading_voice", "edited-reading", "edited_reading", "reading-editor", "reading_editor",
		reportGenerationGuidanceProfileSectionDirectReadingVoice, "section_direct_reading_voice", "section-direct", "section_direct",
		reportGenerationGuidanceProfilePartConnectiveEconomyVoice, "part_connective_economy_voice", "part-connective-economy", "part_connective_economy",
		reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice:
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileCuriosityTightVoice(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileCuriosityTightVoice, "curiosity_tight_voice", "tight-curiosity", "tight_curiosity", "compact-curiosity", "compact_curiosity":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileEditedReadingVoice(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileEditedReadingVoice, "edited_reading_voice", "edited-reading", "edited_reading", "reading-editor", "reading_editor",
		reportGenerationGuidanceProfileSectionDirectReadingVoice, "section_direct_reading_voice", "section-direct", "section_direct",
		reportGenerationGuidanceProfilePartConnectiveEconomyVoice, "part_connective_economy_voice", "part-connective-economy", "part_connective_economy",
		reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice:
		return true
	default:
		return false
	}
}

func requireReportWritingContract(profile string) bool {
	return isReportGenerationGuidanceProfileNarrativeContract(profile)
}

func longFormCompositionStrategy(profile string) string {
	if isReportGenerationGuidanceProfileNarrativeContract(profile) {
		return reporting.LongFormCompositionNarrativeEdit
	}
	return reporting.LongFormCompositionPreserveMarkdown
}
