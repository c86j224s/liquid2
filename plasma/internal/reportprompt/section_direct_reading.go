package reportprompt

import "strings"

const reportGenerationGuidanceProfileSectionDirectReadingVoice = "section-direct-reading-voice"

func isReportGenerationGuidanceProfileSectionDirectReadingVoice(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionDirectReadingVoice,
		"section_direct_reading_voice", "section-direct", "section_direct",
		reportGenerationGuidanceProfilePartConnectiveEconomyVoice,
		"part_connective_economy_voice", "part-connective-economy", "part_connective_economy",
		reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice:
		return true
	default:
		return false
	}
}

func reportSectionDirectWritingGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileSectionDirectReadingVoice(profile) &&
		!isReportGenerationGuidanceProfileLongFormDefault(profile) {
		return ""
	}
	guidance := `Section direct-writing guidance:
- Write the Section as the subject itself, not as an explanation of where the reader is in the report.
- Let the heading carry the Section's structural role. Begin with the actual claim, mechanism, scene, contrast, consequence, or evidence-backed question instead of announcing what this Section or Part will cover.
- Connect ideas through their substance. References to document position such as "이 부", "이 절", "다음 절", "다음 질문", "앞에서", or "뒤에서" are useful only when the content relationship would otherwise be unclear.
- Do not preview the next Section or recap the previous Section merely to make the outline visible. End when this Section's own reasoning is complete.
- Keep claim-specific source boundaries, caveats, examples, numbers, and mechanisms. This guidance removes outline narration; it does not ask for a shorter or less detailed Section.
- Do not mention section direct writing, hidden guidance, or experiment labels in the report.`
	if isReportGenerationGuidanceProfileLongFormDefault(profile) {
		guidance += `

Default long-form Section writing guidance:
- Every sentence should advance a claim, fact, mechanism, distinction, consequence, or limit.
- Omit abstract restatement that only renames the Section purpose, report structure, or previous sentence.
- Prefer concrete subjects and verbs when the sources provide them.
- Use contrast only when the sources support a real distinction; do not add ornamental "not merely A but B" framing.
- Stop paragraphs when their reasoning is complete, without adding a redundant takeaway sentence.
- Do not regularize paragraph length or closing cadence, and do not treat this guidance as a connective-word blacklist.`
	}
	return guidance
}
