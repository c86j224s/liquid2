package sourceretrieval

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const (
	dialTimeout            = 15 * time.Second
	tlsHandshakeTimeout    = 15 * time.Second
	responseHeaderTimeout  = 45 * time.Second
	fetchTimeout           = 60 * time.Second
	maxResponseHeaderBytes = 64 << 10
	maxRedirects           = 5
)

// NewClient returns the source retrieval client used by URL, PDF, and media
// adapters. It rejects credential-bearing redirects and blocked networks.
func NewClient() *http.Client {
	dialer := &net.Dialer{Timeout: dialTimeout}
	return &http.Client{
		Timeout: fetchTimeout,
		Transport: &http.Transport{
			Proxy:                  nil,
			DialContext:            secureDialContext(dialer, net.DefaultResolver),
			TLSHandshakeTimeout:    tlsHandshakeTimeout,
			ResponseHeaderTimeout:  responseHeaderTimeout,
			MaxResponseHeaderBytes: maxResponseHeaderBytes,
		},
		CheckRedirect: validateRedirect,
	}
}

func validateRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	if req.URL == nil || (req.URL.Scheme != "http" && req.URL.Scheme != "https") {
		return fmt.Errorf("redirected to a non-http URL")
	}
	if req.URL.User != nil {
		return fmt.Errorf("redirected to a URL with credentials")
	}
	return nil
}

func secureDialContext(dialer *net.Dialer, resolver *net.Resolver) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid source URL address", producterror.ErrInvalidInput)
		}
		ips, err := resolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("%w: source URL host lookup failed: %v", producterror.ErrInvalidInput, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("%w: source URL host has no addresses", producterror.ErrInvalidInput)
		}
		for _, ip := range ips {
			if blockedIP(ip) {
				return nil, fmt.Errorf("%w: source URL resolves to blocked address %s", producterror.ErrInvalidInput, ip.String())
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
}

func blockedIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() {
		return true
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() ||
		netip.MustParsePrefix("100.64.0.0/10").Contains(ip)
}
