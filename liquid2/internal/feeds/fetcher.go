package feeds

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/c86j224s/liquid2/internal/ingest"
)

const MaxFetchBytes = 1024 * 1024

type HTTPFetcher struct {
	guard    ingest.URLGuard
	client   *http.Client
	maxBytes int64
}

type HTTPFetcherOption func(*HTTPFetcher)

func NewHTTPFetcher(options ...HTTPFetcherOption) *HTTPFetcher {
	fetcher := &HTTPFetcher{
		guard:    ingest.NewURLGuard(),
		maxBytes: MaxFetchBytes,
	}
	for _, option := range options {
		option(fetcher)
	}
	if fetcher.client == nil {
		fetcher.client = ingest.NewSafeHTTPClient(fetcher.guard)
	}
	return fetcher
}

func WithHTTPClient(client *http.Client) HTTPFetcherOption {
	return func(fetcher *HTTPFetcher) {
		if client != nil {
			fetcher.client = client
		}
	}
}

func WithMaxFetchBytes(maxBytes int64) HTTPFetcherOption {
	return func(fetcher *HTTPFetcher) {
		if maxBytes > 0 {
			fetcher.maxBytes = maxBytes
		}
	}
}

func WithURLGuard(guard ingest.URLGuard) HTTPFetcherOption {
	return func(fetcher *HTTPFetcher) {
		fetcher.guard = guard
	}
}

func (fetcher *HTTPFetcher) Fetch(ctx context.Context, rawURL string) (FetchedFeed, error) {
	normalized, err := fetcher.guard.Normalize(ctx, rawURL)
	if err != nil {
		return FetchedFeed{}, fetchFailed("url rejected", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return FetchedFeed{}, fetchFailed("request creation failed", err)
	}
	response, err := fetcher.client.Do(request)
	if err != nil {
		return FetchedFeed{}, fetchFailed("request failed", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return FetchedFeed{}, fetchFailed("unexpected status " + strconv.Itoa(response.StatusCode))
	}
	data, err := readLimited(response.Body, fetcher.maxBytes)
	if err != nil {
		return FetchedFeed{}, err
	}
	return FetchedFeed{URL: response.Request.URL.String(), Data: data}, nil
}

func readLimited(reader io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, fetchFailed("read response", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fetchFailed("response exceeds " + strconv.FormatInt(maxBytes, 10) + " bytes")
	}
	return data, nil
}
