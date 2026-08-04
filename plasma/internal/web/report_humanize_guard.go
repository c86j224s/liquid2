package web

import "github.com/c86j224s/liquid2/plasma/internal/reporthumanize"

func validateHumanizedMarkdown(original string, humanized string) error {
	return reporthumanize.ValidateMarkdown(original, humanized)
}
