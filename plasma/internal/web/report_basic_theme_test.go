package web

import (
	"os/exec"
	"strings"
	"testing"
)

func TestSelfContainedReportThemeContract(t *testing.T) {
	css := selfContainedReportCSS()
	for _, expected := range []string{
		"color-scheme:light",
		"--bg:#fff",
		"--text:#111827",
		"--accent:#1d4ed8",
		"--heading-1:#172554",
		"--heading-2:#1e3a8a",
		"--heading-3:#1e40af",
		"--heading-4:#1d4ed8",
		"--heading-5:#2563eb",
		"--heading-6:#0077b6",
		"body.dark{color-scheme:dark",
		"--bg:#050505",
		"--text:#f8fafc",
		"--accent:#d4af37",
		"--heading-1:#987922",
		"--heading-2:#a17f22",
		"--heading-3:#b58e27",
		"--heading-4:#c89d2c",
		"--heading-5:#d4af37",
		"--heading-6:#f0c75e",
		`font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans KR","Apple SD Gothic Neo",sans-serif`,
		`font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,"Liberation Mono",monospace`,
		"h1{margin:0;font-size:clamp(28px,5vw,58px);font-weight:700;line-height:1.08;max-width:980px;color:var(--heading-1)}h2{font-size:24px;font-weight:600;color:var(--heading-2)}h3{font-size:20px;font-weight:600;color:var(--heading-3)}h4{font-size:18px;font-weight:500;color:var(--heading-4)}h5{font-size:16px;font-weight:500;color:var(--heading-5)}h6{font-size:15px;font-weight:500;color:var(--heading-6)}",
		".metric,.media-panel,.notes,.report-body{background:transparent;border:0;border-radius:0}",
		".report-body h1{margin-top:3.25rem}.report-body h1:first-child{margin-top:0",
		".report-body h2.marked,.report-body h3.marked{text-decoration:underline",
		".report-body table{display:block;width:100%;max-width:100%;overflow-x:auto",
		"button:focus-visible,a:focus-visible,summary:focus-visible{outline:3px solid var(--focus)",
		"pre{max-width:100%;overflow:auto;padding:14px;border-left:3px solid var(--accent);background:var(--code-bg);border-radius:0}",
		".report-body th{background:var(--code-bg);color:var(--heading-2);font-size:13px;font-weight:600}",
		".gallery figcaption span{font-size:13px;color:var(--muted);word-break:break-word}",
		".report-body>:is(h1,h2,h3,h4,h5,h6,p,ul,ol,blockquote){max-width:72ch}",
	} {
		if !strings.Contains(css, expected) {
			t.Fatalf("basic HTML theme is missing %q", expected)
		}
	}
	for _, wideContent := range []string{
		".report-body table{",
		".report-body img{",
		".gallery{",
	} {
		idx := strings.Index(css, wideContent)
		if idx == -1 {
			t.Fatalf("basic HTML theme missing element %q", wideContent)
		}
		element := css[idx:]
		if idx := strings.Index(element, "max-width:72ch"); idx != -1 {
			t.Fatalf("basic HTML theme applied 72ch limit to wide content %q", wideContent)
		}
	}
	for _, forbidden := range []string{
		"linear-gradient",
		"Georgia",
		"Noto Serif KR",
		"body.light",
	} {
		if strings.Contains(css, forbidden) {
			t.Fatalf("basic HTML theme retained %q", forbidden)
		}
	}
}

func TestSelfContainedBasicMermaidThemeContract(t *testing.T) {
	css := selfContainedBasicMermaidCSS()
	for _, expected := range []string{
		".plasma-mermaid-card{background:var(--code-bg);border:0",
		".plasma-mermaid-diagram,.plasma-mermaid-line-legend{color:var(--text)}",
		".plasma-mermaid-line-legend-item:focus-visible{outline:3px solid var(--focus)",
	} {
		if !strings.Contains(css, expected) {
			t.Fatalf("basic HTML Mermaid theme is missing %q", expected)
		}
	}
	cardIdx := strings.Index(css, ".plasma-mermaid-card{")
	if cardIdx == -1 {
		t.Fatalf("basic HTML Mermaid theme missing .plasma-mermaid-card")
	}
	cardElement := css[cardIdx:]
	if strings.Index(cardElement, "max-width:72ch") != -1 {
		t.Fatalf("basic HTML Mermaid theme applied 72ch limit to .plasma-mermaid-card")
	}
}

func TestSelfContainedReportThemeScriptRerendersMermaid(t *testing.T) {
	script := selfContainedReportThemeScript()
	for _, expected := range []string{
		`m.initialize=config=>m.__plasmaThemeInitialize({...config,theme:dark?"dark":"default"})`,
		`m.__plasmaConfigured=false`,
		`document.querySelectorAll(".plasma-mermaid-card")`,
		`figure.replaceWith(pre)`,
		`r.renderPlasmaMermaid(document)`,
		`t.hidden=false`,
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("basic HTML theme script is missing %q", expected)
		}
	}
}

