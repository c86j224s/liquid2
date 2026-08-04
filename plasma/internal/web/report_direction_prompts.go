package web

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

func withReportDirection(prompt, hint string) string {
	block := reportexecution.FormatDirectionHint(hint)
	if block == "" {
		return prompt
	}
	return strings.TrimSpace(prompt) + "\n\n" + block
}
