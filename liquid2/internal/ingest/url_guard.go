package ingest

import (
	"context"
	"net"
	"net/url"
	"strings"
)

type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type URLGuard struct {
	resolver     Resolver
	allowedHosts map[string]struct{}
}

type URLGuardOption func(*URLGuard)

func NewURLGuard(options ...URLGuardOption) URLGuard {
	guard := URLGuard{
		resolver:     net.DefaultResolver,
		allowedHosts: map[string]struct{}{},
	}
	for _, option := range options {
		option(&guard)
	}
	return guard
}

func WithResolver(resolver Resolver) URLGuardOption {
	return func(guard *URLGuard) {
		if resolver != nil {
			guard.resolver = resolver
		}
	}
}

func WithAllowedHostForTest(host string) URLGuardOption {
	return func(guard *URLGuard) {
		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" {
			guard.allowedHosts[host] = struct{}{}
		}
	}
}

func (guard URLGuard) Normalize(ctx context.Context, raw string) (string, error) {
	parsed, err := guard.Validate(ctx, raw)
	if err != nil {
		return "", err
	}
	return parsed.String(), nil
}

func (guard URLGuard) Validate(ctx context.Context, raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, unsafeURL("url is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, unsafeURL("url is malformed", err)
	}
	if parsed.Host == "" {
		return nil, unsafeURL("url is malformed")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, unsafeURL("url scheme must be http or https")
	}
	if parsed.User != nil {
		return nil, unsafeURL("url userinfo is not allowed")
	}
	parsed.Fragment = ""
	host := strings.ToLower(parsed.Hostname())
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil, unsafeURL("url host is not allowed")
	}
	if err := guard.validateHost(ctx, host); err != nil {
		return nil, err
	}
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed, nil
}

func (guard URLGuard) resolveAllowed(ctx context.Context, host string) ([]net.IP, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	if _, ok := guard.allowedHosts[host]; ok {
		return []net.IP{net.ParseIP(host)}, nil
	}
	if ip := net.ParseIP(host); ip != nil {
		if unsafeIP(ip) {
			return nil, unsafeURL("url host resolves to a private or local address")
		}
		return []net.IP{ip}, nil
	}
	addrs, err := guard.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, unsafeURL("url host could not be resolved", err)
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		if unsafeIP(addr.IP) {
			return nil, unsafeURL("url host resolves to a private or local address")
		}
		ips = append(ips, addr.IP)
	}
	if len(ips) == 0 {
		return nil, unsafeURL("url host has no addresses")
	}
	return ips, nil
}

func (guard URLGuard) validateHost(ctx context.Context, host string) error {
	_, err := guard.resolveAllowed(ctx, host)
	return err
}

func unsafeIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return !ip.IsGlobalUnicast() ||
		ip.IsPrivate() ||
		ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}
