package reportprompt

import (
	"strings"
)

func longFormSectionContractPlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileSectionContractFamily(profile) {
		return ""
	}
	guidance := `Section-contract planning guidance:
- Write each part purpose as the part's role in the whole report.
- Write each section purpose as a compact contract with this content in natural prose: central point, reader takeaway, evidence path, and boundary.
- The boundary should say what the section must not drift into, especially broad background, repeated caveats, or sibling-section material.
- Preserve long-form richness: do not collapse source clusters, reduce necessary Part/Section coverage, or shorten the outline merely because the section purposes are more concrete.
- Keep the submitted JSON schema unchanged. Put this contract inside the existing purpose string; do not add new fields.`
	if isReportGenerationGuidanceProfileSectionContractCoverage(profile) {
		guidance += `
- Coverage lock: keep the normal long-form coverage range unless the source packet is genuinely small. For ordinary source packets, prefer preserving or expanding cluster coverage over reducing the outline to a tidier shape.
- Count the planned Sections before submitting. If the plan has fewer than 9 Sections, coverage_notes must state why the source material is too small for the normal range.
- Every major source-backed cluster found through research tools should appear in a section, a coverage note, or a planned omission.`
	}
	return guidance
}

func longFormSectionIntentPlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileSectionIntent(profile) {
		return ""
	}
	return `Section-intent planning guidance:
- Keep the submitted JSON schema unchanged. Put this intent inside the existing purpose string; do not add new fields.
- Write each part purpose as the part's role in the reader's path through the report.
- Write each section purpose as quiet editorial intent: what the reader should come to notice, understand, or question by the end of that section.
- This is not a coverage lock and not a section-count constraint. Let source-backed clusters determine the natural number and size of sections.
- Avoid checklist language. A useful purpose should give the section writer a direction of travel while leaving room for concrete evidence, nuance, and source-backed uncertainty.`
}

func isReportGenerationGuidanceProfileSourceClusterFirst(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSourceClusterFirst, "source_cluster_first", "cluster-first", "cluster_first":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionBrief(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionBrief, "section_brief", "section-writing-brief", "section_writing_brief",
		reportGenerationGuidanceProfileSectionBriefVisualPlan, "section_brief_visual_plan",
		reportGenerationGuidanceProfileSectionBriefNarrativeContract, "section_brief_narrative_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionBriefCluster(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionBriefCluster, "section_brief_cluster_memory", "section-brief-cluster", "section_brief_cluster",
		reportGenerationGuidanceProfileSectionBriefClusterVisualPlan, "section_brief_cluster_memory_visual_plan",
		reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract, "section_brief_cluster_memory_narrative_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionBriefVisualPlan(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionBriefVisualPlan, "section_brief_visual_plan",
		reportGenerationGuidanceProfileSectionBriefNarrativeContract, "section_brief_narrative_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionBriefClusterVisualPlan(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionBriefClusterVisualPlan, "section_brief_cluster_memory_visual_plan",
		reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract, "section_brief_cluster_memory_narrative_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfilePlanReview(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfilePlanReview, "plan_review", "thin-plan-review", "thin_plan_review":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfilePartAssemblyEditTools(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfilePartAssemblyEditTools, "part_assembly_edit_tools", "part-assembly-tools", "part_assembly_tools":
		return true
	default:
		return false
	}
}

func longFormSourceClusterFirstPlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileSourceClusterFirst(profile) {
		return ""
	}
	return `Source-cluster-first planning guidance:
- Before outlining, use the research tools to identify the major source-backed clusters: definitions, mechanisms, examples, numbers, tensions, caveats, comparisons, and missing evidence.
- Build the Parts and Sections only after that cluster pass. The outline should preserve the important clusters instead of choosing the neatest shape first.
- Keep the submitted JSON schema unchanged. Use coverage_notes to record the cluster map: cluster -> planned Section, planned omission, or reason it is out of scope.
- Section purposes should still be readable prose, but they should point to the cluster they preserve and the reader understanding it supports.
- Do not make the report shorter merely because the cluster map is tidy. If a cluster matters, keep room for it.`
}

func longFormSectionBriefPlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileSectionBrief(profile) {
		return ""
	}
	return `Section-brief planning guidance:
- Keep the submitted JSON schema unchanged. Put the brief inside the existing purpose string; do not add new fields.
- Write each Section purpose as a light writing brief, not a rigid template.
- A useful brief should naturally include: what the reader should come to understand, which concrete details or source-backed examples should stay visible, what tension or caveat the section should handle, and which nearby topic should not absorb the section.
- Do not turn those elements into labels. Write one compact prose purpose that gives the section writer a usable direction of travel.
- Preserve long-form richness. A sharper brief must not become a reason to omit useful source clusters.`
}

func longFormSectionBriefClusterMemoryPlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileSectionBriefCluster(profile) {
		return ""
	}
	return `Section-brief cluster-memory planning guidance:
- Keep the submitted JSON schema unchanged. Put the brief inside the existing purpose string; do not add new fields.
- While researching, notice the source-backed clusters that should not disappear: mechanisms, examples, numbers, caveats, comparisons, policy tensions, and missing-evidence boundaries.
- Write each Section purpose as a light prose writing brief that gives the writer reader movement and the most important clusters to keep visible.
- Use coverage_notes only as a memory aid for important clusters inspected and where they are handled. Do not build a separate rigid cluster map.
- Do not make the report shorter because the cluster memory is concise. The memory exists to prevent accidental omissions, not to justify compression.`
}

func longFormPlanReviewPlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfilePlanReview(profile) {
		return ""
	}
	return `Plan-review planning guidance:
- Before submitting the plan, perform one internal thin-plan review.
- Ask whether the outline became too narrow, whether any major source-backed cluster disappeared, whether the Part/Section count is artificially low, and whether caveats are isolated instead of attached to the sections that need them.
- If the review finds a thin plan, revise the plan before the first successful tool submission. Do not submit a weak plan and rely on drafting to repair it.
- Keep the submitted JSON schema unchanged. Use coverage_notes to briefly state what the review preserved or why the source packet is genuinely small.
- This first experiment implements review as pre-submit self-review only; it does not add a separate post-submit review stage.`
}
