package ingest

import (
	"errors"
	"strings"
	"testing"
)

func TestExtractHTMLReadableMarkdown(t *testing.T) {
	body := []byte(`
		<html>
			<head><title>Readable Title</title></head>
			<body>
				<nav>ignore navigation</nav>
				<article>
					<h1>Article Heading</h1>
					<p>` + strings.Repeat("Readable article body with enough content. ", 20) + `</p>
					<p>See <a href="/reference">the reference</a>.</p>
				</article>
			</body>
		</html>
	`)

	content, err := ExtractWithURL("https://example.test/articles/one", "text/html; charset=utf-8", body)
	if err != nil {
		t.Fatalf("extract html: %v", err)
	}
	if content.Title != "Readable Title" {
		t.Fatalf("title = %q", content.Title)
	}
	if content.Format != FormatMarkdown {
		t.Fatalf("format = %q", content.Format)
	}
	if !strings.Contains(content.Content, "# Article Heading") {
		t.Fatalf("content = %q", content.Content)
	}
	if !strings.Contains(content.Content, "the reference") {
		t.Fatalf("content missing link text: %q", content.Content)
	}
}

func TestExtractHTMLSemanticMarkdownFallback(t *testing.T) {
	body := []byte(`
		<html>
			<head><title>Fallback Title</title></head>
			<body>
				<nav>navigation should not be stored</nav>
				<article>
					<h1>Fallback Heading</h1>
					<p>` + strings.Repeat("Fallback article paragraph. ", 12) + `</p>
					<p>See <a href="/docs">the docs</a>.</p>
				</article>
			</body>
		</html>
	`)

	content, err := ExtractWithURL("://bad-url", "text/html", body)
	if err != nil {
		t.Fatalf("extract html: %v", err)
	}
	if content.Title != "Fallback Title" {
		t.Fatalf("title = %q", content.Title)
	}
	if content.Format != FormatMarkdown {
		t.Fatalf("format = %q", content.Format)
	}
	if !strings.Contains(content.Content, "# Fallback Heading") {
		t.Fatalf("content = %q", content.Content)
	}
	if strings.Contains(content.Content, "navigation should not be stored") {
		t.Fatalf("content includes navigation: %q", content.Content)
	}
}

func TestExtractHTMLFallsBackToText(t *testing.T) {
	body := []byte(`
		<html>
			<head><title> Example Title </title></head>
			<body><script>hide()</script></body>
		</html>
	`)

	content, err := ExtractWithURL("://bad-url", "text/html", body)
	if err != nil {
		t.Fatalf("extract html: %v", err)
	}
	if content.Title != "Example Title" {
		t.Fatalf("title = %q", content.Title)
	}
	if content.Format != FormatText {
		t.Fatalf("format = %q", content.Format)
	}
	if content.Content != "Example Title" {
		t.Fatalf("content = %q", content.Content)
	}
}

func TestExtractMarkdownPassthrough(t *testing.T) {
	content, err := Extract("text/markdown", []byte("  # Title\n\nBody  "))
	if err != nil {
		t.Fatalf("extract markdown: %v", err)
	}
	if content.Format != FormatMarkdown {
		t.Fatalf("format = %q", content.Format)
	}
	if content.Content != "# Title\n\nBody" {
		t.Fatalf("content = %q", content.Content)
	}
}

func TestExtractTextPassthrough(t *testing.T) {
	content, err := Extract("", []byte("  plain text  "))
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}
	if content.Format != FormatText {
		t.Fatalf("format = %q", content.Format)
	}
	if content.Content != "plain text" {
		t.Fatalf("content = %q", content.Content)
	}
}

func TestExtractEmptyBodiesFail(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        []byte
	}{
		{name: "html", contentType: "text/html", body: []byte("<html></html>")},
		{name: "markdown", contentType: "text/markdown", body: []byte("  ")},
		{name: "text", contentType: "text/plain", body: []byte("  ")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ExtractWithURL("https://example.test/article", test.contentType, test.body)
			if !errors.Is(err, ErrFetchFailed) {
				t.Fatalf("expected fetch failure, got %v", err)
			}
		})
	}
}

func TestExtractUnsupportedMedia(t *testing.T) {
	_, err := Extract("application/json", []byte(`{"ok":true}`))
	if !errors.Is(err, ErrUnsupportedMedia) {
		t.Fatalf("expected unsupported media, got %v", err)
	}
}
