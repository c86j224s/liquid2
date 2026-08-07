package reportprompt

import "strings"

const reportGenerationGuidanceProfilePartConnectiveEconomyVoice = "part-connective-economy-voice"

func isReportGenerationGuidanceProfilePartConnectiveEconomyVoice(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfilePartConnectiveEconomyVoice,
		"part_connective_economy_voice", "part-connective-economy", "part_connective_economy":
		return true
	default:
		return false
	}
}

func reportPartConnectiveEconomyGuidance(profile string) string {
	if !isReportGenerationGuidanceProfilePartConnectiveEconomyVoice(profile) &&
		!isReportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice(profile) &&
		!isReportGenerationGuidanceProfileLongFormDefault(profile) {
		return ""
	}
	return `Part connective-economy guidance:
- Add connective text only when it changes the reader's understanding of how the immutable Sections relate. The Part title and Section headings already carry structure.
- Leave the intro empty when the first Section begins the subject clearly. Otherwise write at most one short paragraph of no more than two sentences, without previewing the Section inventory or repeating the first Section's opening.
- Default to no transitions. Add one sentence only when the relationship between adjacent Sections would otherwise be unclear; do not recap the previous Section or preview the next one.
- Leave the closing empty unless the Sections together support a new Part-level synthesis that none of them states. When needed, use at most one short paragraph of no more than two sentences and do not preview the next Part.
- Do not restate definitions, evidence, examples, caveats, or conclusions already present in the immutable Sections. This guidance reduces assembly overhead, not source-backed explanation.
- Do not mention Part connective economy, hidden guidance, or experiment labels in the report.`
}

func reportPartAdjacentBoundaryEditGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileLongFormDefault(profile) {
		return ""
	}
	return `Default long-form adjacent-boundary audit:
- At each adjacent Section boundary, read the previous Section's final substantive paragraph, the connective text between them, and the next Section's first substantive paragraph as one passage.
- Apply this audit only when a following Section exists. Do not remove or rewrite the final substantive paragraph of the last Section in a Part under this adjacent-boundary rule.
- If a Section's final paragraph only restates a mechanism, conclusion, distinction, previously established caveat, or reading instruction from that Section, delete the paragraph instead of compressing or rephrasing it.
- Keep the paragraph when it adds a new concrete fact, example, citation, consequence, caveat, or unresolved question.
- After removing a recap, leave the boundary empty unless the next Section answers a specific unresolved question, tension, or dependency already established by the current Section. When that bridge is useful, use at most one sentence naming the open point instead of merely announcing the next topic.
- Preserve Section-internal rhythm and style, and never remove facts, citations, or necessary explanation to make the boundary shorter. The smallest edit remains valid, and no-op remains valid when the boundary already reads cleanly.`
}
