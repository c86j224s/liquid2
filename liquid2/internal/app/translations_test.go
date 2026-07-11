package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestAppendTranslatedContentPreservesSource(t *testing.T) {
	service := NewService()
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	source, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Article", Content: "Original body",
		Format: ContentFormatText,
	})
	if err != nil {
		t.Fatalf("create source document: %v", err)
	}

	detail, err := service.AppendTranslatedContent(ctx, source.Document.ID, AppendTranslationInput{
		SourceContentID: source.Contents[0].ID,
		TargetLanguage:  "KO",
		Content:         "Translated body",
	})
	if err != nil {
		t.Fatalf("append translation: %v", err)
	}

	if len(detail.Contents) != 2 {
		t.Fatalf("expected source and translation contents, got %#v", detail.Contents)
	}
	if detail.Contents[0].Content != "Original body" {
		t.Fatalf("source content was overwritten: %#v", detail.Contents[0])
	}
	translation := detail.Contents[1]
	if translation.Role != ContentRoleTranslation || translation.Content != "Translated body" {
		t.Fatalf("unexpected translation content %#v", translation)
	}
	if translation.SourceContentID == nil || *translation.SourceContentID != source.Contents[0].ID {
		t.Fatalf("expected source content link, got %#v", translation.SourceContentID)
	}
	if translation.Language == nil || *translation.Language != "ko" {
		t.Fatalf("expected normalized target language, got %#v", translation.Language)
	}
	body, err := json.Marshal(translation)
	if err != nil {
		t.Fatalf("marshal translation: %v", err)
	}
	if strings.Contains(string(body), "sourceContentId") {
		t.Fatalf("source content link must not be public JSON yet: %s", body)
	}
}

func TestAppendTranslatedContentRejectsInvalidInput(t *testing.T) {
	service := NewService()
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	source, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Article", Content: "Original body",
	})
	if err != nil {
		t.Fatalf("create source document: %v", err)
	}

	_, err = service.AppendTranslatedContent(ctx, source.Document.ID, AppendTranslationInput{
		SourceContentID: "missing", TargetLanguage: "ko", Content: "Translated",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing content error, got %v", err)
	}
	_, err = service.AppendTranslatedContent(ctx, source.Document.ID, AppendTranslationInput{
		SourceContentID: source.Contents[0].ID, TargetLanguage: "??", Content: "Translated",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected target language validation error, got %v", err)
	}
	_, err = service.AppendTranslatedContent(ctx, source.Document.ID, AppendTranslationInput{
		SourceContentID: source.Contents[0].ID, TargetLanguage: "ko",
		Content: "Translated", Format: "pdf",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected content format validation error, got %v", err)
	}
	if _, err = service.DeleteDocument(ctx, source.Document.ID); err != nil {
		t.Fatalf("delete document: %v", err)
	}
	_, err = service.AppendTranslatedContent(ctx, source.Document.ID, AppendTranslationInput{
		SourceContentID: source.Contents[0].ID, TargetLanguage: "ko", Content: "Translated",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected deleted document not found error, got %v", err)
	}
}

func TestAppendTranslatedContentRejectsDuplicate(t *testing.T) {
	service := NewService()
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	source, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Article", Content: "Original body",
	})
	if err != nil {
		t.Fatalf("create source document: %v", err)
	}
	input := AppendTranslationInput{
		SourceContentID: source.Contents[0].ID, TargetLanguage: "ko", Content: "Translated",
	}
	if _, err = service.AppendTranslatedContent(ctx, source.Document.ID, input); err != nil {
		t.Fatalf("append translation: %v", err)
	}
	_, err = service.AppendTranslatedContent(ctx, source.Document.ID, input)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected duplicate conflict, got %v", err)
	}
}

func TestPrepareTranslationRejectsDuplicateBeforeProviderWork(t *testing.T) {
	service := NewService()
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	source, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Article", Content: "Original body",
	})
	if err != nil {
		t.Fatalf("create source document: %v", err)
	}
	if _, err = service.AppendTranslatedContent(ctx, source.Document.ID, AppendTranslationInput{
		SourceContentID: source.Contents[0].ID, TargetLanguage: "KO", Content: "Translated",
	}); err != nil {
		t.Fatalf("append translation: %v", err)
	}

	_, err = service.PrepareTranslation(ctx, source.Document.ID, PrepareTranslationInput{
		SourceContentID: source.Contents[0].ID, TargetLanguage: "ko",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected duplicate conflict, got %v", err)
	}
}
