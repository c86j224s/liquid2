package reportilphase0

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/c86j224s/liquid2/plasma/internal/pdfdocument"
	"github.com/chromedp/chromedp"
)

func TestRenderPDFRejectsEmptyChromePath(t *testing.T) {
	if _, err := RenderPDF(context.Background(), []byte("<!doctype html><html><body>test</body></html>"), " "); err == nil {
		t.Fatal("empty Chrome path unexpectedly accepted")
	}
}

func installedChromePath(t *testing.T) string {
	t.Helper()
	chromePath := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	chromeInfo, err := os.Stat(chromePath)
	if err != nil || chromeInfo.IsDir() || chromeInfo.Mode()&0o111 == 0 {
		t.Skip("installed macOS Google Chrome is not executable")
	}
	return chromePath
}

func pdfKitPageTexts(t *testing.T, content []byte) []string {
	t.Helper()
	if _, err := exec.LookPath("swift"); err != nil {
		t.Skip("Swift is required for the macOS PDFKit pagination check")
	}
	directory := t.TempDir()
	pdfPath := filepath.Join(directory, "report.pdf")
	scriptPath := filepath.Join(directory, "page-text.swift")
	if err := os.WriteFile(pdfPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	script := `import Foundation
import PDFKit
let document = PDFDocument(url: URL(fileURLWithPath: CommandLine.arguments[1]))!
let pages = (0..<document.pageCount).map { document.page(at: $0)?.string ?? "" }
let data = try! JSONSerialization.data(withJSONObject: pages)
FileHandle.standardOutput.write(data)
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("swift", scriptPath, pdfPath).CombinedOutput()
	if err != nil {
		t.Fatalf("PDFKit page text extraction failed: %v\n%s", err, output)
	}
	var texts []string
	if err := json.Unmarshal(output, &texts); err != nil {
		t.Fatalf("decode PDFKit page text: %v\n%s", err, output)
	}
	return texts
}

func TestRenderHTMLAndPDFWithInstalledChrome(t *testing.T) {
	chromePath := installedChromePath(t)
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc.smoke", RevisionID: "rev.smoke", Title: "Renderer smoke", Language: "en",
		Blocks: []Block{
			{NodeID: "intro", Kind: "prose", Prose: "A small self-contained renderer smoke test."},
			{NodeID: "table", Kind: "table", Table: &Table{Caption: "Values", Columns: []string{"Name", "Value"}, Rows: [][]string{{"Alpha", "1"}, {"Beta", "2"}}}},
			{NodeID: "equation", Kind: "equation", Equation: &Equation{Expression: `\[ T_{\mathrm{charged}} = \sum_{i \in \{0,1\}} \mathrm{total\_thought\_tokens}_i + n_o p_o \]`, Notation: "latex"}},
		},
		Provenance: map[string]string{"test": "installed-chrome-smoke"},
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	htmlText := string(html)
	for _, required := range []string{
		"<!doctype html>", "Content-Security-Policy", "default-src 'none'", "script-src 'unsafe-inline'",
		`data-tex="T_{\mathrm{charged}} = \sum_{i \in \{0,1\}} \mathrm{total\_thought\_tokens}_i + n_o p_o"`,
		`output:"mathml"`, `data-math-rendered="false"`,
	} {
		if !strings.Contains(htmlText, required) {
			t.Fatalf("HTML missing %q", required)
		}
	}
	lower := strings.ToLower(htmlText)
	for _, forbidden := range []string{"<link", "src=\"http:", "src=\"https:", "url(http", "url(https", `<mtext>\[`, `<code>\[`} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("HTML contains forbidden external or raw-equation content %q", forbidden)
		}
	}
	result, err := RenderPDF(context.Background(), html, chromePath)
	if err != nil {
		t.Fatal(err)
	}
	if result.RendererProduct == "" || result.RendererRevision == "" {
		t.Fatalf("renderer identity missing: %#v", result)
	}
	if !strings.HasPrefix(string(result.Content), "%PDF-") {
		t.Fatal("PDF magic missing")
	}
	pdfInfo, err := pdfdocument.Inspect(result.Content)
	if err != nil {
		t.Fatal(err)
	}
	if pdfInfo.PageCount <= 0 {
		t.Fatalf("page count = %d", pdfInfo.PageCount)
	}
	pdfText := strings.Join(pdfKitPageTexts(t, result.Content), "\n")
	compactPDFText := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, pdfText)
	for _, forbidden := range []string{`\[`, `\]`, `\mathrm`, `\sum`, `\{0,1\}`} {
		if strings.Contains(pdfText, forbidden) {
			t.Fatalf("PDF exposed raw LaTeX %q: %q", forbidden, pdfText)
		}
	}
	for _, required := range []string{"charged", "total_thought_tokens", "∑", "0,1"} {
		if !strings.Contains(compactPDFText, required) {
			t.Fatalf("PDF projection lost typeset equation component %q: %q", required, pdfText)
		}
	}
	document.Blocks[2].Equation.Expression = `\[ T_{\mathrm{charged}} = ` + strings.Repeat(`\mathrm{total\_thought\_tokens}_i + `, 8) + `n_o p_o \]`
	wideHTML, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	wideResult, err := RenderPDF(context.Background(), wideHTML, chromePath)
	if err != nil {
		t.Fatal(err)
	}
	widePages := pdfKitPageTexts(t, wideResult.Content)
	wideText := strings.Join(widePages, "\n")
	compactWideText := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, wideText)
	if strings.Contains(wideText, `\mathrm`) ||
		!strings.Contains(compactWideText, "total_thought_tokens") ||
		!strings.Contains(compactWideText, "𝑛𝑜𝑝𝑜") {
		t.Fatalf("wide equation was clipped or exposed as raw LaTeX: %q", wideText)
	}
}

func TestRenderHTMLAndPDFRenderMermaidWithInstalledChrome(t *testing.T) {
	chromePath := installedChromePath(t)
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc.mermaid", RevisionID: "rev.mermaid", Title: "초보자 전투 흐름", Language: "ko",
		Blocks: []Block{{
			NodeID: "flow", Kind: "code", Language: "mermaid",
			Code: "flowchart TD\n  A[첫 내정 시작] --> B{병량이 충분한가?}\n  B -->|예| C[첫 전투 준비]\n  B -->|아니오| D[농지와 시장 확충]\n  D --> A",
		}},
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	htmlText := string(html)
	for _, required := range []string{
		`data-mermaid-rendered="false"`, `pre > code.language-mermaid`, `securityLevel: "strict"`,
		`renderPlasmaMermaid`, `Mermaid renderer returned no SVG`,
	} {
		if !strings.Contains(htmlText, required) {
			t.Fatalf("HTML missing Mermaid contract %q", required)
		}
	}
	if strings.Contains(htmlText, `src="/static/`) {
		t.Fatal("self-contained HTML references Plasma static assets")
	}

	allocatorCtx, allocatorCancel := chromedp.NewExecAllocator(context.Background(), append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(chromePath), chromedp.DisableGPU)...)
	defer allocatorCancel()
	browserCtx, browserCancel := chromedp.NewContext(allocatorCtx)
	defer browserCancel()
	htmlPath := filepath.Join(t.TempDir(), "mermaid.html")
	if err := os.WriteFile(htmlPath, html, 0o600); err != nil {
		t.Fatal(err)
	}
	var status, renderedText string
	var svgCount, rawCount int
	if err := chromedp.Run(browserCtx,
		chromedp.Navigate("file://"+htmlPath),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Poll(`document.body?.dataset.mermaidRendered !== "false"`, nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Evaluate(`document.body.dataset.mermaidRendered`, &status),
		chromedp.Evaluate(`document.querySelectorAll(".plasma-mermaid-diagram svg").length`, &svgCount),
		chromedp.Evaluate(`document.querySelectorAll("pre > code.language-mermaid, pre > code.lang-mermaid").length`, &rawCount),
		chromedp.Evaluate(`document.querySelector(".plasma-mermaid-diagram")?.textContent || ""`, &renderedText),
	); err != nil {
		t.Fatal(err)
	}
	if status != "true" || svgCount != 1 || rawCount != 0 {
		t.Fatalf("Mermaid browser render status=%q svg=%d raw=%d", status, svgCount, rawCount)
	}
	for _, label := range []string{"첫 내정 시작", "병량이 충분한가", "첫 전투 준비", "농지와 시장 확충"} {
		if !strings.Contains(renderedText, label) {
			t.Fatalf("rendered Mermaid lost label %q: %q", label, renderedText)
		}
	}

	result, err := RenderPDF(context.Background(), html, chromePath)
	if err != nil {
		t.Fatal(err)
	}
	pdfText := strings.Join(pdfKitPageTexts(t, result.Content), "\n")
	for _, forbidden := range []string{"flowchart TD", "A[첫 내정 시작]", "B -->|예|"} {
		if strings.Contains(pdfText, forbidden) {
			t.Fatalf("PDF exposed raw Mermaid source %q: %q", forbidden, pdfText)
		}
	}
	for _, label := range []string{"첫 내정 시작", "병량이 충분한가", "첫 전투 준비", "농지와 시장 확충"} {
		if !strings.Contains(strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, pdfText), strings.ReplaceAll(label, " ", "")) {
			t.Fatalf("PDF lost rendered Mermaid label %q: %q", label, pdfText)
		}
	}
}

func TestRenderPDFRejectsMermaidRenderErrors(t *testing.T) {
	chromePath := installedChromePath(t)
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc.bad-mermaid", RevisionID: "rev.bad-mermaid", Title: "Bad Mermaid", Language: "en",
		Blocks: []Block{{NodeID: "diagram", Kind: "code", Language: "mermaid", Code: "flowchart TD\n A --><"}},
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RenderPDF(context.Background(), html, chromePath); err == nil || !strings.Contains(err.Error(), "Mermaid render failed") {
		t.Fatalf("invalid Mermaid PDF render = %v", err)
	}
}

func TestRenderPDFRejectsEquationRenderErrors(t *testing.T) {
	chromePath := installedChromePath(t)
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc.bad-equation", RevisionID: "rev.bad-equation", Title: "Bad equation", Language: "en",
		Blocks: []Block{{NodeID: "equation", Kind: "equation", Equation: &Equation{Expression: `\notacommand{`, Notation: "latex"}}},
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RenderPDF(context.Background(), html, chromePath); err == nil || !strings.Contains(err.Error(), "equation render failed") {
		t.Fatalf("invalid equation PDF render = %v", err)
	}
}

func TestRenderHTMLUsesStrongPrintFigurePagination(t *testing.T) {
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc.figure.print", RevisionID: "rev.figure.print", Title: "Figure print", Language: "en",
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	htmlText := string(html)
	if !strings.Contains(htmlText, `figure,table,.equation{break-inside:avoid;page-break-inside:avoid}`) {
		t.Fatal("print CSS does not carry modern and legacy figure page-break controls")
	}
	if strings.Contains(htmlText, `figure,table,pre,.equation{break-inside:avoid;page-break-inside:avoid}`) {
		t.Fatal("print CSS keeps code blocks together and can shrink them below readable size")
	}
}

func TestRenderPDFDoesNotShrinkLongCodeBlocks(t *testing.T) {
	chromePath := installedChromePath(t)
	codeLines := []string{
		`Introductory line 01`,
		`Introductory line 02`,
		`Introductory line 03`,
		`Introductory line 04`,
		`Introductory line 05`,
		`Introductory line 06`,
		`Introductory line 07`,
		`Introductory line 08`,
		`Introductory line 09`,
		`Introductory line 10`,
		`Introductory line 11`,
		`Introductory line 12`,
		`Introductory line 13`,
		`Introductory line 14`,
		`Introductory line 15`,
		`Introductory line 16`,
		`Introductory line 17`,
		`Introductory line 18`,
		`Introductory line 19`,
		`Introductory line 20`,
		`{`,
		`  "stateful_first_request": {`,
		`    "store": true,`,
		`    "input": "first turn"`,
		`  },`,
		`  "stateful_follow_up": {`,
		`    "previous_interaction_id": "interaction ID",`,
		`    "input": "next turn"`,
		`  },`,
		`  "stateless_follow_up": {`,
		`    "input": [`,
		`      "complete history",`,
		`      "all thought blocks",`,
		`      "tool signatures",`,
		`      "current turn"`,
		`    ]`,
		`  },`,
		`  "stateless_model_switch": {`,
		`    "model": "next model",`,
		`    "input": [`,
		`      "previous thought blocks",`,
		`      "tool signatures",`,
		`      "current turn"`,
		`    ]`,
		`  }`,
		`}`,
	}
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc.code.print", RevisionID: "rev.code.print", Title: "Code print", Language: "en",
		Blocks: []Block{
			{NodeID: "intro", Kind: "prose", Prose: strings.Repeat("Long report prose before the code projection. ", 180)},
			{NodeID: "code", Kind: "code", Language: "json", Code: strings.Join(codeLines, "\n")},
		},
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	result, err := RenderPDF(context.Background(), html, chromePath)
	if err != nil {
		t.Fatal(err)
	}
	pages := pdfKitPageTexts(t, result.Content)
	if len(pages) < 2 {
		t.Fatalf("long code block stayed on one page and may have been shrunk: %d pages", len(pages))
	}
	pdfText := strings.Join(pages, "\n")
	for _, required := range []string{`stateful_first_request`, `stateless_model_switch`, `previous thought blocks`} {
		if !strings.Contains(pdfText, required) {
			t.Fatalf("PDF projection lost long code content %q: %q", required, pdfText)
		}
	}
}

func TestRenderPDFPreservesKoreanTextLayout(t *testing.T) {
	chromePath := installedChromePath(t)
	prose := "다케다성은 효고현 아사고시의 산 정상에 세워진 산성 유적이다. 계곡을 내려다보는 높은 자리와 지형을 따라 쌓은 석축이 하나의 방어 체계를 이룬다."
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc.korean", RevisionID: "rev.korean", Title: "다케다성의 역사와 경관", Language: "ko",
		Blocks: []Block{
			{NodeID: "section.korean", Kind: "section", Level: 2, Title: "산 위에 세워진 성"},
			{NodeID: "prose.korean", ParentNodeID: "section.korean", Kind: "prose", Prose: prose},
		},
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), `@media print{:root,body{font-family:"Nanum Gothic","Noto Sans CJK KR","Noto Sans KR","Arial Unicode MS",Arial,sans-serif`) {
		t.Fatal("print CSS does not pin a Korean-capable primary font")
	}
	result, err := RenderPDF(context.Background(), html, chromePath)
	if err != nil {
		t.Fatal(err)
	}
	pdfText := strings.Join(pdfKitPageTexts(t, result.Content), "\n")
	compactPDFText := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, pdfText)
	for _, value := range []string{document.Title, document.Blocks[0].Title, prose} {
		compactValue := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, value)
		if !strings.Contains(compactPDFText, compactValue) {
			t.Fatalf("PDF projection split or reordered Korean text %q in %q", value, pdfText)
		}
	}
}

