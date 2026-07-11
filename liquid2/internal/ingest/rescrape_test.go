package ingest

import (
	"context"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
)

func TestServiceRescrapeRefreshesExistingDocument(t *testing.T) {
	ctx := context.Background()
	documents := app.NewService()
	source, err := documents.CreateScrapedDocument(ctx, app.ScrapedDocumentInput{
		URL: "https://example.com/final", SourceURL: "https://example.com/start",
		Title: "Article", Content: "Old body", Format: app.ContentFormatText,
	})
	if err != nil {
		t.Fatalf("create scraped document: %v", err)
	}
	fetcher := &rescrapeFetcher{
		page: FetchedPage{
			URL: "https://example.com/final2", Title: "Ignored title",
			Content: "# New body", Format: app.ContentFormatMarkdown,
		},
	}
	service := NewService(documents, WithFetcher(fetcher))

	detail, err := service.Rescrape(ctx, source.Document.ID)
	if err != nil {
		t.Fatalf("re-scrape: %v", err)
	}

	if fetcher.rawURL != "https://example.com/start" {
		t.Fatalf("expected original source URL fetch, got %q", fetcher.rawURL)
	}
	if detail.Document.Title != "Article" {
		t.Fatalf("expected existing title to be preserved, got %q", detail.Document.Title)
	}
	if detail.Contents[0].Content != "# New body" || detail.Contents[0].Format != app.ContentFormatMarkdown {
		t.Fatalf("expected refreshed content, got %#v", detail.Contents[0])
	}
}

type rescrapeFetcher struct {
	rawURL string
	page   FetchedPage
}

func (fetcher *rescrapeFetcher) Fetch(_ context.Context, rawURL string) (FetchedPage, error) {
	fetcher.rawURL = rawURL
	return fetcher.page, nil
}
