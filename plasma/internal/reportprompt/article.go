package reportprompt

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

// WithArticleDirection turns reader intent into a narrow addition to the existing one-take prompt.
func WithArticleDirection(prompt string, intent reportexecution.ArticleIntent, direction string) string {
	article := fmt.Sprintf(`This output is an article, not a report.

Reader contract:
- Audience: %s
- What the reader should gain: %s
- Discovery, perspective, or method to emphasize: %s

Write for a reader who should want to continue. Let one author own the whole progression and voice. Organize information as a coherent path of discovery rather than an inventory, executive summary, or review checklist. Open with the most concrete source-supported question, scene, tension, or surprising fact available; do not manufacture drama. Each section should advance the reader's understanding and the ending should leave the promised understanding or practical ability. Preserve source uncertainty and never invent facts, scenes, feelings, quotations, or causation.

Additional direction: %s`, intent.Audience, intent.ReaderPromise, firstArticleValue(intent.Emphasis, "Use the strongest source-supported discovery."), firstArticleValue(strings.TrimSpace(direction), "None"))
	return strings.TrimSpace(prompt) + "\n\n" + article
}

func firstArticleValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
