package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	htmlpkg "html"
	"net/url"
	"strconv"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (server *Server) renderDesignedReportHTML(sourceArtifact app.RawArtifact, model designedReportContentModel, images []reportInlineImage, notes []string) ([]byte, error) {
	mathHead, err := selfContainedMathHead()
	if err != nil {
		return nil, err
	}
	mathScripts, err := selfContainedMathScripts()
	if err != nil {
		return nil, err
	}
	title := firstNonEmpty(model.Title, reportArtifactTitle(sourceArtifact))
	kicker := firstNonEmpty(model.Kicker, model.VisualIdentity.StyleKey, "Designed Report")
	subtitle := firstNonEmpty(model.Subtitle, model.Thesis, "저장된 Markdown 리포트 artifact를 바탕으로 재구성한 self-contained interactive HTML입니다.")
	wordCount := len(strings.Fields(string(sourceArtifact.Content)))
	shape := firstNonEmpty(model.CompositionShape.ShapeKey, "reference_visual_map")
	visualCount := len(model.VisualUnits)
	var out bytes.Buffer
	out.WriteString("<!doctype html>\n<html lang=\"ko\">\n<head>\n<meta charset=\"utf-8\">\n")
	out.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	out.WriteString("<title>" + htmlpkg.EscapeString(title) + "</title>\n")
	out.WriteString(designedReportCSS())
	out.WriteString(mathHead)
	out.WriteString("</head>\n<body>\n")
	out.WriteString("<header class=\"designed-hero visual-map-hero theme-" + htmlpkg.EscapeString(designedStyleClass(model.VisualIdentity.StyleKey)) + "\" id=\"top\"><div class=\"hero-copy\"><p class=\"eyebrow\">" + htmlpkg.EscapeString(kicker) + "</p><h1>" + htmlpkg.EscapeString(title) + "</h1><p class=\"subtitle\">" + htmlpkg.EscapeString(subtitle) + "</p>")
	if model.Thesis != "" {
		out.WriteString("<p class=\"thesis\">" + htmlpkg.EscapeString(model.Thesis) + "</p>")
	}
	if model.VisualIdentity.Motif != "" || model.VisualIdentity.PaletteNote != "" {
		out.WriteString("<p class=\"motif-note\"><strong>" + htmlpkg.EscapeString(model.VisualIdentity.Motif) + "</strong>")
		if model.VisualIdentity.PaletteNote != "" {
			out.WriteString("<span>" + htmlpkg.EscapeString(model.VisualIdentity.PaletteNote) + "</span>")
		}
		out.WriteString("</p>")
	}
	out.WriteString("</div>")
	renderDesignedHeroVisual(&out, model)
	out.WriteString("</header>\n")
	out.WriteString("<main class=\"designed-shell\">\n")
	out.WriteString("<aside class=\"designed-rail\"><button id=\"themeToggle\" type=\"button\">테마 전환</button><div class=\"metric\"><span>리포트 단어</span><strong>" + strconv.Itoa(wordCount) + "</strong></div><div class=\"metric\"><span>탭</span><strong>" + strconv.Itoa(len(model.Tabs)) + "</strong></div><div class=\"metric\"><span>시각 단위</span><strong>" + strconv.Itoa(visualCount) + "</strong></div><div class=\"metric\"><span>형태</span><strong class=\"metric-text\">" + htmlpkg.EscapeString(designedShapeLabel(shape)) + "</strong></div><nav>")
	out.WriteString("<a href=\"#overview\">개요</a>")
	for index, tab := range model.Tabs {
		label := firstNonEmpty(tab.Label, fmt.Sprintf("섹션 %d", index+1))
		out.WriteString("<a href=\"#tab-" + strconv.Itoa(index+1) + "\">" + htmlpkg.EscapeString(label) + "</a>")
	}
	if len(images) > 0 {
		out.WriteString("<a href=\"#media\">미디어</a>")
	}
	out.WriteString("<a href=\"#sources\">출처와 한계</a>")
	out.WriteString("</nav></aside>\n")
	out.WriteString("<section class=\"designed-content\">\n")
	renderDesignedOverview(&out, model)
	if len(model.VisualUnits) > 1 {
		renderDesignedVisualUnits(&out, model.VisualUnits[1:])
	}
	imageByRef := designedInlineImageMap(images)
	usedImages := map[string]bool{}
	renderDesignedTabs(&out, model.Tabs, imageByRef, usedImages)
	renderDesignedMedia(&out, remainingDesignedInlineImages(images, usedImages), len(images))
	renderDesignedSources(&out, model, sourceArtifact, notes)
	out.WriteString("</section>\n</main>\n")
	out.WriteString("<script>const b=document.body,t=document.getElementById('themeToggle');t?.addEventListener('click',()=>b.classList.toggle('light'));document.querySelectorAll('[data-tab-target]').forEach(btn=>btn.addEventListener('click',()=>{document.querySelector(btn.dataset.tabTarget)?.scrollIntoView({behavior:'smooth',block:'start'});}));</script>\n")
	out.WriteString(mathScripts)
	out.WriteString("</body>\n</html>\n")
	return out.Bytes(), nil
}

func renderDesignedHeroVisual(out *bytes.Buffer, model designedReportContentModel) {
	lead := designedLeadVisual(model)
	nodes := lead.Nodes
	title := firstNonEmpty(lead.Title, model.HeroVisual.Title, "핵심 관계도")
	left := firstNonEmpty(model.HeroVisual.LeftLabel, "맥락")
	right := firstNonEmpty(model.HeroVisual.RightLabel, "의미")
	out.WriteString("<div class=\"hero-visual\"><div class=\"visual-labels\"><span>" + htmlpkg.EscapeString(left) + "</span><strong>" + htmlpkg.EscapeString(title) + "</strong><span>" + htmlpkg.EscapeString(right) + "</span></div>")
	if lead.Question != "" {
		out.WriteString("<p class=\"visual-question\">" + htmlpkg.EscapeString(lead.Question) + "</p>")
	}
	if len(nodes) == 0 {
		out.WriteString("<p class=\"muted\">시각화할 핵심 관계가 콘텐츠 모델에 포함되지 않았습니다.</p></div>")
		return
	}
	renderConnectedHeroMap(out, nodes, "hero-map")
	if lead.Caption != "" {
		out.WriteString("<p class=\"caption\">" + htmlpkg.EscapeString(lead.Caption) + "</p>")
	}
	out.WriteString("</div>")
}

func designedLeadVisual(model designedReportContentModel) designedReportVisual {
	if len(model.VisualUnits) > 0 && len(model.VisualUnits[0].Nodes) > 0 {
		return model.VisualUnits[0]
	}
	return designedReportVisual{
		Title:    model.HeroVisual.Title,
		Kind:     "map",
		Question: firstNonEmpty(model.CompositionShape.Rationale, model.Thesis),
		Nodes:    model.HeroVisual.Nodes,
		Caption:  model.HeroVisual.RightLabel,
	}
}

