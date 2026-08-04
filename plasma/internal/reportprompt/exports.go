package reportprompt

const (
	ProfileG2                                   = reportGenerationGuidanceProfileG2
	ProfileNone                                 = reportGenerationGuidanceProfileNone
	ProfileSectionContract                      = reportGenerationGuidanceProfileSectionContract
	ProfileSectionContractCoverage              = reportGenerationGuidanceProfileSectionContractCoverage
	ProfileSectionIntent                        = reportGenerationGuidanceProfileSectionIntent
	ProfileSourceClusterFirst                   = reportGenerationGuidanceProfileSourceClusterFirst
	ProfileSectionBrief                         = reportGenerationGuidanceProfileSectionBrief
	ProfileSectionBriefCluster                  = reportGenerationGuidanceProfileSectionBriefCluster
	ProfileSectionBriefVisualPlan               = reportGenerationGuidanceProfileSectionBriefVisualPlan
	ProfileSectionBriefClusterVisualPlan        = reportGenerationGuidanceProfileSectionBriefClusterVisualPlan
	ProfilePlanReview                           = reportGenerationGuidanceProfilePlanReview
	ProfilePartAssemblyEditTools                = reportGenerationGuidanceProfilePartAssemblyEditTools
	ProfileVisualSupplement                     = reportGenerationGuidanceProfileVisualSupplement
	ProfileVisualPlan                           = reportGenerationGuidanceProfileVisualPlan
	ProfileDefault                              = reportGenerationGuidanceProfileDefault
	ProfileLongFormDefault                      = reportGenerationGuidanceProfileLongFormDefault
	ProfileNarrativeContract                    = reportGenerationGuidanceProfileNarrativeContract
	ProfileReaderParagraphContract              = reportGenerationGuidanceProfileReaderParagraphContract
	ProfileCuriosityLedExplanation              = reportGenerationGuidanceProfileCuriosityLedExplanation
	ProfileCuriosityNaturalVoice                = reportGenerationGuidanceProfileCuriosityNaturalVoice
	ProfileCuriosityTightVoice                  = reportGenerationGuidanceProfileCuriosityTightVoice
	ProfileEditedReadingVoice                   = reportGenerationGuidanceProfileEditedReadingVoice
	ProfileSectionBriefNarrativeContract        = reportGenerationGuidanceProfileSectionBriefNarrativeContract
	ProfileSectionBriefClusterNarrativeContract = reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract
	ProfileVisualTypeManual                     = reportGenerationGuidanceProfileVisualTypeManual
	ProfileVisualEvidenceFit                    = reportGenerationGuidanceProfileVisualEvidenceFit
	ProfileVisualReadingAidPreferred            = reportGenerationGuidanceProfileVisualReadingAidPreferred
	ProfileVisualReaderIntent                   = reportGenerationGuidanceProfileVisualReaderIntent
	ProfileVisualClaritySeeking                 = reportGenerationGuidanceProfileVisualClaritySeeking
	ProfileVisualAffordancePriming              = reportGenerationGuidanceProfileVisualAffordancePriming
	ProfilePartConnectiveEconomyVoice           = reportGenerationGuidanceProfilePartConnectiveEconomyVoice
	ProfileSectionDirectReadingVoice            = reportGenerationGuidanceProfileSectionDirectReadingVoice
	ProfilePartConnectiveSubjectDirectSynthesis = reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice
	MermaidValidationRuleText                   = reportMermaidValidationRule
)

// LongFormExperimentalPlanningGuidance returns planning-only long-form experiment guidance.
func LongFormExperimentalPlanningGuidance(profile string) string {
	return longFormExperimentalPlanningGuidance(profile)
}

// SectionDirectWritingGuidance returns section direct-writing guidance for matching profiles.
func SectionDirectWritingGuidance(profile string) string {
	return reportSectionDirectWritingGuidance(profile)
}

// PartConnectiveEconomyGuidance returns part connective guidance for matching profiles.
func PartConnectiveEconomyGuidance(profile string) string {
	return reportPartConnectiveEconomyGuidance(profile)
}

// SubjectDirectSynthesisPlanningGuidance returns subject-direct planning guidance.
func SubjectDirectSynthesisPlanningGuidance(profile string) string {
	return reportSubjectDirectSynthesisPlanningGuidance(profile)
}

// SubjectDirectSynthesisSectionGuidance returns subject-direct section guidance.
func SubjectDirectSynthesisSectionGuidance(profile string) string {
	return reportSubjectDirectSynthesisSectionGuidance(profile)
}

// VisualAidPlanningGuidance returns visual-aid planning guidance for matching profiles.
func VisualAidPlanningGuidance(profile string) string {
	return reportVisualAidPlanningGuidance(profile)
}

// IsNarrativeContract reports whether a profile requires the narrative contract family.
func IsNarrativeContract(profile string) bool {
	return isReportGenerationGuidanceProfileNarrativeContract(profile)
}

// IsPartAssemblyEditTools reports whether a profile enables part assembly edit tools.
func IsPartAssemblyEditTools(profile string) bool {
	return isReportGenerationGuidanceProfilePartAssemblyEditTools(profile)
}

// IsVisualPlan reports whether a profile is in the visual-plan family used by legacy callers.
func IsVisualPlan(profile string) bool {
	return isReportGenerationGuidanceProfileVisualPlan(profile)
}

// IsPartConnectiveEconomyVoice reports whether a profile is the part-connective economy family.
func IsPartConnectiveEconomyVoice(profile string) bool {
	return isReportGenerationGuidanceProfilePartConnectiveEconomyVoice(profile)
}

// IsPartConnectiveSubjectDirectSynthesis reports whether a profile is the subject-direct synthesis voice.
func IsPartConnectiveSubjectDirectSynthesis(profile string) bool {
	return isReportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice(profile)
}

// RequireReportWritingContract reports whether report planning must include the writing contract.
func RequireReportWritingContract(profile string) bool {
	return requireReportWritingContract(profile)
}

// LongFormCompositionStrategy selects the long-form composition strategy for a guidance profile.
func LongFormCompositionStrategy(profile string) string {
	return longFormCompositionStrategy(profile)
}

// NormalizePostReportHumanize normalizes the post-report humanize flag.
func NormalizePostReportHumanize(value string) string {
	return normalizePostReportHumanize(value)
}
