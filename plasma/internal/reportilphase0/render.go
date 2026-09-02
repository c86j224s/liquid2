package reportilphase0

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportmathassets"
	"github.com/c86j224s/liquid2/plasma/internal/reportmermaidassets"
)

// RenderMarkdown creates the linear manuscript projection. Renderers preserve
// authored prose bytes and only add target syntax around typed structure.
func RenderMarkdown(document Document) ([]byte, []DegradationReceipt, error) {
	var out strings.Builder
	out.WriteString("# ")
	out.WriteString(document.Title)
	out.WriteString("\n\n")
	receipts := []DegradationReceipt{}
	readerEvidence := readerEvidenceRefs(document)
	for _, block := range document.Blocks {
		evidenceRefs := readerEvidence[block.NodeID]
		evidenceRendered := false
		switch block.Kind {
		case "section":
			out.WriteString(strings.Repeat("#", block.Level))
			out.WriteByte(' ')
			out.WriteString(block.Title)
			out.WriteString("\n\n")
		case "prose":
			out.WriteString(block.Prose)
			renderMarkdownInlineEvidenceRefs(&out, evidenceRefs, document.References)
			evidenceRendered = true
			out.WriteString("\n\n")
		case "list":
			for index, item := range block.Items {
				out.WriteString("- ")
				out.WriteString(item)
				if index == len(block.Items)-1 {
					renderMarkdownInlineEvidenceRefs(&out, evidenceRefs, document.References)
					evidenceRendered = true
				}
				out.WriteByte('\n')
			}
			out.WriteByte('\n')
		case "quote":
			lines := strings.Split(block.Prose, "\n")
			for index, line := range lines {
				out.WriteString("> ")
				out.WriteString(line)
				if index == len(lines)-1 {
					renderMarkdownInlineEvidenceRefs(&out, evidenceRefs, document.References)
					evidenceRendered = true
				}
				out.WriteByte('\n')
			}
			out.WriteByte('\n')
		case "callout":
			out.WriteString("> **참고:** ")
			out.WriteString(block.Prose)
			renderMarkdownInlineEvidenceRefs(&out, evidenceRefs, document.References)
			evidenceRendered = true
			out.WriteString("\n\n")
			receipts = append(receipts, DegradationReceipt{Target: "markdown", NodeID: block.NodeID, Capability: "callout", Outcome: "transformed", Reason: "callout lowered to a labeled block quote"})
		case "code":
			fence := markdownFence(block.Code)
			out.WriteString(fence)
			out.WriteString(block.Language)
			out.WriteByte('\n')
			out.WriteString(block.Code)
			if !strings.HasSuffix(block.Code, "\n") {
				out.WriteByte('\n')
			}
			out.WriteString(fence)
			out.WriteString("\n\n")
		case "table":
			renderMarkdownTable(&out, block.Table)
		case "equation":
			out.WriteString("$$\n")
			out.WriteString(normalizeDisplayLaTeX(block.Equation.Expression))
			out.WriteString("\n$$\n\n")
		case "figure":
			asset := findAsset(document.Assets, block.Figure.AssetID)
			out.WriteString("![")
			out.WriteString(escapeMarkdownLabel(block.Figure.Alt))
			if asset.ArtifactID != "" {
				out.WriteString("](artifact:")
				out.WriteString(asset.ArtifactID)
				out.WriteString(")")
			} else {
				out.WriteString("](data:")
				out.WriteString(asset.MediaType)
				out.WriteString(";base64,")
				out.WriteString(asset.DataBase64)
				out.WriteString(")")
			}
			out.WriteString("\n\n*")
			out.WriteString(block.Figure.Caption)
			if asset.SourcePageURL != "" {
				out.WriteString(" — [출처](")
				out.WriteString(asset.SourcePageURL)
				out.WriteString(")")
			}
			out.WriteString("*\n\n")
		case "raw":
			fence := markdownFence(string(block.Extension))
			out.WriteString(fence)
			out.WriteString("json\n")
			out.Write(block.Extension)
			out.WriteByte('\n')
			out.WriteString(fence)
			out.WriteString("\n\n")
			receipts = append(receipts, DegradationReceipt{Target: "markdown", NodeID: block.NodeID, Capability: "raw_extension", Outcome: "fallback", Reason: "opaque extension preserved as JSON code"})
		default:
			return nil, nil, fmt.Errorf("Markdown renderer does not support block kind %q", block.Kind)
		}
		if !evidenceRendered {
			renderMarkdownEvidenceRefs(&out, evidenceRefs, document.References)
		}
	}
	receipts = append(receipts, renderMarkdownReferences(&out, document.References, document.Language)...)
	return []byte(out.String()), receipts, nil
}

