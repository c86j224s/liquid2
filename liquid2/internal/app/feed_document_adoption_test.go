package app

import (
	"context"
	"testing"
)

func TestImportFeedItemsAdoptsOrphanRSSDocument(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	orphanID := seedOrphanRSSDocument(t, ctx, service, "https://example.com/a", FolderSystemRoleFeeds)
	feed, err := service.CreateFeed(ctx, CreateFeedInput{URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}

	result, err := service.ImportFeedItems(ctx, feed.ID, []FeedImportItem{{
		Title: "First", URL: "https://example.com/a",
		GUID: "guid-1", ContentHash: "hash-1",
	}})
	if err != nil {
		t.Fatalf("import feed items: %v", err)
	}
	if result.Imported != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected import result %#v", result)
	}
	assertSingleFeedItemDocument(t, ctx, service, feed, orphanID)
	docs, err := service.ListDocuments(ctx, DocumentFilters{Kind: DocumentKindRSSItem})
	if err != nil {
		t.Fatalf("list rss docs: %v", err)
	}
	if len(docs.Items) != 1 || docs.Items[0].ID != orphanID {
		t.Fatalf("expected orphan adoption without duplicate, got %#v", docs.Items)
	}
	if docs.Items[0].FolderID == nil || *docs.Items[0].FolderID != *feed.FolderID {
		t.Fatalf("expected adopted document in feed folder, got %#v", docs.Items[0])
	}
}

func TestImportFeedItemsAdoptsTrashOrphanWithoutMoving(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	orphanID := seedOrphanRSSDocument(t, ctx, service, "https://example.com/a", FolderSystemRoleTrash)
	feed, err := service.CreateFeed(ctx, CreateFeedInput{URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}

	result, err := service.ImportFeedItems(ctx, feed.ID, []FeedImportItem{{
		Title: "First", URL: "https://example.com/a",
		GUID: "guid-1", ContentHash: "hash-1",
	}})
	if err != nil {
		t.Fatalf("import feed items: %v", err)
	}
	if result.Imported != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected import result %#v", result)
	}
	assertSingleFeedItemDocument(t, ctx, service, feed, orphanID)
	docs, err := service.ListDocuments(ctx, DocumentFilters{Kind: DocumentKindRSSItem})
	if err != nil {
		t.Fatalf("list visible rss docs: %v", err)
	}
	if len(docs.Items) != 0 {
		t.Fatalf("expected adopted trash document to stay hidden, got %#v", docs.Items)
	}
	docs, err = service.ListDocuments(ctx, DocumentFilters{Kind: DocumentKindRSSItem, IncludeTrash: true})
	if err != nil {
		t.Fatalf("list rss docs with trash: %v", err)
	}
	if len(docs.Items) != 1 || docs.Items[0].ID != orphanID {
		t.Fatalf("expected adopted trash document, got %#v", docs.Items)
	}
	trash := findFolderByRole(mustListFolders(t, ctx, service), FolderSystemRoleTrash)
	if docs.Items[0].FolderID == nil || *docs.Items[0].FolderID != trash.ID {
		t.Fatalf("expected trash folder to be preserved, got %#v", docs.Items[0])
	}
}

func TestImportFeedItemsMovesFolderlessOrphanToFeedFolder(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	orphanID := seedOrphanRSSDocument(t, ctx, service, "https://example.com/a", "")
	feed, err := service.CreateFeed(ctx, CreateFeedInput{URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}

	if _, err = service.ImportFeedItems(ctx, feed.ID, []FeedImportItem{{
		Title: "First", URL: "https://example.com/a",
		GUID: "guid-1", ContentHash: "hash-1",
	}}); err != nil {
		t.Fatalf("import feed items: %v", err)
	}
	docs, err := service.ListDocuments(ctx, DocumentFilters{Kind: DocumentKindRSSItem})
	if err != nil {
		t.Fatalf("list rss docs: %v", err)
	}
	if len(docs.Items) != 1 || docs.Items[0].ID != orphanID {
		t.Fatalf("expected folderless orphan adoption, got %#v", docs.Items)
	}
	if docs.Items[0].FolderID == nil || *docs.Items[0].FolderID != *feed.FolderID {
		t.Fatalf("expected adopted document in feed folder, got %#v", docs.Items[0])
	}
}

func seedOrphanRSSDocument(t *testing.T, ctx context.Context, service *Service, sourceURL string, folderRole string) string {
	t.Helper()
	var id string
	err := service.repo.Update(ctx, func(tx RepositoryTx) error {
		var folderID *string
		if folderRole == FolderSystemRoleTrash {
			folderValue := ensureTrashFolder(tx)
			folderID = &folderValue
		}
		if folderRole == FolderSystemRoleFeeds {
			folderValue := ensureFeedsFolder(tx)
			folderID = &folderValue
		}
		id = tx.NextID("doc")
		tx.PutDocument(documentRecord{
			meta: DocumentMetadata{
				ID: id, Title: "First", Kind: DocumentKindRSSItem, FolderID: folderID,
				CanonicalURL: optionalString(sourceURL), SourceURL: optionalString(sourceURL),
				Status: DocumentStatusUnread, CreatedAt: tx.Now(), UpdatedAt: tx.Now(),
			},
			contents: []DocumentContent{contentRecord(tx, "old body", ContentFormatText)},
			blobs:    []BlobMetadata{},
			blobData: map[string][]byte{},
			tagIDs:   []string{},
		})
		return nil
	})
	if err != nil {
		t.Fatalf("seed orphan rss document: %v", err)
	}
	return id
}

func assertSingleFeedItemDocument(t *testing.T, ctx context.Context, service *Service, feed Feed, documentID string) {
	t.Helper()
	err := service.repo.View(ctx, func(tx RepositoryReader) error {
		items := tx.FeedItems(feed.ID)
		if len(items) != 1 {
			t.Fatalf("expected one feed item, got %#v", items)
		}
		if items[0].DocumentID != documentID {
			t.Fatalf("expected feed item document %q, got %#v", documentID, items[0])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view feed items: %v", err)
	}
}

func mustListFolders(t *testing.T, ctx context.Context, service *Service) []Folder {
	t.Helper()
	folders, err := service.ListFolders(ctx)
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	return folders
}