func renderConnectedHeroMap(out *bytes.Buffer, nodes []designedReportNode, id string) {
	if len(nodes) == 0 {
		out.WriteString("<p class=\"muted\">시각화할 노드가 없습니다.</p>")
		return
	}
	width := 880
	height := 420
	count := minInt(len(nodes), 6)
	points := designedHeroMapPoints(count, width, height)
	out.WriteString("<svg class=\"hero-map-svg\" id=\"" + htmlpkg.EscapeString(id) + "\" viewBox=\"0 0 " + strconv.Itoa(width) + " " + strconv.Itoa(height) + "\" role=\"img\" aria-label=\"핵심 관계도\">")
	out.WriteString("<defs><linearGradient id=\"" + htmlpkg.EscapeString(id) + "-line\" x1=\"0\" x2=\"1\"><stop offset=\"0\" stop-color=\"#1f7a68\"/><stop offset=\"1\" stop-color=\"#d78a25\"/></linearGradient><filter id=\"" + htmlpkg.EscapeString(id) + "-shadow\"><feDropShadow dx=\"0\" dy=\"10\" stdDeviation=\"12\" flood-opacity=\"0.16\"/></filter></defs>")
	out.WriteString("<g class=\"hero-map-edges\">")
	for index := 1; index < count; index++ {
		from := points[index-1]
		to := points[index]
		midX := (from.X + to.X) / 2
		out.WriteString("<path d=\"M" + strconv.Itoa(from.X) + " " + strconv.Itoa(from.Y) + " C" + strconv.Itoa(midX) + " " + strconv.Itoa(from.Y) + " " + strconv.Itoa(midX) + " " + strconv.Itoa(to.Y) + " " + strconv.Itoa(to.X) + " " + strconv.Itoa(to.Y) + "\"/>")
	}
	out.WriteString("</g><g class=\"hero-map-nodes\">")
	for index := 0; index < count; index++ {
		node := nodes[index]
		point := points[index]
		tone := normalizeDesignedTone(node.Tone)
		out.WriteString("<g class=\"hero-map-node tone-" + htmlpkg.EscapeString(tone) + "\" transform=\"translate(" + strconv.Itoa(point.X) + " " + strconv.Itoa(point.Y) + ")\">")
		out.WriteString("<rect x=\"-118\" y=\"-48\" width=\"236\" height=\"96\" rx=\"18\"/>")
		out.WriteString("<text class=\"hero-map-index\" x=\"-98\" y=\"-20\">" + fmt.Sprintf("%02d", index+1) + "</text>")
		out.WriteString("<text class=\"hero-map-label\" x=\"-98\" y=\"2\">" + htmlpkg.EscapeString(shortSVGText(node.Label, 16)) + "</text>")
		body := shortSVGText(node.Body, 58)
		lineOne, lineTwo := splitSVGText(body, 30)
		out.WriteString("<text class=\"hero-map-body\" x=\"-98\" y=\"25\">" + htmlpkg.EscapeString(lineOne) + "</text>")
		if lineTwo != "" {
			out.WriteString("<text class=\"hero-map-body\" x=\"-98\" y=\"42\">" + htmlpkg.EscapeString(lineTwo) + "</text>")
		}
		out.WriteString("</g>")
	}
	out.WriteString("</g></svg>")
	out.WriteString("<ol class=\"hero-map-readable\">")
	for index := 0; index < count; index++ {
		node := nodes[index]
		out.WriteString("<li><strong>" + fmt.Sprintf("%02d", index+1) + " " + htmlpkg.EscapeString(node.Label) + "</strong><span>" + htmlpkg.EscapeString(node.Body) + "</span></li>")
	}
	out.WriteString("</ol>")
}

type designedMapPoint struct {
	X int
	Y int
}

func designedHeroMapPoints(count int, width int, height int) []designedMapPoint {
	switch count {
	case 1:
		return []designedMapPoint{{X: width / 2, Y: height / 2}}
	case 2:
		return []designedMapPoint{{X: 170, Y: height / 2}, {X: width - 170, Y: height / 2}}
	case 3:
		return []designedMapPoint{{X: 150, Y: 120}, {X: width / 2, Y: 280}, {X: width - 150, Y: 120}}
	case 4:
		return []designedMapPoint{{X: 150, Y: 110}, {X: 350, Y: 285}, {X: 560, Y: 135}, {X: width - 150, Y: 285}}
	case 5:
		return []designedMapPoint{{X: 145, Y: 112}, {X: 315, Y: 285}, {X: width / 2, Y: 112}, {X: 565, Y: 285}, {X: width - 145, Y: 112}}
	default:
		return []designedMapPoint{{X: 140, Y: 110}, {X: 285, Y: 292}, {X: 425, Y: 112}, {X: 570, Y: 292}, {X: 735, Y: 112}, {X: width - 150, Y: 292}}
	}
}

func shortSVGText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return strings.TrimSpace(string(runes[:maxRunes-1])) + "..."
}

func splitSVGText(value string, limit int) (string, string) {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes), ""
	}
	breakAt := limit
	for index := limit; index > limit/2; index-- {
		if runes[index] == ' ' {
			breakAt = index
			break
		}
	}
	return strings.TrimSpace(string(runes[:breakAt])), strings.TrimSpace(string(runes[breakAt:]))
}

func renderDesignedOverview(out *bytes.Buffer, model designedReportContentModel) {
	out.WriteString("<section id=\"overview\" class=\"overview-panel\"><div class=\"section-heading\"><p>Overview</p><h2>읽기 전에 잡아야 할 중심축</h2></div>")
	if model.Thesis != "" {
		out.WriteString("<p class=\"overview-thesis\">" + htmlpkg.EscapeString(model.Thesis) + "</p>")
	}
	if len(model.Markers) > 0 {
		out.WriteString("<div class=\"marker-grid\">")
		for _, marker := range model.Markers {
			out.WriteString("<div class=\"marker\"><span>" + htmlpkg.EscapeString(marker.Label) + "</span><strong>" + htmlpkg.EscapeString(marker.Value) + "</strong><p>" + htmlpkg.EscapeString(marker.Note) + "</p></div>")
		}
		out.WriteString("</div>")
	}
	if len(model.Tabs) > 0 {
		out.WriteString("<div class=\"tab-jump-row\">")
		for index, tab := range model.Tabs {
			label := firstNonEmpty(tab.Label, fmt.Sprintf("섹션 %d", index+1))
			out.WriteString("<button type=\"button\" data-tab-target=\"#tab-" + strconv.Itoa(index+1) + "\">" + htmlpkg.EscapeString(label) + "</button>")
		}
		out.WriteString("</div>")
	}
	out.WriteString("</section>\n")
}

func renderDesignedVisualUnits(out *bytes.Buffer, visuals []designedReportVisual) {
	if len(visuals) == 0 {
		return
	}
	out.WriteString("<section class=\"visual-stack\" aria-label=\"시각 요약\">")
	for index, visual := range visuals {
		kind := normalizeDesignedVisualKind(visual.Kind)
		out.WriteString("<article class=\"visual-card visual-card-" + htmlpkg.EscapeString(kind) + "\"><div class=\"section-heading\"><p>" + htmlpkg.EscapeString(designedVisualKindLabel(kind)) + "</p><h2>" + htmlpkg.EscapeString(firstNonEmpty(visual.Title, fmt.Sprintf("시각 요약 %d", index+1))) + "</h2></div>")
		if visual.Question != "" {
			out.WriteString("<p class=\"visual-question\">" + htmlpkg.EscapeString(visual.Question) + "</p>")
		}
		renderDesignedVisualGrammar(out, visual, "visual-"+strconv.Itoa(index+1))
		if visual.Caption != "" {
			out.WriteString("<p class=\"caption\">" + htmlpkg.EscapeString(visual.Caption) + "</p>")
		}
		out.WriteString("</article>")
	}
	out.WriteString("</section>\n")
}