// RenderHTML materializes static semantic HTML with embedded assets and no
// executable runtime or external subresource dependency.
func RenderHTML(document Document) ([]byte, []DegradationReceipt, error) {
	var body strings.Builder
	receipts := []DegradationReceipt{}
	readerEvidence := readerEvidenceRefs(document)
	lastContentIndex := -1
	if len(document.References) > 0 {
		for index := len(document.Blocks) - 1; index >= 0; index-- {
			if document.Blocks[index].Kind != "section" {
				lastContentIndex = index
				break
			}
		}
	}
	for blockIndex, block := range document.Blocks {
		if blockIndex == lastContentIndex {
			body.WriteString(`<div class="report-tail">`)
		}
		evidenceRefs := readerEvidence[block.NodeID]
		body.WriteString(`<div id="`)
		body.WriteString(html.EscapeString(block.NodeID))
		body.WriteString(`" data-node-id="`)
		body.WriteString(html.EscapeString(block.NodeID))
		body.WriteString(`">`)
		switch block.Kind {
		case "section":
			fmt.Fprintf(&body, "<h%d>%s</h%d>", block.Level, html.EscapeString(block.Title), block.Level)
		case "prose":
			body.WriteString("<p>")
			body.WriteString(renderProseHTML(block.Prose))
			renderHTMLEvidenceRefs(&body, evidenceRefs, document.References, document.Language)
			body.WriteString("</p>")
		case "list":
			body.WriteString("<ul>")
			for _, item := range block.Items {
				body.WriteString("<li>")
				body.WriteString(html.EscapeString(item))
				body.WriteString("</li>")
			}
			body.WriteString("</ul>")
		case "quote":
			body.WriteString("<blockquote><p>")
			body.WriteString(renderProseHTML(block.Prose))
			body.WriteString("</p></blockquote>")
		case "callout":
			body.WriteString(`<aside class="callout" aria-label="참고"><p>`)
			body.WriteString(renderProseHTML(block.Prose))
			body.WriteString("</p></aside>")
		case "code":
			body.WriteString(`<pre><code`)
			if block.Language != "" {
				body.WriteString(` class="language-`)
				body.WriteString(html.EscapeString(block.Language))
				body.WriteString(`"`)
			}
			body.WriteString(">")
			body.WriteString(html.EscapeString(block.Code))
			body.WriteString("</code></pre>")
		case "table":
			renderHTMLTable(&body, block.Table)
		case "equation":
			expression := normalizeDisplayLaTeX(block.Equation.Expression)
			body.WriteString(`<div class="equation" role="math" aria-label="Mathematical equation" data-tex="`)
			body.WriteString(html.EscapeString(expression))
			body.WriteString(`"><code>`)
			body.WriteString(html.EscapeString(expression))
			body.WriteString(`</code></div>`)
		case "figure":
			asset := findAsset(document.Assets, block.Figure.AssetID)
			body.WriteString("<figure><img src=\"data:")
			body.WriteString(html.EscapeString(asset.MediaType))
			body.WriteString(";base64,")
			body.WriteString(asset.DataBase64)
			body.WriteString(`" alt="`)
			body.WriteString(html.EscapeString(block.Figure.Alt))
			body.WriteString(`"><figcaption>`)
			body.WriteString(html.EscapeString(block.Figure.Caption))
			body.WriteString("</figcaption></figure>")
		case "raw":
			body.WriteString(`<pre class="opaque-extension"><code>`)
			body.WriteString(html.EscapeString(string(block.Extension)))
			body.WriteString("</code></pre>")
			receipts = append(receipts, DegradationReceipt{Target: "html", NodeID: block.NodeID, Capability: "raw_extension", Outcome: "fallback", Reason: "opaque extension preserved as escaped JSON"})
		default:
			return nil, nil, fmt.Errorf("HTML renderer does not support block kind %q", block.Kind)
		}
		if block.Kind != "prose" {
			renderHTMLEvidenceRefs(&body, evidenceRefs, document.References, document.Language)
		}
		body.WriteString("</div>\n")
	}
	renderHTMLReferences(&body, document.References, document.Language)
	if lastContentIndex >= 0 {
		body.WriteString("</div>\n")
	}

	bodyHTML := body.String()
	hasEquations := strings.Contains(bodyHTML, `class="equation"`)
	hasMermaid := strings.Contains(bodyHTML, `class="language-mermaid"`)
	var out bytes.Buffer
	out.WriteString("<!doctype html>\n<html lang=\"")
	out.WriteString(html.EscapeString(document.Language))
	out.WriteString("\"><head><meta charset=\"utf-8\"><meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; style-src 'unsafe-inline'; ")
	if hasEquations || hasMermaid {
		out.WriteString("script-src 'unsafe-inline'; ")
	}
	out.WriteString("img-src data:; font-src data:; base-uri 'none'; form-action 'none'\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><meta name=\"color-scheme\" content=\"light dark\"><title>")
	out.WriteString(html.EscapeString(document.Title))
	out.WriteString("</title><style>")
	out.WriteString(staticCSS)
	if hasMermaid {
		out.WriteString(mermaidCompatCSS)
		out.Write(reportmermaidassets.Stylesheet())
	}
	out.WriteString("</style></head><body data-math-rendered=\"")
	if hasEquations {
		out.WriteString("false")
	} else {
		out.WriteString("true")
	}
	out.WriteString("\" data-mermaid-rendered=\"")
	if hasMermaid {
		out.WriteString("false")
	} else {
		out.WriteString("true")
	}
	out.WriteString("\"><main><h1>")
	out.WriteString(html.EscapeString(document.Title))
	out.WriteString("</h1>\n")
	out.WriteString(bodyHTML)
	out.WriteString("</main>")
	if hasEquations {
		writeInlineScript(&out, reportmathassets.KaTeXRuntime())
		writeInlineScript(&out, []byte(equationBootstrap))
	}
	if hasMermaid {
		writeInlineScript(&out, reportmermaidassets.DOMPurifyRuntime())
		writeInlineScript(&out, []byte(`window.Plasma={reports:{}};`))
		writeInlineScript(&out, reportmermaidassets.MermaidRuntime())
		writeInlineScript(&out, reportmermaidassets.LegendRenderer())
		writeInlineScript(&out, reportmermaidassets.Renderer())
		writeInlineScript(&out, []byte(mermaidBootstrap))
	}
	out.WriteString("</body></html>\n")
	return out.Bytes(), receipts, nil
}

