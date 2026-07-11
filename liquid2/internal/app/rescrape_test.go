package app

import (
	"context"
	"errors"
	"testing"
)

func TestReplaceRescrapedContentPreservesExtractedContentID(t *testing.T) {
	ctx := context.Background()
	service := NewService()
	source, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/final", SourceURL: "https://example.com/start",
		Title: "Article", Content: "Old body", Format: ContentFormatText,
	})
	if err != nil {
		t.Fatalf("create scraped document: %v", err)
	}
	sourceContentID := source.Contents[0].ID
	if _, err := service.AppendTranslatedContent(ctx, source.Document.ID, AppendTranslationInput{
		SourceContentID: sourceContentID, TargetLanguage: "ko", Content: "Translated",
	}); err != nil {
		t.Fatalf("append translation: %v", err)
	}

	detail, err := service.ReplaceRescrapedContent(ctx, source.Document.ID, RescrapedContentInput{
		URL: "https://example.com/final2", SourceURL: "https://example.com/start",
		Content: "# New body", Format: ContentFormatMarkdown,
	})
	if err != nil {
		t.Fatalf("replace rescraped content: %v", err)
	}

	if detail.Contents[0].ID != sourceContentID {
		t.Fatalf("expected extracted content ID to be preserved, got %#v", detail.Contents[0])
	}
	if detail.Contents[0].Content != "# New body" || detail.Contents[0].Format != ContentFormatMarkdown {
		t.Fatalf("expected refreshed markdown content, got %#v", detail.Contents[0])
	}
	if len(detail.Contents) != 2 || detail.Contents[1].SourceContentID == nil ||
		*detail.Contents[1].SourceContentID != sourceContentID {
		t.Fatalf("expected translation source link to remain, got %#v", detail.Contents)
	}
	if detail.Document.CanonicalURL == nil || *detail.Document.CanonicalURL != "https://example.com/final2" {
		t.Fatalf("expected canonical URL refresh, got %#v", detail.Document.CanonicalURL)
	}

	versions, err := service.ListDocumentVersions(ctx, source.Document.ID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 2 || versions[1].Contents[0].Content != "Old body" {
		t.Fatalf("expected old content snapshot, got %#v", versions)
	}
}

func TestPrepareRescrapeRejectsNonScrapedDocuments(t *testing.T) {
	ctx := context.Background()
	service := NewService()
	detail, err := service.CreateBookmarkDocument(ctx, BookmarkDocumentInput{
		URL: "https://example.com/a", Title: "Bookmark",
	})
	if err != nil {
		t.Fatalf("create bookmark: %v", err)
	}

	_, err = service.PrepareRescrape(ctx, detail.Document.ID)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
