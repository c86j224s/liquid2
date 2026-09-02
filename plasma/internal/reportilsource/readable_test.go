package reportilsource

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractHTMLReturnsVisibleTextWithoutMarkupLocatorsOrHiddenContent(t *testing.T) {
	content := []byte(`<!doctype html>
<html>
<head><title>Hidden title</title><style>.secret { color: red }</style></head>
<body>
<script>window.secret = "script secret";</script>
<h1 data-path="/private/report">Visible heading</h1>
<p>First <strong>visible</strong> paragraph.</p>
<ul><li>Repeated fact</li><li>Repeated fact</li></ul>
<a href="https://example.com/private-locator">Public label</a>
<table><tr><th>Key</th><th>Value</th></tr><tr><td>A</td><td>B</td></tr></table>
<svg><text>Hidden SVG text</text></svg>
</body>
</html>`)

	readable, err := Extract(content, "text/html; charset=utf-8")
	if err != nil {
		t.Fatal(err)
	}
	if readable.Extraction != "html_visible_text" ||
		readable.ByteSize != 107 ||
		readable.SHA256 != "a2a12fbf6a639fc99c3f1b8d1352c7c0f178418831180c6bc1094e38f8dde25b" {
		t.Fatalf("HTML canonical readable receipt changed: %#v", readable)
	}
	if readable.DocumentTitle != "Hidden title" || readable.PrimaryHeading != "Visible heading" {
		t.Fatalf("HTML citation metadata = %#v", readable)
	}
	for _, want := range []string{"Visible heading", "First visible paragraph.", "Public label", "Key | Value", "A | B"} {
		if !strings.Contains(readable.Text, want) {
			t.Fatalf("visible text lacks %q: %q", want, readable.Text)
		}
	}
	if strings.Count(readable.Text, "Repeated fact") != 2 {
		t.Fatalf("repeated source prose was deduplicated: %q", readable.Text)
	}
	for _, forbidden := range []string{"Hidden title", "script secret", "Hidden SVG text", "data-path", "/private/report", "private-locator", "<h1", "href="} {
		if strings.Contains(readable.Text, forbidden) {
			t.Fatalf("HTML readable text leaked %q: %q", forbidden, readable.Text)
		}
	}
	encoded, err := json.Marshal(readable)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["DocumentTitle"]; ok {
		t.Fatalf("HTML document title was serialized: %s", encoded)
	}
	if _, ok := fields["PrimaryHeading"]; ok {
		t.Fatalf("HTML primary heading was serialized: %s", encoded)
	}
	if strings.Contains(string(encoded), "Hidden title") {
		t.Fatalf("hidden HTML title was serialized: %s", encoded)
	}
}

func TestExtractRejectsUnsupportedEmptyOversizedAndUnreadableContent(t *testing.T) {
	cases := []struct {
		name      string
		content   []byte
		mediaType string
	}{
		{name: "unsupported", content: []byte("image"), mediaType: "image/png"},
		{name: "empty", content: []byte("  \n\t"), mediaType: "text/plain"},
		{name: "oversized", content: []byte(strings.Repeat("x", MaxReadableBytes+1)), mediaType: "text/plain"},
		{name: "invalid UTF-8", content: []byte{0xff, 0xfe}, mediaType: "text/plain"},
		{name: "garbled extracted replacement text", content: []byte("heading�body"), mediaType: "text/plain"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Extract(tc.content, tc.mediaType); err == nil {
				t.Fatal("unsafe source content was accepted")
			}
		})
	}
}

