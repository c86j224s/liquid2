package reportilphase0

const (
	AuthoringModeStandard = "standard"
	AuthoringModeLongForm = "long_form"
)

func normalizeAuthoringMode(value string) string {
	if value == AuthoringModeLongForm {
		return AuthoringModeLongForm
	}
	return AuthoringModeStandard
}

func authorSectionLimit(mode string) int {
	if normalizeAuthoringMode(mode) == AuthoringModeLongForm {
		return 16
	}
	return 8
}

func authoringModeLabel(mode string) string {
	if normalizeAuthoringMode(mode) == AuthoringModeLongForm {
		return "long-form"
	}
	return "standard-length"
}

func authoringLengthGuidance(mode string) string {
	if normalizeAuthoringMode(mode) == AuthoringModeLongForm {
		return `- Write a genuinely long-form report: develop the answer across substantial connected passages, preserve materially useful detail, and give important chronology, mechanisms, relationships, cases, and tensions enough space to become understandable. Do not pad, repeat, or turn length into a source inventory.
- Use four to sixteen reader-facing sections when the material supports them. Prefer fewer substantial sections over many thin headings. The final section must contain the real conclusion.`
	}
	return `- Use two to eight reader-facing sections. Prefer substantial connected passages over thin headings. The final section must contain the real conclusion.`
}

func editorialMemoryDepthGuidance(mode string) string {
	if normalizeAuthoringMode(mode) == AuthoringModeLongForm {
		return "The downstream manuscript is long-form. Preserve supporting accounts that materially deepen chronology, mechanism, relationship, comparison, or context rather than retaining only the shortest central answer."
	}
	return "The downstream manuscript is standard-length. Preserve every account that materially improves the complete answer without turning memory into an inventory."
}
