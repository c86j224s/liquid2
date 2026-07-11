package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestSQLiteRepositoryPersistsFeedsItemsAndJobs(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "liquid2.db")
	service, closeService := newSQLiteService(t, ctx, dbPath)

	feed, err := service.CreateFeed(ctx, CreateFeedInput{URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	document, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Feed item", Kind: DocumentKindBookmark})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	publishedAt := int64(1759990000000)
	err = service.repo.Update(ctx, func(tx RepositoryTx) error {
		tx.PutFeedItem(FeedItem{
			ID: "feed_item_1", FeedID: feed.ID, DocumentID: document.Document.ID,
			GUID: stringPtr("guid-1"), URL: "https://example.com/a",
			PublishedAt: &publishedAt, CreatedAt: tx.Now(),
		})
		tx.PutJob(Job{
			ID: "job_1", Kind: JobKindPollFeed, Status: JobStatusQueued,
			PayloadJSON: "{}", CreatedAt: tx.Now(), UpdatedAt: tx.Now(),
		})
		return nil
	})
	if err != nil {
		t.Fatalf("seed feed item and job: %v", err)
	}
	closeService()

	service, closeService = newSQLiteService(t, ctx, dbPath)
	defer closeService()
	feeds, err := service.ListFeeds(ctx)
	if err != nil {
		t.Fatalf("list feeds: %v", err)
	}
	if len(feeds) != 1 || feeds[0].ID != feed.ID {
		t.Fatalf("unexpected feeds %#v", feeds)
	}
	detail, err := service.GetDocument(ctx, document.Document.ID)
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	if detail.Document.PublishedAt == nil || *detail.Document.PublishedAt != publishedAt {
		t.Fatalf("expected persisted published_at %d, got %#v", publishedAt, detail.Document.PublishedAt)
	}
	list, err := service.ListJobs(ctx, JobFilters{Kind: JobKindPollFeed})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != "job_1" {
		t.Fatalf("unexpected jobs %#v", list.Items)
	}
	err = service.repo.View(ctx, func(tx RepositoryReader) error {
		items := tx.FeedItems(feed.ID)
		if len(items) != 1 || items[0].ID != "feed_item_1" {
			t.Fatalf("unexpected feed items %#v", items)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view feed items: %v", err)
	}
}

func stringPtr(value string) *string {
	return &value
}

func TestSQLiteRepositoryRejectsDuplicateFeedItem(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()
	feed, err := service.CreateFeed(ctx, CreateFeedInput{URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	doc, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "A", Kind: DocumentKindBookmark})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	err = service.repo.Update(ctx, func(tx RepositoryTx) error {
		item := FeedItem{
			ID: "item_1", FeedID: feed.ID, DocumentID: doc.Document.ID,
			GUID: stringPtr("guid-1"), URL: "https://example.com/a", CreatedAt: tx.Now(),
		}
		tx.PutFeedItem(item)
		item.ID = "item_2"
		tx.PutFeedItem(item)
		return nil
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected duplicate feed item conflict, got %v", err)
	}
}

func TestSQLiteRepositoryClearsFeedFolderOnDelete(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()
	folder, err := service.CreateFolder(ctx, FolderInput{Name: "Custom feeds"})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	feed, err := service.CreateFeed(ctx, CreateFeedInput{
		URL: "https://example.com/feed.xml",
	})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	feed, err = service.UpdateFeed(ctx, feed.ID, UpdateFeedInput{FolderID: &folder.ID})
	if err != nil {
		t.Fatalf("move feed folder: %v", err)
	}
	if err := service.DeleteFolder(ctx, folder.ID, "move_to_uncategorized"); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	feeds, err := service.ListFeeds(ctx)
	if err != nil {
		t.Fatalf("list feeds: %v", err)
	}
	if len(feeds) != 1 || feeds[0].ID != feed.ID || feeds[0].FolderID != nil {
		t.Fatalf("expected feed moved out of deleted folder, got %#v", feeds)
	}
}

func TestSQLiteRepositoryAdoptsOrphanRSSDocument(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()
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
}
