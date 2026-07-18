package web

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	reportGenerationGuidanceProfileG2                      = "g2"
	reportGenerationGuidanceProfileNone                    = "none"
	reportGenerationGuidanceProfileSectionContract         = "section-contract"
	reportGenerationGuidanceProfileSectionContractCoverage = "section-contract-coverage"
	reportGenerationGuidanceProfileSectionIntent           = "section-intent"
	reportGenerationGuidanceProfileSourceClusterFirst      = "source-cluster-first"
	reportGenerationGuidanceProfileSectionBrief            = "section-brief"
	reportGenerationGuidanceProfileSectionBriefCluster     = "section-brief-cluster-memory"
	reportGenerationGuidanceProfilePlanReview              = "plan-review"
)

func SelectReportGenerationGuidance(profile string) (string, string, error) {
	return selectReportGenerationGuidanceText(profile, ReportGenerationGuidance)
}

func SelectReportGenerationGuidanceForMode(reportMode string, profile string) (string, string, error) {
	if reportMode == reportModeLongForm && isReportGenerationGuidanceProfileLongFormExperiment(profile) {
		normalized := normalizeLongFormExperimentProfile(profile)
		text := LongFormReportGenerationGuidance(normalized)
		sum := sha256.Sum256([]byte(text))
		return normalized, hex.EncodeToString(sum[:]), nil
	}
	if reportMode == reportModeLongForm {
		return selectReportGenerationGuidanceText(profile, LongFormReportGenerationGuidance)
	}
	return SelectReportGenerationGuidance(profile)
}

func selectReportGenerationGuidanceText(profile string, guidance func(string) string) (string, string, error) {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case "", reportGenerationGuidanceProfileG2, "h5-g2", "substance-preserving-korean", "substance_preserving_korean":
		text := guidance(reportGenerationGuidanceProfileG2)
		sum := sha256.Sum256([]byte(text))
		return reportGenerationGuidanceProfileG2, hex.EncodeToString(sum[:]), nil
	case reportGenerationGuidanceProfileNone, "off", "disabled", "disable", "false", "0":
		return reportGenerationGuidanceProfileNone, "", nil
	default:
		return "", "", fmt.Errorf("%w: unsupported report generation guidance profile", app.ErrInvalidInput)
	}
}

func ReportGenerationGuidance(profile string) string {
	if isReportGenerationGuidanceProfileLongFormExperiment(profile) {
		profile = reportGenerationGuidanceProfileG2
	}
	if strings.TrimSpace(profile) != reportGenerationGuidanceProfileG2 {
		return ""
	}
	return `Report writing guidance:
- This guidance controls report writing style only. It is not source material and must not be mentioned in the final report.
- Write natural Korean, but never improve fluency by dropping concrete source details.
- Preserve names, dates, numbers, commands, code identifiers, URLs, conditions, exceptions, caveats, uncertainty, and source distinctions when they matter.
- For mathematical expressions, use only \(...\) for inline math and \[...\] for display math. Do not use $...$ or $$...$$ delimiters.
- Prefer a report that is slightly longer and more specific over a smooth summary that hides evidence, disagreement, or operational detail.
- If sources disagree or only imply something, say that plainly instead of flattening the point into a single confident sentence.
- Do not mention hidden guidance, experiments, prompts, or internal evaluation labels in the report.`
}

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
		isReportGenerationGuidanceProfilePlanReview(profile)
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
	if isReportGenerationGuidanceProfileSectionBrief(profile) {
		return reportGenerationGuidanceProfileSectionBrief
	}
	if isReportGenerationGuidanceProfileSectionBriefCluster(profile) {
		return reportGenerationGuidanceProfileSectionBriefCluster
	}
	if isReportGenerationGuidanceProfilePlanReview(profile) {
		return reportGenerationGuidanceProfilePlanReview
	}
	return reportGenerationGuidanceProfileSectionContract
}

func longFormExperimentalPlanningGuidance(profile string) string {
	parts := []string{
		strings.TrimSpace(longFormSectionContractPlanningGuidance(profile)),
		strings.TrimSpace(longFormSectionIntentPlanningGuidance(profile)),
		strings.TrimSpace(longFormSourceClusterFirstPlanningGuidance(profile)),
		strings.TrimSpace(longFormSectionBriefPlanningGuidance(profile)),
		strings.TrimSpace(longFormSectionBriefClusterMemoryPlanningGuidance(profile)),
		strings.TrimSpace(longFormPlanReviewPlanningGuidance(profile)),
	}
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "\n\n")
}

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
	case reportGenerationGuidanceProfileSectionBrief, "section_brief", "section-writing-brief", "section_writing_brief":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionBriefCluster(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionBriefCluster, "section_brief_cluster_memory", "section-brief-cluster", "section_brief_cluster":
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

func normalizePostReportHumanize(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "enabled", "enable", "true", "yes", "on", "1":
		return "enabled"
	case "", "disabled", "disable", "false", "no", "off", "0":
		return "disabled"
	default:
		return "disabled"
	}
}