func TestRenderPDFDoesNotLeaveOneReferenceOnFinalPage(t *testing.T) {
	chromePath := installedChromePath(t)
	longParagraph := strings.Repeat("The castle shows how mountain terrain supported defense and regional control. ", 72)
	conclusion := strings.Repeat("The final judgment connects the castle to regional order. ", 3)
	document := Document{
		SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
		DocumentID: "doc.references", RevisionID: "rev.references", Title: "Castle and regional order", Language: "en",
		Blocks: []Block{
			{NodeID: "section.opening", Kind: "section", Level: 2, Title: "Castle and region"},
			{NodeID: "prose.opening", ParentNodeID: "section.opening", Kind: "prose", Prose: longParagraph},
			{NodeID: "section.meaning", Kind: "section", Level: 2, Title: "Historical meaning"},
			{NodeID: "prose.meaning", ParentNodeID: "section.meaning", Kind: "prose", Prose: conclusion},
		},
	}
	for index := 1; index <= 6; index++ {
		document.References = append(document.References, Reference{
			RefID:        fmt.Sprintf("ref.%d", index),
			Kind:         "footnote",
			Target:       fmt.Sprintf("accepted-source:%03d", index),
			VisibleLabel: fmt.Sprintf("Official history source %d - castle, silver mine, and route survey", index),
			Locator:      "Frozen source catalog",
		})
	}
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`.report-tail{break-inside:avoid-page}section[aria-labelledby="references"]{break-before:auto;font-size:.86em`,
		`section[aria-labelledby="references"] ol{margin:.25em 0 0;padding-left:1.5em;columns:2;column-gap:2.2em}`,
		`section[aria-labelledby="references"] li:nth-last-child(2){break-after:avoid-page}`,
	} {
		if !strings.Contains(string(html), required) {
			t.Fatalf("print CSS lacks %q", required)
		}
	}
	result, err := RenderPDF(context.Background(), html, chromePath)
	if err != nil {
		t.Fatal(err)
	}
	pageTexts := pdfKitPageTexts(t, result.Content)
	if len(pageTexts) < 2 {
		t.Fatalf("pagination fixture produced %d pages", len(pageTexts))
	}
	last := pageTexts[len(pageTexts)-1]
	if !strings.Contains(last, "Official history source 6") {
		t.Fatalf("final PDF page does not exercise the reference tail: %q", last)
	}
	if !strings.Contains(last, "Official history source 5") {
		t.Fatalf("final PDF page isolates one reference: %q", last)
	}
	if !strings.Contains(last, "The final judgment connects the castle") {
		t.Fatalf("final PDF page contains references without report prose: %q", last)
	}
}

