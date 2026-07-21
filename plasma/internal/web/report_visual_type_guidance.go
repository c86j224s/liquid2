package web

import "strings"

const reportGenerationGuidanceProfileVisualTypeManual = "visual-type-manual"

func isReportGenerationGuidanceProfileVisualTypeManual(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileVisualTypeManual, "visual_type_manual", "visual-type-selection", "visual_type_selection":
		return true
	default:
		return false
	}
}

func reportVisualTypeSelectionPlanningGuidance() string {
	return `Visual type selection planning guidance:
- When a section needs a visual aid, name the intended type in the existing section purpose or coverage_notes: Markdown table, flowchart dependency graph, sequence diagram, state diagram, timeline, entity/class diagram, or source-backed chart.
- For source-backed numeric material such as stock-style tables, industry statistics, or agent benchmark matrices, keep a compact Markdown table for exact values, then consider one simple chart when an ordered series, region comparison, or benchmark trade-off would be easier to scan visually.
- For ordered numeric series, plan xychart-beta only when the source gives the x-axis labels and y-values. For benchmark trade-offs, plan quadrantChart only when the source provides two defensible axes; otherwise keep the table. For composition shares, plan pie only when the source provides parts of a whole.
- For complex architecture dependency graphs, prefer a stable flowchart or graph with subsystem grouping and labeled dependencies. Use classDiagram or erDiagram only when the source explicitly defines types, entities, schemas, or ownership relationships.
- For protocols, handoffs, and actor interactions, prefer sequenceDiagram. For lifecycle or status transitions, prefer stateDiagram-v2. For chronological change or dated milestones, prefer timeline.
- Treat C4Context, sankey-beta, block-beta, packet-beta, and requirementDiagram as compatibility-sensitive. Plan a stable flowchart or table fallback unless the report truly needs that grammar.`
}

func reportVisualTypeSelectionWritingGuidance() string {
	return `- Match the visual type to the source structure instead of using the same table or diagram shape everywhere.
- For quantitative datasets, never invent values, trends, percentages, axes, or benchmark comparisons that are not in the source. If the source has explicit ordered values or explicit comparison axes, a simple source-backed chart can supplement the table; if a chart would require inference, use a table plus prose.
- For complex architecture dependency graphs, use a simple flowchart or graph when it preserves the relationships better than prose. Keep node names short, group related services when useful, label important dependency edges, and explain the dependency risk or consequence after the diagram.
- For dense benchmark or industry-stat tables, keep the table readable: compare only the columns that support the section's point, move secondary detail back into prose, and add a chart only for the one trend or trade-off the reader most needs to see.
- If Mermaid syntax becomes fragile, simplify the diagram or replace it with a Markdown table rather than shipping a clever diagram that may fail to render.`
}
