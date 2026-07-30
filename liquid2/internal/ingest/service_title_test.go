package ingest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
)

func TestBookmarkPrefersUserTitleWithoutFetchingRemoteTitle(t *testing.T) {
	documents := app.NewService()
	t.Cleanup(func() { _ = documents.Close() })
	titleFetcher := &recordingTitleFetcher{title: "Remote title"}
	service := NewService(
		documents,
		WithGuard(NewURLGuard(WithAllowedHostForTest("example.com"))),
		WithTitleFetcher(titleFetcher),
	)

	detail, err := service.Bookmark(context.Background(), BookmarkInput{
		URL:   "https://example.com/articles/hello-world.html",
		Title: " User title ",
	})
	if err != nil {
		t.Fatalf("bookmark: %v", err)
	}
	if detail.Document.Title != "User title" {
		t.Fatalf("expected user title, got %q", detail.Document.Title)
	}
	if titleFetcher.calls != 0 {
		t.Fatalf("expected no remote title fetch, got %d calls", titleFetcher.calls)
	}
}

func TestBookmarkUsesRemoteTitleWhenInputTitleMissing(t *testing.T) {
	documents := app.NewService()
	t.Cleanup(func() { _ = documents.Close() })
	titleFetcher := &recordingTitleFetcher{title: "Remote page title"}
	service := NewService(
		documents,
		WithGuard(NewURLGuard(WithAllowedHostForTest("example.com"))),
		WithTitleFetcher(titleFetcher),
	)

	detail, err := service.Bookmark(context.Background(), BookmarkInput{
		URL: "https://example.com/articles/hello-world.html?utm=ignored",
	})
	if err != nil {
		t.Fatalf("bookmark: %v", err)
	}
	if detail.Document.Title != "Remote page title" {
		t.Fatalf("expected remote title, got %q", detail.Document.Title)
	}
	if titleFetcher.rawURL != "https://example.com/articles/hello-world.html?utm=ignored" {
		t.Fatalf("expected normalized URL, got %q", titleFetcher.rawURL)
	}
}

func TestBookmarkUsesURLTitleFallback(t *testing.T) {
	documents := app.NewService()
	t.Cleanup(func() { _ = documents.Close() })
	titleFetcher := &recordingTitleFetcher{err: errors.New("network failed")}
	service := NewService(
		documents,
		WithGuard(NewURLGuard(WithAllowedHostForTest("example.com"))),
		WithTitleFetcher(titleFetcher),
	)

	detail, err := service.Bookmark(context.Background(), BookmarkInput{
		URL: "https://example.com/articles/hello-world.html?utm=ignored",
	})
	if err != nil {
		t.Fatalf("bookmark: %v", err)
	}
	if detail.Document.Title != "hello world" {
		t.Fatalf("expected URL-derived title, got %q", detail.Document.Title)
	}
}

func TestBookmarkUsesURLTitleFallbackWhenRemoteTitleIsTooLong(t *testing.T) {
	documents := app.NewService()
	t.Cleanup(func() { _ = documents.Close() })
	service := NewService(
		documents,
		WithGuard(NewURLGuard(WithAllowedHostForTest("example.com"))),
		WithTitleFetcher(&recordingTitleFetcher{title: strings.Repeat("가", maxFetchedTitleRunes+1)}),
	)

	detail, err := service.Bookmark(context.Background(), BookmarkInput{
		URL: "https://example.com/articles/hello-world.html",
	})
	if err != nil {
		t.Fatalf("bookmark: %v", err)
	}
	if detail.Document.Title != "hello world" {
		t.Fatalf("expected URL-derived title, got %q", detail.Document.Title)
	}
}

func TestScrapeUsesURLTitleFallbackWhenPageTitleMissing(t *testing.T) {
	documents := app.NewService()
	t.Cleanup(func() { _ = documents.Close() })
	service := NewService(
		documents,
		WithFetcher(blankTitleFetcher{}),
	)

	detail, err := service.Scrape(context.Background(), ScrapeInput{
		URL: "https://example.com/articles/hello-world.html",
	})
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	if detail.Document.Title != "hello world" {
		t.Fatalf("expected URL-derived title, got %q", detail.Document.Title)
	}
}

type blankTitleFetcher struct{}

func (blankTitleFetcher) Fetch(_ context.Context, rawURL string) (FetchedPage, error) {
	return FetchedPage{
		URL: rawURL, Content: "Readable body", Format: FormatText,
	}, nil
}

type recordingTitleFetcher struct {
	calls  int
	rawURL string
	title  string
	err    error
}

func (fetcher *recordingTitleFetcher) FetchTitle(_ context.Context, rawURL string) (string, error) {
	fetcher.calls++
	fetcher.rawURL = rawURL
	return fetcher.title, fetcher.err
}
