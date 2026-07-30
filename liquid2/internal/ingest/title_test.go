package ingest

import "testing"

func TestTitleFromURLDerivesReadablePathTitle(t *testing.T) {
	got := titleFromURL("https://www.example.com/articles/hello-world.html?utm=ignored")
	if got != "hello world" {
		t.Fatalf("expected readable path title, got %q", got)
	}
}

func TestTitleFromURLFallsBackToHost(t *testing.T) {
	tests := map[string]string{
		"https://example.com/":            "example.com",
		"https://www.example.com/a":       "example.com",
		"https://example.com/posts/12345": "example.com",
		"https://example.com/index.html":  "example.com",
		"not a url":                       "Untitled document",
	}
	for rawURL, want := range tests {
		if got := titleFromURL(rawURL); got != want {
			t.Fatalf("titleFromURL(%q) = %q, want %q", rawURL, got, want)
		}
	}
}
