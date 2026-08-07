package reportprompt

import (
	"strings"
)

func ReportGenerationPlanningGuidance(profile string) string {
	parts := []string{
		strings.TrimSpace(reportVisualAidPlanningGuidance(profile)),
		strings.TrimSpace(reportNarrativeContractPlanningGuidance(profile)),
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

// ReportGenerationGuidance는 일반 보고서 작성 agent에 전달할 본문 작성 지침을 만든다.
func ReportGenerationGuidance(profile string) string {
	narrativeContract := isReportGenerationGuidanceProfileNarrativeContract(profile)
	if isReportGenerationGuidanceProfileLongFormExperiment(profile) {
		if narrativeContract {
			profile = reportGenerationGuidanceProfileNarrativeContract
		} else if isReportGenerationGuidanceProfileLongFormVisualPlan(profile) || isReportGenerationGuidanceProfilePartAssemblyEditTools(profile) {
			profile = reportGenerationGuidanceProfileVisualPlan
		} else {
			profile = reportGenerationGuidanceProfileG2
		}
	}
	guidance := baseReportGenerationGuidance()
	if isReportGenerationGuidanceProfileVisualAid(profile) {
		guidance += "\n\n" + reportVisualAidWritingGuidance(profile)
	}
	if isReportGenerationGuidanceProfileNarrativeContract(profile) {
		guidance += "\n\n" + reportNarrativeContractWritingGuidance(profile)
	}
	if strings.TrimSpace(profile) != reportGenerationGuidanceProfileG2 && !isReportGenerationGuidanceProfileVisualAid(profile) && !isReportGenerationGuidanceProfileNarrativeContract(profile) {
		return ""
	}
	return guidance
}

// PlannedReportGenerationGuidance builds writer guidance for the planned general-report path.
// It keeps general-report policy out of one-take and long-form prompts.
func PlannedReportGenerationGuidance(profile string) string {
	return strings.TrimSpace(strings.Join([]string{
		ReportGenerationGuidance(profile),
		reportPlannedNarrativeWritingGuidance(profile),
	}, "\n\n"))
}

func baseReportGenerationGuidance() string {
	return `Report writing guidance:
- This guidance controls report writing style only. It is not source material and must not be mentioned in the final report.
- Write natural Korean, but never improve fluency by dropping concrete source details.
- Preserve names, dates, numbers, commands, code identifiers, URLs, conditions, exceptions, caveats, uncertainty, and source distinctions when they matter.
- For mathematical expressions, use only \(...\) for inline math and \[...\] for display math. Do not use $...$ or $$...$$ delimiters.
- Prefer a report that is slightly longer and more specific over a smooth summary that hides evidence, disagreement, or operational detail.
- If sources disagree or only imply something, say that plainly instead of flattening the point into a single confident sentence.
- Do not mention hidden guidance, experiments, prompts, or internal evaluation labels in the report.`
}
