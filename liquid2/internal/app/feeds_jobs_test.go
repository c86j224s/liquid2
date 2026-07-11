package app

import (
	"context"
	"errors"
	"testing"
)

func TestFeedCRUD(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })

	enabled := false
	title := "Example"
	feed, err := service.CreateFeed(ctx, CreateFeedInput{
		URL: "https://example.com/feed.xml", Title: &title, Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	if feed.Enabled || feed.Title == nil || *feed.Title != title || feed.FolderID == nil {
		t.Fatalf("unexpected feed %#v", feed)
	}
	if _, err := service.CreateFeed(ctx, CreateFeedInput{URL: feed.URL}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected duplicate feed conflict, got %v", err)
	}

	enabled = true
	nextURL := "https://example.com/next.xml"
	updated, err := service.UpdateFeed(ctx, feed.ID, UpdateFeedInput{URL: &nextURL, Enabled: &enabled})
	if err != nil {
		t.Fatalf("update feed: %v", err)
	}
	if !updated.Enabled || updated.URL != nextURL {
		t.Fatalf("unexpected updated feed %#v", updated)
	}
	feeds, err := service.ListFeeds(ctx)
	if err != nil {
		t.Fatalf("list feeds: %v", err)
	}
	if len(feeds) != 1 || feeds[0].ID != feed.ID {
		t.Fatalf("unexpected feeds %#v", feeds)
	}
	if err := service.DeleteFeed(ctx, feed.ID); err != nil {
		t.Fatalf("delete feed: %v", err)
	}
	if _, err := service.UpdateFeed(ctx, feed.ID, UpdateFeedInput{URL: &nextURL}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing feed after delete, got %v", err)
	}
}

func TestDeleteFolderMovesFeedReferences(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })

	parent, err := service.CreateFolder(ctx, FolderInput{Name: "Parent"})
	if err != nil {
		t.Fatalf("create parent folder: %v", err)
	}
	child, err := service.CreateFolder(ctx, FolderInput{Name: "Child", ParentID: &parent.ID})
	if err != nil {
		t.Fatalf("create child folder: %v", err)
	}
	feed, err := service.CreateFeed(ctx, CreateFeedInput{
		URL: "https://example.com/feed.xml",
	})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	feed, err = service.UpdateFeed(ctx, feed.ID, UpdateFeedInput{FolderID: &child.ID})
	if err != nil {
		t.Fatalf("move feed folder: %v", err)
	}

	if err := service.DeleteFolder(ctx, child.ID, "move_to_parent"); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	feeds, err := service.ListFeeds(ctx)
	if err != nil {
		t.Fatalf("list feeds: %v", err)
	}
	if len(feeds) != 1 || feeds[0].ID != feed.ID || feeds[0].FolderID == nil || *feeds[0].FolderID != parent.ID {
		t.Fatalf("expected feed moved to parent folder, got %#v", feeds)
	}
}

func TestJobListAndGet(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })

	err := service.repo.Update(ctx, func(tx RepositoryTx) error {
		tx.PutJob(Job{
			ID: "job_1", Kind: JobKindPollFeed, Status: JobStatusQueued,
			PayloadJSON: "{}", CreatedAt: tx.Now(), UpdatedAt: tx.Now(),
		})
		return nil
	})
	if err != nil {
		t.Fatalf("seed job: %v", err)
	}
	job, err := service.GetJob(ctx, "job_1")
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if job.Kind != JobKindPollFeed || job.Status != JobStatusQueued {
		t.Fatalf("unexpected job %#v", job)
	}
	list, err := service.ListJobs(ctx, JobFilters{Kind: JobKindPollFeed})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != "job_1" {
		t.Fatalf("unexpected jobs %#v", list.Items)
	}
}

func TestMemoryRepositoryRejectsDuplicateFeedItem(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
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
