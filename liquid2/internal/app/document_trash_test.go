package app

import (
	"context"
	"errors"
	"testing"
)

func TestListFoldersEnsuresTrashSystemFolder(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })

	folders, err := service.ListFolders(context.Background())
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	trash := findFolderByRole(folders, FolderSystemRoleTrash)
	if trash == nil || trash.Name != trashDocumentFolderName {
		t.Fatalf("expected trash system folder, got %#v", folders)
	}
}

func TestMoveDocumentToTrashHidesFromDefaultList(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	detail, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Discard"})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	detail, err = service.MoveDocumentToTrash(ctx, detail.Document.ID)
	if err != nil {
		t.Fatalf("move to trash: %v", err)
	}
	if detail.Document.FolderID == nil {
		t.Fatal("expected trash folder")
	}
	list, err := service.ListDocuments(ctx, DocumentFilters{})
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(list.Items) != 0 || list.TotalCount != 0 {
		t.Fatalf("expected trash hidden from default list, got %#v", list)
	}
	trashList, err := service.ListDocuments(ctx, DocumentFilters{
		FolderID: *detail.Document.FolderID,
	})
	if err != nil {
		t.Fatalf("list trash documents: %v", err)
	}
	if len(trashList.Items) != 1 || trashList.Items[0].ID != detail.Document.ID {
		t.Fatalf("expected trash document, got %#v", trashList)
	}
}

func TestTrashDescendantDocumentsHideFromDefaultList(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	detail, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Discard"})
	if err != nil {
		t.Fatalf("create document: %v", err)
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
	scoped, err := service.ListDocuments(ctx, DocumentFilters{FolderID: childID})
	if err != nil {
		t.Fatalf("list scoped documents: %v", err)
	}
	if len(scoped.Items) != 1 || scoped.Items[0].ID != detail.Document.ID {
		t.Fatalf("expected scoped trash child document, got %#v", scoped)
	}
}

func TestFolderCannotMoveUnderDescendant(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	parent, err := service.CreateFolder(ctx, FolderInput{Name: "Parent"})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	child, err := service.CreateFolder(ctx, FolderInput{
		Name: "Child", ParentID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	_, err = service.UpdateFolder(ctx, parent.ID, FolderInput{
		Name: "Parent", ParentID: &child.ID,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestTrashCannotBeAssignedDirectly(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	folders, err := service.ListFolders(ctx)
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	trash := findFolderByRole(folders, FolderSystemRoleTrash)
	if trash == nil {
		t.Fatalf("expected trash folder, got %#v", folders)
	}

	_, err = service.CreateBookmarkDocument(ctx, BookmarkDocumentInput{
		URL: "https://example.com", Title: "Discard", FolderID: trash.ID,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected bookmark validation error, got %v", err)
	}
	detail, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Keep"})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	_, err = service.UpdateDocument(ctx, detail.Document.ID, UpdateDocumentInput{
		FolderID: &trash.ID,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected update validation error, got %v", err)
	}
	_, err = service.CreateFeed(ctx, CreateFeedInput{
		URL: "https://example.com/feed.xml", FolderID: &trash.ID,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected feed validation error, got %v", err)
	}
}

func TestSystemFoldersCannotBeEditedOrDeleted(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	folders, err := service.ListFolders(ctx)
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	trash := findFolderByRole(folders, FolderSystemRoleTrash)
	if trash == nil {
		t.Fatalf("expected trash folder, got %#v", folders)
	}

	_, err = service.UpdateFolder(ctx, trash.ID, FolderInput{Name: "Bin"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	err = service.DeleteFolder(ctx, trash.ID, "reject_if_not_empty")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func findFolderByRole(folders []Folder, role string) *Folder {
	for _, folder := range folders {
		if folder.SystemRole != nil && *folder.SystemRole == role {
			return &folder
		}
		if match := findFolderByRole(folder.Children, role); match != nil {
			return match
		}
	}
	return nil
}
