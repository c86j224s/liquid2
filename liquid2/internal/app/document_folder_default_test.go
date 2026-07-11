package app

import (
	"context"
	"errors"
	"testing"
)

func TestCreateDocumentAssignsDefaultFolder(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()

	detail, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Loose"})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	if detail.Document.FolderID == nil {
		t.Fatal("expected default folder")
	}
	folders, err := service.ListFolders(ctx)
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	if len(folders) != 3 ||
		folders[0].Name != defaultDocumentFolderName ||
		folders[1].Name != feedsSystemFolderName ||
		folders[2].Name != trashDocumentFolderName {
		t.Fatalf("expected default inbox folder, got %#v", folders)
	}
}

func TestUpdateDocumentEmptyFolderMovesToDefault(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	folder, err := service.CreateFolder(ctx, FolderInput{Name: "Research"})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	detail, err := service.CreateBookmarkDocument(ctx, BookmarkDocumentInput{
		URL: "https://example.com", Title: "Saved", FolderID: folder.ID,
	})
	if err != nil {
		t.Fatalf("create bookmark: %v", err)
	}

	empty := ""
	detail, err = service.UpdateDocument(ctx, detail.Document.ID, UpdateDocumentInput{FolderID: &empty})
	if err != nil {
		t.Fatalf("clear folder: %v", err)
	}
	if detail.Document.FolderID == nil || *detail.Document.FolderID == folder.ID {
		t.Fatalf("expected default folder, got %#v", detail.Document.FolderID)
	}
}

func TestDeleteRootFolderMovesDocumentsToDefault(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	folder, err := service.CreateFolder(ctx, FolderInput{Name: "Research"})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	detail, err := service.CreateBookmarkDocument(ctx, BookmarkDocumentInput{
		URL: "https://example.com", Title: "Saved", FolderID: folder.ID,
	})
	if err != nil {
		t.Fatalf("create bookmark: %v", err)
	}

	if err := service.DeleteFolder(ctx, folder.ID, "move_to_uncategorized"); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	detail, err = service.GetDocument(ctx, detail.Document.ID)
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	if detail.Document.FolderID == nil || *detail.Document.FolderID == folder.ID {
		t.Fatalf("expected document moved to default folder, got %#v", detail.Document.FolderID)
	}
}

func TestDeleteDefaultFolderWithDocumentsConflicts(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	detail, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Loose"})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	err = service.DeleteFolder(ctx, *detail.Document.FolderID, "move_to_uncategorized")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}
