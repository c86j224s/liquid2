package app

import (
	"context"
	"testing"
)

func TestDocumentHistoryCapturesTitleMutation(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	detail, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Original", Content: "Body",
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	sameTitle := "Original"
	if _, err = service.UpdateDocument(ctx, detail.Document.ID, UpdateDocumentInput{Title: &sameTitle}); err != nil {
		t.Fatalf("update with same title: %v", err)
	}
	assertDocumentVersionCount(t, service, detail.Document.ID, 0)

	newTitle := "Renamed"
	if _, err = service.UpdateDocument(ctx, detail.Document.ID, UpdateDocumentInput{Title: &newTitle}); err != nil {
		t.Fatalf("rename document: %v", err)
	}
	versions := assertDocumentVersionCount(t, service, detail.Document.ID, 1)
	version := versions[0]
	if version.MutationKind != DocumentMutationTitle || version.Sequence != 1 {
		t.Fatalf("unexpected title version metadata: %#v", version)
	}
	if version.Title != "Original" || version.Metadata.Title != "Original" {
		t.Fatalf("expected old title snapshot, got %#v", version)
	}
	if len(version.Contents) != 1 || version.Contents[0].Content != "Body" {
		t.Fatalf("expected old content snapshot, got %#v", version.Contents)
	}

	thirdTitle := "Third"
	if _, err = service.UpdateDocument(ctx, detail.Document.ID, UpdateDocumentInput{Title: &thirdTitle}); err != nil {
		t.Fatalf("rename document again: %v", err)
	}
	versions = assertDocumentVersionCount(t, service, detail.Document.ID, 2)
	if versions[1].Sequence != 2 || versions[1].Title != "Renamed" {
		t.Fatalf("expected second sequence with previous title, got %#v", versions[1])
	}
}

func TestDocumentHistoryCapturesTranslationAppend(t *testing.T) {
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
		SourceContentID: source.Contents[0].ID,
		TargetLanguage:  "ko",
		Content:         "Translated body",
	}); err != nil {
		t.Fatalf("append translation: %v", err)
	}
	versions := assertDocumentVersionCount(t, service, source.Document.ID, 1)
	version := versions[0]
	if version.MutationKind != DocumentMutationContent {
		t.Fatalf("expected content mutation version, got %#v", version)
	}
	if len(version.Contents) != 1 || version.Contents[0].Content != "Original body" {
		t.Fatalf("expected pre-translation content snapshot, got %#v", version.Contents)
	}
}

func TestDocumentHistoryIgnoresMetadataOnlyAndCreationPaths(t *testing.T) {
	service := NewService()
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	detail, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Article", Content: "Body",
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	assertDocumentVersionCount(t, service, detail.Document.ID, 0)

	if _, err = service.MarkDocumentRead(ctx, detail.Document.ID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	rating := 4
	if _, err = service.SetDocumentRating(ctx, detail.Document.ID, &rating); err != nil {
		t.Fatalf("set rating: %v", err)
	}
	folder, err := service.CreateFolder(ctx, FolderInput{Name: "Archive"})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if _, err = service.UpdateDocument(ctx, detail.Document.ID, UpdateDocumentInput{FolderID: &folder.ID}); err != nil {
		t.Fatalf("move document: %v", err)
	}
	tag, err := service.CreateTag(ctx, "Research")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if _, err = service.ReplaceDocumentTags(ctx, detail.Document.ID, []string{tag.ID}); err != nil {
		t.Fatalf("replace tags: %v", err)
	}
	note, err := service.CreateDocumentNote(ctx, detail.Document.ID, CreateNoteInput{Body: "note", Format: "text"})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if _, err = service.UpdateDocumentNote(ctx, detail.Document.ID, note.ID, CreateNoteInput{Body: "updated", Format: "text"}); err != nil {
		t.Fatalf("update note: %v", err)
	}
	if _, err = service.DeleteDocumentNote(ctx, detail.Document.ID, note.ID); err != nil {
		t.Fatalf("delete note: %v", err)
	}
	if _, err = service.DeleteDocument(ctx, detail.Document.ID); err != nil {
		t.Fatalf("soft delete document: %v", err)
	}
	assertDocumentVersionCount(t, service, detail.Document.ID, 0)
}

func assertDocumentVersionCount(t *testing.T, service *Service, documentID string, want int) []DocumentVersion {
	t.Helper()
	versions, err := service.ListDocumentVersions(context.Background(), documentID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != want {
		t.Fatalf("expected %d versions, got %#v", want, versions)
	}
	return versions
}
