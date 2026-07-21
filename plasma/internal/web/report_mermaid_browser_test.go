package web

import (
	"os/exec"
	"strings"
	"testing"
)

func TestReportMermaidStaticAssetOrderAndLazyRuntime(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	ordered := []string{
		"vendor/purify.min.js",
		"vendor/katex/katex.min.js",
		"report_math.js",
		"report_mermaid_legend.js",
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
		"static/report_mermaid_legend.js",
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
	legend := string(mustReadStatic(t, "static/report_mermaid_legend.js"))
	style := string(mustReadStatic(t, "static/report_mermaid.css"))
	for _, expected := range []string{
		"window.renderPlasmaMermaid?.(log)",
		`if (kind === "markdown") window.renderPlasmaMermaid?.($("detailBody"))`,
		`/static/vendor/mermaid.min.js`,
		`securityLevel: "strict"`,
		`startOnLoad: false`,
		`pre > code.language-mermaid`,
		`DOMPurify.sanitize`,
		`applyPlasmaMermaidLineLegend?.(figure, source)`,
		`plasmaMermaidLineLegendLabels`,
		`bindLineSeries(svg, labels.length)`,
		`line-plot-`,
		`plasma-mermaid-line-series--active`,
		`normalizeMermaidSVGLabels`,
		`foreignObject`,
		`plasma-mermaid-label`,
		`plasma-mermaid-line-legend`,
		`plasma-mermaid-card`,
	} {
		if !strings.Contains(app+"\n"+runtime+"\n"+legend+"\n"+style, expected) {
			t.Fatalf("expected Mermaid rendering contract %q", expected)
		}
	}
	for _, forbidden := range []string{
		`kind === "html") window.renderPlasmaMermaid`,
		`src="/static/vendor/mermaid.min.js"`,
	} {
		if strings.Contains(app+"\n"+runtime+"\n"+legend, forbidden) {
			t.Fatalf("Mermaid support should not include %q", forbidden)
		}
	}
}

func TestReportMermaidLineLegendParser(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	fixture := `
const fs = require("fs"), vm = require("vm");
const context = { window: {} };
vm.createContext(context);
vm.runInContext(fs.readFileSync("static/report_mermaid_legend.js", "utf8"), context);
const labels = context.window.plasmaMermaidLineLegendLabels(` + "`" + `xychart-beta
  title "TIOBE"
  x-axis [2020, 2021]
  line "Python" [3, 1]
  line "C++" [4, 3]
  line 'Rust' [18, 10]
  line JavaScript [7, 6]
` + "`" + `);
if (JSON.stringify(labels) !== JSON.stringify(["Python", "C++", "Rust", "JavaScript"])) process.exit(1);
const flowchartLabels = context.window.plasmaMermaidLineLegendLabels("flowchart TD\\n  A --> B");
if (flowchartLabels.length !== 0) process.exit(2);
const unlabeled = context.window.plasmaMermaidLineLegendLabels("xychart-beta\\n  line [1, 2]");
if (unlabeled.length !== 0) process.exit(3);
if (typeof context.window.applyPlasmaMermaidLineLegend !== "function") process.exit(4);
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("Mermaid legend parser fixture: %v: %s", err, out)
	}
}
