package feeds

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/c86j224s/liquid2/internal/app"
	"golang.org/x/net/html"
)

var htmlContentTags = map[string]struct{}{
	"a": {}, "article": {}, "blockquote": {}, "br": {}, "code": {},
	"div": {}, "em": {}, "h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {},
	"h6": {}, "img": {}, "li": {}, "ol": {}, "p": {}, "pre": {},
	"section": {}, "span": {}, "strong": {}, "table": {}, "tbody": {},
	"td": {}, "th": {}, "thead": {}, "tr": {}, "ul": {},
}

type Normalizer struct{}

func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

func (normalizer *Normalizer) Normalize(feedURL string, parsed ParsedFeed) []app.FeedImportItem {
	items := make([]app.FeedImportItem, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		normalized, ok := normalizer.normalizeItem(feedURL, item)
		if ok {
			items = append(items, normalized)
		}
	}
	return items
}

func (normalizer *Normalizer) normalizeItem(feedURL string, item ParsedItem) (app.FeedImportItem, bool) {
	itemURL, ok := normalizeItemURL(feedURL, item.URL, item.GUID)
	if !ok {
		return app.FeedImportItem{}, false
	}
	title := strings.TrimSpace(item.Title)
	if title == "" {
		title = titleFromURL(itemURL)
	}
	content := strings.TrimSpace(item.Content)
	return app.FeedImportItem{
		Title: title, URL: itemURL, CanonicalURL: itemURL, SourceURL: itemURL,
		GUID: strings.TrimSpace(item.GUID), ContentHash: contentHash(itemURL, title, item.GUID, content),
		PublishedAt: item.PublishedAt, Content: content, Format: contentFormat(content),
	}, true
}

func normalizeItemURL(feedURL string, rawURL string, guid string) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		rawURL = strings.TrimSpace(guid)
	}
	base, err := url.Parse(feedURL)
	if err != nil {
		return "", false
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	if !parsed.IsAbs() {
		parsed = base.ResolveReference(parsed)
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	if parsed.Hostname() == "" {
		return "", false
	}
	return parsed.String(), true
}

func contentHash(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte(strings.TrimSpace(part)))
		hash.Write([]byte{0})
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func contentFormat(content string) string {
	tokenizer := html.NewTokenizer(strings.NewReader(content))
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			return app.ContentFormatText
		case html.StartTagToken, html.SelfClosingTagToken:
			name, _ := tokenizer.TagName()
			if _, ok := htmlContentTags[strings.ToLower(string(name))]; ok {
				return app.ContentFormatHTML
			}
		}
	}
}

func titleFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return "Untitled feed item"
	}
	path := strings.Trim(strings.TrimSpace(parsed.Path), "/")
	if path == "" {
		return parsed.Hostname()
	}
	parts := strings.Split(path, "/")
	title, err := url.PathUnescape(parts[len(parts)-1])
	if err != nil {
		title = parts[len(parts)-1]
	}
	title = strings.TrimSpace(strings.ReplaceAll(title, "-", " "))
	if title == "" {
		return parsed.Hostname()
	}
	if _, err := strconv.Atoi(title); err == nil {
		return parsed.Hostname()
	}
	return title
}
