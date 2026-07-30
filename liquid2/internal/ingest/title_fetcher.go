package ingest

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const MaxTitleFetchBytes = 256 * 1024

type TitleFetcher interface {
	FetchTitle(ctx context.Context, rawURL string) (string, error)
}

type HTTPTitleFetcher struct {
	guard    URLGuard
	client   *http.Client
	maxBytes int64
}

type HTTPTitleFetcherOption func(*HTTPTitleFetcher)

func NewHTTPTitleFetcher(options ...HTTPTitleFetcherOption) *HTTPTitleFetcher {
	fetcher := &HTTPTitleFetcher{
		guard:    NewURLGuard(),
		maxBytes: MaxTitleFetchBytes,
	}
	for _, option := range options {
		option(fetcher)
	}
	if fetcher.client == nil {
		fetcher.client = safeHTTPClient(fetcher.guard)
		fetcher.client.Timeout = 5 * time.Second
	}
	return fetcher
}

func WithTitleURLGuard(guard URLGuard) HTTPTitleFetcherOption {
	return func(fetcher *HTTPTitleFetcher) {
		fetcher.guard = guard
	}
}

func WithTitleMaxFetchBytes(maxBytes int64) HTTPTitleFetcherOption {
	return func(fetcher *HTTPTitleFetcher) {
		if maxBytes > 0 {
			fetcher.maxBytes = maxBytes
		}
	}
}

func (fetcher *HTTPTitleFetcher) FetchTitle(ctx context.Context, rawURL string) (string, error) {
	normalized, err := fetcher.guard.Normalize(ctx, rawURL)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return "", unsafeURL("url is malformed", err)
	}
	response, err := fetcher.client.Do(request)
	if err != nil {
		return "", fetchFailed("request failed", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", fetchFailed("unexpected status " + strconv.Itoa(response.StatusCode))
	}
	contentType := mediaType(response.Header.Get("Content-Type"))
	if contentType != "" && contentType != "text/html" && contentType != "application/xhtml+xml" {
		return "", nil
	}
	data, err := readTitleLimited(response.Body, fetcher.maxBytes)
	if err != nil {
		return "", err
	}
	return extractHTMLMetadataTitle(data), nil
}

func readTitleLimited(reader io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes))
	if err != nil {
		return nil, fetchFailed("read title response", err)
	}
	return data, nil
}

func extractHTMLMetadataTitle(data []byte) string {
	tokenizer := html.NewTokenizer(bytes.NewReader(data))
	var elementTitle string
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			return elementTitle
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			switch token.DataAtom.String() {
			case "meta":
				if title := metaTitle(token); title != "" {
					return title
				}
			case "title":
				if elementTitle == "" {
					elementTitle = readTitleElement(tokenizer)
				}
			}
		}
	}
}

func metaTitle(token html.Token) string {
	var property string
	var content string
	for _, attr := range token.Attr {
		switch strings.ToLower(strings.TrimSpace(attr.Key)) {
		case "property":
			property = strings.ToLower(strings.TrimSpace(attr.Val))
		case "content":
			content = attr.Val
		}
	}
	if property != "og:title" {
		return ""
	}
	return collapseText(content)
}

func readTitleElement(tokenizer *html.Tokenizer) string {
	var builder strings.Builder
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			return collapseText(builder.String())
		case html.TextToken:
			builder.Write(tokenizer.Text())
		case html.EndTagToken:
			token := tokenizer.Token()
			if token.DataAtom.String() == "title" {
				return collapseText(builder.String())
			}
		}
	}
}
