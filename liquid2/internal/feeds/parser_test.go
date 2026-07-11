package feeds

import (
	"context"
	"testing"
)

func TestGofeedParserParsesRSSItems(t *testing.T) {
	parser := NewGofeedParser()
	parsed, err := parser.Parse(context.Background(), FetchedFeed{Data: []byte(sampleRSS())})
	if err != nil {
		t.Fatalf("parse rss: %v", err)
	}
	if parsed.Title != "Example Feed" || len(parsed.Items) != 1 {
		t.Fatalf("unexpected parsed feed %#v", parsed)
	}
	item := parsed.Items[0]
	if item.Title != "First" || item.URL != "https://example.com/a" || item.GUID != "guid-1" {
		t.Fatalf("unexpected parsed item %#v", item)
	}
	if item.Content == "" || item.PublishedAt == nil {
		t.Fatalf("expected content and published time, got %#v", item)
	}
}

func sampleRSS() string {
	return `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Example Feed</title>
    <item>
      <title>First</title>
      <link>https://example.com/a</link>
      <guid>guid-1</guid>
      <description><![CDATA[<p>Hello</p>]]></description>
      <pubDate>Thu, 09 Oct 2025 17:53:20 +0900</pubDate>
    </item>
  </channel>
</rss>`
}
