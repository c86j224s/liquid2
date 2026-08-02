package browserrender

import (
	"strings"
	"testing"
	"time"
)

func TestReadableDocumentPrefersMainAndDropsScripts(t *testing.T) {
	body := strings.Repeat("<p>Rendered article paragraph with enough meaningful source detail.</p>", 20)
	result, err := readableDocument(`<html><head><title>Doc Title</title></head><body><nav>Navigation</nav><main><h1>Main Title</h1><script>secret()</script>`+body+`</main></body></html>`, "https://example.com/doc", time.Date(2026, 8, 1, 2, 3, 4, 0, time.UTC))
	if err != nil {
		t.Fatalf("readableDocument returned error: %v", err)
	}
	content := string(result.Content)
	if result.Title != "Main Title" || result.MediaType != MediaTypeHTML || result.FinalURL != "https://example.com/doc" {
		t.Fatalf("unexpected render result metadata: %#v", result)
	}
	if strings.Contains(content, "secret()") || strings.Contains(content, "Navigation") {
		t.Fatalf("rendered content should drop scripts and prefer main: %s", content)
	}
	if result.TextLength < minReadableTextLength {
		t.Fatalf("expected readable text length, got %d", result.TextLength)
	}
}

func TestReadableDocumentRejectsThinRenderedBody(t *testing.T) {
	_, err := readableDocument(`<html><body><main><h1>Only title</h1></main></body></html>`, "https://example.com/doc", time.Now())
	if err != ErrNoReadableBody {
		t.Fatalf("expected ErrNoReadableBody, got %v", err)
	}
}
