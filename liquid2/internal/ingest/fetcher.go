package ingest

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

const MaxFetchBytes = 1024 * 1024

type Fetcher interface {
	Fetch(ctx context.Context, rawURL string) (FetchedPage, error)
}

type FetchedPage struct {
	URL     string
	Title   string
	Content string
	Format  string
}

type HTTPFetcher struct {
	guard     URLGuard
	client    *http.Client
	extractor Extractor
	maxBytes  int64
}

type HTTPFetcherOption func(*HTTPFetcher)

func NewHTTPFetcher(options ...HTTPFetcherOption) *HTTPFetcher {
	fetcher := &HTTPFetcher{
		guard:     NewURLGuard(),
		extractor: DefaultExtractor{},
		maxBytes:  MaxFetchBytes,
	}
	for _, option := range options {
		option(fetcher)
	}
	if fetcher.client == nil {
		fetcher.client = safeHTTPClient(fetcher.guard)
	}
	if fetcher.extractor == nil {
		fetcher.extractor = DefaultExtractor{}
	}
	return fetcher
}

func NewSafeHTTPClient(guard URLGuard) *http.Client {
	return safeHTTPClient(guard)
}

func WithURLGuard(guard URLGuard) HTTPFetcherOption {
	return func(fetcher *HTTPFetcher) {
		fetcher.guard = guard
	}
}

func WithMaxFetchBytes(maxBytes int64) HTTPFetcherOption {
	return func(fetcher *HTTPFetcher) {
		if maxBytes > 0 {
			fetcher.maxBytes = maxBytes
		}
	}
}

func WithExtractor(extractor Extractor) HTTPFetcherOption {
	return func(fetcher *HTTPFetcher) {
		if extractor != nil {
			fetcher.extractor = extractor
		}
	}
}

func (fetcher *HTTPFetcher) Fetch(ctx context.Context, rawURL string) (FetchedPage, error) {
	normalized, err := fetcher.guard.Normalize(ctx, rawURL)
	if err != nil {
		return FetchedPage{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return FetchedPage{}, unsafeURL("url is malformed", err)
	}
	response, err := fetcher.client.Do(request)
	if err != nil {
		return FetchedPage{}, fetchFailed("request failed", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return FetchedPage{}, fetchFailed("unexpected status " + strconv.Itoa(response.StatusCode))
	}
	data, err := readLimited(response.Body, fetcher.maxBytes)
	if err != nil {
		return FetchedPage{}, err
	}
	finalURL := response.Request.URL.String()
	extracted, err := fetcher.extractor.Extract(finalURL, response.Header.Get("Content-Type"), data)
	if err != nil {
		return FetchedPage{}, err
	}
	return FetchedPage{
		URL: finalURL, Title: extracted.Title,
		Content: extracted.Content, Format: extracted.Format,
	}, nil
}

func safeHTTPClient(guard URLGuard) *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			Proxy:       nil,
			DialContext: safeDialContext(guard),
		},
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fetchFailed("too many redirects")
			}
			_, err := guard.Validate(request.Context(), request.URL.String())
			return err
		},
	}
}

func safeDialContext(guard URLGuard) func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fetchFailed("dial address is malformed", err)
		}
		ips, err := guard.resolveAllowed(ctx, host)
		if err != nil {
			return nil, err
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
}

func readLimited(reader io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, fetchFailed("read response", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, payloadTooLarge("response exceeds 1MB")
	}
	return data, nil
}
