package reporthumanize

import "github.com/c86j224s/liquid2/plasma/internal/reporting"

// ValidateMarkdown enforces the same structural fidelity guard used by H5 before
// it accepts a finalized patch artifact.
func ValidateMarkdown(original string, humanized string) error {
	return reporting.ValidateHumanizedMarkdown(original, humanized)
}