func normalizeDisplayLaTeX(expression string) string {
	normalized := strings.TrimSpace(expression)
	if strings.HasPrefix(normalized, `\[`) && strings.HasSuffix(normalized, `\]`) {
		return strings.TrimSpace(normalized[2 : len(normalized)-2])
	}
	return normalized
}

const equationBootstrap = `(function(){const body=document.body;try{document.querySelectorAll(".equation[data-tex]").forEach((node)=>{const tex=node.dataset.tex||"";node.innerHTML=katex.renderToString(tex,{displayMode:true,throwOnError:true,trust:false,output:"mathml",maxSize:20,maxExpand:1000});const math=node.querySelector("math");if(math){math.setAttribute("data-fit-scale","1");const available=node.clientWidth;const overflow=math.scrollWidth;if(overflow>available&&available>0){const scale=Math.max(.42,available/overflow);math.style.zoom=String(scale);math.setAttribute("data-fit-scale",String(scale))}}node.removeAttribute("data-tex")});body.dataset.mathRendered="true"}catch(error){body.dataset.mathRendered="error";body.dataset.mathError=String(error&&error.message||error)}})();`

const mermaidCompatCSS = `:root{--line2:#bdd0e4;--surface:#fff;--danger:#b91c1c;--muted:#4b5d75}.plasma-mermaid-diagram{color:#172033}@media(prefers-color-scheme:dark){:root{--line2:#66552e;--surface:#151821;--danger:#fca5a5;--muted:#d8ceb7}.plasma-mermaid-diagram{color:#f7f2e5}}`

