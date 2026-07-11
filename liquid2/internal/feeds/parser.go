package feeds

import (
	"bytes"
	"context"
	"strings"

	"github.com/mmcdole/gofeed"
)

type GofeedParser struct {
	parser *gofeed.Parser
}

func NewGofeedParser() *GofeedParser {
	return &GofeedParser{parser: gofeed.NewParser()}
}

func (parser *GofeedParser) Parse(ctx context.Context, fetched FetchedFeed) (ParsedFeed, error) {
	if err := ctx.Err(); err != nil {
		return ParsedFeed{}, err
	}
	feed, err := parser.parser.Parse(bytes.NewReader(fetched.Data))
	if err != nil {
		return ParsedFeed{}, parseFailed("parse feed", err)
	}
	parsed := ParsedFeed{Title: strings.TrimSpace(feed.Title)}
	for _, item := range feed.Items {
		if item == nil {
			continue
		}
		parsed.Items = append(parsed.Items, parsedItem(item))
	}
	return parsed, nil
}

func parsedItem(item *gofeed.Item) ParsedItem {
	publishedAt := item.PublishedParsed
	if publishedAt == nil {
		publishedAt = item.UpdatedParsed
	}
	content := strings.TrimSpace(item.Content)
	if content == "" {
		content = strings.TrimSpace(item.Description)
	}
	var publishedMillis *int64
	if publishedAt != nil {
		value := publishedAt.UnixMilli()
		publishedMillis = &value
	}
	itemURL := strings.TrimSpace(item.Link)
	if itemURL == "" && len(item.Links) > 0 {
		itemURL = strings.TrimSpace(item.Links[0])
	}
	return ParsedItem{
		Title: strings.TrimSpace(item.Title), URL: itemURL,
		GUID: strings.TrimSpace(item.GUID), Content: content,
		PublishedAt: publishedMillis,
	}
}
