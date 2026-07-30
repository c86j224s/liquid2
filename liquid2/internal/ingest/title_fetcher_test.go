package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestHTTPTitleFetcherDefaultsToFiveSecondTimeoutAnd256KiBLimit(t *testing.T) {
	fetcher := NewHTTPTitleFetcher()
	if fetcher.client.Timeout != 5*time.Second {
		t.Fatalf("timeout = %v", fetcher.client.Timeout)
	}
	if fetcher.maxBytes != MaxTitleFetchBytes {
		t.Fatalf("max bytes = %d", fetcher.maxBytes)
	}
}

func TestHTTPTitleFetcherPrefersOGTitle(t *testing.T) {
	fetcher, server := newTestTitleFetcher(t, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write([]byte(`<html><head><title>Element title</title><meta property="og:title" content=" OG title &amp; more "></head></html>`))
	}))
	defer server.Close()

	title, err := fetcher.FetchTitle(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetch title: %v", err)
	}
	if title != "OG title & more" {
		t.Fatalf("title = %q", title)
	}
}

func TestHTTPTitleFetcherUsesTitleElement(t *testing.T) {
	fetcher, server := newTestTitleFetcher(t, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`<html><head><title> Element title </title></head></html>`))
	}))
	defer server.Close()

	title, err := fetcher.FetchTitle(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetch title: %v", err)
	}
	if title != "Element title" {
		t.Fatalf("title = %q", title)
	}
}

func TestHTTPTitleFetcherReturnsNoTitleForNonHTML(t *testing.T) {
	fetcher, server := newTestTitleFetcher(t, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"title":"Ignored"}`))
	}))
	defer server.Close()

	title, err := fetcher.FetchTitle(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetch title: %v", err)
	}
	if title != "" {
		t.Fatalf("title = %q", title)
	}
}

func TestHTTPTitleFetcherUsesTitleWithinLimitedRead(t *testing.T) {
	fetcher, server := newTestTitleFetcher(t, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		_, _ = writer.Write([]byte(`<html><head><title>Early title</title></head><body>` +
			strings.Repeat("a", MaxTitleFetchBytes) + `</body></html>`))
	}))
	defer server.Close()

	title, err := fetcher.FetchTitle(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetch title: %v", err)
	}
	if title != "Early title" {
		t.Fatalf("title = %q", title)
	}
}

func newTestTitleFetcher(t *testing.T, handler http.Handler) (*HTTPTitleFetcher, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	parsedServerURL, err := url.Parse(server.URL)
	if err != nil {
		server.Close()
		t.Fatalf("parse server URL: %v", err)
	}
	fetcher := NewHTTPTitleFetcher(
		WithTitleURLGuard(NewURLGuard(WithAllowedHostForTest(parsedServerURL.Hostname()))),
	)
	return fetcher, server
}
