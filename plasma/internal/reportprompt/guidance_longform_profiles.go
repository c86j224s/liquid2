package reportprompt

import (
	"strings"
)

func isReportGenerationGuidanceProfileSectionIntent(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionIntent, "section_intent", "reader-intent", "reader_intent", "section-reader-intent", "section_reader_intent":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionContract(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionContract, "section_contract", "sectioncontract", "section-purpose-contract", "section_purpose_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionContractCoverage(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionContractCoverage, "section_contract_coverage", "section-contract-coverage-locked", "section_contract_coverage_locked":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionContractFamily(profile string) bool {
	return isReportGenerationGuidanceProfileSectionContract(profile) || isReportGenerationGuidanceProfileSectionContractCoverage(profile)
}

func isReportGenerationGuidanceProfileLongFormExperiment(profile string) bool {
	return isReportGenerationGuidanceProfileSectionContractFamily(profile) ||
		isReportGenerationGuidanceProfileSectionIntent(profile) ||
		isReportGenerationGuidanceProfileSourceClusterFirst(profile) ||
		isReportGenerationGuidanceProfileSectionBrief(profile) ||
		isReportGenerationGuidanceProfileSectionBriefCluster(profile) ||
		isReportGenerationGuidanceProfileLongFormVisualPlan(profile) ||
		isReportGenerationGuidanceProfilePlanReview(profile) ||
		isReportGenerationGuidanceProfilePartAssemblyEditTools(profile)
}

func normalizeLongFormExperimentProfile(profile string) string {
	if isReportGenerationGuidanceProfileSectionContractCoverage(profile) {
		return reportGenerationGuidanceProfileSectionContractCoverage
	}
	if isReportGenerationGuidanceProfileSectionIntent(profile) {
		return reportGenerationGuidanceProfileSectionIntent
	}
	if isReportGenerationGuidanceProfileSourceClusterFirst(profile) {
		return reportGenerationGuidanceProfileSourceClusterFirst
	}
	if isReportGenerationGuidanceProfileSectionBriefNarrativeContract(profile) {
		return reportGenerationGuidanceProfileSectionBriefNarrativeContract
	}
	if isReportGenerationGuidanceProfileSectionBriefClusterNarrativeContract(profile) {
		return reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract
	}
	if isReportGenerationGuidanceProfileSectionBriefVisualPlan(profile) {
		return reportGenerationGuidanceProfileSectionBriefVisualPlan
	}
	if isReportGenerationGuidanceProfileSectionBriefClusterVisualPlan(profile) {
		return reportGenerationGuidanceProfileSectionBriefClusterVisualPlan
	}
	if isReportGenerationGuidanceProfileSectionBrief(profile) {
		return reportGenerationGuidanceProfileSectionBrief
	}
	if isReportGenerationGuidanceProfileSectionBriefCluster(profile) {
		return reportGenerationGuidanceProfileSectionBriefCluster
	}
	if isReportGenerationGuidanceProfilePlanReview(profile) {
		return reportGenerationGuidanceProfilePlanReview
	}
	if isReportGenerationGuidanceProfilePartAssemblyEditTools(profile) {
		return reportGenerationGuidanceProfilePartAssemblyEditTools
	}
	return reportGenerationGuidanceProfileSectionContract
}

func longFormExperimentalPlanningGuidance(profile string) string {
	if isReportGenerationGuidanceProfilePartAssemblyEditTools(profile) {
		return reportVisualAidPlanningGuidance(reportGenerationGuidanceProfileVisualPlan)
	}
	parts := []string{
		strings.TrimSpace(reportVisualAidPlanningGuidance(profile)),
		strings.TrimSpace(longFormSectionContractPlanningGuidance(profile)),
		strings.TrimSpace(longFormSectionIntentPlanningGuidance(profile)),
		strings.TrimSpace(longFormSourceClusterFirstPlanningGuidance(profile)),
		strings.TrimSpace(longFormSectionBriefPlanningGuidance(profile)),
		strings.TrimSpace(longFormSectionBriefClusterMemoryPlanningGuidance(profile)),
		strings.TrimSpace(longFormPlanReviewPlanningGuidance(profile)),
		strings.TrimSpace(reportSubjectDirectSynthesisPlanningGuidance(profile)),
		strings.TrimSpace(reportNarrativeContractPlanningGuidance(profile)),
	}
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "\n\n")
}
