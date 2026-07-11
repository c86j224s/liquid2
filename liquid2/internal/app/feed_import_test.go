package app

import (
	"context"
	"errors"
	"testing"
)

func TestImportFeedItemsCreatesRSSDocumentsOnce(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	feed, err := service.CreateFeed(ctx, CreateFeedInput{
		URL: "https://example.com/feed.xml",
	})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	publishedAt := int64(1759990000000)
	items := []FeedImportItem{{
		Title: "First", URL: "https://example.com/a", SourceURL: "https://example.com/a?utm=1",
		GUID: "guid-1", ContentHash: "hash-1", PublishedAt: &publishedAt, Content: "body",
	}}
	result, err := service.ImportFeedItems(ctx, feed.ID, items)
	if err != nil {
		t.Fatalf("import feed items: %v", err)
	}
	if result.Imported != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected first result %#v", result)
	}
	result, err = service.ImportFeedItems(ctx, feed.ID, items)
	if err != nil {
		t.Fatalf("repeat import feed items: %v", err)
	}
	if result.Imported != 0 || result.Skipped != 1 {
		t.Fatalf("unexpected repeat result %#v", result)
	}
	docs, err := service.ListDocuments(ctx, DocumentFilters{Kind: DocumentKindRSSItem})
	if err != nil {
		t.Fatalf("list rss docs: %v", err)
	}
	if len(docs.Items) != 1 || docs.Items[0].FolderID == nil || *docs.Items[0].FolderID != *feed.FolderID {
		t.Fatalf("unexpected imported docs %#v", docs.Items)
	}
	if docs.Items[0].PublishedAt == nil || *docs.Items[0].PublishedAt != publishedAt {
		t.Fatalf("expected summary published_at %d, got %#v", publishedAt, docs.Items[0].PublishedAt)
	}
	detail, err := service.GetDocument(ctx, docs.Items[0].ID)
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	if detail.Document.PublishedAt == nil || *detail.Document.PublishedAt != publishedAt {
		t.Fatalf("expected detail published_at %d, got %#v", publishedAt, detail.Document.PublishedAt)
	}
}

func TestImportFeedItemsSkipsTrashMovedRSSDocument(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	feed, err := service.CreateFeed(ctx, CreateFeedInput{URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	items := []FeedImportItem{{
		Title: "First", URL: "https://example.com/a",
		GUID: "guid-1", ContentHash: "hash-1",
	}}
	result, err := service.ImportFeedItems(ctx, feed.ID, items)
	if err != nil {
		t.Fatalf("import feed items: %v", err)
	}
	if result.Imported != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected first result %#v", result)
	}
	docs, err := service.ListDocuments(ctx, DocumentFilters{Kind: DocumentKindRSSItem})
	if err != nil {
		t.Fatalf("list rss docs: %v", err)
	}
	if len(docs.Items) != 1 {
		t.Fatalf("expected one rss document, got %#v", docs.Items)
	}
	if _, err = service.MoveDocumentToTrash(ctx, docs.Items[0].ID); err != nil {
		t.Fatalf("move to trash: %v", err)
	}

	result, err = service.ImportFeedItems(ctx, feed.ID, items)
	if err != nil {
		t.Fatalf("repeat import feed items: %v", err)
	}
	if result.Imported != 0 || result.Skipped != 1 {
		t.Fatalf("expected trash-preserved duplicate skip, got %#v", result)
	}
}

func TestImportFeedItemsRejectsDisabledFeed(t *testing.T) {
	ctx := context.Background()
	disabled := false
	service := NewService()
	t.Cleanup(func() { _ = service.Close() })
	feed, err := service.CreateFeed(ctx, CreateFeedInput{
		URL: "https://example.com/feed.xml", Enabled: &disabled,
	})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	_, err = service.ImportFeedItems(ctx, feed.ID, []FeedImportItem{{
		Title: "First", URL: "https://example.com/a",
	}})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected disabled feed conflict, got %v", err)
	}
}
