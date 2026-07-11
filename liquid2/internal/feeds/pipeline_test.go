package feeds

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
)

type fakeFetcher struct {
	data []byte
}

func (fetcher fakeFetcher) Fetch(_ context.Context, rawURL string) (FetchedFeed, error) {
	return FetchedFeed{URL: rawURL, Data: fetcher.data}, nil
}

func TestPipelineImportsFeedItemsOnce(t *testing.T) {
	ctx := context.Background()
	service := app.NewService(app.WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	feed, err := service.CreateFeed(ctx, app.CreateFeedInput{URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	payload, err := EncodePollFeedPayload(feed.ID)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	pipeline := NewPipeline(service, fakeFetcher{data: []byte(sampleRSS())}, NewGofeedParser())
	job := jobs.Job{ID: "job_1", Kind: jobs.KindPollFeed, PayloadJSON: payload}
	if err := pipeline.Handle(ctx, job); err != nil {
		t.Fatalf("handle feed job: %v", err)
	}
	if err := pipeline.Handle(ctx, job); err != nil {
		t.Fatalf("repeat feed job: %v", err)
	}
	docs, err := service.ListDocuments(ctx, app.DocumentFilters{Kind: app.DocumentKindRSSItem})
	if err != nil {
		t.Fatalf("list docs: %v", err)
	}
	if len(docs.Items) != 1 {
		t.Fatalf("expected one imported document, got %#v", docs.Items)
	}
}

func TestPipelineImportsWithSQLiteRepositoryOnce(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	service := app.NewService(app.WithRepository(app.NewSQLiteRepository(store)))
	t.Cleanup(func() { _ = service.Close() })
	feed, err := service.CreateFeed(ctx, app.CreateFeedInput{URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	payload, err := EncodePollFeedPayload(feed.ID)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	pipeline := NewPipeline(service, fakeFetcher{data: []byte(sampleRSS())}, NewGofeedParser())
	job := jobs.Job{ID: "job_1", Kind: jobs.KindPollFeed, PayloadJSON: payload}
	for range 2 {
		if err := pipeline.Handle(ctx, job); err != nil {
			t.Fatalf("handle feed job: %v", err)
		}
	}
	docs, err := service.ListDocuments(ctx, app.DocumentFilters{Kind: app.DocumentKindRSSItem})
	if err != nil {
		t.Fatalf("list docs: %v", err)
	}
	if len(docs.Items) != 1 {
		t.Fatalf("expected one imported document, got %#v", docs.Items)
	}
}