const mermaidBootstrap = `(function(){const body=document.body;Promise.resolve(window.Plasma?.reports?.renderPlasmaMermaid?.(document.body)).then((rendered)=>{if(rendered!==true)throw new Error("one or more Mermaid diagrams failed");if(document.querySelector("pre > code.language-mermaid, pre > code.lang-mermaid"))throw new Error("raw Mermaid blocks remain");if(!document.querySelector(".plasma-mermaid-diagram svg"))throw new Error("Mermaid renderer returned no SVG");body.dataset.mermaidRendered="true"}).catch((error)=>{body.dataset.mermaidRendered="error";body.dataset.mermaidError=String(error&&error.message||error)})})();`

func writeInlineScript(out *bytes.Buffer, script []byte) {
	out.WriteString("<script>")
	replacer := strings.NewReplacer("</script", `<\/script`, "</SCRIPT", `<\/SCRIPT`)
	out.WriteString(replacer.Replace(string(script)))
	out.WriteString("</script>")
}

func renderMarkdownTable(out *strings.Builder, table *Table) {
	if table.Caption != "" {
		out.WriteString("**")
		out.WriteString(table.Caption)
		out.WriteString("**\n\n")
	}
	out.WriteString("| ")
	out.WriteString(strings.Join(escapedTableCells(table.Columns), " | "))
	out.WriteString(" |\n|")
	for range table.Columns {
		out.WriteString(" --- |")
	}
	out.WriteByte('\n')
	for _, row := range table.Rows {
		out.WriteString("| ")
		out.WriteString(strings.Join(escapedTableCells(row), " | "))
		out.WriteString(" |\n")
	}
	out.WriteByte('\n')
}

func renderHTMLTable(out *strings.Builder, table *Table) {
	label := strings.TrimSpace(table.Caption)
	if label == "" {
		label = "Data table"
	}
	out.WriteString(`<div class="table-scroll" role="region" aria-label="`)
	out.WriteString(html.EscapeString(label))
	out.WriteString(`" tabindex="0" data-columns="`)
	fmt.Fprintf(out, "%d", len(table.Columns))
	out.WriteString(`"><table>`)
	if table.Caption != "" {
		out.WriteString("<caption>")
		out.WriteString(html.EscapeString(table.Caption))
		out.WriteString("</caption>")
	}
	out.WriteString("<thead><tr>")
	for _, column := range table.Columns {
		out.WriteString(`<th scope="col">`)
		out.WriteString(html.EscapeString(column))
		out.WriteString("</th>")
	}
	out.WriteString("</tr></thead><tbody>")
	for _, row := range table.Rows {
		out.WriteString("<tr>")
		for _, cell := range row {
			out.WriteString("<td>")
			out.WriteString(html.EscapeString(cell))
			out.WriteString("</td>")
		}
		out.WriteString("</tr>")
	}
	out.WriteString("</tbody></table></div>")
}

