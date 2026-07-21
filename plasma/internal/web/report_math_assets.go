package web

import (
	"embed"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

//go:embed static/vendor/katex/katex.min.js static/vendor/katex/katex.min.css static/vendor/katex/fonts/*.woff2 static/vendor/markdown-it.min.js static/vendor/markdown-it-texmath.js static/vendor/purify.min.js static/vendor/mermaid.min.js static/report_math.js static/report_math.css static/report_mermaid_legend.js static/report_mermaid.js static/report_mermaid.css
var reportMathAssets embed.FS

var reportMathFonts = []string{
	"KaTeX_AMS-Regular.woff2", "KaTeX_Caligraphic-Bold.woff2", "KaTeX_Caligraphic-Regular.woff2",
	"KaTeX_Fraktur-Bold.woff2", "KaTeX_Fraktur-Regular.woff2", "KaTeX_Main-Bold.woff2",
	"KaTeX_Main-BoldItalic.woff2", "KaTeX_Main-Italic.woff2", "KaTeX_Main-Regular.woff2",
	"KaTeX_Math-BoldItalic.woff2", "KaTeX_Math-Italic.woff2", "KaTeX_SansSerif-Bold.woff2",
	"KaTeX_SansSerif-Italic.woff2", "KaTeX_SansSerif-Regular.woff2", "KaTeX_Script-Regular.woff2",
	"KaTeX_Size1-Regular.woff2", "KaTeX_Size2-Regular.woff2", "KaTeX_Size3-Regular.woff2",
	"KaTeX_Size4-Regular.woff2", "KaTeX_Typewriter-Regular.woff2",
}

const selfContainedReportRendererVersion = "html5-frontend-bracket-math-mermaid-legend-20260721"

var reportMathCSSURL = regexp.MustCompile(`url\(fonts/([^)]+)\)`)

func selfContainedMathHead() (string, error) {
	css, err := reportMathAssets.ReadFile("static/vendor/katex/katex.min.css")
	if err != nil {
		return "", err
	}
	fontData := make(map[string]string, len(reportMathFonts))
	for _, name := range reportMathFonts {
		data, readErr := reportMathAssets.ReadFile("static/vendor/katex/fonts/" + name)
		if readErr != nil {
			return "", readErr
		}
		fontData[name] = "data:font/woff2;base64," + base64.StdEncoding.EncodeToString(data)
	}
	missing := ""
	rewritten := reportMathCSSURL.ReplaceAllStringFunc(string(css), func(match string) string {
		parts := reportMathCSSURL.FindStringSubmatch(match)
		uri, ok := fontData[parts[1]]
		if !ok {
			missing = parts[1]
			return match
		}
		return "url(" + uri + ")"
	})
	if missing != "" || strings.Contains(rewritten, "url(") && reportMathCSSURL.MatchString(rewritten) {
		return "", fmt.Errorf("unresolved KaTeX font URL %q", missing)
	}
	custom, err := reportMathAssets.ReadFile("static/report_math.css")
	if err != nil {
		return "", err
	}
	return "<style>" + safeRawElement(rewritten+"\n"+string(custom), "style") + "</style>\n", nil
}

func selfContainedMermaidHead() (string, error) {
	css, err := reportMathAssets.ReadFile("static/report_mermaid.css")
	if err != nil {
		return "", err
	}
	compat := `:root{--line2:var(--line,#d1d5db);--surface:var(--panel,#fff);--danger:var(--warn,#b91c1c)}.plasma-mermaid-diagram{color:var(--ink,var(--text,#1f2937))}`
	return "<style>" + safeRawElement(compat+"\n"+string(css), "style") + "</style>\n", nil
}

func selfContainedMarkdownScripts() (string, error) {
	return selfContainedReportScriptsWithBootstrap(`(()=>{const run=()=>{const source=document.getElementById("report-markdown");const target=document.getElementById("report-body");if(source&&target&&window.renderPlasmaMarkdown)window.renderPlasmaMarkdown(target,JSON.parse(source.textContent))};document.readyState==="loading"?document.addEventListener("DOMContentLoaded",run,{once:true}):run()})();`)
}

func selfContainedDesignedScripts() (string, error) {
	return selfContainedReportScriptsWithBootstrap(`(()=>{const run=()=>{document.querySelectorAll("[data-designed-markdown]").forEach(node=>{const source=node.querySelector('script[type="application/json"]');if(!source||!window.renderPlasmaMarkdown)return;try{window.renderPlasmaMarkdown(node,JSON.parse(source.textContent))}catch(_error){}});window.renderDesignedTextMath&&window.renderDesignedTextMath(document.body)};document.readyState==="loading"?document.addEventListener("DOMContentLoaded",run,{once:true}):run()})();`)
}

func selfContainedReportScriptsWithBootstrap(bootstrap string) (string, error) {
	markdown, err := reportMathAssets.ReadFile("static/vendor/markdown-it.min.js")
	if err != nil {
		return "", err
	}
	texmath, err := reportMathAssets.ReadFile("static/vendor/markdown-it-texmath.js")
	if err != nil {
		return "", err
	}
	purify, err := reportMathAssets.ReadFile("static/vendor/purify.min.js")
	if err != nil {
		return "", err
	}
	runtime, err := reportMathAssets.ReadFile("static/vendor/katex/katex.min.js")
	if err != nil {
		return "", err
	}
	renderer, err := reportMathAssets.ReadFile("static/report_math.js")
	if err != nil {
		return "", err
	}
	scripts := "<script>" + safeRawElement(string(markdown), "script") + "</script>\n" +
		"<script>" + safeRawElement(string(texmath), "script") + "</script>\n" +
		"<script>" + safeRawElement(string(purify), "script") + "</script>\n" +
		"<script>" + safeRawElement(string(runtime), "script") + "</script>\n" +
		"<script>" + safeRawElement(string(renderer), "script") + "</script>\n"
	mermaidRuntime, err := reportMathAssets.ReadFile("static/vendor/mermaid.min.js")
	if err != nil {
		return "", err
	}
	mermaidLegend, err := reportMathAssets.ReadFile("static/report_mermaid_legend.js")
	if err != nil {
		return "", err
	}
	mermaidRenderer, err := reportMathAssets.ReadFile("static/report_mermaid.js")
	if err != nil {
		return "", err
	}
	scripts += "<script>" + safeRawElement(string(mermaidRuntime), "script") + "</script>\n" +
		"<script>" + safeRawElement(string(mermaidLegend), "script") + "</script>\n" +
		"<script>" + safeRawElement(string(mermaidRenderer), "script") + "</script>\n"
	scripts += "<script>" + safeRawElement(bootstrap, "script") + "</script>\n"
	return scripts, nil
}

func safeRawElement(value, element string) string {
	re := regexp.MustCompile(`(?i)</` + regexp.QuoteMeta(element))
	return re.ReplaceAllString(value, `<\/`+element)
}
