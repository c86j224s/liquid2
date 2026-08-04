package sourceretrieval

import "testing"

func TestNormalizeURLIdentity(t *testing.T) {
	got, err := Normalize("  HTTPS://EXAMPLE.COM/docs?q=1#section  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://example.com/docs?q=1" {
		t.Fatalf("unexpected normalized URL: %q", got)
	}
}

func TestNormalizeRejectsCredentialsAndNonHTTP(t *testing.T) {
	for _, raw := range []string{"https://user@example.com/docs", "file:///tmp/docs"} {
		if _, err := Normalize(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}