func renderMarkdownInlineEvidenceRefs(out *strings.Builder, refIDs []string, references []Reference) {
	if len(refIDs) == 0 {
		return
	}
	out.WriteString("<sup>")
	for index, refID := range refIDs {
		if index > 0 {
			out.WriteString(", ")
		}
		out.WriteString("[")
		out.WriteString(referenceMarker(references, refID))
		out.WriteString("]")
	}
	out.WriteString("</sup>")
}

func renderMarkdownEvidenceRefs(out *strings.Builder, refIDs []string, references []Reference) {
	if len(refIDs) == 0 {
		return
	}
	renderMarkdownInlineEvidenceRefs(out, refIDs, references)
	out.WriteString("\n\n")
}

// readerEvidenceRefs lowers immutable block-level provenance into one compact,
// ordered citation group at the end of each reader-facing section.
func readerEvidenceRefs(document Document) map[string][]string {
	refsByBlock := make(map[string][]string)
	var sectionBlocks []Block
	flush := func() {
		if len(sectionBlocks) == 0 {
			return
		}
		seen := map[string]bool{}
		for _, block := range sectionBlocks {
			for _, refID := range block.EvidenceRefs {
				seen[refID] = true
			}
		}
		refs := make([]string, 0, len(seen))
		for _, ref := range document.References {
			if seen[ref.RefID] {
				refs = append(refs, ref.RefID)
			}
		}
		if len(refs) > 0 {
			target := sectionBlocks[len(sectionBlocks)-1]
			refsByBlock[target.NodeID] = refs
		}
		sectionBlocks = nil
	}
	for _, block := range document.Blocks {
		if block.Kind == "section" {
			flush()
			continue
		}
		sectionBlocks = append(sectionBlocks, block)
	}
	flush()
	return refsByBlock
}

func renderHTMLEvidenceRefs(out *strings.Builder, refIDs []string, references []Reference, language string) {
	if len(refIDs) == 0 {
		return
	}
	out.WriteString(`<sup class="evidence-refs" aria-label="`)
	out.WriteString(html.EscapeString(referenceHeading(language)))
	out.WriteString(`">`)
	for index, refID := range refIDs {
		if index > 0 {
			out.WriteString(", ")
		}
		out.WriteString(`<a href="#`)
		out.WriteString(html.EscapeString(refID))
		out.WriteString(`" title="`)
		out.WriteString(html.EscapeString(referenceLabel(references, refID)))
		out.WriteString(`">`)
		out.WriteString(referenceMarker(references, refID))
		out.WriteString(`</a>`)
	}
	out.WriteString("</sup>")
}

func referenceMarker(references []Reference, refID string) string {
	for index, ref := range references {
		if ref.RefID == refID {
			return fmt.Sprintf("%d", index+1)
		}
	}
	return "?"
}

func referenceHeading(language string) string {
	if strings.EqualFold(language, "ko") || strings.HasPrefix(strings.ToLower(language), "ko-") {
		return "근거"
	}
	return "References"
}

func referenceLabel(references []Reference, refID string) string {
	for _, ref := range references {
		if ref.RefID == refID {
			if ref.VisibleLabel != "" {
				return ref.VisibleLabel
			}
			return ref.RefID
		}
	}
	return refID
}

