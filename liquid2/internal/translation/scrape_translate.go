package translation

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/ingest"
)

type Scraper interface {
	Scrape(context.Context, ingest.ScrapeInput) (app.DocumentDetail, error)
}

type DocumentTranslator interface {
	TranslateDocument(context.Context, EnqueueDocumentInput) (app.Job, error)
}

type ScrapeTranslateInput struct {
	URL            string
	TargetLanguage string
	FolderID       string
	TagIDs         []string
}

type ScrapeTranslateResult struct {
	Document app.DocumentDetail
	Job      app.Job
}

func ScrapeTranslate(
	ctx context.Context,
	scraper Scraper,
	translator DocumentTranslator,
	input ScrapeTranslateInput,
) (ScrapeTranslateResult, error) {
	if translator == nil {
		return ScrapeTranslateResult{}, ErrTranslationUnavailable
	}
	targetLanguage, err := app.NormalizeContentLanguage(input.TargetLanguage)
	if err != nil {
		return ScrapeTranslateResult{}, err
	}
	detail, err := scraper.Scrape(ctx, ingest.ScrapeInput{
		URL: input.URL, FolderID: input.FolderID, TagIDs: input.TagIDs,
	})
	if err != nil {
		return ScrapeTranslateResult{}, err
	}
	sourceContentID, err := firstContentID(detail)
	if err != nil {
		return ScrapeTranslateResult{}, err
	}
	job, err := translator.TranslateDocument(ctx, EnqueueDocumentInput{
		DocumentID: detail.Document.ID, SourceContentID: sourceContentID,
		TargetLanguage: targetLanguage,
	})
	if err != nil {
		return ScrapeTranslateResult{}, err
	}
	return ScrapeTranslateResult{Document: detail, Job: job}, nil
}

func firstContentID(detail app.DocumentDetail) (string, error) {
	for _, content := range detail.Contents {
		if content.ID != "" {
			return content.ID, nil
		}
	}
	return "", fmt.Errorf("%w: scraped document content is missing", app.ErrValidation)
}
