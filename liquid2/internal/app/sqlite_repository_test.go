package app

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"

	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
)

func TestSQLiteRepositoryPersistsDocumentLibrary(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "liquid2.db")
	service, closeService := newSQLiteService(t, ctx, dbPath)

	folder, err := service.CreateFolder(ctx, FolderInput{Name: "Research", SortOrder: 10})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	tag, err := service.CreateTag(ctx, "SQLite")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	detail, err := service.CreateBookmarkDocument(ctx, BookmarkDocumentInput{
		URL: "https://example.com/a", SourceURL: "https://example.com/a#x",
		Title: "Example", FolderID: folder.ID, TagIDs: []string{tag.ID},
	})
	if err != nil {
		t.Fatalf("create bookmark: %v", err)
	}
	if _, err := service.CreateDocumentNote(ctx, detail.Document.ID, CreateNoteInput{Body: "Remember", Format: "text"}); err != nil {
		t.Fatalf("create note: %v", err)
	}
	closeService()

	service, closeService = newSQLiteService(t, ctx, dbPath)
	defer closeService()
	detail, err = service.GetDocument(ctx, detail.Document.ID)
	if err != nil {
		t.Fatalf("get persisted document: %v", err)
	}
	if detail.Document.Title != "Example" || detail.Document.FolderID == nil || *detail.Document.FolderID != folder.ID {
		t.Fatalf("unexpected persisted document %#v", detail.Document)
	}
	if detail.Contents != nil {
		t.Fatalf("expected nil bookmark contents, got %#v", detail.Contents)
	}
	if len(detail.Tags) != 1 || detail.Tags[0].Slug != "sqlite" {
		t.Fatalf("unexpected persisted tags %#v", detail.Tags)
	}
	notes, err := service.ListDocumentNotes(ctx, detail.Document.ID)
	if err != nil {
		t.Fatalf("list persisted notes: %v", err)
	}
	if len(notes.Items) != 1 || notes.Items[0].Body != "Remember" {
		t.Fatalf("unexpected persisted notes %#v", notes.Items)
	}
}

func TestSQLiteRepositoryStoresUploadedBlobBytes(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()

	detail, err := service.CreateUploadedDocument(ctx, UploadedDocumentInput{
		Filename: "note.txt", MimeType: "text/plain",
		Data: []byte("Stored body"), Content: "Stored body", Format: ContentFormatText,
	})
	if err != nil {
		t.Fatalf("create uploaded document: %v", err)
	}
	blobID := detail.Blobs[0].ID
	err = service.repo.View(ctx, func(tx RepositoryReader) error {
		record, ok := tx.Document(detail.Document.ID)
		if !ok {
			return fmt.Errorf("document missing")
		}
		if !bytes.Equal(record.blobData[blobID], []byte("Stored body")) {
			return fmt.Errorf("unexpected blob data %q", string(record.blobData[blobID]))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view uploaded blob: %v", err)
	}
}

func TestSQLiteRepositoryUpdatesAndDeletes(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()
	folder, err := service.CreateFolder(ctx, FolderInput{Name: "Research"})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	tag, err := service.CreateTag(ctx, "Backend")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	detail, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Article", Content: "Body",
		Format: ContentFormatText, FolderID: folder.ID, TagIDs: []string{tag.ID},
	})
	if err != nil {
		t.Fatalf("create scraped document: %v", err)
	}
	rating := 5
	if _, err := service.SetDocumentRating(ctx, detail.Document.ID, &rating); err != nil {
		t.Fatalf("set rating: %v", err)
	}
	if _, err := service.MarkDocumentRead(ctx, detail.Document.ID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	note, err := service.CreateDocumentNote(ctx, detail.Document.ID, CreateNoteInput{Body: "Draft", Format: "text"})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if _, err := service.UpdateDocumentNote(ctx, detail.Document.ID, note.ID, CreateNoteInput{Body: "Final", Format: "markdown"}); err != nil {
		t.Fatalf("update note: %v", err)
	}
	if _, err := service.DeleteDocumentNote(ctx, detail.Document.ID, note.ID); err != nil {
		t.Fatalf("delete note: %v", err)
	}
	if err := service.DeleteFolder(ctx, folder.ID, "move_to_uncategorized"); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	detail, err = service.GetDocument(ctx, detail.Document.ID)
	if err != nil {
		t.Fatalf("get updated document: %v", err)
	}
	if detail.Document.Status != DocumentStatusRead || detail.Document.Rating == nil || *detail.Document.Rating != rating {
		t.Fatalf("unexpected updated document %#v", detail.Document)
	}
	if detail.Document.FolderID == nil || *detail.Document.FolderID == folder.ID {
		t.Fatalf("expected document moved to fallback folder, got %#v", detail.Document.FolderID)
	}
	notes, err := service.ListDocumentNotes(ctx, detail.Document.ID)
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if len(notes.Items) != 0 {
		t.Fatalf("expected deleted note hidden, got %#v", notes.Items)
	}
}

func newSQLiteService(t *testing.T, ctx context.Context, dbPath string) (*Service, func()) {
	t.Helper()
	store, err := sqlitestore.Open(ctx, dbPath, sqlitestore.WithLogger(slog.Default()))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate sqlite store: %v", err)
	}
	repo := NewSQLiteRepository(store, WithSQLiteRepositoryClock(func() int64 { return 1760000000000 }))
	service := NewService(WithRepository(repo))
	return service, func() {
		if err := service.Close(); err != nil {
			t.Fatalf("close service: %v", err)
		}
		if err := store.Close(); err != nil {
			t.Fatalf("close sqlite store: %v", err)
		}
	}
}
