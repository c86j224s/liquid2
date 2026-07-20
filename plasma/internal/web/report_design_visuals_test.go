package web

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
	"golang.org/x/net/html"
)

func TestDesignedReportVisualKindNormalization(t *testing.T) {
	model := normalizeDesignedReportContentModel(designedReportContentModel{
		VisualUnits: []designedReportVisual{
			{Title: "A", Kind: "Decision Tree", Nodes: []designedReportNode{{Label: "A", Body: "B"}}},
			{Title: "B", Kind: "dependency-path", Nodes: []designedReportNode{{Label: "A", Body: "B"}}},
			{Title: "C", Kind: "unknown", Nodes: []designedReportNode{{Label: "A", Body: "B"}}},
		},
	})
	got := []string{
		model.VisualUnits[0].Kind,
		model.VisualUnits[1].Kind,
		model.VisualUnits[2].Kind,
	}
	want := []string{"decision_tree", "dependency_path", "map"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("kind %d: got %q, want %q", index, got[index], want[index])
		}
	}
}

func TestDesignedReportVisualUnitsDispatchGrammar(t *testing.T) {
	visuals := []designedReportVisual{
		testDesignedVisual("일정", "timeline"),
		testDesignedVisual("근거", "evidence_chain"),
		testDesignedVisual("판단", "tradeoff_matrix"),
		testDesignedVisual("반복", "loop"),
		testDesignedVisual("관계", "unknown"),
	}
	var out bytes.Buffer
	renderDesignedVisualUnits(&out, visuals)
	content := out.String()
	for _, expected := range []string{
		"visual-timeline",
		"visual-evidence-chain",
		"visual-matrix",
		"visual-loop",
		"relationship-svg",
		"Relationship map",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected rendered visual units to contain %q:\n%s", expected, content)
		}
	}
}

func TestDesignedReportSourceTeXPromptAndVersionContract(t *testing.T) {
	prompt := agentDesignedHTMLContentModelPrompt("수식 리포트", `본문 \(E=mc^2\), \[x^2+y^2=z^2\]를 보존합니다.`, nil)
	for _, expected := range []string{
		"Use only \\(...\\) for inline math and \\[...\\] for display math.",
		"Do not rewrite, translate, invent, or place formulas only in SVG text.",
		`\(E=mc^2\)`,
		`\[x^2+y^2=z^2\]`,
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected designed report source TeX contract %q:\n%s", expected, prompt)
		}
	}

	const expectedVersion = "dh31-source-markdown-visuals-20260721"
	if designedReportRendererVersion != expectedVersion {
		t.Fatalf("designed renderer version = %q, want %q", designedReportRendererVersion, expectedVersion)
	}
	appJS := string(mustReadStatic(t, "static/app.js"))
	if !strings.Contains(appJS, `const DESIGNED_REPORT_RENDERER_VERSION = "`+expectedVersion+`";`) {
		t.Fatalf("browser designed renderer version is not synchronized with %q", expectedVersion)
	}
}

