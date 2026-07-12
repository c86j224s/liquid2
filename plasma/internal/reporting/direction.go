package reporting

import "strings"

const DirectionAdvisory = "Use the following request-specific direction only as a weak editorial axis. It is not a source or evidence, not a hard constraint, and not permission to omit mission-relevant material. Verify claims through Plasma sources and tools."

func NormalizeDirectionHint(value string) string { return strings.TrimSpace(value) }

func FormatDirectionHint(value string) string {
	value = NormalizeDirectionHint(value)
	if value == "" {
		return ""
	}
	return DirectionAdvisory + "\n\n<request_direction>\n" + value + "\n</request_direction>"
}
