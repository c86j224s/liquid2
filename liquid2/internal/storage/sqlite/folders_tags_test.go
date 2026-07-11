package sqlite

import (
	"testing"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func TestFolderTreeAndSiblingUniqueness(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()

	root, err := q.CreateFolder(ctx, sqlitedb.CreateFolderParams{
		ID: "folder_root", Name: "Root", CreatedAt: 1000, UpdatedAt: 1000,
	})
	if err != nil {
		t.Fatalf("create root folder: %v", err)
	}
	_, err = q.CreateFolder(ctx, sqlitedb.CreateFolderParams{
		ID: "folder_root_dup", Name: "Root", CreatedAt: 1000, UpdatedAt: 1000,
	})
	if err == nil {
		t.Fatal("expected duplicate root folder error")
	}
	_, err = q.CreateFolder(ctx, sqlitedb.CreateFolderParams{
		ID: "folder_child", ParentID: nullString(root.ID), Name: "Child",
		CreatedAt: 1000, UpdatedAt: 1000,
	})
	if err != nil {
		t.Fatalf("create child folder: %v", err)
	}
	_, err = q.CreateFolder(ctx, sqlitedb.CreateFolderParams{
		ID: "folder_dup", ParentID: nullString(root.ID), Name: "Child",
		CreatedAt: 1000, UpdatedAt: 1000,
	})
	if err == nil {
		t.Fatal("expected duplicate sibling folder error")
	}
}

func TestTagAssignment(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()
	createTestDocument(t, ctx, q, "doc_1")

	tag, err := q.CreateTag(ctx, sqlitedb.CreateTagParams{
		ID: "tag_1", Name: "SQLite", Slug: "sqlite", CreatedAt: 1000,
	})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	_, err = q.CreateTag(ctx, sqlitedb.CreateTagParams{
		ID: "tag_2", Name: "SQLite again", Slug: "sqlite", CreatedAt: 1000,
	})
	if err == nil {
		t.Fatal("expected duplicate tag slug error")
	}
	if err := q.AssignDocumentTag(ctx, sqlitedb.AssignDocumentTagParams{
		DocumentID: "doc_1", TagID: tag.ID,
	}); err != nil {
		t.Fatalf("assign tag: %v", err)
	}
	if err := q.AssignDocumentTag(ctx, sqlitedb.AssignDocumentTagParams{
		DocumentID: "doc_1", TagID: tag.ID,
	}); err == nil {
		t.Fatal("expected duplicate document tag error")
	}
}
