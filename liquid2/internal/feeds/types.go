package feeds

import (
	"context"

	"github.com/c86j224s/liquid2/internal/app"
)

type Fetcher interface {
	Fetch(ctx context.Context, rawURL string) (FetchedFeed, error)
}

type Parser interface {
	Parse(ctx context.Context, fetched FetchedFeed) (ParsedFeed, error)
}

type FetchedFeed struct {
	URL  string
	Data []byte
}

type ParsedFeed struct {
	Title string
	Items []ParsedItem
}

type ParsedItem struct {
	Title       string
	URL         string
	GUID        string
	Content     string
	PublishedAt *int64
}

type fetchedContext struct {
	Feed    app.Feed
	Fetched FetchedFeed
}

type parsedContext struct {
	Feed    app.Feed
	Fetched FetchedFeed
	Parsed  ParsedFeed
}

type normalizedContext struct {
	Feed  app.Feed
	Items []app.FeedImportItem
}
