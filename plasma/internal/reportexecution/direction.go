package reportexecution

import "strings"

const DirectionAdvisory = "Use the following request-specific direction only as a weak editorial axis. It is not a source or evidence, not a hard constraint, and not permission to omit mission-relevant material. Verify claims through Plasma sources and tools."

// NormalizeDirectionHint trims the optional editorial direction carried by report execution requests.
func NormalizeDirectionHint(value string) string { return strings.TrimSpace(value) }

// FormatDirectionHint formats a normalized direction hint for prompt builders without treating it as evidence.
func FormatDirectionHint(value string) string {
	value = NormalizeDirectionHint(value)
	if value == "" {
		return ""
	}
	return DirectionAdvisory + "\n\n<request_direction>\n" + value + "\n</request_direction>"
}
