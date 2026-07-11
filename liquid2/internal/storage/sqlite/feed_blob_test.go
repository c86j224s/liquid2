package sqlite

import (
	"bytes"
	"testing"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func TestBlobSizeLimit(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()
	createTestDocument(t, ctx, q, "doc_1")

	_, err := q.CreateBlob(ctx, sqlitedb.CreateBlobParams{
		ID: "blob_1", DocumentID: "doc_1", Filename: "large.bin",
		MimeType: "application/octet-stream", Size: 1048577,
		Sha256: validHash(), Data: bytes.Repeat([]byte("x"), 1), CreatedAt: 1000,
	})
	if err == nil {
		t.Fatal("expected blob size constraint error")
	}

	_, err = q.CreateBlob(ctx, sqlitedb.CreateBlobParams{
		ID: "blob_2", DocumentID: "doc_1", Filename: "mismatch.bin",
		MimeType: "application/octet-stream", Size: 1,
		Sha256: validHash(), Data: []byte("xx"), CreatedAt: 1000,
	})
	if err == nil {
		t.Fatal("expected blob size/data mismatch error")
	}

	_, err = q.CreateBlob(ctx, sqlitedb.CreateBlobParams{
		ID: "blob_3", DocumentID: "doc_1", Filename: "large-data.bin",
		MimeType: "application/octet-stream", Size: 1048577,
		Sha256: validHash(), Data: bytes.Repeat([]byte("x"), 1048577), CreatedAt: 1000,
	})
	if err == nil {
		t.Fatal("expected blob data size constraint error")
	}
}

func TestFeedItemDeduplication(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()
	createTestDocument(t, ctx, q, "doc_1")
	createTestDocument(t, ctx, q, "doc_2")
	createTestDocument(t, ctx, q, "doc_3")

	feed, err := q.CreateFeed(ctx, sqlitedb.CreateFeedParams{
		ID: "feed_1", Url: "https://example.com/feed.xml", Enabled: 1,
		CreatedAt: 1000, UpdatedAt: 1000,
	})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}

	item := sqlitedb.CreateFeedItemParams{
		ID: "item_1", FeedID: feed.ID, DocumentID: "doc_1", Guid: nullString("g1"),
		Url: "https://example.com/a", CanonicalUrl: nullString("https://example.com/a"),
		ContentHash: nullString("hash-a"), CreatedAt: 1000,
	}
	if _, err := q.CreateFeedItem(ctx, item); err != nil {
		t.Fatalf("create feed item: %v", err)
	}
	item.ID = "item_2"
	item.DocumentID = "doc_2"
	if _, err := q.CreateFeedItem(ctx, item); err == nil {
		t.Fatal("expected duplicate feed item error")
	}

	urlOnly := sqlitedb.CreateFeedItemParams{
		ID: "item_3", FeedID: feed.ID, DocumentID: "doc_3",
		Url: "https://example.com/raw-only", CreatedAt: 1000,
	}
	if _, err := q.CreateFeedItem(ctx, urlOnly); err != nil {
		t.Fatalf("create URL-only feed item: %v", err)
	}
	urlOnly.ID = "item_4"
	urlOnly.DocumentID = "doc_2"
	if _, err := q.CreateFeedItem(ctx, urlOnly); err == nil {
		t.Fatal("expected duplicate feed item URL error")
	}
}

func TestFeedURLUniqueness(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()

	params := sqlitedb.CreateFeedParams{
		ID: "feed_1", Url: "https://example.com/feed.xml", Enabled: 1,
		CreatedAt: 1000, UpdatedAt: 1000,
	}
	if _, err := q.CreateFeed(ctx, params); err != nil {
		t.Fatalf("create feed: %v", err)
	}
	params.ID = "feed_2"
	if _, err := q.CreateFeed(ctx, params); err == nil {
		t.Fatal("expected duplicate feed URL error")
	}
}
