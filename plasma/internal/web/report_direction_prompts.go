package web

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func withReportDirection(prompt, hint string) string {
	block := reporting.FormatDirectionHint(hint)
	if block == "" {
		return prompt
	}
	return strings.TrimSpace(prompt) + "\n\n" + block
}
