package httptransport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/ingest"
)

func ingestionTestRouter(service *app.Service) http.Handler {
	return ingestionTestRouterWithTranslator(service, nil)
}

func ingestionTestRouterWithTranslator(service *app.Service, translator DocumentTranslator) http.Handler {
	guard := ingest.NewURLGuard(ingest.WithResolver(testResolver{
		"example.com": {net.ParseIP("93.184.216.34")},
	}))
	ingestion := ingest.NewService(service, ingest.WithGuard(guard), ingest.WithFetcher(fakeFetcher{}))
	if translator == nil {
		return NewRouter(service, WithIngestion(ingestion))
	}
	return NewRouter(service, WithIngestion(ingestion), WithDocumentTranslator(translator))
}

type fakeFetcher struct{}

func (fakeFetcher) Fetch(_ context.Context, rawURL string) (ingest.FetchedPage, error) {
	return ingest.FetchedPage{
		URL: rawURL, Title: "Fetched title", Content: "Readable body", Format: ingest.FormatText,
	}, nil
}

type failingFetcher struct{}

func (failingFetcher) Fetch(_ context.Context, _ string) (ingest.FetchedPage, error) {
	cause := errors.New("dial tcp https://example.com/a?token=secret: connection refused")
	return ingest.FetchedPage{}, fmt.Errorf("%w: request failed: %w", ingest.ErrFetchFailed, cause)
}

type testResolver map[string][]net.IP

func (resolver testResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips := resolver[host]
	if len(ips) == 0 {
		return nil, errors.New("not found")
	}
	addrs := make([]net.IPAddr, 0, len(ips))
	for _, ip := range ips {
		addrs = append(addrs, net.IPAddr{IP: ip})
	}
	return addrs, nil
}
