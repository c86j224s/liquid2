package app

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
)

func TestSQLiteRepositoryPreservesContentCreatedAt(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close sqlite store: %v", err)
		}
	}()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate sqlite store: %v", err)
	}
	now := int64(1760000000000)
	repo := NewSQLiteRepository(store, WithSQLiteRepositoryClock(func() int64 { return now }))
	service := NewService(WithRepository(repo))
	defer func() {
		if err := service.Close(); err != nil {
			t.Fatalf("close service: %v", err)
		}
	}()

	detail, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Article", Content: "Body",
		Format: ContentFormatText,
	})
	if err != nil {
		t.Fatalf("create scraped document: %v", err)
	}
	rows, err := store.Queries().ListDocumentContents(ctx, detail.Document.ID)
	if err != nil {
		t.Fatalf("list contents: %v", err)
	}
	if len(rows) != 1 || rows[0].CreatedAt != now {
		t.Fatalf("unexpected initial content rows %#v", rows)
	}

	now += 60000
	if _, err := service.MarkDocumentRead(ctx, detail.Document.ID); err != nil {
		t.Fatalf("mark document read: %v", err)
	}
	rows, err = store.Queries().ListDocumentContents(ctx, detail.Document.ID)
	if err != nil {
		t.Fatalf("list contents after update: %v", err)
	}
	if len(rows) != 1 || rows[0].CreatedAt != 1760000000000 {
		t.Fatalf("content created_at was not preserved: %#v", rows)
	}
}

func TestSQLiteRepositoryPreservesTranslationSourceContent(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()

	source, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Article", Content: "Original body",
	})
	if err != nil {
		t.Fatalf("create scraped document: %v", err)
	}
	detail, err := service.AppendTranslatedContent(ctx, source.Document.ID, AppendTranslationInput{
		SourceContentID: source.Contents[0].ID,
		TargetLanguage:  "ko",
		Content:         "Translated searchable phrase",
	})
	if err != nil {
		t.Fatalf("append translation: %v", err)
	}
	if len(detail.Contents) != 2 {
		t.Fatalf("expected source and translation contents, got %#v", detail.Contents)
	}
	translation := detail.Contents[1]
	if translation.SourceContentID == nil || *translation.SourceContentID != source.Contents[0].ID {
		t.Fatalf("expected source content link, got %#v", translation.SourceContentID)
	}

	reloaded, err := service.GetDocument(ctx, source.Document.ID)
	if err != nil {
		t.Fatalf("reload document: %v", err)
	}
	translation = reloaded.Contents[1]
	if translation.SourceContentID == nil || *translation.SourceContentID != source.Contents[0].ID {
		t.Fatalf("expected persisted source content link, got %#v", translation.SourceContentID)
	}
	list, err := service.ListDocuments(ctx, DocumentFilters{Query: "searchable"})
	if err != nil {
		t.Fatalf("search translated content: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != source.Document.ID {
		t.Fatalf("expected translated content search hit, got %#v", list.Items)
	}
	if strings.Contains(list.Items[0].Title, "Translated") {
		t.Fatalf("translation should not replace metadata: %#v", list.Items[0])
	}
	_, err = service.AppendTranslatedContent(ctx, source.Document.ID, AppendTranslationInput{
		SourceContentID: source.Contents[0].ID, TargetLanguage: "ja",
		Content: "Invalid format", Format: "pdf",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected app validation before sqlite constraint, got %v", err)
	}
}
