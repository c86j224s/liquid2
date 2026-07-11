package app

import (
	"context"
	"errors"
	"testing"
)

func TestListFoldersEnsuresFeedsSystemFolder(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })

	folders, err := service.ListFolders(context.Background())
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	feeds := findFolderByRole(folders, FolderSystemRoleFeeds)
	if feeds == nil || feeds.Name != feedsSystemFolderName {
		t.Fatalf("expected feeds system folder, got %#v", folders)
	}
}

func TestCreateFeedCreatesFolderUnderFeeds(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	title := "Example Feed"

	feed, err := service.CreateFeed(ctx, CreateFeedInput{
		URL: "https://example.com/feed.xml", Title: &title,
	})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	if feed.FolderID == nil {
		t.Fatal("expected feed folder")
	}
	folders, err := service.ListFolders(ctx)
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	feeds := findFolderByRole(folders, FolderSystemRoleFeeds)
	if feeds == nil || len(feeds.Children) != 1 || feeds.Children[0].ID != *feed.FolderID {
		t.Fatalf("expected feed child folder under feeds, got %#v", folders)
	}
	if feeds.Children[0].Name != title {
		t.Fatalf("expected feed folder title, got %#v", feeds.Children[0])
	}
}

func TestCreateFeedRejectsManualFolder(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	folder, err := service.CreateFolder(ctx, FolderInput{Name: "Manual"})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}

	_, err = service.CreateFeed(ctx, CreateFeedInput{
		URL: "https://example.com/feed.xml", FolderID: &folder.ID,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestFeedsFolderCannotBeAssignedToManualDocuments(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	folders, err := service.ListFolders(ctx)
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	feeds := findFolderByRole(folders, FolderSystemRoleFeeds)
	if feeds == nil {
		t.Fatalf("expected feeds folder, got %#v", folders)
	}

	_, err = service.CreateBookmarkDocument(ctx, BookmarkDocumentInput{
		URL: "https://example.com", Title: "Manual", FolderID: feeds.ID,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected bookmark validation error, got %v", err)
	}
	detail, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Manual"})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	_, err = service.UpdateDocument(ctx, detail.Document.ID, UpdateDocumentInput{
		FolderID: &feeds.ID,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected update validation error, got %v", err)
	}
}
