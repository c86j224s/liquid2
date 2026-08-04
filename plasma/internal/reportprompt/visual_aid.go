package reportprompt

import (
	"strings"
)

func reportVisualAidWritingGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileVisualAid(profile) {
		return ""
	}
	guidance := `Report visual-aid guidance:
- Use tables or Mermaid diagrams only when they help a reader understand comparison, sequence, flow, dependency, condition, timeline, hierarchy, or trade-off better than prose alone.
- A visual aid must supplement the prose, not replace it. Introduce why it is useful, then explain the takeaway after it.
- Keep prose source-grounded and specific. Do not add decorative visuals, filler tables, or diagrams that merely repeat adjacent paragraphs.
- If no natural visual structure appears after reading the sources, do not include a table or Mermaid diagram.`
	if isReportGenerationGuidanceProfileVisualPlanFamily(profile) {
		guidance += `
- Follow the generation plan's visual-aid intent when it is useful, but do not force a planned table or diagram if later source reads show prose is clearer.
- When a planned visual aid is used, make its purpose obvious in the nearby prose.`
	}
	if isReportGenerationGuidanceProfileVisualPlanFamily(profile) {
		guidance += "\n" + reportVisualTypeSelectionWritingGuidance()
	}
	if isReportGenerationGuidanceProfileVisualEvidenceFit(profile) {
		guidance += "\n" + reportVisualEvidenceFitWritingGuidance()
	}
	if isReportGenerationGuidanceProfileVisualReadingAidPreferred(profile) {
		guidance += "\n" + reportVisualReadingAidPreferenceWritingGuidance()
	}
	if isReportGenerationGuidanceProfileVisualReaderIntent(profile) {
		guidance += "\n" + reportVisualReaderIntentWritingGuidance()
	}
	if isReportGenerationGuidanceProfileVisualClaritySeeking(profile) {
		guidance += "\n" + reportVisualClaritySeekingWritingGuidance()
	}
	if isReportGenerationGuidanceProfileProductAffordancePriming(profile) {
		guidance += "\n" + reportVisualAffordancePrimingWritingGuidance()
	}
	return guidance
}

func reportVisualAidPlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileVisualPlanFamily(profile) {
		return ""
	}
	guidance := `Visual-aid planning guidance:
- While planning, decide whether any section naturally benefits from a table or Mermaid diagram.
- Keep the submitted plan schema unchanged. Put visual-aid intent inside the existing section purpose or coverage_notes; do not add new fields.
- State the purpose of each planned visual aid in plain prose: comparison, process, dependency, timeline, decision path, or trade-off.
- Plan zero visual aids when the report would read better as prose.`
	if isReportGenerationGuidanceProfileVisualPlanFamily(profile) {
		guidance += "\n" + reportVisualTypeSelectionPlanningGuidance()
	}
	if isReportGenerationGuidanceProfileVisualEvidenceFit(profile) {
		guidance += "\n" + reportVisualEvidenceFitPlanningGuidance()
	}
	if isReportGenerationGuidanceProfileVisualReadingAidPreferred(profile) {
		guidance += "\n" + reportVisualReadingAidPreferencePlanningGuidance()
	}
	if isReportGenerationGuidanceProfileVisualReaderIntent(profile) {
		guidance += "\n" + reportVisualReaderIntentPlanningGuidance()
	}
	if isReportGenerationGuidanceProfileVisualClaritySeeking(profile) {
		guidance += "\n" + reportVisualClaritySeekingPlanningGuidance()
	}
	if isReportGenerationGuidanceProfileProductAffordancePriming(profile) {
		guidance += "\n" + reportVisualAffordancePrimingPlanningGuidance()
	}
	return guidance
}

func isReportGenerationGuidanceProfileVisualAid(profile string) bool {
	return isReportGenerationGuidanceProfileVisualSupplement(profile) ||
		isReportGenerationGuidanceProfileVisualPlan(profile) ||
		isReportGenerationGuidanceProfileNarrativeContract(profile) ||
		isReportGenerationGuidanceProfileVisualTypeManual(profile) ||
		isReportGenerationGuidanceProfileVisualEvidenceFit(profile) ||
		isReportGenerationGuidanceProfileVisualReadingAidPreferred(profile) ||
		isReportGenerationGuidanceProfileVisualReaderIntent(profile) ||
		isReportGenerationGuidanceProfileVisualClaritySeeking(profile) ||
		isReportGenerationGuidanceProfileVisualAffordancePriming(profile)
}

func isReportGenerationGuidanceProfileVisualSupplement(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileVisualSupplement, "visual_supplement", "visual-aids", "visual_aids":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileVisualPlan(profile string) bool {
	if isReportGenerationGuidanceProfileNarrativeContract(profile) {
		return true
	}
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileVisualPlan, "visual_plan", "planned-visual-units", "planned_visual_units",
		reportGenerationGuidanceProfileSectionBriefVisualPlan, "section_brief_visual_plan",
		reportGenerationGuidanceProfileSectionBriefClusterVisualPlan, "section_brief_cluster_memory_visual_plan":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileVisualPlanFamily(profile string) bool {
	return isReportGenerationGuidanceProfileVisualPlan(profile) ||
		isReportGenerationGuidanceProfileVisualTypeManual(profile) ||
		isReportGenerationGuidanceProfileVisualEvidenceFit(profile) ||
		isReportGenerationGuidanceProfileVisualReadingAidPreferred(profile) ||
		isReportGenerationGuidanceProfileVisualReaderIntent(profile) ||
		isReportGenerationGuidanceProfileVisualClaritySeeking(profile) ||
		isReportGenerationGuidanceProfileVisualAffordancePriming(profile)
}

func isReportGenerationGuidanceProfileProductAffordancePriming(profile string) bool {
	return isReportGenerationGuidanceProfileVisualPlan(profile) ||
		isReportGenerationGuidanceProfileVisualAffordancePriming(profile)
}

func isReportGenerationGuidanceProfileLongFormVisualPlan(profile string) bool {
	return isReportGenerationGuidanceProfileSectionBriefVisualPlan(profile) ||
		isReportGenerationGuidanceProfileSectionBriefClusterVisualPlan(profile)
}