func renderRelationshipSVG(out *bytes.Buffer, nodes []designedReportNode, id string) {
	if len(nodes) == 0 {
		out.WriteString("<p class=\"muted\">시각화할 노드가 없습니다.</p>")
		return
	}
	width := 960
	height := 130 + len(nodes)*74
	out.WriteString("<svg class=\"relationship-svg\" id=\"" + htmlpkg.EscapeString(id) + "\" viewBox=\"0 0 " + strconv.Itoa(width) + " " + strconv.Itoa(height) + "\" role=\"img\" aria-label=\"관계도\">")
	out.WriteString("<defs><linearGradient id=\"" + htmlpkg.EscapeString(id) + "-g\" x1=\"0\" x2=\"1\"><stop offset=\"0\" stop-color=\"#2d7ff9\"/><stop offset=\"1\" stop-color=\"#e7a23b\"/></linearGradient></defs>")
	centerX := 120
	for index, node := range nodes {
		y := 80 + index*74
		if index > 0 {
			out.WriteString("<path d=\"M" + strconv.Itoa(centerX) + " " + strconv.Itoa(y-54) + " L" + strconv.Itoa(centerX) + " " + strconv.Itoa(y-22) + "\" stroke=\"url(#" + htmlpkg.EscapeString(id) + "-g)\" stroke-width=\"3\" stroke-linecap=\"round\"/>")
		}
		out.WriteString("<circle cx=\"" + strconv.Itoa(centerX) + "\" cy=\"" + strconv.Itoa(y) + "\" r=\"22\" class=\"svg-node svg-" + htmlpkg.EscapeString(node.Tone) + "\"/>")
		out.WriteString("<text x=\"" + strconv.Itoa(centerX) + "\" y=\"" + strconv.Itoa(y+5) + "\" text-anchor=\"middle\" class=\"svg-index\">" + strconv.Itoa(index+1) + "</text>")
		out.WriteString("<text x=\"180\" y=\"" + strconv.Itoa(y-6) + "\" class=\"svg-label\">" + htmlpkg.EscapeString(node.Label) + "</text>")
		out.WriteString("<foreignObject x=\"180\" y=\"" + strconv.Itoa(y+6) + "\" width=\"720\" height=\"48\"><p xmlns=\"http://www.w3.org/1999/xhtml\" class=\"svg-body\">" + htmlpkg.EscapeString(node.Body) + "</p></foreignObject>")
	}
	out.WriteString("</svg>")
}

func renderDesignedTabs(out *bytes.Buffer, tabs []designedReportTab, imageByRef map[string]reportInlineImage, usedImages map[string]bool) {
	for index, tab := range tabs {
		id := "tab-" + strconv.Itoa(index+1)
		label := firstNonEmpty(tab.Label, fmt.Sprintf("파트 %d", index+1))
		out.WriteString("<section id=\"" + id + "\" class=\"tab-section\"><div class=\"section-heading\"><p>" + htmlpkg.EscapeString(label) + "</p><h2>" + htmlpkg.EscapeString(firstNonEmpty(tab.Question, tab.Summary, label)) + "</h2></div>")
		if tab.Summary != "" {
			out.WriteString("<p class=\"tab-summary\">" + htmlpkg.EscapeString(tab.Summary) + "</p>")
		}
		if tab.Takeaway != "" {
			out.WriteString("<p class=\"takeaway\"><strong>핵심:</strong> " + htmlpkg.EscapeString(tab.Takeaway) + "</p>")
		}
		for sectionIndex, section := range tab.Sections {
			renderDesignedSection(out, section, sectionIndex, imageByRef, usedImages)
		}
		out.WriteString("</section>\n")
	}
}

func renderDesignedSection(out *bytes.Buffer, section designedReportSection, index int, imageByRef map[string]reportInlineImage, usedImages map[string]bool) {
	component := strings.ToLower(strings.TrimSpace(section.Component))
	if component == "" {
		component = "analysis"
	}
	out.WriteString("<article class=\"content-card component-" + htmlpkg.EscapeString(component) + "\"><div class=\"content-card-head\"><span>" + fmt.Sprintf("%02d", index+1) + "</span><h3>" + htmlpkg.EscapeString(firstNonEmpty(section.Heading, "세부 항목")) + "</h3></div>")
	renderDesignedInlineImages(out, section.Images, imageByRef, usedImages, "before_body")
	for _, paragraph := range section.Body {
		out.WriteString("<p>" + htmlpkg.EscapeString(paragraph) + "</p>")
	}
	if len(section.Bullets) > 0 {
		out.WriteString("<ul>")
		for _, bullet := range section.Bullets {
			out.WriteString("<li>" + htmlpkg.EscapeString(bullet) + "</li>")
		}
		out.WriteString("</ul>")
	}
	renderDesignedTable(out, section.Table)
	renderDesignedDiagram(out, section.Diagram)
	renderDesignedInlineImages(out, section.Images, imageByRef, usedImages, "after_body")
	if section.Caveat != "" {
		out.WriteString("<p class=\"caveat\"><strong>주의:</strong> " + htmlpkg.EscapeString(section.Caveat) + "</p>")
	}
	if section.SourceNote != "" {
		out.WriteString("<p class=\"source-note\">" + htmlpkg.EscapeString(section.SourceNote) + "</p>")
	}
	out.WriteString("</article>")
}

func renderDesignedInlineImages(out *bytes.Buffer, placements []designedReportImagePlacement, imageByRef map[string]reportInlineImage, usedImages map[string]bool, position string) {
	if len(placements) == 0 || len(imageByRef) == 0 {
		return
	}
	wrote := false
	for _, placement := range placements {
		if normalizeDesignedImagePlacement(placement.Placement) != position {
			continue
		}
		ref := strings.TrimSpace(placement.ImageRef)
		image, ok := imageByRef[ref]
		if !ok {
			continue
		}
		if usedImages[ref] {
			continue
		}
		if !wrote {
			out.WriteString("<div class=\"inline-image-strip\">")
			wrote = true
		}
		usedImages[ref] = true
		caption := firstNonEmpty(placement.Caption, image.Title, image.Caption())
		out.WriteString("<figure class=\"inline-report-image\"><img loading=\"lazy\" src=\"" + htmlpkg.EscapeString(image.DataURI) + "\" alt=\"" + htmlpkg.EscapeString(image.Title) + "\"><figcaption><strong>" + htmlpkg.EscapeString(image.Title) + "</strong><span>" + htmlpkg.EscapeString(caption) + "</span></figcaption></figure>")
	}
	if wrote {
		out.WriteString("</div>")
	}
}

func normalizeDesignedImagePlacement(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "before_body", "before":
		return "before_body"
	default:
		return "after_body"
	}
}

func renderDesignedTable(out *bytes.Buffer, table designedReportTable) {
	if len(table.Columns) == 0 || len(table.Rows) == 0 {
		return
	}
	out.WriteString("<div class=\"table-wrap\"><table><thead><tr>")
	for _, column := range table.Columns {
		out.WriteString("<th>" + htmlpkg.EscapeString(column) + "</th>")
	}
	out.WriteString("</tr></thead><tbody>")
	for _, row := range table.Rows {
		out.WriteString("<tr>")
		for index := range table.Columns {
			cell := ""
			if index < len(row) {
				cell = row[index]
			}
			out.WriteString("<td>" + htmlpkg.EscapeString(cell) + "</td>")
		}
		out.WriteString("</tr>")
	}
	out.WriteString("</tbody></table></div>")
}

func renderDesignedDiagram(out *bytes.Buffer, diagram designedReportDiagram) {
	if len(diagram.Steps) == 0 {
		return
	}
	out.WriteString("<div class=\"mini-diagram\"><h4>" + htmlpkg.EscapeString(firstNonEmpty(diagram.Title, "흐름")) + "</h4>")
	for _, step := range diagram.Steps {
		out.WriteString("<div class=\"mini-step node-" + htmlpkg.EscapeString(step.Tone) + "\"><strong>" + htmlpkg.EscapeString(step.Label) + "</strong><span>" + htmlpkg.EscapeString(step.Body) + "</span></div>")
	}
	out.WriteString("</div>")
}

