package reportprompt

import (
	"strings"
)

func LongFormReportGenerationGuidance(profile string) string {
	base := strings.TrimSpace(ReportGenerationGuidance(profile))
	if base == "" {
		return ""
	}
	guidance := base + `

Long-form human-writer guidance:
- Write each section as a person explaining the material to another person, not as a system reporting that it inspected a session.
- Prefer clear, concrete topic sentences and natural paragraph-to-paragraph flow over formulaic phrases such as "this report confirms" or "based on the provided material".
- Keep caveats, limits, and source boundaries, but weave them into the argument instead of repeating the same disclaimer frame.
- Vary sentence length, split overloaded sentences, and let the report sound like edited prose while preserving all source-backed substance.`
	if isReportGenerationGuidanceProfileSectionContractFamily(profile) {
		guidance += `

Long-form section-contract guidance:
- During planning, write each Section purpose as a compact writing contract, not a vague topic label.
- The contract should state the section's central point, reader takeaway, evidence path, and boundary: what this section should not expand into.
- Keep long-form richness. A sharper contract must not collapse important source clusters, reduce necessary Part/Section coverage, or make the report short by omission.
- During section drafting, use that contract to keep the section centered. Do not turn the section into a source inventory or a generic background survey.
- Preserve source-backed caveats, but attach them to the section's argument instead of repeating a detached disclaimer frame.`
	}
	if isReportGenerationGuidanceProfileSectionIntent(profile) {
		guidance += `

Long-form section-intent guidance:
- Treat each Section as a reader movement, not as a checklist: by the end, the reader should notice one concrete shift, tension, implication, or distinction.
- Use the Section purpose as quiet editorial intent. It should help the writer feel why the section exists, without forcing a rigid structure or reducing source coverage.
- Let source-backed clusters keep their natural size. Do not make the report shorter merely because the intent is clearer.
- During drafting, write toward the intended reader understanding while preserving concrete source details, caveats, and unresolved questions.`
	}
	if isReportGenerationGuidanceProfileSectionContractCoverage(profile) {
		guidance += `

Long-form section-contract coverage guidance:
- During planning, keep baseline long-form coverage density unless the mission material is genuinely small. For ordinary source packets, target 3-5 Parts and 9-14 Sections; fewer than 9 Sections requires an explicit source-size reason in coverage_notes.
- Treat coverage preservation as stronger than outline neatness. Do not reduce Parts or Sections merely because the section purposes are sharper.
- Map each major source-backed cluster to a Part, Section, coverage note, or planned omission. A cleaner outline is not a reason to drop a cluster.
- Section purposes should still be compact contracts, but they must organize the same richness rather than replacing it with a narrower report.
- During section drafting, use the contract to choose the section's spine while preserving concrete details, examples, tensions, and caveats that belong to that cluster.`
	}
	if isReportGenerationGuidanceProfileSourceClusterFirst(profile) {
		guidance += `

Long-form source-cluster-first guidance:
- Treat the plan's source-cluster map as the report's coverage memory. Do not write a smoother section by dropping a mapped cluster that belongs to it.
- During drafting, turn each relevant cluster into source-backed explanation, not a checklist or inventory.
- Preserve concrete examples, mechanisms, numbers, caveats, and unresolved questions that make the cluster worth covering.`
	}
	if isReportGenerationGuidanceProfileSectionBrief(profile) {
		guidance += `

Long-form section-brief guidance:
- Treat each Section purpose as a light writing brief. It should orient the section without forcing a template.
- Use the brief to preserve the section's intended reader movement, concrete details, tension, and adjacent-topic boundary.
- Do not satisfy the brief by merely naming those elements. Write natural prose that uses them to explain the source-backed material.`
	}
	if isReportGenerationGuidanceProfileSectionBriefCluster(profile) {
		guidance += `

Long-form section-brief cluster-memory guidance:
- Treat the Section brief as both writing direction and memory for important source-backed clusters.
- Keep clusters visible through concrete explanation, not by listing them. A cluster can be a mechanism, example, number, caveat, comparison, policy tension, or missing-evidence boundary.
- Do not infer that a small cluster map means the report should become short. Use it to avoid accidental omissions while preserving natural flow.`
	}
	if isReportGenerationGuidanceProfilePlanReview(profile) {
		guidance += `

Long-form plan-review guidance:
- Draft against the reviewed plan rather than narrowing it during writing.
- If the plan preserved a source cluster or caveat, keep it visible unless the source read proves it irrelevant.
- Do not compensate for a cleaner outline by shortening the actual report.`
	}
	return guidance
}
