package web

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

func longFormArticleContract(intent reportexecution.ArticleIntent) string {
	return fmt.Sprintf(`Audience: %s
Reader promise: %s
Discovery, perspective, or method to emphasize: %s
Target length: booklet-length; use the source-supported 6-14 chapter range already enforced by the long-form product contract.`,
		strings.TrimSpace(intent.Audience),
		strings.TrimSpace(intent.ReaderPromise),
		firstNonEmpty(strings.TrimSpace(intent.Emphasis), "Use the strongest source-supported discovery."),
	)
}