func renderDesignedMedia(out *bytes.Buffer, images []reportInlineImage, totalImageCount int) {
	if totalImageCount == 0 {
		return
	}
	out.WriteString("<section id=\"media\" class=\"media-panel\"><div class=\"section-heading\"><p>Media</p><h2>포함된 이미지</h2></div>")
	if len(images) == 0 {
		out.WriteString("<p class=\"muted\">모든 이미지는 관련 본문 섹션 안에 배치되었습니다.</p></section>\n")
		return
	}
	out.WriteString("<div class=\"designed-gallery\">")
	for _, image := range images {
		out.WriteString("<figure><img loading=\"lazy\" src=\"" + htmlpkg.EscapeString(image.DataURI) + "\" alt=\"" + htmlpkg.EscapeString(image.Title) + "\"><figcaption><strong>" + htmlpkg.EscapeString(image.Title) + "</strong><span>" + htmlpkg.EscapeString(image.Caption()) + "</span></figcaption></figure>")
	}
	out.WriteString("</div></section>\n")
}

func renderDesignedSources(out *bytes.Buffer, model designedReportContentModel, sourceArtifact app.RawArtifact, notes []string) {
	out.WriteString("<section id=\"sources\" class=\"sources-panel\"><div class=\"section-heading\"><p>Sources</p><h2>출처와 한계</h2></div>")
	out.WriteString("<div class=\"source-origin\"><span>원본 리포트 artifact</span><code>" + htmlpkg.EscapeString(sourceArtifact.ArtifactID) + "</code></div>")
	if len(model.Sources) > 0 {
		out.WriteString("<div class=\"source-list\">")
		for _, source := range model.Sources {
			label := firstNonEmpty(source.Label, source.Href, "source")
			out.WriteString("<div class=\"source-row\"><strong>")
			if isSafeHTTPURL(source.Href) {
				out.WriteString("<a href=\"" + htmlpkg.EscapeString(source.Href) + "\" rel=\"noreferrer noopener\" target=\"_blank\">" + htmlpkg.EscapeString(label) + "</a>")
			} else {
				out.WriteString(htmlpkg.EscapeString(label))
			}
			out.WriteString("</strong><p>" + htmlpkg.EscapeString(source.Note) + "</p></div>")
		}
		out.WriteString("</div>")
	}
	if len(model.Caveats) > 0 || len(notes) > 0 {
		out.WriteString("<div class=\"caveat-list\"><h3>주의와 생성 노트</h3><ul>")
		for _, caveat := range model.Caveats {
			out.WriteString("<li>" + htmlpkg.EscapeString(caveat) + "</li>")
		}
		for _, note := range notes {
			out.WriteString("<li>" + htmlpkg.EscapeString(note) + "</li>")
		}
		out.WriteString("</ul></div>")
	}
	if len(model.Glossary) > 0 {
		out.WriteString("<div class=\"glossary\"><h3>용어</h3>")
		for _, item := range model.Glossary {
			out.WriteString("<div><dt>" + htmlpkg.EscapeString(item.Term) + "</dt><dd>" + htmlpkg.EscapeString(item.Definition) + "</dd></div>")
		}
		out.WriteString("</div>")
	}
	out.WriteString("</section>\n")
}

func designedStyleClass(styleKey string) string {
	switch strings.ToLower(strings.TrimSpace(styleKey)) {
	case "archive", "blueprint", "newsroom", "cinematic", "product", "atlas":
		return strings.ToLower(strings.TrimSpace(styleKey))
	default:
		return "atlas"
	}
}

func designedShapeLabel(shape string) string {
	switch strings.ToLower(strings.TrimSpace(shape)) {
	case "scroll_narrative":
		return "서사"
	case "decision_dashboard":
		return "판단"
	case "field_guide":
		return "가이드"
	default:
		return "관계도"
	}
}

func isSafeHTTPURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func designedReportCSS() string {
	return `<style>
:root{color-scheme:dark;--bg:#0d1015;--panel:#151b22;--panel2:#1d2630;--ink:#f4efe7;--muted:#aeb8c2;--line:#33404f;--accent:#f0b84b;--accent2:#5fc6b3;--warn:#ee7b6f;--good:#83d18f;--blue:#6ba7ff}
*,*:before,*:after{box-sizing:border-box}
body{margin:0;overflow-x:hidden;background:radial-gradient(circle at 20% 0%,rgba(95,198,179,.14),transparent 28%),linear-gradient(180deg,#0d1015,#12161d 45%,#0d1015);color:var(--ink);font:15px/1.72 "Apple SD Gothic Neo","Noto Sans KR",ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;letter-spacing:0}
body.light{--bg:#f5f2ea;--panel:#fffdf8;--panel2:#f0ece2;--ink:#1d242b;--muted:#626d78;--line:#d6cfc1;--accent:#9b5d13;--accent2:#287467;--warn:#a53a31;--good:#317a3b;--blue:#245da8;background:#f5f2ea}
button{border:1px solid var(--line);background:rgba(255,255,255,.04);color:var(--ink);border-radius:7px;padding:9px 12px;cursor:pointer}
.designed-hero{display:grid;grid-template-columns:minmax(0,.9fr) minmax(420px,1.1fr);gap:30px;align-items:center;min-height:min(68vh,680px);padding:38px clamp(18px,5vw,72px) 30px;border-bottom:1px solid var(--line)}
.visual-map-hero{background:radial-gradient(circle at 76% 18%,rgba(240,184,75,.18),transparent 30rem),linear-gradient(135deg,rgba(95,198,179,.13),transparent 42%),linear-gradient(180deg,#111820,#0d1015)}
body.light .visual-map-hero{background:radial-gradient(circle at 76% 18%,rgba(155,93,19,.15),transparent 28rem),linear-gradient(135deg,rgba(40,116,103,.11),transparent 42%),linear-gradient(180deg,#fffaf0,#f4efe2)}
.eyebrow{margin:0 0 12px;color:var(--accent);text-transform:uppercase;font:800 12px/1.2 ui-monospace,SFMono-Regular,monospace}.hero-copy h1{margin:0;max-width:980px;font-size:clamp(36px,7vw,88px);line-height:.98;letter-spacing:0;overflow-wrap:anywhere}.subtitle,.thesis,.motif-note,.visual-question,.caption,.content-card,.source-row,.hero-map-readable{overflow-wrap:anywhere}.subtitle{max-width:820px;margin:18px 0 0;color:var(--muted);font-size:clamp(16px,2vw,21px)}.thesis{margin:24px 0 0;max-width:840px;padding-left:18px;border-left:3px solid var(--accent);font-size:18px}
.motif-note{display:grid;gap:5px;margin:18px 0 0;padding:12px 14px;border:1px solid var(--line);border-radius:12px;background:rgba(255,255,255,.045);color:var(--muted)}.motif-note strong{color:var(--ink)}.motif-note span{font-size:13px}
.hero-visual{background:linear-gradient(180deg,rgba(255,255,255,.09),rgba(255,255,255,.025));border:1px solid var(--line);border-radius:18px;padding:18px;box-shadow:0 30px 80px rgba(0,0,0,.24);min-width:0}.visual-labels{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:10px}.visual-labels span{color:var(--muted);font-size:12px}.visual-labels strong{color:var(--accent)}
.hero-map-svg{display:block;width:100%;height:auto;min-height:320px;border:1px solid var(--line);border-radius:14px;background:rgba(255,255,255,.035);margin-top:12px}.hero-map-edges path{fill:none;stroke:url(#hero-map-line);stroke-width:4;stroke-linecap:round;opacity:.8}.hero-map-node rect{fill:var(--panel);stroke:var(--line);stroke-width:2;filter:url(#hero-map-shadow)}.hero-map-node.tone-accent rect{stroke:var(--accent)}.hero-map-node.tone-warn rect{stroke:var(--warn)}.hero-map-node.tone-good rect{stroke:var(--good)}.hero-map-index{fill:var(--accent);font:900 13px ui-monospace,monospace}.hero-map-label{fill:var(--ink);font:900 17px "Apple SD Gothic Neo","Noto Sans KR",system-ui}.hero-map-body{fill:var(--muted);font:13px "Apple SD Gothic Neo","Noto Sans KR",system-ui}.hero-map-readable{display:none}
.node-ladder{display:grid;gap:10px}.node{display:grid;grid-template-columns:44px 1fr;gap:12px;padding:12px;border:1px solid var(--line);border-radius:10px;background:rgba(255,255,255,.035)}.node-index{display:grid;place-items:center;border-radius:999px;background:var(--panel2);color:var(--accent);font:800 12px/1 ui-monospace,monospace}.node strong{display:block;margin-bottom:4px}.node p{margin:0;color:var(--muted);font-size:13px}.node-accent{border-color:color-mix(in srgb,var(--accent) 60%,var(--line))}.node-warn{border-color:color-mix(in srgb,var(--warn) 65%,var(--line))}.node-good{border-color:color-mix(in srgb,var(--good) 65%,var(--line))}
.designed-shell{display:grid;grid-template-columns:240px minmax(0,1fr);gap:30px;max-width:1440px;margin:0 auto;padding:28px clamp(16px,4vw,56px) 80px}.designed-rail{position:sticky;top:18px;align-self:start;display:grid;gap:12px}.metric{border:1px solid var(--line);background:var(--panel);border-radius:10px;padding:13px}.metric span{display:block;color:var(--muted);font-size:12px}.metric strong{font-size:28px;color:var(--accent)}.metric .metric-text{font-size:18px}nav{display:grid;gap:7px;margin-top:6px}nav a{color:var(--accent2);text-decoration:none;border-left:2px solid var(--line);padding:5px 0 5px 10px}
.designed-content{display:grid;gap:22px;min-width:0}.overview-panel,.visual-card,.tab-section,.media-panel,.sources-panel{border:1px solid var(--line);background:rgba(21,27,34,.88);border-radius:14px;padding:clamp(20px,3vw,34px)}body.light .overview-panel,body.light .visual-card,body.light .tab-section,body.light .media-panel,body.light .sources-panel{background:var(--panel)}
.section-heading p{margin:0 0 6px;color:var(--accent);font:800 12px/1.2 ui-monospace,monospace;text-transform:uppercase}.section-heading h2{margin:0;font-size:clamp(24px,3vw,42px);line-height:1.08}.overview-thesis,.tab-summary{font-size:18px;color:var(--ink);max-width:900px}.marker-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px;margin-top:22px}.marker{border:1px solid var(--line);border-radius:10px;padding:14px;background:rgba(255,255,255,.035)}.marker span{font-size:12px;color:var(--muted)}.marker strong{display:block;font-size:26px;color:var(--accent);margin:4px 0}.marker p{margin:0;color:var(--muted);font-size:13px}.tab-jump-row{display:flex;flex-wrap:wrap;gap:8px;margin-top:20px}
.visual-stack{display:grid;gap:18px}.visual-question,.caption,.muted{color:var(--muted)}.relationship-svg{display:block;width:100%;height:auto;margin-top:16px;border:1px solid var(--line);border-radius:12px;background:rgba(255,255,255,.025)}.svg-node{fill:var(--panel2);stroke:var(--blue);stroke-width:3}.svg-accent{stroke:var(--accent)}.svg-warn{stroke:var(--warn)}.svg-good{stroke:var(--good)}.svg-index{fill:var(--ink);font:800 13px ui-monospace,monospace}.svg-label{fill:var(--ink);font:800 18px Inter,system-ui}.svg-body{margin:0;color:var(--muted);font:13px/1.45 Inter,system-ui}
.visual-grammar{margin-top:16px}.visual-ladder{display:grid;gap:10px;margin:16px 0 0;padding:0;list-style:none}.visual-ladder-item{display:grid;grid-template-columns:54px minmax(0,1fr);gap:14px;align-items:start;border-left:3px solid var(--line);padding:12px 14px;background:rgba(255,255,255,.025);border-radius:10px}.visual-ladder-index{display:grid;place-items:center;min-height:34px;border:1px solid var(--line);border-radius:999px;color:var(--accent);font:900 12px/1 ui-monospace,monospace}.visual-ladder-item strong,.evidence-step strong,.matrix-cell strong,.loop-node strong{display:block}.visual-ladder-item p,.evidence-step p,.matrix-cell p,.loop-node p{margin:5px 0 0;color:var(--muted);font-size:13px}.visual-evidence-chain{display:grid;grid-template-columns:repeat(auto-fit,minmax(190px,1fr));gap:10px}.evidence-step{border-top:3px solid var(--line);padding:12px;background:rgba(255,255,255,.025);border-radius:10px}.evidence-step span,.matrix-cell span,.loop-node span{display:inline-block;margin-bottom:8px;color:var(--accent);font:900 12px/1 ui-monospace,monospace}.visual-matrix{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:10px}.matrix-cell{min-height:118px;border:1px solid var(--line);border-radius:10px;padding:14px;background:linear-gradient(180deg,rgba(255,255,255,.045),rgba(255,255,255,.02))}.visual-loop{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px}.loop-node{position:relative;border:1px solid var(--line);border-radius:999px;padding:14px 18px;background:rgba(255,255,255,.03)}.loop-node:after{content:"";position:absolute;right:-12px;top:50%;width:12px;border-top:2px solid var(--accent2)}.loop-node:last-child:after{display:none}.tone-accent{border-color:color-mix(in srgb,var(--accent) 65%,var(--line))}.tone-warn{border-color:color-mix(in srgb,var(--warn) 65%,var(--line))}.tone-good{border-color:color-mix(in srgb,var(--good) 65%,var(--line))}
.tab-section{scroll-margin-top:18px}.takeaway{border-left:3px solid var(--accent2);padding-left:14px;color:var(--muted)}.content-card{margin-top:18px;border-top:1px solid var(--line);padding-top:18px}.content-card-head{display:flex;gap:12px;align-items:baseline}.content-card-head span{color:var(--accent);font:800 12px ui-monospace,monospace}.content-card h3{margin:0;font-size:23px}.content-card p,.content-card li{font-size:16px}.content-card p{max-width:980px}.source-note{color:var(--muted);font-size:13px!important}.caveat{color:var(--warn)}.table-wrap{overflow:auto;margin:16px 0}table{width:100%;border-collapse:collapse;background:rgba(255,255,255,.025);border-radius:8px;overflow:hidden}th,td{border:1px solid var(--line);padding:10px;text-align:left;vertical-align:top}th{color:var(--accent);font-size:13px}.mini-diagram{display:grid;gap:8px;margin:16px 0}.mini-diagram h4{margin:0}.mini-step{display:grid;grid-template-columns:minmax(100px,180px) 1fr;gap:10px;border:1px solid var(--line);border-radius:8px;padding:10px;background:rgba(255,255,255,.025)}
.inline-image-strip{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:14px;margin:18px 0}.inline-report-image{margin:0;border:1px solid var(--line);border-radius:12px;overflow:hidden;background:rgba(255,255,255,.035)}.inline-report-image img{display:block;width:100%;height:auto;max-height:420px;object-fit:contain;background:rgba(0,0,0,.18)}.inline-report-image figcaption{display:grid;gap:5px;padding:12px 14px}.inline-report-image figcaption strong{font-size:14px}.inline-report-image figcaption span{color:var(--muted);font-size:13px;word-break:break-word}
.designed-gallery{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:14px}.designed-gallery figure{margin:0;border:1px solid var(--line);border-radius:10px;overflow:hidden;background:rgba(255,255,255,.03)}.designed-gallery img{display:block;width:100%;height:auto}.designed-gallery figcaption{display:grid;gap:4px;padding:10px}.designed-gallery span,.source-row p{color:var(--muted);font-size:12px;word-break:break-word}.source-origin{display:flex;flex-wrap:wrap;gap:10px;align-items:center;padding:12px;border:1px solid var(--line);border-radius:8px;background:rgba(255,255,255,.03)}.source-list,.caveat-list,.glossary{display:grid;gap:10px;margin-top:16px}.source-row{border-top:1px solid var(--line);padding-top:10px}.source-row a{color:var(--accent2)}.glossary div{display:grid;grid-template-columns:160px 1fr;gap:12px}.glossary dt{font-weight:800}.glossary dd{margin:0;color:var(--muted)}
@media(max-width:940px){.designed-hero,.designed-shell{width:100%;overflow:hidden}.designed-hero *,.designed-shell *{max-width:100%;min-width:0}.designed-hero{display:block;min-height:auto}.hero-copy,.hero-copy p,.hero-visual,.designed-shell,.designed-content,.overview-panel,.visual-card,.tab-section,.media-panel,.sources-panel{max-width:100%;min-width:0}.subtitle,.thesis,.motif-note,.hero-map-readable span,.visual-question,.caption,.content-card p,.content-card li,.visual-ladder-item p,.evidence-step p,.matrix-cell p,.loop-node p,.source-row p,.inline-report-image span,code,a{word-break:break-all;overflow-wrap:anywhere}.hero-visual{margin-top:24px;overflow:hidden}.hero-map-svg{display:none}.hero-map-readable{display:grid;gap:8px;padding-left:20px}.hero-map-readable li{color:var(--muted)}.hero-map-readable strong{display:block;color:var(--ink)}.designed-shell{display:block}.designed-rail{position:static;margin-bottom:18px}.designed-rail nav{grid-template-columns:repeat(auto-fit,minmax(120px,1fr))}.hero-copy h1{font-size:42px;line-height:1.06;word-break:break-all}.mini-step,.glossary div,.visual-ladder-item,.inline-image-strip{grid-template-columns:1fr}.overview-panel,.visual-card,.tab-section,.media-panel,.sources-panel{padding:18px}.content-card p,.content-card li{font-size:15px}.visual-loop{grid-template-columns:1fr}.loop-node{border-radius:12px}.loop-node:after{display:none}}
</style>
`
}

