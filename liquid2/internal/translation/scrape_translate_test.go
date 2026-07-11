package translation

import (
	"context"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/ingest"
)

type fakeScraper struct {
	detail app.DocumentDetail
	err    error
	calls  int
}

func (scraper *fakeScraper) Scrape(_ context.Context, _ ingest.ScrapeInput) (app.DocumentDetail, error) {
	scraper.calls++
	if scraper.err != nil {
		return app.DocumentDetail{}, scraper.err
	}
	return scraper.detail, nil
}

type fakeTranslator struct {
	job   app.Job
	err   error
	input EnqueueDocumentInput
	calls int
}

func (translator *fakeTranslator) TranslateDocument(_ context.Context, input EnqueueDocumentInput) (app.Job, error) {
	translator.calls++
	translator.input = input
	if translator.err != nil {
		return app.Job{}, translator.err
	}
	return translator.job, nil
}

func TestScrapeTranslateScrapesThenEnqueuesTranslation(t *testing.T) {
	scraper := &fakeScraper{detail: scrapedDetail()}
	translator := &fakeTranslator{job: app.Job{
		ID: "job_1", Kind: app.JobKindTranslateDocument, Status: app.JobStatusQueued,
	}}

	result, err := ScrapeTranslate(context.Background(), scraper, translator, ScrapeTranslateInput{
		URL: "https://example.com/a", TargetLanguage: "KO",
	})
	if err != nil {
		t.Fatalf("scrape translate: %v", err)
	}
	if result.Document.Document.ID != "doc_1" || result.Job.ID != "job_1" {
		t.Fatalf("unexpected result %#v", result)
	}
	if translator.input.DocumentID != "doc_1" || translator.input.SourceContentID != "content_1" {
		t.Fatalf("unexpected translator input %#v", translator.input)
	}
	if translator.input.TargetLanguage != "ko" {
		t.Fatalf("expected normalized language, got %q", translator.input.TargetLanguage)
	}
}

func TestScrapeTranslateRequiresTranslatorBeforeScrape(t *testing.T) {
	scraper := &fakeScraper{detail: scrapedDetail()}
	_, err := ScrapeTranslate(context.Background(), scraper, nil, ScrapeTranslateInput{
		URL: "https://example.com/a", TargetLanguage: "ko",
	})
	if !errors.Is(err, ErrTranslationUnavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
	if scraper.calls != 0 {
		t.Fatalf("translator unavailable should not scrape, got %d calls", scraper.calls)
	}
}

func TestScrapeTranslateValidatesLanguageBeforeScrape(t *testing.T) {
	scraper := &fakeScraper{detail: scrapedDetail()}
	translator := &fakeTranslator{}
	_, err := ScrapeTranslate(context.Background(), scraper, translator, ScrapeTranslateInput{
		URL: "https://example.com/a", TargetLanguage: "??",
	})
	if !errors.Is(err, app.ErrValidation) {
		t.Fatalf("expected validation, got %v", err)
	}
	if scraper.calls != 0 || translator.calls != 0 {
		t.Fatalf("invalid language should not scrape or enqueue")
	}
}

func scrapedDetail() app.DocumentDetail {
	return app.DocumentDetail{
		Document: app.DocumentMetadata{ID: "doc_1", Title: "Article"},
		Contents: []app.DocumentContent{{
			ID: "content_1", Role: app.ContentRoleExtracted,
			Format: app.ContentFormatText, Content: "Readable body",
		}},
	}
}
