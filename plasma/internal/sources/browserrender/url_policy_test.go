package browserrender

import (
	"context"
	"errors"
	"testing"
)

func TestAllowURLRejectsBlockedAddresses(t *testing.T) {
	renderer := NewRenderer(Options{})
	for _, rawURL := range []string{
		"http://127.0.0.1/",
		"http://10.0.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"file:///etc/passwd",
	} {
		t.Run(rawURL, func(t *testing.T) {
			err := renderer.allowURL(context.Background(), rawURL)
			if !errors.Is(err, ErrBlockedURL) {
				t.Fatalf("expected ErrBlockedURL, got %v", err)
			}
		})
	}
}
