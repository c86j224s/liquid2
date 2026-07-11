package app

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteRepositoryPersistsDocumentVersions(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "liquid2.db")
	service, closeService := newSQLiteService(t, ctx, dbPath)
	source, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Original", Content: "Body",
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	newTitle := "Renamed"
	if _, err = service.UpdateDocument(ctx, source.Document.ID, UpdateDocumentInput{Title: &newTitle}); err != nil {
		t.Fatalf("rename document: %v", err)
	}
	closeService()

	service, closeService = newSQLiteService(t, ctx, dbPath)
	defer closeService()
	versions, err := service.ListDocumentVersions(ctx, source.Document.ID)
	if err != nil {
		t.Fatalf("list versions after reload: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected one persisted version, got %#v", versions)
	}
	version := versions[0]
	if version.MutationKind != DocumentMutationTitle || version.Title != "Original" {
		t.Fatalf("unexpected persisted version: %#v", version)
	}
	if len(version.Contents) != 1 || version.Contents[0].Content != "Body" {
		t.Fatalf("expected persisted content snapshot, got %#v", version.Contents)
	}
}

func TestSQLiteRepositoryPreservesNilVersionContents(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "liquid2.db")
	service, closeService := newSQLiteService(t, ctx, dbPath)
	source, err := service.CreateBookmarkDocument(ctx, BookmarkDocumentInput{
		URL: "https://example.com/a", Title: "Bookmark",
	})
	if err != nil {
		t.Fatalf("create bookmark: %v", err)
	}
	newTitle := "Renamed"
	if _, err = service.UpdateDocument(ctx, source.Document.ID, UpdateDocumentInput{Title: &newTitle}); err != nil {
		t.Fatalf("rename bookmark: %v", err)
	}
	closeService()

	service, closeService = newSQLiteService(t, ctx, dbPath)
	defer closeService()
	versions, err := service.ListDocumentVersions(ctx, source.Document.ID)
	if err != nil {
		t.Fatalf("list versions after reload: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected one persisted version, got %#v", versions)
	}
	if versions[0].Contents != nil {
		t.Fatalf("expected nil content snapshot, got %#v", versions[0].Contents)
	}
}

func TestSQLiteRepositoryPersistsVersionContentSourceLinks(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "liquid2.db")
	service, closeService := newSQLiteService(t, ctx, dbPath)
	source, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Original", Content: "Body",
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	if _, err = service.AppendTranslatedContent(ctx, source.Document.ID, AppendTranslationInput{
		SourceContentID: source.Contents[0].ID, TargetLanguage: "ko", Content: "Korean",
	}); err != nil {
		t.Fatalf("append first translation: %v", err)
	}
	if _, err = service.AppendTranslatedContent(ctx, source.Document.ID, AppendTranslationInput{
		SourceContentID: source.Contents[0].ID, TargetLanguage: "ja", Content: "Japanese",
	}); err != nil {
		t.Fatalf("append second translation: %v", err)
	}
	closeService()

	service, closeService = newSQLiteService(t, ctx, dbPath)
	defer closeService()
	versions, err := service.ListDocumentVersions(ctx, source.Document.ID)
	if err != nil {
		t.Fatalf("list versions after reload: %v", err)
	}
	if len(versions) != 2 || len(versions[1].Contents) != 2 {
		t.Fatalf("expected second snapshot to include first translation, got %#v", versions)
	}
	translation := versions[1].Contents[1]
	if translation.SourceContentID == nil || *translation.SourceContentID != source.Contents[0].ID {
		t.Fatalf("expected persisted source content link, got %#v", translation)
	}
}
