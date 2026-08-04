package sourceretrieval

import (
	"net/http"
	"net/netip"
	"net/url"
	"testing"
)

func TestBlockedIPPolicy(t *testing.T) {
	for _, value := range []string{"127.0.0.1", "10.0.0.1", "169.254.1.1", "100.64.0.1", "::1"} {
		if !blockedIP(netip.MustParseAddr(value)) {
			t.Errorf("expected %s to be blocked", value)
		}
	}
	if blockedIP(netip.MustParseAddr("203.0.113.1")) {
		t.Fatal("public test address must not be classified as a blocked network")
	}
}

func TestRedirectPolicyRejectsCredentialsAndExcessHops(t *testing.T) {
	request := &http.Request{URL: &url.URL{Scheme: "https", Host: "example.com", User: url.User("user")}}
	if err := validateRedirect(request, nil); err == nil {
		t.Fatal("credential-bearing redirect must be rejected")
	}
	request.URL.User = nil
	if err := validateRedirect(request, make([]*http.Request, maxRedirects)); err == nil {
		t.Fatal("redirect limit must be enforced")
	}
}
