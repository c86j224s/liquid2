package web

import "strings"

const reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice = "part-connective-subject-direct-synthesis-voice"

func isReportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice:
		return true
	default:
		return false
	}
}

func reportSubjectDirectSynthesisPlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice(profile) {
		return ""
	}
	return `Subject-direct synthesis planning guidance:
- Keep the submitted JSON schema unchanged. Put this direction inside writing_contract, Part purposes, Section purposes, coverage_notes, and planned_omissions only.
- Shape writing_contract.tone_and_shape around direct explanation of the subject, not narration of how source material talks about it.
- Shape writing_contract.reading_path so the first move is the first real subject claim, mechanism, tension, example, number, or boundary the reader needs, not a report-purpose or source-tour sentence.
- Write each Part and Section purpose toward the first subject move it must make: the claim about the subject, the mechanism that explains it, the example or number that anchors it, and the caveat, comparison, disagreement, or authority boundary that materially limits it.
- Preserve citations, mechanisms, examples, numbers, caveats, comparisons, unresolved tensions, source-backed uncertainty, and necessary source identity in the plan's existing fields. Direct subject voice is not permission to compress evidence or erase provenance.`
}

func reportSubjectDirectSynthesisSectionGuidance(profile string) string {
	if !isReportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice(profile) {
		return ""
	}
	return `Subject-direct synthesis Section guidance:
- State the source-backed claim about the subject as the sentence's actual subject and predicate. Do not make "the source", "the document", "the report", "the analysis", or "the material" the grammatical subject unless the source's identity materially changes the claim.
- Keep source identity in surface prose when version differences, inter-source disagreement, source-flagged uncertainty or instability, measurement method, or authority scope changes what the reader should believe. Otherwise keep provenance in citations and evidence references rather than turning the sentence into source narration.
- Rewrite source-relative statements into subject-relative explanation while preserving citations, mechanisms, examples, numbers, caveats, comparisons, and unresolved tensions.
- Do not flatten disagreement, authority boundaries, measurement conditions, missing evidence, or weak signals into a single subject claim. Put the boundary beside the claim it limits.
- Do not mention subject-direct synthesis, hidden guidance, or experiment labels in the report.`
}
