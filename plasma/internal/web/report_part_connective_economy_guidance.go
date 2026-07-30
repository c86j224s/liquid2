package web

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
		!isReportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice(profile) {
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