func renderMarkdownReferences(out *strings.Builder, references []Reference, language string) []DegradationReceipt {
	if len(references) == 0 {
		return nil
	}
	receipts := []DegradationReceipt{}
	out.WriteString("## ")
	out.WriteString(referenceHeading(language))
	out.WriteString("\n\n")
	for index, ref := range references {
		label := ref.VisibleLabel
		if label == "" {
			label = ref.RefID
		}
		fmt.Fprintf(out, "%d. ", index+1)
		switch ref.Kind {
		case "cross_reference":
			out.WriteString(escapeMarkdownLabel(label))
			out.WriteString(" — document node `")
			out.WriteString(ref.Target)
			out.WriteByte('`')
			receipts = append(receipts, DegradationReceipt{Target: "markdown", NodeID: ref.Target, Capability: "stable_cross_reference", Outcome: "fallback", Reason: "portable Markdown has no target-independent stable node anchor"})
		case "footnote":
			if publicURL := publicCitationURL(ref.Locator); publicURL != "" {
				out.WriteString("[")
				out.WriteString(escapeMarkdownLabel(label))
				out.WriteString("](")
				out.WriteString(escapeMarkdownDestination(publicURL))
				out.WriteByte(')')
			} else {
				out.WriteString(escapeMarkdownLabel(label))
			}
		default:
			out.WriteString("[")
			out.WriteString(escapeMarkdownLabel(label))
			out.WriteString("](")
			out.WriteString(escapeMarkdownDestination(ref.Target))
			out.WriteByte(')')
		}
		out.WriteByte('\n')
	}
	out.WriteByte('\n')
	return receipts
}

func renderHTMLReferences(out *strings.Builder, references []Reference, language string) {
	if len(references) == 0 {
		return
	}
	out.WriteString(`<section aria-labelledby="references"><h2 id="references">`)
	out.WriteString(html.EscapeString(referenceHeading(language)))
	out.WriteString(`</h2><ol>`)
	for _, ref := range references {
		label := ref.VisibleLabel
		if label == "" {
			label = ref.RefID
		}
		target := safeReferenceURL(ref.Target)
		if ref.Kind == "cross_reference" {
			target = "#" + ref.Target
		}
		out.WriteString(`<li id="`)
		out.WriteString(html.EscapeString(ref.RefID))
		out.WriteString(`">`)
		if ref.Kind == "footnote" {
			if publicURL := publicCitationURL(ref.Locator); publicURL != "" {
				out.WriteString(`<a href="`)
				out.WriteString(html.EscapeString(publicURL))
				out.WriteString(`">`)
				out.WriteString(html.EscapeString(label))
				out.WriteString("</a>")
			} else {
				out.WriteString(html.EscapeString(label))
			}
		} else {
			out.WriteString(`<a href="`)
			out.WriteString(html.EscapeString(target))
			out.WriteString(`">`)
			out.WriteString(html.EscapeString(label))
			out.WriteString("</a>")
		}
		out.WriteString("</li>")
	}
	out.WriteString("</ol></section>")
}

func findAsset(assets []Asset, assetID string) Asset {
	for _, asset := range assets {
		if asset.AssetID == assetID {
			return asset
		}
	}
	panic("validated asset missing: " + assetID)
}

func renderProseHTML(prose string) string {
	return strings.ReplaceAll(html.EscapeString(prose), "\n", "<br>\n")
}

func markdownFence(code string) string {
	longest := 0
	current := 0
	for _, r := range code {
		if r == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	if longest < 3 {
		longest = 3
	} else {
		longest++
	}
	return strings.Repeat("`", longest)
}

func escapedTableCells(cells []string) []string {
	out := make([]string, len(cells))
	for index, cell := range cells {
		out[index] = strings.ReplaceAll(strings.ReplaceAll(cell, "|", `\|`), "\n", "<br>")
	}
	return out
}

func escapeMarkdownLabel(value string) string {
	return strings.NewReplacer("\\", `\\`, "[", `\[`, "]", `\]`).Replace(value)
}

func escapeMarkdownDestination(value string) string {
	return strings.NewReplacer("\\", `%5C`, "(", `%28`, ")", `%29`, " ", `%20`).Replace(value)
}

func safeReferenceURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "#"
	}
	return parsed.String()
}

