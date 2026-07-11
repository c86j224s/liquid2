package ingest

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestURLGuardRejectsPrivateTargets(t *testing.T) {
	guard := NewURLGuard(WithResolver(staticResolver{
		"internal.test": {net.ParseIP("192.168.1.10")},
	}))
	for _, raw := range []string{
		"http://127.0.0.1/page",
		"http://localhost/page",
		"http://169.254.169.254/latest/meta-data",
		"https://internal.test/page",
	} {
		if _, err := guard.Normalize(context.Background(), raw); !errors.Is(err, ErrUnsafeURL) {
			t.Fatalf("expected unsafe URL for %s, got %v", raw, err)
		}
	}
}

func TestURLGuardNormalizesPublicURL(t *testing.T) {
	guard := NewURLGuard(WithResolver(staticResolver{
		"example.com": {net.ParseIP("93.184.216.34")},
	}))

	normalized, err := guard.Normalize(context.Background(), "HTTPS://Example.com/a?x=1#section")
	if err != nil {
		t.Fatalf("normalize URL: %v", err)
	}
	if normalized != "https://example.com/a?x=1" {
		t.Fatalf("unexpected normalized URL %q", normalized)
	}
}

type staticResolver map[string][]net.IP

func (resolver staticResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
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
