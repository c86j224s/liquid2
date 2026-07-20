package web

import (
	"strings"
	"testing"
)

func TestReportMermaidStaticAssetOrderAndLazyRuntime(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	ordered := []string{
		"vendor/purify.min.js",
		"vendor/katex/katex.min.js",
		"report_math.js",
		"report_mermaid.js",
		"app.js",
	}
	last := -1
	for _, asset := range ordered {
		at := strings.Index(index, asset)
		if at < 0 || at <= last {
			t.Fatalf("Mermaid support asset %q missing or out of order", asset)
		}
		last = at
	}
	if strings.Contains(index, "vendor/mermaid.min.js") {
		t.Fatal("Mermaid runtime should be lazy-loaded, not included in the initial page")
	}
	for _, asset := range []string{
		"static/report_mermaid.js",
		"static/report_mermaid.css",
		"static/vendor/mermaid.min.js",
		"static/vendor/mermaid.LICENSE",
	} {
		if len(mustReadStatic(t, asset)) == 0 {
			t.Fatalf("empty static asset %q", asset)
		}
	}
}

func TestReportMermaidMarkdownSurfaces(t *testing.T) {
	app := string(mustReadStatic(t, "static/app.js"))
	runtime := string(mustReadStatic(t, "static/report_mermaid.js"))
	style := string(mustReadStatic(t, "static/report_mermaid.css"))
	for _, expected := range []string{
		"window.renderPlasmaMermaid?.(log)",
		`if (kind === "markdown") window.renderPlasmaMermaid?.($("detailBody"))`,
		`/static/vendor/mermaid.min.js`,
		`securityLevel: "strict"`,
		`startOnLoad: false`,
		`pre > code.language-mermaid`,
		`DOMPurify.sanitize`,
		`normalizeMermaidSVGLabels`,
		`foreignObject`,
		`plasma-mermaid-label`,
		`plasma-mermaid-card`,
	} {
		if !strings.Contains(app+"\n"+runtime+"\n"+style, expected) {
			t.Fatalf("expected Mermaid rendering contract %q", expected)
		}
	}
	for _, forbidden := range []string{
		`kind === "html") window.renderPlasmaMermaid`,
		`src="/static/vendor/mermaid.min.js"`,
	} {
		if strings.Contains(app+"\n"+runtime, forbidden) {
			t.Fatalf("Mermaid support should not include %q", forbidden)
		}
	}
}
