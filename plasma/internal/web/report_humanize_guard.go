package web

import "github.com/c86j224s/liquid2/plasma/internal/reporting"

func validateHumanizedMarkdown(original string, humanized string) error {
	return reporting.ValidateHumanizedMarkdown(original, humanized)
}
