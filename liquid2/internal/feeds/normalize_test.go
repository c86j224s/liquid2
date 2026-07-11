package feeds

import (
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
)

func TestNormalizerResolvesItemURLs(t *testing.T) {
	normalizer := NewNormalizer()
	items := normalizer.Normalize("https://example.com/feed.xml", ParsedFeed{Items: []ParsedItem{{
		Title: "First", URL: "/posts/first?utm=1#section", GUID: "guid-1", Content: "<p>Hello</p>",
	}}})
	if len(items) != 1 {
		t.Fatalf("expected one normalized item, got %#v", items)
	}
	item := items[0]
	if item.URL != "https://example.com/posts/first?utm=1" || item.SourceURL != item.URL {
		t.Fatalf("unexpected item urls %#v", item)
	}
	if item.Format != app.ContentFormatHTML || item.ContentHash == "" {
		t.Fatalf("unexpected content metadata %#v", item)
	}
}

func TestNormalizerSkipsItemsWithoutHTTPURL(t *testing.T) {
	normalizer := NewNormalizer()
	items := normalizer.Normalize("https://example.com/feed.xml", ParsedFeed{Items: []ParsedItem{{
		Title: "Bad", URL: "mailto:test@example.com", GUID: "guid-1",
	}}})
	if len(items) != 0 {
		t.Fatalf("expected item to be skipped, got %#v", items)
	}
}

func TestContentFormatDoesNotTreatComparisonsAsHTML(t *testing.T) {
	if got := contentFormat("price < 100 > 50"); got != app.ContentFormatText {
		t.Fatalf("expected text format, got %q", got)
	}
	if got := contentFormat("<p>Hello</p>"); got != app.ContentFormatHTML {
		t.Fatalf("expected html format, got %q", got)
	}
}
