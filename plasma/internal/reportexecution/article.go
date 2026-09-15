package reportexecution

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const OutputKindArticle = "article"

// ArticleIntent is the reader-facing input carried by the existing report lifecycle.
type ArticleIntent struct {
	Audience      string `json:"audience"`
	ReaderPromise string `json:"reader_promise"`
	Emphasis      string `json:"emphasis,omitempty"`
}

func normalizeArticleIntent(intent ArticleIntent) ArticleIntent {
	intent.Audience = strings.TrimSpace(intent.Audience)
	intent.ReaderPromise = strings.TrimSpace(intent.ReaderPromise)
	intent.Emphasis = strings.TrimSpace(intent.Emphasis)
	return intent
}

func validateArticleRequest(req DraftRequest) error {
	outputKind, intent := req.OutputKind, req.ArticleIntent
	if outputKind == "" {
		if intent != (ArticleIntent{}) {
			return fmt.Errorf("%w: article intent requires article output", producterror.ErrInvalidInput)
		}
		return nil
	}
	if outputKind != OutputKindArticle {
		return fmt.Errorf("%w: unsupported output kind", producterror.ErrInvalidInput)
	}
	shortArticle := req.PipelineFamily == "" && req.ReportMode == ModeOneTake
	longArticle := req.PipelineFamily == "report_il_experimental" && req.ReportMode == ModeLongForm
	if !shortArticle && !longArticle {
		return fmt.Errorf("%w: article output requires the existing one-take or long-form IL path", producterror.ErrInvalidInput)
	}
	if longArticle && req.ExecutionStrategy != "serial" && req.ExecutionStrategy != "section_fanout" {
		return fmt.Errorf("%w: long-form article execution strategy is invalid", producterror.ErrInvalidInput)
	}
	for _, value := range []string{intent.Audience, intent.ReaderPromise} {
		if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > 500 {
			return fmt.Errorf("%w: article audience and reader promise are required", producterror.ErrInvalidInput)
		}
	}
	if !utf8.ValidString(intent.Emphasis) || utf8.RuneCountInString(intent.Emphasis) > 500 {
		return fmt.Errorf("%w: article emphasis is invalid", producterror.ErrInvalidInput)
	}
	return nil
}
