package ingest

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSafeHTTPClientRejectsUnsafeRedirect(t *testing.T) {
	client := safeHTTPClient(NewURLGuard())
	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/private", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	err = client.CheckRedirect(request, nil)
	if !errors.Is(err, ErrUnsafeURL) {
		t.Fatalf("expected unsafe redirect rejection, got %v", err)
	}
}

func TestSafeDialContextRechecksResolvedAddress(t *testing.T) {
	resolver := &rebindingResolver{}
	guard := NewURLGuard(WithResolver(resolver))
	ctx := context.Background()

	if _, err := guard.Normalize(ctx, "http://example.test/a"); err != nil {
		t.Fatalf("normalize initial public address: %v", err)
	}
	_, err := safeDialContext(guard)(ctx, "tcp", "example.test:80")
	if !errors.Is(err, ErrUnsafeURL) {
		t.Fatalf("expected unsafe dial-time address rejection, got %v", err)
	}
	if resolver.calls < 2 {
		t.Fatalf("expected dial path to resolve host again, got %d calls", resolver.calls)
	}
}

func TestHTTPFetcherPassesFinalURLToExtractor(t *testing.T) {
	extractor := &recordingExtractor{
		result: ExtractedContent{
			Title:   "Extracted",
			Content: "Body",
			Format:  FormatMarkdown,
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/start":
			http.Redirect(writer, request, "/final", http.StatusFound)
		case "/final":
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = writer.Write([]byte(`<article><a href="/relative">Relative</a></article>`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	parsedServerURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server url: %v", err)
	}
	fetcher := NewHTTPFetcher(
		WithURLGuard(NewURLGuard(WithAllowedHostForTest(parsedServerURL.Hostname()))),
		WithExtractor(extractor),
	)

	page, err := fetcher.Fetch(context.Background(), server.URL+"/start")
	if err != nil {
		t.Fatalf("fetch redirected page: %v", err)
	}
	if page.URL != server.URL+"/final" {
		t.Fatalf("page URL = %q", page.URL)
	}
	if extractor.pageURL != server.URL+"/final" {
		t.Fatalf("extractor URL = %q", extractor.pageURL)
	}
	if extractor.contentType != "text/html; charset=utf-8" {
		t.Fatalf("content type = %q", extractor.contentType)
	}
	if !strings.Contains(string(extractor.data), `href="/relative"`) {
		t.Fatalf("extractor data = %q", string(extractor.data))
	}
}

func TestHTTPFetcherDefaultsNilExtractor(t *testing.T) {
	fetcher := NewHTTPFetcher(WithExtractor(nil))
	if fetcher.extractor == nil {
		t.Fatal("expected default extractor")
	}
}

type rebindingResolver struct {
	calls int
}

func (resolver *rebindingResolver) LookupIPAddr(_ context.Context, _ string) ([]net.IPAddr, error) {
	resolver.calls++
	if resolver.calls == 1 {
		return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
	}
	return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
}

type recordingExtractor struct {
	pageURL     string
	contentType string
	data        []byte
	result      ExtractedContent
}

func (extractor *recordingExtractor) Extract(pageURL string, contentType string, data []byte) (ExtractedContent, error) {
	extractor.pageURL = pageURL
	extractor.contentType = contentType
	extractor.data = append([]byte(nil), data...)
	return extractor.result, nil
}