const staticCSS = `
:root{font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#13213c;background:#f7faff;line-height:1.65}
body{margin:0;background:#f7faff;color:#172033}main{max-width:850px;margin:0 auto;padding:64px 42px 96px;background:#fff;box-shadow:0 18px 60px rgba(36,74,130,.10)}
h1,h2,h3,h4,h5,h6{color:#123d78;line-height:1.25;margin:1.8em 0 .65em}h1{font-size:2.4rem;margin-top:0}h2{color:#1d5da3}h3{color:#3379bb}
p{margin:.8em 0}a{color:#1c65aa}.evidence-refs{margin-left:.18em;color:#4b5d75;font-size:.72em;line-height:0}.evidence-refs a{white-space:nowrap;text-decoration:none}blockquote,.callout{margin:1.2em 0;padding:.8em 1.1em;border-left:4px solid #559bd3;background:#edf6ff}pre{overflow:auto;padding:1em;background:#0f172a;color:#f8fafc;border-radius:8px}code{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}.equation{margin:1.35em 0;padding:.85em 1em;overflow-x:auto;text-align:center;background:#f3f7fc;border:1px solid #c9d9eb;border-radius:8px}.equation math{display:block;max-width:100%;font-size:1.08em}.equation code{display:block;font-size:1.04em;white-space:pre}
.table-scroll{max-width:100%;margin:1.4em 0;overflow-x:auto;overscroll-behavior-inline:contain;-webkit-overflow-scrolling:touch}.table-scroll:focus-visible{outline:3px solid #559bd3;outline-offset:3px}table{border-collapse:collapse;width:100%;margin:0}caption{font-weight:700;margin-bottom:.5em}th,td{border:1px solid #bdd0e4;padding:.55em;text-align:left;vertical-align:top}th{background:#e8f2fc}figure{margin:1.6em 0}img{display:block;max-width:100%;height:auto;margin:auto}figcaption{margin-top:.55em;text-align:center;color:#4b5d75}
@media(max-width:640px){main{padding:32px 20px;box-shadow:none}h1{font-size:1.9rem}.table-scroll[data-columns="3"] table{min-width:42rem}.table-scroll[data-columns="4"] table{min-width:56rem}.table-scroll th,.table-scroll td{min-width:11rem}}
@media(prefers-color-scheme:dark){:root,body{background:#080b12;color:#f7f2e5}main{background:#10141c;box-shadow:none}h1,h2,h3,h4,h5,h6{color:#d9ad52}a{color:#efc66d}blockquote,.callout{background:#171b24;border-left-color:#c99a3f}.equation{background:#151821;border-color:#66552e}.table-scroll:focus-visible{outline-color:#d9ad52}th,td{border-color:#66552e}th{background:#221e16}figcaption,.evidence-refs{color:#d8ceb7}}
@media print{:root,body{font-family:"Nanum Gothic","Noto Sans CJK KR","Noto Sans KR","Arial Unicode MS",Arial,sans-serif;background:#fff;color:#111}main{max-width:none;margin:0;padding:0;box-shadow:none}a{color:#111;text-decoration:none}p{orphans:3;widows:3}.evidence-refs{font-size:.68em}pre{font-size:.78em;line-height:1.42;white-space:pre-wrap;overflow-wrap:anywhere}.equation{overflow:visible}.equation math{font-size:1em}.table-scroll{max-width:none;margin:1.4em 0;overflow:visible}.table-scroll[data-columns] table{width:100%;min-width:0}.table-scroll th,.table-scroll td{min-width:0}figure,table,.equation{break-inside:avoid;page-break-inside:avoid}h1,h2,h3{break-after:avoid}.report-tail{break-inside:avoid-page}section[aria-labelledby="references"]{break-before:auto;font-size:.86em;line-height:1.42}section[aria-labelledby="references"] h2{margin-top:1.2em;margin-bottom:.45em}section[aria-labelledby="references"] ol{margin:.25em 0 0;padding-left:1.5em;columns:2;column-gap:2.2em}section[aria-labelledby="references"] li{margin:.16em 0;break-inside:avoid}section[aria-labelledby="references"] li:nth-last-child(2){break-after:avoid-page}img{max-height:220mm}}
@page{size:A4;margin:18mm 17mm 20mm}
`
