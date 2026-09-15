package confluenceaccess

import "testing"

func TestAPITokenSiteURLBoundary(t *testing.T) {
	for _, value := range []string{"http://example.atlassian.net", "https://user:secret@example.atlassian.net", "https://example.com", "https://example.atlassian.net/wiki/spaces"} {
		if _, err := NormalizeConfluenceAPITokenSiteURL(value); err == nil {
			t.Fatalf("accepted invalid site %q", value)
		}
	}
	for _, value := range []string{"https://example.atlassian.net", "https://example.atlassian.net/wiki"} {
		if _, err := NormalizeConfluenceAPITokenSiteURL(value); err != nil {
			t.Fatalf("site=%q: %v", value, err)
		}
	}
	if _, err := NormalizeConfluenceAPITokenAPIBaseURLForSite("https://other.atlassian.net/wiki", "https://example.atlassian.net"); err == nil {
		t.Fatal("accepted mismatched host")
	}
}