func TestReportFirstAuthoredContentSurvivesPDFProjection(t *testing.T) {
	chromePath := installedChromePath(t)
	catalog := testSourceCatalog(t, "mis_report_first_pdf")
	draft := reportFirstAuthorDraft{
		Title:    "Takeda Castle and Ikuno Silver Mine",
		Language: "en",
		Sections: []reportFirstSectionDraft{
			{
				Title: "A mountain position in regional history",
				Blocks: []reportFirstBlockDraft{
					{
						Kind:               "prose",
						Prose:              "Takeda Castle matters because defense, routes, and silver production met in one regional setting.",
						EvidenceSourceKeys: []string{"source_001"},
					},
					{
						Kind:               "list",
						Items:              []string{"The traditions name 1433 and 1443.", "Stone walls reshaped the site after 1585."},
						EvidenceSourceKeys: []string{"source_001"},
					},
					{
						Kind: "table",
						Table: &documentTableDraft{
							Caption: stringPointer("Chronology at a glance"),
							Column1: "Date",
							Column2: "Recorded development",
							Rows: []documentTableRowDraft{
								{Cell1: "1585", Cell2: "A new phase of stone construction"},
							},
						},
						EvidenceSourceKeys: []string{"source_001"},
					},
				},
			},
			{
				Title: "The evidence boundary and final judgment",
				Blocks: []reportFirstBlockDraft{
					{
						Kind:               "prose",
						Prose:              "The surviving evidence does not prove direct mine administration by the castle.",
						EvidenceSourceKeys: []string{"source_001"},
					},
					{
						Kind:               "prose",
						Prose:              "Its stronger significance is the visible convergence of military position, movement, and production.",
						EvidenceSourceKeys: []string{"source_001"},
					},
				},
			},
		},
	}
	_, document, err := compileReportFirstAuthorDraft(
		draft,
		"narrative_pdf",
		"doc_pdf",
		"Explain the relationship between Takeda Castle and Ikuno Silver Mine.",
		catalog,
		testSourceReadReceipt(catalog),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	document.RevisionID = "rev_pdf"
	html, _, err := RenderHTML(document)
	if err != nil {
		t.Fatal(err)
	}
	result, err := RenderPDF(context.Background(), html, chromePath)
	if err != nil {
		t.Fatal(err)
	}
	pdfText := strings.Join(pdfKitPageTexts(t, result.Content), "\n")
	compactPDFText := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, pdfText)
	for _, authoredValue := range []string{
		draft.Title,
		draft.Sections[0].Title,
		draft.Sections[0].Blocks[0].Prose,
		draft.Sections[0].Blocks[1].Items[0],
		*draft.Sections[0].Blocks[2].Table.Caption,
		draft.Sections[0].Blocks[2].Table.Rows[0].Cell2,
		draft.Sections[1].Blocks[0].Prose,
		draft.Sections[1].Blocks[1].Prose,
	} {
		compactAuthoredValue := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, authoredValue)
		if !strings.Contains(compactPDFText, compactAuthoredValue) {
			t.Fatalf("PDF projection lost authored value %q in %q", authoredValue, pdfText)
		}
	}
}
