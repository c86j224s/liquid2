package browserrender

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

func (renderer *Renderer) allowURL(ctx context.Context, rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed == nil || parsed.Scheme == "" {
		return fmt.Errorf("%w: invalid URL", ErrBlockedURL)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	case "about", "blob", "data":
		return nil
	default:
		return fmt.Errorf("%w: unsupported URL scheme", ErrBlockedURL)
	}
	if parsed.User != nil {
		return fmt.Errorf("%w: URL credentials are not allowed", ErrBlockedURL)
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return fmt.Errorf("%w: URL host is required", ErrBlockedURL)
	}
	resolveCtx, cancel := context.WithTimeout(ctx, renderer.resolveTime)
	defer cancel()
	ips, err := net.DefaultResolver.LookupNetIP(resolveCtx, "ip", host)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("%w: URL host lookup failed", ErrBlockedURL)
	}
	for _, ip := range ips {
		if blockedBrowserRenderIP(ip) {
			return fmt.Errorf("%w: URL resolves to blocked address", ErrBlockedURL)
		}
	}
	return nil
}

func blockedBrowserRenderIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() {
		return true
	}
	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return true
	}
	if netip.MustParsePrefix("100.64.0.0/10").Contains(ip) {
		return true
	}
	return false
}