func TestDesignedReportHTMLDOMSmoke(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	server := NewServer(svc, Options{}).(*Server)
	model := normalizeDesignedReportContentModel(designedReportContentModel{
		Kicker:   "Designed Report",
		Title:    "렌더러 검증 리포트",
		Subtitle: "여러 시각 문법을 한 HTML 안에서 검증합니다.",
		Thesis:   `시각 문법은 \(E=mc^2\) 관계를 따라 선택됩니다.`,
		VisualUnits: []designedReportVisual{
			testDesignedVisual("핵심 관계", "map"),
			testDesignedVisual("일정", "timeline"),
			testDesignedVisual("근거", "evidence_chain"),
			testDesignedVisual("판단", "tradeoff_matrix"),
			testDesignedVisual("반복", "loop"),
		},
		Tabs: []designedReportTab{{
			Label:    "검증",
			Question: "무엇이 렌더링되는가",
			Sections: []designedReportSection{{
				Heading:    "출처 보존",
				Body:       []string{`이 HTML은 \[x^2+y^2=z^2\] 관계를 표시합니다.`},
				Table:      designedReportTable{Columns: []string{"항목", `값 \(x\)`}, Rows: [][]string{{"결과", `\(y\)`}}},
				Caveat:     `잘못된 \(\notacommand{\) 수식은 원문으로 남습니다.`,
				SourceNote: `원본 \(M\) Markdown 리포트 기반`,
			}},
		}},
		Sources: []designedReportSource{{
			Label: "원본",
			Href:  "https://example.com/report",
			Note:  `테스트용 \(S\) 안전 URL`,
		}},
		Caveats: []string{`테스트 \(C\) fixture는 실제 판단 자료가 아닙니다.`},
	})
	model.VisualUnits[0].Nodes[0].Label = `SVG \(x\)`
	content, err := server.renderDesignedReportHTML(app.RawArtifact{
		ArtifactID: "art_dom_smoke_md",
		MissionID:  "mis_dom_smoke",
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "dom-smoke.md",
		Content:    []byte("# 렌더러 검증 리포트\n\n원본 Markdown 리포트입니다.\n"),
	}, model, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(strings.NewReader(string(content)))
	if err != nil {
		t.Fatalf("designed HTML should parse as DOM: %v\n%s", err, content)
	}
	for _, className := range []string{"hero-map-svg", "visual-timeline", "visual-evidence-chain", "visual-matrix", "visual-loop", "sources-panel"} {
		if !domHasClass(doc, className) {
			t.Fatalf("expected DOM class %q in designed HTML:\n%s", className, content)
		}
	}
	if domHasExternalResource(doc) {
		t.Fatalf("designed HTML should not auto-load external resources:\n%s", content)
	}
	htmlContent := string(content)
	for _, expected := range []string{"renderDesignedTextMath(document.body)", "renderPlasmaMarkdown(node,JSON.parse(source.textContent))", "renderPlasmaMermaid(root)", "data:font/woff2;base64,", `version:"0.17.0"`, `<text class="hero-map-label" x="-98" y="2">SVG \(x\)</text>`} {
		if !strings.Contains(htmlContent, expected) {
			t.Fatalf("expected designed math contract %q", expected)
		}
	}
	if strings.Contains(htmlContent, `<text class="hero-map-label" x="-98" y="2"><span class="plasma-math`) {
		t.Fatal("SVG text label received HTML math markup")
	}
}

func TestDesignedReportCSSIncludesMobileOverflowGuard(t *testing.T) {
	css := designedReportCSS()
	mediaStart := strings.Index(css, "@media(max-width:940px)")
	if mediaStart < 0 {
		t.Fatal("expected designed report CSS to include mobile media query")
	}
	mediaCSS := css[mediaStart:]
	for _, expected := range []string{
		".designed-hero,.designed-shell{width:100%;overflow:hidden}",
		".designed-hero *,.designed-shell *{max-width:100%;min-width:0}",
		".hero-map-svg{display:none}",
		".hero-map-readable{display:grid",
		"word-break:break-all",
	} {
		if !strings.Contains(mediaCSS, expected) {
			t.Fatalf("expected designed report mobile CSS to contain %q", expected)
		}
	}
	if !strings.Contains(css, "overflow-x:hidden") {
		t.Fatal("expected designed report CSS to hide horizontal body overflow")
	}
}

func testDesignedVisual(title string, kind string) designedReportVisual {
	return designedReportVisual{
		Title:    title,
		Kind:     kind,
		Question: "무엇을 읽어야 하는가",
		Nodes: []designedReportNode{
			{Label: "첫 단계", Body: "보고서의 첫 근거를 확인합니다.", Tone: "accent"},
			{Label: "둘째 단계", Body: "한계와 판단을 분리합니다.", Tone: "warn"},
		},
		Caption: "검증용 시각 단위입니다.",
	}
}

func domHasClass(node *html.Node, className string) bool {
	if node.Type == html.ElementNode {
		for _, attr := range node.Attr {
			if attr.Key != "class" {
				continue
			}
			for _, candidate := range strings.Fields(attr.Val) {
				if candidate == className {
					return true
				}
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if domHasClass(child, className) {
			return true
		}
	}
	return false
}

func domHasExternalResource(node *html.Node) bool {
	if node.Type == html.ElementNode {
		if node.Data == "iframe" || node.Data == "link" {
			return true
		}
		for _, attr := range node.Attr {
			if node.Data == "script" && attr.Key == "src" && strings.TrimSpace(attr.Val) != "" {
				return true
			}
			if (node.Data == "img" || node.Data == "source") && attr.Key == "src" && strings.HasPrefix(strings.TrimSpace(attr.Val), "http") {
				return true
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if domHasExternalResource(child) {
			return true
		}
	}
	return false
}