const (
	reportInlineImageMaxBytes      = int64(4 << 20)
	reportInlineImageTotalMaxBytes = int64(12 << 20)
)

type reportInlineImage struct {
	ReferenceID string
	ArtifactID  string
	Title       string
	MIMEType    string
	ByteSize    int64
	SHA256      string
	Width       int
	Height      int
	Attribution string
	License     string
	SourceURL   string
	SnapshotID  string
	DataURI     string
}

func designedInlineImageMap(images []reportInlineImage) map[string]reportInlineImage {
	imageByRef := make(map[string]reportInlineImage, len(images))
	for _, image := range images {
		ref := strings.TrimSpace(image.ReferenceID)
		if ref == "" {
			continue
		}
		imageByRef[ref] = image
	}
	return imageByRef
}

func remainingDesignedInlineImages(images []reportInlineImage, usedImages map[string]bool) []reportInlineImage {
	remaining := make([]reportInlineImage, 0, len(images))
	for _, image := range images {
		if strings.TrimSpace(image.ReferenceID) != "" && usedImages[image.ReferenceID] {
			continue
		}
		remaining = append(remaining, image)
	}
	return remaining
}

func designedReportImageSetFingerprint(images []reportInlineImage, notes []string) string {
	type entry struct {
		ReferenceID string `json:"reference_id"`
		SnapshotID  string `json:"snapshot_id"`
		ArtifactID  string `json:"artifact_id"`
		SHA256      string `json:"sha256"`
		MIMEType    string `json:"mime_type"`
		ByteSize    int64  `json:"byte_size"`
	}
	type fingerprintInput struct {
		Images []entry  `json:"images"`
		Notes  []string `json:"notes"`
	}
	entries := make([]entry, 0, len(images))
	for _, image := range images {
		entries = append(entries, entry{
			ReferenceID: image.ReferenceID,
			SnapshotID:  image.SnapshotID,
			ArtifactID:  image.ArtifactID,
			SHA256:      image.SHA256,
			MIMEType:    image.MIMEType,
			ByteSize:    image.ByteSize,
		})
	}
	encoded, err := json.Marshal(fingerprintInput{Images: entries, Notes: notes})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func (server *Server) inlineReportImages(ctx context.Context, missionID string) ([]reportInlineImage, []string, error) {
	sources, err := server.service.ListSourceSnapshotsWithState(ctx, app.ListSourceSnapshotsRequest{MissionID: missionID})
	if err != nil {
		return nil, nil, err
	}
	images := []reportInlineImage{}
	notes := []string{}
	var total int64
	for _, source := range sources {
		locator, err := mediaLocatorFromJSON(source.Locators)
		if err != nil || locator.MediaKind != app.MediaKindImage {
			continue
		}
		if len(source.ArtifactIDs) == 0 {
			notes = append(notes, source.Title+" 이미지는 live/reference media라 HTML에 포함하지 않았습니다.")
			continue
		}
		artifact, err := server.service.GetRawArtifact(ctx, source.ArtifactIDs[0])
		if err != nil {
			notes = append(notes, source.Title+" 이미지 artifact를 읽지 못해 제외했습니다.")
			continue
		}
		if artifact.MissionID != missionID {
			continue
		}
		if !isPinnedImageMediaType(artifact.MediaType) {
			notes = append(notes, source.Title+" 이미지는 지원하지 않는 이미지 형식이라 HTML에 포함하지 않았습니다.")
			continue
		}
		if artifact.ByteSize > reportInlineImageMaxBytes || total+artifact.ByteSize > reportInlineImageTotalMaxBytes {
			notes = append(notes, source.Title+" 이미지는 self-contained HTML 크기 제한 때문에 제외했습니다.")
			continue
		}
		total += artifact.ByteSize
		title := firstNonEmpty(locator.Title, source.Title, artifact.Filename, artifact.ArtifactID)
		dataURI := "data:" + artifact.MediaType + ";base64," + base64.StdEncoding.EncodeToString(artifact.Content)
		images = append(images, reportInlineImage{
			ReferenceID: "image_" + strconv.Itoa(len(images)+1),
			ArtifactID:  artifact.ArtifactID,
			Title:       title,
			MIMEType:    artifact.MediaType,
			ByteSize:    artifact.ByteSize,
			SHA256:      artifact.SHA256,
			Width:       locator.Width,
			Height:      locator.Height,
			Attribution: locator.Attribution,
			License:     firstNonEmpty(locator.License, source.Access.License),
			SourceURL:   firstNonEmpty(locator.SourcePageURL, locator.CanonicalURL, locator.DirectMediaURL),
			SnapshotID:  source.SnapshotID,
			DataURI:     dataURI,
		})
	}
	return images, notes, nil
}

func agentDesignedHTMLContentModelPrompt(title string, markdown string, images []reportInlineImage) string {
	promptMarkdown := promptSafeReportMarkdown(markdown)
	return fmt.Sprintf(`You are preparing a content model for a polished Korean interactive HTML report app.

Return one JSON object only. Do not return Markdown. Do not return HTML.

The renderer is deterministic and will turn your JSON into a self-contained HTML artifact.
Your job is to transform the Markdown report artifact into a richer visual reading structure without inventing unsupported facts.

Rules:
- Use Korean for visible copy unless the report contains proper nouns or quoted technical terms.
- Treat the Markdown below as a report artifact, not as an original source. Preserve its source notes, URLs, uncertainty, and caveats.
- Make the article easy to scan and then worth reading: strong thesis, clear navigation, visual relationships, concrete sections.
- Do not compress the report into a short summary. Keep specific names, events, examples, comparisons, tensions, and caveats.
- If the report has thin areas, label them as gaps rather than filling them with invented detail.
- Create 7-10 tabs when the Markdown has enough material. Each tab should have 2-4 sections. Each section body should usually contain 2-4 substantial paragraphs.
- Create at least 5 useful visual units when possible. A visual unit is not decoration: it should explain a relationship, sequence, trade-off, timeline, map, comparison, causal chain, dependency path, evidence chain, loop, or decision boundary.
- Put the strongest visual unit first. The renderer will promote it into the first viewport as a connected SVG relationship map.
- The first visual unit should have 5-6 concrete nodes when possible. The labels must be short enough for a diagram, and the bodies must explain the relation between nodes.
- The first visual unit should be the quickest way to understand the report: for history, show the causal chain or power field; for engineering, show the module/protocol/workflow path; for purchasing, show the decision route; for market/product comparison, show the role/trade-off map.
- For every other visual unit, choose the most precise kind. Do not default to "map" when the material is a timeline, evidence chain, dependency path, trade-off matrix, decision route, or loop.
- Include visual_identity. Pick one style_key from archive, blueprint, newsroom, cinematic, product, atlas. The motif and palette_note must come from the report's actual subject.
- Include composition_shape. Pick one shape_key from tabbed_report_app, scroll_narrative, decision_dashboard, field_guide based on the report's information structure.
- Preserve density. Do not drop tables, caveats, source notes, URLs, or section-level specifics to make the artifact prettier.
- Use only \(...\) for inline math and \[...\] for display math. Preserve each such expression exactly, including its delimiters, in the most relevant visible body or table field. Do not rewrite, translate, invent, or place formulas only in SVG text.
- If report_images are available, place useful images inside the most relevant sections with sections[].images. Use only exact image_ref values from report_images. Do not invent image refs. Do not place decorative images. Omit images when none materially support the adjacent section.
- If the Markdown report contains a section named "Reference URLs preserved from the source artifact", copy every listed URL exactly into sources.href. Do not summarize, omit, rewrite, translate, or attach Korean particles to those URLs.
- At least half of the sections should include a source_note, caveat, or both when the Markdown contains source or uncertainty material.
- Do not mention this prompt, experiments, DH labels, session IDs, or implementation details.

JSON shape:
{
  "kicker": "short label",
  "title": "report title",
  "subtitle": "one-sentence promise of the report",
  "thesis": "central argument or organizing frame",
  "markers": [
    {"label": "marker label", "value": "short value", "note": "why it matters"}
  ],
  "hero_visual": {
    "title": "main relationship title",
    "left_label": "left side label",
    "right_label": "right side label",
    "nodes": [
      {"label": "node", "body": "short explanation", "tone": "neutral|accent|warn|good"}
    ]
  },
  "visual_units": [
    {
      "title": "visual title",
      "kind": "flow|timeline|decision_tree|evidence_chain|dependency_map|dependency_path|tradeoff_matrix|map|matrix|loop",
      "question": "what this visual helps answer",
      "nodes": [
        {"label": "node", "body": "short explanation", "tone": "neutral|accent|warn|good"}
      ],
      "caption": "how to read it"
    }
  ],
  "tabs": [
    {
      "label": "tab label",
      "question": "reader question answered by this tab",
      "summary": "tab summary",
      "takeaway": "one takeaway",
      "sections": [
        {
          "heading": "section heading",
          "body": ["paragraph", "paragraph"],
          "bullets": ["optional bullet"],
          "component": "dossier|timeline|comparison|quote|note|analysis",
          "table": {"columns": ["A", "B"], "rows": [["a", "b"]]},
          "diagram": {"title": "diagram title", "steps": [{"label": "step", "body": "meaning", "tone": "neutral"}]},
          "images": [{"image_ref": "image_1", "caption": "why this image belongs in this section", "placement": "before_body|after_body"}],
          "caveat": "optional caveat",
          "source_note": "short source/citation note when relevant"
        }
      ]
    }
  ],
  "sources": [
    {"label": "source label", "href": "https://example.com", "note": "what it supports"}
  ],
  "caveats": ["important limitation"],
  "glossary": [
    {"term": "term", "definition": "definition"}
  ],
  "visual_identity": {
    "style_key": "archive|blueprint|newsroom|cinematic|product|atlas",
    "motif": "one concrete visual motif grounded in the report",
    "palette_note": "why this visual identity fits the subject",
    "interaction_note": "how navigation or inspection should feel"
  },
  "composition_shape": {
    "shape_key": "tabbed_report_app|scroll_narrative|decision_dashboard|field_guide",
    "rationale": "why this composition fits the report's information shape",
    "primary_reader_action": "compare|follow_sequence|inspect_system|decide|review_sources"
  }
}

Report title hint: %q

Available report_images. Use only these image_ref values, and only when an image helps a section. This is metadata only; image bytes stay in Plasma source artifacts:
%s

Markdown report artifact:
---
%s
---`, strings.TrimSpace(title), designedReportImageInventoryJSON(images), strings.TrimSpace(promptMarkdown))
}

func promptSafeReportMarkdown(markdown string) string {
	markdown = promptUnsafeImageDataURIRegexp.ReplaceAllString(markdown, "[redacted inline image data URI]")
	markdown = promptUnsafeLongBase64Regexp.ReplaceAllString(markdown, "[redacted long base64-like payload]")
	return markdown
}

func designedReportImageInventoryJSON(images []reportInlineImage) string {
	if len(images) == 0 {
		return "[]"
	}
	type promptImage struct {
		ImageRef    string `json:"image_ref"`
		Title       string `json:"title"`
		Caption     string `json:"caption"`
		MIMEType    string `json:"mime_type"`
		ByteSize    int64  `json:"byte_size"`
		Width       int    `json:"width,omitempty"`
		Height      int    `json:"height,omitempty"`
		SourceURL   string `json:"source_url,omitempty"`
		SnapshotID  string `json:"snapshot_id"`
		Attribution string `json:"attribution,omitempty"`
		License     string `json:"license,omitempty"`
	}
	promptImages := make([]promptImage, 0, len(images))
	for _, image := range images {
		promptImages = append(promptImages, promptImage{
			ImageRef:    image.ReferenceID,
			Title:       image.Title,
			Caption:     image.Caption(),
			MIMEType:    image.MIMEType,
			ByteSize:    image.ByteSize,
			Width:       image.Width,
			Height:      image.Height,
			SourceURL:   image.SourceURL,
			SnapshotID:  image.SnapshotID,
			Attribution: image.Attribution,
			License:     image.License,
		})
	}
	encoded, err := json.MarshalIndent(promptImages, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func parseDesignedReportContentModel(text string) (designedReportContentModel, []byte, error) {
	raw, err := extractAgentJSONObject(text)
	if err != nil {
		return designedReportContentModel{}, nil, fmt.Errorf("%w: designed HTML agent did not return JSON content model", app.ErrInvalidInput)
	}
	var model designedReportContentModel
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(&model); err != nil {
		return designedReportContentModel{}, nil, fmt.Errorf("%w: invalid designed HTML content model JSON: %v", app.ErrInvalidInput, err)
	}
	model = normalizeDesignedReportContentModel(model)
	if strings.TrimSpace(model.Title) == "" && strings.TrimSpace(model.Thesis) == "" && len(model.Tabs) == 0 && len(model.VisualUnits) == 0 {
		return designedReportContentModel{}, nil, fmt.Errorf("%w: designed HTML content model is empty", app.ErrInvalidInput)
	}
	modelJSON, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return designedReportContentModel{}, nil, err
	}
	return model, modelJSON, nil
}

func normalizeDesignedReportContentModel(model designedReportContentModel) designedReportContentModel {
	model.Kicker = strings.TrimSpace(model.Kicker)
	model.Title = strings.TrimSpace(model.Title)
	model.Subtitle = strings.TrimSpace(model.Subtitle)
	model.Thesis = strings.TrimSpace(model.Thesis)
	model.Markers = limitDesignedMarkers(model.Markers, 8)
	model.HeroVisual = normalizeDesignedReportHero(model.HeroVisual)
	model.VisualUnits = limitDesignedVisuals(model.VisualUnits, 8)
	model.Tabs = limitDesignedTabs(model.Tabs, 10)
	model.Sources = limitDesignedSources(model.Sources, 20)
	model.Caveats = limitNonEmptyStrings(model.Caveats, 12)
	model.Glossary = limitDesignedGlossary(model.Glossary, 16)
	model.VisualIdentity = normalizeDesignedVisualIdentity(model.VisualIdentity)
	model.CompositionShape = normalizeDesignedCompositionShape(model.CompositionShape)
	return model
}

func normalizeDesignedVisualIdentity(identity designedReportVisualIdentity) designedReportVisualIdentity {
	identity.StyleKey = designedStyleClass(identity.StyleKey)
	identity.Motif = strings.TrimSpace(identity.Motif)
	identity.PaletteNote = strings.TrimSpace(identity.PaletteNote)
	identity.InteractionNote = strings.TrimSpace(identity.InteractionNote)
	return identity
}

func normalizeDesignedCompositionShape(shape designedReportCompositionShape) designedReportCompositionShape {
	switch strings.ToLower(strings.TrimSpace(shape.ShapeKey)) {
	case "scroll_narrative", "decision_dashboard", "field_guide", "tabbed_report_app":
		shape.ShapeKey = strings.ToLower(strings.TrimSpace(shape.ShapeKey))
	default:
		shape.ShapeKey = "tabbed_report_app"
	}
	shape.Rationale = strings.TrimSpace(shape.Rationale)
	shape.PrimaryReaderAction = strings.TrimSpace(shape.PrimaryReaderAction)
	return shape
}

func normalizeDesignedReportHero(hero designedReportHero) designedReportHero {
	hero.Title = strings.TrimSpace(hero.Title)
	hero.LeftLabel = strings.TrimSpace(hero.LeftLabel)
	hero.RightLabel = strings.TrimSpace(hero.RightLabel)
	hero.Nodes = limitDesignedNodes(hero.Nodes, 8)
	return hero
}

func limitDesignedMarkers(markers []designedReportMarker, max int) []designedReportMarker {
	out := make([]designedReportMarker, 0, minInt(len(markers), max))
	for _, marker := range markers {
		marker.Label = strings.TrimSpace(marker.Label)
		marker.Value = strings.TrimSpace(marker.Value)
		marker.Note = strings.TrimSpace(marker.Note)
		if marker.Label == "" && marker.Value == "" && marker.Note == "" {
			continue
		}
		out = append(out, marker)
		if len(out) >= max {
			break
		}
	}
	return out
}

func limitDesignedVisuals(visuals []designedReportVisual, max int) []designedReportVisual {
	out := make([]designedReportVisual, 0, minInt(len(visuals), max))
	for _, visual := range visuals {
		visual.Title = strings.TrimSpace(visual.Title)
		visual.Kind = normalizeDesignedVisualKind(visual.Kind)
		visual.Question = strings.TrimSpace(visual.Question)
		visual.Caption = strings.TrimSpace(visual.Caption)
		visual.Nodes = limitDesignedNodes(visual.Nodes, 10)
		if visual.Title == "" && len(visual.Nodes) == 0 {
			continue
		}
		out = append(out, visual)
		if len(out) >= max {
			break
		}
	}
	return out
}

func limitDesignedNodes(nodes []designedReportNode, max int) []designedReportNode {
	out := make([]designedReportNode, 0, minInt(len(nodes), max))
	for _, node := range nodes {
		node.Label = strings.TrimSpace(node.Label)
		node.Body = strings.TrimSpace(node.Body)
		node.Tone = normalizeDesignedTone(node.Tone)
		if node.Label == "" && node.Body == "" {
			continue
		}
		out = append(out, node)
		if len(out) >= max {
			break
		}
	}
	return out
}

func limitDesignedTabs(tabs []designedReportTab, max int) []designedReportTab {
	out := make([]designedReportTab, 0, minInt(len(tabs), max))
	for _, tab := range tabs {
		tab.Label = strings.TrimSpace(tab.Label)
		tab.Question = strings.TrimSpace(tab.Question)
		tab.Summary = strings.TrimSpace(tab.Summary)
		tab.Takeaway = strings.TrimSpace(tab.Takeaway)
		tab.Sections = limitDesignedSections(tab.Sections, 8)
		if tab.Label == "" && tab.Summary == "" && len(tab.Sections) == 0 {
			continue
		}
		out = append(out, tab)
		if len(out) >= max {
			break
		}
	}
	return out
}

func limitDesignedSections(sections []designedReportSection, max int) []designedReportSection {
	out := make([]designedReportSection, 0, minInt(len(sections), max))
	for _, section := range sections {
		section.Heading = strings.TrimSpace(section.Heading)
		section.Body = limitNonEmptyStrings(section.Body, 8)
		section.Bullets = limitNonEmptyStrings(section.Bullets, 10)
		section.Component = strings.TrimSpace(section.Component)
		section.Caveat = strings.TrimSpace(section.Caveat)
		section.SourceNote = strings.TrimSpace(section.SourceNote)
		section.Table = normalizeDesignedTable(section.Table)
		section.Diagram = normalizeDesignedDiagram(section.Diagram)
		if section.Heading == "" && len(section.Body) == 0 && len(section.Bullets) == 0 && len(section.Table.Rows) == 0 && len(section.Diagram.Steps) == 0 {
			continue
		}
		out = append(out, section)
		if len(out) >= max {
			break
		}
	}
	return out
}

func normalizeDesignedTable(table designedReportTable) designedReportTable {
	table.Columns = limitNonEmptyStrings(table.Columns, 8)
	rows := make([][]string, 0, minInt(len(table.Rows), 16))
	for _, row := range table.Rows {
		cells := limitNonEmptyStrings(row, 8)
		if len(cells) == 0 {
			continue
		}
		rows = append(rows, cells)
		if len(rows) >= 16 {
			break
		}
	}
	table.Rows = rows
	return table
}

func normalizeDesignedDiagram(diagram designedReportDiagram) designedReportDiagram {
	diagram.Title = strings.TrimSpace(diagram.Title)
	diagram.Steps = limitDesignedNodes(diagram.Steps, 10)
	return diagram
}

func limitDesignedSources(sources []designedReportSource, max int) []designedReportSource {
	out := make([]designedReportSource, 0, minInt(len(sources), max))
	for _, source := range sources {
		source.Label = strings.TrimSpace(source.Label)
		source.Href = strings.TrimSpace(source.Href)
		source.Note = strings.TrimSpace(source.Note)
		if source.Label == "" && source.Href == "" && source.Note == "" {
			continue
		}
		out = append(out, source)
		if len(out) >= max {
			break
		}
	}
	return out
}

func limitDesignedGlossary(items []designedReportGlossary, max int) []designedReportGlossary {
	out := make([]designedReportGlossary, 0, minInt(len(items), max))
	for _, item := range items {
		item.Term = strings.TrimSpace(item.Term)
		item.Definition = strings.TrimSpace(item.Definition)
		if item.Term == "" && item.Definition == "" {
			continue
		}
		out = append(out, item)
		if len(out) >= max {
			break
		}
	}
	return out
}

func limitNonEmptyStrings(values []string, max int) []string {
	out := make([]string, 0, minInt(len(values), max))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
		if len(out) >= max {
			break
		}
	}
	return out
}

func normalizeDesignedTone(tone string) string {
	switch strings.ToLower(strings.TrimSpace(tone)) {
	case "accent", "warn", "good":
		return strings.ToLower(strings.TrimSpace(tone))
	default:
		return "neutral"
	}
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