func TestSelfContainedReportThemeToggle(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := strings.TrimSuffix(strings.TrimPrefix(selfContainedReportThemeScript(), "<script>"), "</script>\n")
	fixture := `
const classes = new Set();
const source = "graph TD\\n  A-->B";
const figures = [];
const replacedFigures = [];
const figureReplacements = [];
const replacements = [];
let currentNode;
function makeNode(tagName) {
  const node = {
    tagName: tagName.toUpperCase(), className: "", textContent: "", children: [], parentElement: null,
    appendChild(child) { child.parentElement = this; this.children.push(child); return child; },
    querySelector(selector) {
      if (selector === ".plasma-mermaid-raw") return this.children.find((child) => child.className === "plasma-mermaid-raw") || null;
      return null;
    },
    replaceWith(next) {
      this.replacedWith = next;
      replacements.push({ from: this, to: next });
      if (this.className === "plasma-mermaid-card") figureReplacements.push({ from: this, to: next });
      if (currentNode === this) currentNode = next;
    }
  };
  return node;
}
function makeFigure(track = true) {
  const figure = makeNode("figure");
  figure.className = "plasma-mermaid-card";
  const raw = makeNode("code");
  raw.className = "plasma-mermaid-raw";
  raw.textContent = source;
  figure.appendChild(raw);
  if (track) figures.push(figure);
  return figure;
}
currentNode = makeFigure();
const initializeThemes = [];
let renderCalls = 0;
const mermaid = {
  initialize(config) { initializeThemes.push(config.theme); }
};
const button = {
  hidden: true, attributes: {}, textContent: "",
  setAttribute(name, value) { this.attributes[name] = value; },
  addEventListener(name, callback) { if (name === "click") this.click = callback; }
};
global.window = {
  mermaid,
  Plasma: { reports: {
    renderPlasmaMermaid(root) {
      renderCalls++;
      mermaid.initialize({ theme: "fixture" });
      root.querySelectorAll("pre > code.language-mermaid, pre > code.lang-mermaid").forEach((code) => {
        const figure = makeFigure(false);
        figure.children[0].textContent = code.textContent;
        replacedFigures.push(figure);
        code.parentElement.replaceWith(figure);
      });
    }
  }}
};
global.document = {
  querySelectorAll(selector) {
    if (selector === ".plasma-mermaid-card") return currentNode.className === "plasma-mermaid-card" ? [currentNode] : [];
    if (selector === "pre > code.language-mermaid, pre > code.lang-mermaid") {
      if (currentNode.tagName !== "PRE") return [];
      return currentNode.children.filter((child) => child.className === "language-mermaid");
    }
    return [];
  },
  createElement(tagName) { return makeNode(tagName); },
  body: { classList: {
    toggle(name, on) { if (on) classes.add(name); else classes.delete(name); },
    contains(name) { return classes.has(name); }
  }},
  getElementById(id) { return id === "themeToggle" ? button : null; }
};
` + script + `
if (button.hidden) process.exit(1);
if (classes.has("dark")) process.exit(2);
if (button.attributes["aria-pressed"] !== "false" || button.textContent !== "다크 모드") process.exit(3);
button.click();
if (!classes.has("dark")) process.exit(4);
if (button.attributes["aria-pressed"] !== "true" || button.attributes["aria-label"] !== "라이트 모드 켜기" || button.textContent !== "라이트 모드") process.exit(5);
button.click();
if (classes.has("dark") || button.attributes["aria-pressed"] !== "false") process.exit(6);
if (renderCalls !== 2 || initializeThemes.join(",") !== "dark,default") process.exit(7);
if (renderCalls !== 2 || replacedFigures.length !== 2 || figureReplacements.length !== 2) process.exit(8);
for (const figure of replacedFigures) {
  if (figure.className !== "plasma-mermaid-card" || figure.children[0].className !== "plasma-mermaid-raw" || figure.children[0].textContent !== source) process.exit(9);
}
for (const replacement of figureReplacements) {
  const pre = replacement.to;
  if (pre.tagName !== "PRE" || pre.children.length !== 1 || pre.children[0].textContent !== source || pre.children[0].className !== "language-mermaid") process.exit(9);
}
if (currentNode.className !== "plasma-mermaid-card" || currentNode.children[0].className !== "plasma-mermaid-raw" || currentNode.children[0].textContent !== source) process.exit(10);
`
	if out, err := exec.Command("node", "-e", fixture).CombinedOutput(); err != nil {
		t.Fatalf("basic HTML theme toggle fixture: %v: %s", err, out)
	}
}

func TestSelfContainedReportRendererVersionIncludesTheme(t *testing.T) {
	if selfContainedReportRendererVersion != "html8-heading-palette-20260814" {
		t.Fatalf("basic renderer version = %q", selfContainedReportRendererVersion)
	}
}
