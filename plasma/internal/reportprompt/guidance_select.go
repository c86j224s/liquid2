package reportprompt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

func SelectReportGenerationGuidance(profile string) (string, string, error) {
	return selectReportGenerationGuidanceText(profile, ReportGenerationGuidance)
}

// SelectReportGenerationGuidanceForMode는 리포트 모드와 profile을 함께 고려해 작성 지침을 고른다.
func SelectReportGenerationGuidanceForMode(reportMode string, profile string) (string, string, error) {
	if reportMode == reportexecution.ModeLongForm && strings.TrimSpace(profile) == "" {
		profile = reportGenerationGuidanceProfileLongFormDefault
	}
	if isReportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice(profile) {
		if reportMode != reportexecution.ModeLongForm {
			return "", "", fmt.Errorf("%w: subject-direct synthesis voice is supported only for long-form reports", producterror.ErrInvalidInput)
		}
		normalized := reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice
		text := strings.TrimSpace(strings.Join([]string{
			LongFormReportGenerationGuidance(normalized),
			reportSectionDirectWritingGuidance(normalized),
			reportPartConnectiveEconomyGuidance(normalized),
			reportSubjectDirectSynthesisSectionGuidance(normalized),
		}, "\n\n"))
		sum := sha256.Sum256([]byte(text))
		return normalized, hex.EncodeToString(sum[:]), nil
	}
	if isReportGenerationGuidanceProfilePartConnectiveEconomyVoice(profile) {
		if reportMode != reportexecution.ModeLongForm {
			return "", "", fmt.Errorf("%w: Part connective economy is supported only for long-form reports", producterror.ErrInvalidInput)
		}
		normalized := reportGenerationGuidanceProfilePartConnectiveEconomyVoice
		text := strings.TrimSpace(strings.Join([]string{
			LongFormReportGenerationGuidance(normalized),
			reportSectionDirectWritingGuidance(normalized),
			reportPartConnectiveEconomyGuidance(normalized),
		}, "\n\n"))
		sum := sha256.Sum256([]byte(text))
		return normalized, hex.EncodeToString(sum[:]), nil
	}
	if isReportGenerationGuidanceProfileSectionDirectReadingVoice(profile) {
		if reportMode != reportexecution.ModeLongForm {
			return "", "", fmt.Errorf("%w: section-direct reading voice is supported only for long-form reports", producterror.ErrInvalidInput)
		}
		normalized := reportGenerationGuidanceProfileSectionDirectReadingVoice
		text := strings.TrimSpace(strings.Join([]string{
			LongFormReportGenerationGuidance(normalized),
			reportSectionDirectWritingGuidance(normalized),
		}, "\n\n"))
		sum := sha256.Sum256([]byte(text))
		return normalized, hex.EncodeToString(sum[:]), nil
	}
	if reportMode == reportexecution.ModeLongForm && isReportGenerationGuidanceProfileLongFormExperiment(profile) {
		normalized := normalizeLongFormExperimentProfile(profile)
		text := LongFormReportGenerationGuidance(normalized)
		sum := sha256.Sum256([]byte(text))
		return normalized, hex.EncodeToString(sum[:]), nil
	}
	if reportMode == reportexecution.ModeLongForm {
		return selectReportGenerationGuidanceText(profile, LongFormReportGenerationGuidance)
	}
	return SelectReportGenerationGuidance(profile)
}

func selectReportGenerationGuidanceText(profile string, guidance func(string) string) (string, string, error) {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case "":
		text := guidance(reportGenerationGuidanceProfileDefault)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileDefault, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileG2, "h5-g2", "substance-preserving-korean", "substance_preserving_korean":
		text := guidance(reportGenerationGuidanceProfileG2)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileG2, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileVisualSupplement, "visual_supplement", "visual-aids", "visual_aids":
		text := guidance(reportGenerationGuidanceProfileVisualSupplement)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileVisualSupplement, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileVisualPlan, "visual_plan", "planned-visual-units", "planned_visual_units":
		text := guidance(reportGenerationGuidanceProfileVisualPlan)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileVisualPlan, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileNarrativeContract, "narrative_contract", "reader-first-editor", "reader_first_editor":
		text := guidance(reportGenerationGuidanceProfileNarrativeContract)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileNarrativeContract, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileReaderParagraphContract, "reader_paragraph_contract", "direct-explanation-contract", "direct_explanation_contract":
		text := guidance(reportGenerationGuidanceProfileReaderParagraphContract)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileReaderParagraphContract, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileCuriosityLedExplanation, "curiosity_led_explanation", "curiosity-explanation", "curiosity_explanation", "processed-reading-artifact", "processed_reading_artifact":
		text := guidance(reportGenerationGuidanceProfileCuriosityLedExplanation)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileCuriosityLedExplanation, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileCuriosityNaturalVoice, "curiosity_natural_voice", "natural-curiosity", "natural_curiosity":
		text := guidance(reportGenerationGuidanceProfileCuriosityNaturalVoice)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileCuriosityNaturalVoice, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileCuriosityTightVoice, "curiosity_tight_voice", "tight-curiosity", "tight_curiosity", "compact-curiosity", "compact_curiosity":
		text := guidance(reportGenerationGuidanceProfileCuriosityTightVoice)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileCuriosityTightVoice, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileEditedReadingVoice, "edited_reading_voice", "edited-reading", "edited_reading", "reading-editor", "reading_editor":
		text := guidance(reportGenerationGuidanceProfileEditedReadingVoice)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileEditedReadingVoice, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileVisualTypeManual, "visual_type_manual", "visual-type-selection", "visual_type_selection":
		text := guidance(reportGenerationGuidanceProfileVisualTypeManual)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileVisualTypeManual, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileVisualEvidenceFit, "visual_evidence_fit", "evidence-fit-visuals", "evidence_fit_visuals":
		text := guidance(reportGenerationGuidanceProfileVisualEvidenceFit)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileVisualEvidenceFit, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileVisualReadingAidPreferred, "visual_reading_aid_preferred", "visual-preferred", "visual_preferred":
		text := guidance(reportGenerationGuidanceProfileVisualReadingAidPreferred)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileVisualReadingAidPreferred, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileVisualReaderIntent, "visual_reader_intent", "reader-intent-visuals", "reader_intent_visuals":
		text := guidance(reportGenerationGuidanceProfileVisualReaderIntent)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileVisualReaderIntent, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileVisualClaritySeeking, "visual_clarity_seeking", "clarity-seeking-visuals", "clarity_seeking_visuals":
		text := guidance(reportGenerationGuidanceProfileVisualClaritySeeking)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileVisualClaritySeeking, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileVisualAffordancePriming, "visual_affordance_priming", "affordance-primed-visuals", "affordance_primed_visuals":
		text := guidance(reportGenerationGuidanceProfileVisualAffordancePriming)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileVisualAffordancePriming, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileNone, "off", "disabled", "disable", "false", "0":
		return reportGenerationGuidanceProfileNone, "", nil
	default:
		return "", "", fmt.Errorf("%w: unsupported report generation guidance profile", producterror.ErrInvalidInput)
	}
}
