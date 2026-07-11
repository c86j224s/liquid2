package app

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteRepositoryHidesTrashDescendantDocuments(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()
	detail, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/discard", Title: "Discard",
		Content: "trash descendant search body", Format: ContentFormatText,
	})
	if err != nil {
		t.Fatalf("create scraped document: %v", err)
	}
	var childID string
	err = service.repo.Update(ctx, func(tx RepositoryTx) error {
		trashID := ensureTrashFolder(tx)
		childID = "folder_trash_child"
		now := tx.Now()
		tx.PutFolder(Folder{
			ID: childID, ParentID: &trashID, Name: "Trash Child",
			CreatedAt: now, UpdatedAt: now,
		})
		record, _ := tx.Document(detail.Document.ID)
		record.meta.FolderID = &childID
		record.meta.UpdatedAt = now
		tx.PutDocument(record)
		return nil
	})
	if err != nil {
		t.Fatalf("seed trash child: %v", err)
	}

	list, err := service.ListDocuments(ctx, DocumentFilters{})
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(list.Items) != 0 || list.TotalCount != 0 {
		t.Fatalf("expected trash child hidden, got %#v", list)
	}
	search, err := service.ListDocuments(ctx, DocumentFilters{Query: "descendant"})
	if err != nil {
		t.Fatalf("search documents: %v", err)
	}
	if len(search.Items) != 0 || search.TotalCount != 0 {
		t.Fatalf("expected trash child search hidden, got %#v", search)
	}
	scoped, err := service.ListDocuments(ctx, DocumentFilters{FolderID: childID})
	if err != nil {
		t.Fatalf("list scoped documents: %v", err)
	}
	if len(scoped.Items) != 1 || scoped.Items[0].ID != detail.Document.ID {
		t.Fatalf("expected scoped trash child document, got %#v", scoped)
	}
}
