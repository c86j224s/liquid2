package sqlite

import (
	"testing"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func TestDocumentCRUD(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()

	createTestDocument(t, ctx, q, "doc_1")

	updated, err := q.UpdateDocumentMetadata(ctx, sqlitedb.UpdateDocumentMetadataParams{
		ID:        "doc_1",
		Title:     "Updated",
		FolderID:  nullString("folder_test"),
		UpdatedAt: 2000,
	})
	if err != nil {
		t.Fatalf("update document: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("expected updated title, got %q", updated.Title)
	}

	deleted, err := q.SoftDeleteDocument(ctx, sqlitedb.SoftDeleteDocumentParams{
		ID:        "doc_1",
		DeletedAt: nullInt(3000),
		UpdatedAt: 3000,
	})
	if err != nil {
		t.Fatalf("soft delete document: %v", err)
	}
	if !deleted.DeletedAt.Valid || deleted.DeletedAt.Int64 != 3000 {
		t.Fatalf("expected deleted_at 3000, got %+v", deleted.DeletedAt)
	}
}

func TestDocumentConstraints(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()
	createTestFolder(t, ctx, q, "folder_test")

	_, err := q.CreateDocument(ctx, sqlitedb.CreateDocumentParams{
		ID:        "bad_status",
		Title:     "Bad",
		Kind:      "bookmark",
		FolderID:  nullString("folder_test"),
		Status:    "archived",
		CreatedAt: 1000,
		UpdatedAt: 1000,
	})
	if err == nil {
		t.Fatal("expected invalid status constraint error")
	}

	_, err = q.CreateDocument(ctx, sqlitedb.CreateDocumentParams{
		ID:        "bad_rating",
		Title:     "Bad",
		Kind:      "bookmark",
		FolderID:  nullString("folder_test"),
		Status:    "unread",
		Rating:    nullInt(6),
		CreatedAt: 1000,
		UpdatedAt: 1000,
	})
	if err == nil {
		t.Fatal("expected invalid rating constraint error")
	}
}