func TestExtractRedactsRestrictedProviderSpansAndPreservesSurroundingText(t *testing.T) {
	cases := []struct {
		name      string
		content   string
		mediaType string
		redacted  string
		secret    string
	}{
		{name: "credential", content: "HTTP authentication\nAuthorization: secret-value\nNext section", mediaType: "text/plain", redacted: redactedCredential, secret: "secret-value"},
		{name: "JSON credential", content: `{"api_key":"secret-value"}`, mediaType: "application/json", redacted: redactedCredential, secret: "secret-value"},
		{name: "private URL", content: "Query http://127.0.0.1/private for status", mediaType: "text/plain", redacted: redactedURL, secret: "http://127.0.0.1/private"},
		{name: "userinfo URL", content: "Example https://user:password@example.org/ for authentication", mediaType: "text/plain", redacted: redactedURL, secret: "user:password"},
		{name: "escaped private URL", content: `{"url":"http:\/\/172.16.0.2\/report"}`, mediaType: "application/json", redacted: redactedURL, secret: "172.16.0.2"},
		{name: "Tailnet URL", content: "Open https://node.ts.net/report, then continue", mediaType: "text/plain", redacted: redactedURL, secret: "node.ts.net"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			readable, err := Extract([]byte(tc.content), tc.mediaType)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(readable.Text, tc.redacted) || strings.Contains(readable.Text, tc.secret) {
				t.Fatalf("redacted readable = %q", readable.Text)
			}
			if err := ValidateProviderContent(readable.Text); err != nil {
				t.Fatalf("redacted readable remains unsafe: %v", err)
			}
		})
	}
}

func TestExtractRedactsNetworkDocumentationExamplesWithoutDroppingSource(t *testing.T) {
	content := []byte(`HTTP tunneling
Proxy-Authorization: basic aGVsbG86d29ybGQ=
Kubernetes service variables use tcp://10.0.0.11:6379.
Query kube-proxy at http://localhost:10249/proxyMode.
Public reference https://kubernetes.io/docs/ remains readable.`)
	readable, err := Extract(content, "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"HTTP tunneling", redactedCredential, redactedURL, "Public reference https://kubernetes.io/docs/ remains readable."} {
		if !strings.Contains(readable.Text, want) {
			t.Fatalf("network documentation lost %q: %q", want, readable.Text)
		}
	}
	for _, forbidden := range []string{"aGVsbG86d29ybGQ=", "10.0.0.11", "localhost:10249"} {
		if strings.Contains(readable.Text, forbidden) {
			t.Fatalf("network documentation leaked %q: %q", forbidden, readable.Text)
		}
	}
}

func TestValidateProviderContentStillRejectsUnredactedMaterial(t *testing.T) {
	for _, content := range []string{
		"Authorization: secret-value",
		"Bearer secret-value",
		"https://user:password@example.com/a",
		"http://localhost:8080/a",
		"http://10.0.0.1/a",
		"https://node.ts.net/a",
	} {
		if err := ValidateProviderContent(content); err == nil {
			t.Fatalf("unredacted material was accepted: %q", content)
		}
	}
}

func TestExtractNormalizesIsolatedForbiddenControls(t *testing.T) {
	readable, err := Extract([]byte("heading\x1ebody\x00tail"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if readable.Text != "heading body tail" || readable.ByteSize != len(readable.Text) {
		t.Fatalf("normalized readable = %#v", readable)
	}
}

func TestExtractStoredTextPreservesRepeatedLines(t *testing.T) {
	readable, err := Extract([]byte("same line\nsame line\nfinal line"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if readable.Extraction != "stored_text" || strings.Count(readable.Text, "same line") != 2 {
		t.Fatalf("stored text changed repeated prose: %#v", readable)
	}
}

func TestExtractAllowsPublicURLsInsideMarkdownAndJSON(t *testing.T) {
	for _, tc := range []struct {
		content   string
		mediaType string
	}{
		{content: `[public](https://example.com/report)`, mediaType: "text/markdown"},
		{content: `{"url":"https:\/\/example.com\/report"}`, mediaType: "application/json"},
		{content: `custom://example.com/report`, mediaType: "text/plain"},
	} {
		if _, err := Extract([]byte(tc.content), tc.mediaType); err != nil {
			t.Fatalf("public URL was rejected: %v", err)
		}
	}
}
