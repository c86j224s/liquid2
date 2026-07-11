package web

import (
	"bytes"
	"fmt"
	htmlpkg "html"
	"strconv"
	"strings"
)

func normalizeDesignedVisualKind(kind string) string {
	normalized := strings.ToLower(strings.TrimSpace(kind))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	switch normalized {
	case "flow", "timeline", "decision_tree", "decision_route", "evidence_chain", "dependency_map", "dependency_path", "tradeoff_matrix", "map", "matrix", "loop":
		return normalized
	default:
		return "map"
	}
}

func designedVisualKindLabel(kind string) string {
	switch normalizeDesignedVisualKind(kind) {
	case "timeline":
		return "Timeline"
	case "decision_tree", "decision_route":
		return "Decision route"
	case "evidence_chain":
		return "Evidence chain"
	case "dependency_map", "dependency_path":
		return "Dependency path"
	case "tradeoff_matrix", "matrix":
		return "Trade-off matrix"
	case "loop":
		return "Feedback loop"
	case "flow":
		return "Flow"
	default:
		return "Relationship map"
	}
}

func renderDesignedVisualGrammar(out *bytes.Buffer, visual designedReportVisual, id string) {
	switch normalizeDesignedVisualKind(visual.Kind) {
	case "timeline", "flow", "decision_tree", "decision_route", "dependency_map", "dependency_path":
		renderDesignedVisualLadder(out, visual, id)
	case "evidence_chain":
		renderDesignedEvidenceChain(out, visual, id)
	case "tradeoff_matrix", "matrix":
		renderDesignedTradeoffMatrix(out, visual, id)
	case "loop":
		renderDesignedLoop(out, visual, id)
	default:
		renderRelationshipSVG(out, visual.Nodes, id)
	}
}

func renderDesignedVisualLadder(out *bytes.Buffer, visual designedReportVisual, id string) {
	if len(visual.Nodes) == 0 {
		out.WriteString("<p class=\"muted\">시각화할 노드가 없습니다.</p>")
		return
	}
	kind := normalizeDesignedVisualKind(visual.Kind)
	out.WriteString("<ol class=\"visual-grammar visual-ladder visual-" + htmlpkg.EscapeString(kind) + "\" id=\"" + htmlpkg.EscapeString(id) + "\">")
	for index, node := range visual.Nodes {
		tone := normalizeDesignedTone(node.Tone)
		out.WriteString("<li class=\"visual-ladder-item tone-" + htmlpkg.EscapeString(tone) + "\"><span class=\"visual-ladder-index\">" + fmt.Sprintf("%02d", index+1) + "</span><div><strong>" + htmlpkg.EscapeString(node.Label) + "</strong><p>" + htmlpkg.EscapeString(node.Body) + "</p></div></li>")
	}
	out.WriteString("</ol>")
}

func renderDesignedEvidenceChain(out *bytes.Buffer, visual designedReportVisual, id string) {
	if len(visual.Nodes) == 0 {
		out.WriteString("<p class=\"muted\">시각화할 노드가 없습니다.</p>")
		return
	}
	out.WriteString("<div class=\"visual-grammar visual-evidence-chain\" id=\"" + htmlpkg.EscapeString(id) + "\">")
	for index, node := range visual.Nodes {
		tone := normalizeDesignedTone(node.Tone)
		out.WriteString("<section class=\"evidence-step tone-" + htmlpkg.EscapeString(tone) + "\"><span>" + fmt.Sprintf("E%02d", index+1) + "</span><strong>" + htmlpkg.EscapeString(node.Label) + "</strong><p>" + htmlpkg.EscapeString(node.Body) + "</p></section>")
	}
	out.WriteString("</div>")
}

func renderDesignedTradeoffMatrix(out *bytes.Buffer, visual designedReportVisual, id string) {
	if len(visual.Nodes) == 0 {
		out.WriteString("<p class=\"muted\">시각화할 노드가 없습니다.</p>")
		return
	}
	out.WriteString("<div class=\"visual-grammar visual-matrix\" id=\"" + htmlpkg.EscapeString(id) + "\">")
	for index, node := range visual.Nodes {
		tone := normalizeDesignedTone(node.Tone)
		out.WriteString("<div class=\"matrix-cell tone-" + htmlpkg.EscapeString(tone) + "\"><span>" + strconv.Itoa(index+1) + "</span><strong>" + htmlpkg.EscapeString(node.Label) + "</strong><p>" + htmlpkg.EscapeString(node.Body) + "</p></div>")
	}
	out.WriteString("</div>")
}

func renderDesignedLoop(out *bytes.Buffer, visual designedReportVisual, id string) {
	if len(visual.Nodes) == 0 {
		out.WriteString("<p class=\"muted\">시각화할 노드가 없습니다.</p>")
		return
	}
	out.WriteString("<div class=\"visual-grammar visual-loop\" id=\"" + htmlpkg.EscapeString(id) + "\">")
	for index, node := range visual.Nodes {
		tone := normalizeDesignedTone(node.Tone)
		out.WriteString("<div class=\"loop-node tone-" + htmlpkg.EscapeString(tone) + "\"><span>" + fmt.Sprintf("%02d", index+1) + "</span><strong>" + htmlpkg.EscapeString(node.Label) + "</strong><p>" + htmlpkg.EscapeString(node.Body) + "</p></div>")
	}
	out.WriteString("</div>")
}
