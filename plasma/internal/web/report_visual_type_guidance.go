package web

import "strings"

const (
	reportGenerationGuidanceProfileVisualTypeManual          = "visual-type-manual"
	reportGenerationGuidanceProfileVisualEvidenceFit         = "visual-evidence-fit"
	reportGenerationGuidanceProfileVisualReadingAidPreferred = "visual-reading-aid-preferred"
	reportGenerationGuidanceProfileVisualReaderIntent        = "visual-reader-intent"
	reportGenerationGuidanceProfileVisualClaritySeeking      = "visual-clarity-seeking"
	reportGenerationGuidanceProfileVisualAffordancePriming   = "visual-affordance-priming"
)

func isReportGenerationGuidanceProfileVisualTypeManual(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileVisualTypeManual, "visual_type_manual", "visual-type-selection", "visual_type_selection":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileVisualEvidenceFit(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileVisualEvidenceFit, "visual_evidence_fit", "evidence-fit-visuals", "evidence_fit_visuals":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileVisualReadingAidPreferred(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileVisualReadingAidPreferred, "visual_reading_aid_preferred", "visual-preferred", "visual_preferred":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileVisualReaderIntent(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileVisualReaderIntent, "visual_reader_intent", "reader-intent-visuals", "reader_intent_visuals":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileVisualClaritySeeking(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileVisualClaritySeeking, "visual_clarity_seeking", "clarity-seeking-visuals", "clarity_seeking_visuals":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileVisualAffordancePriming(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileVisualAffordancePriming, "visual_affordance_priming", "affordance-primed-visuals", "affordance_primed_visuals":
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

func reportVisualEvidenceFitPlanningGuidance() string {
	return `Visual evidence-fit planning guidance:
- Treat visual aids as reader aids, not proof artifacts. Plan them when they make source-backed structure, flow, contrast, or uncertainty easier to understand.
- Do not reject a structure, flow, relationship, dependency, or qualitative comparison merely because the source lacks exact numeric values. If the same point can be responsibly explained in prose, it can often be diagrammed as an interpretive aid.
- While planning each visual, keep its evidence level clear in the existing section purpose or coverage_notes: exact value reproduction, qualitative comparison, or interpretive structure.
- For qualitative or interpretive visuals, plan nearby wording that makes the evidence level clear without turning the report into repeated disclaimers.`
}

func reportVisualEvidenceFitWritingGuidance() string {
	return `- Match the visual's claim strength to the source evidence. Exact numeric charts may reproduce source values; qualitative charts should use qualitative labels such as high, medium, low, relative strength, or directional change; interpretive diagrams should show structure, flow, or relationships without implying measured magnitude.
- A missing exact number is not by itself a reason to avoid a useful diagram. It is a reason to avoid pretending that the diagram is more precise than the source.
- When a visual is interpretive, add a short nearby note or sentence that says it is a source-based interpretation or reading aid. Do this once where useful, not as repetitive boilerplate.
- Never let a visual make a stronger claim than the prose could defend from the same sources. If the relation is uncertain, show or say that uncertainty directly.`
}

func reportVisualReadingAidPreferencePlanningGuidance() string {
	return `Visual reading-aid preference planning guidance:
- When a section contains a relationship, sequence, dependency, lifecycle, comparison, trade-off, timeline, or uncertainty structure, prefer planning one compact visual aid over adding another explanatory paragraph if the visual would make the structure easier to scan.
- Do not treat stock-chart-grade exactness as the default requirement for charts. Use the source's own resolution: exact values, ranges, indexed movement, directional change, relative strength, qualitative labels, or interpretive structure.
- Put the visual's intended reading job in the existing section purpose or coverage_notes: orient the reader, compare cases, show dependency, show lifecycle, show timing, or show uncertainty.
- Plan no visual aid when the section has no natural structure to externalize or when the visual would only decorate a simple point.`
}

func reportVisualReadingAidPreferenceWritingGuidance() string {
	return `- For structure-heavy material, prefer a compact visual aid as the organizing surface, then use prose to explain the takeaway instead of repeating the same structure in a long paragraph.
- If the source supports a relationship, sequence, dependency, lifecycle, comparison, trade-off, timeline, or uncertainty structure, do not omit a useful visual solely because the evidence is approximate, directional, qualitative, or interpretive.
- Keep the visual honest to the source's resolution. Use exact values when given, ranges when given, indexed or directional labels when only movement is supported, and qualitative labels when only relative strength is supported.
- A visual should reduce the reader's cognitive load. If it would add decoration, duplicate an already simple sentence, or imply unsupported certainty, skip it.`
}

func reportVisualReaderIntentPlanningGuidance() string {
	return `Visual reader-intent planning guidance:
- Before planning a visual aid, name the central reader task for the section: what source-backed material should become easier to inspect than prose alone?
- Plan a visual aid when it helps the reader see the section's central material directly: ordered changes, repeated comparisons, timing, dependencies, lifecycle states, trade-offs, or grouped categories.
- Do not plan a visual merely because a caution, methodology note, or inference boundary can be diagrammed. Unless that boundary is the section's main subject, keep it in prose and use visuals for the source material the reader came to understand.
- For numeric or ordered source material, prefer visuals that stay close to the source values, timing, or comparison axes: compact table, source-backed chart, or timeline. Do not let a meta-level explanation diagram replace the source-near visual.
- Put this reader task inside the existing section purpose or coverage_notes. Keep the plan schema unchanged.`
}

func reportVisualReaderIntentWritingGuidance() string {
	return `- Decide from the reader's task, not from a desire to include a diagram. Use a visual aid only when it makes the section's central source-backed material easier to inspect than prose alone.
- For numeric or ordered material, keep visuals close to the source values, sequence, timing, or comparison axes. Prefer compact tables, source-backed charts, or timelines over meta-level diagrams about what can or cannot be inferred.
- Keep inference boundaries, caveats, and methodology cautions in prose unless they are the actual subject of the section. They should not displace a more useful source-near visual.
- When a visual is used, introduce the reader task it solves and then explain the takeaway. If the visual does not reduce the reader's work, remove it.`
}

func reportVisualClaritySeekingPlanningGuidance() string {
	return `Visual clarity-seeking planning guidance:
- As you shape each section, actively look for a visual surface that would make the reader grasp the source-backed point faster or more clearly.
- A good candidate visual lets the reader see a pattern, sequence, comparison, dependency, trade-off, scenario, range, uncertainty, or category structure that prose would otherwise describe at length.
- Choose the visual form that best serves that reader task: table for exact lookup, chart for values over axes, timeline for chronology, flowchart, sequence, or state diagram for movement and dependencies, and matrix-style comparison for trade-offs.
- Match the visual's precision to the source resolution: exact values, ranges, directional movement, qualitative strength, or interpretive structure.
- Put the intended clarity job inside the existing section purpose or coverage_notes. Keep the plan schema unchanged.`
}

func reportVisualClaritySeekingWritingGuidance() string {
	return `- Before drafting each section, actively look for whether a compact visual can make the source-backed point faster or clearer for the reader.
- Use the visual as an explanation surface: introduce what to notice, show the structure compactly, then explain the takeaway in prose.
- Let the source resolution set the visual's form and precision: exact values, ranges, directional movement, qualitative strength, or interpretive structure can all be useful when represented honestly.
- Keep this selection process as writing context. In the final report, let the chosen visual and nearby prose carry the explanation naturally.`
}

func reportVisualAffordancePrimingPlanningGuidance() string {
	return `Visual affordance priming planning guidance:
- Let the source's shape suggest the visual aid before defaulting to prose. Chronology invites a timeline; dependency or blast radius invites a flowchart; actor handoff invites a sequence diagram; lifecycle or status change invites a state diagram; ordered numeric movement invites a source-backed chart; trade-off or scenario comparison invites a compact matrix or table.
- When a section's central evidence is a sequence of dated, monthly, quarterly, or phase-based anchors, a Mermaid timeline is usually the fastest way for the reader to see order, lag, and pending decisions. Use a table alongside it only when exact lookup is the main reader task.
- While planning each section, name the dominant source shape and the visual surface it naturally affords inside the existing purpose or coverage_notes.
- Use the mapping as a reader-orientation aid, not as a quota. The chosen visual should make the section's central source material easier to inspect.
- Keep exact values, ranges, timing anchors, actors, states, and dependency names close to the source. Keep the plan schema unchanged.`
}

func reportVisualAffordancePrimingWritingGuidance() string {
	return `- Before writing each section, notice the dominant source shape: chronology, dependency, actor handoff, lifecycle, ordered values, scenario range, or trade-off.
- Let that shape pull the natural visual forward: timeline for timing, flowchart for dependency, sequence for handoff, state diagram for lifecycle, source-backed chart for ordered values, and matrix or table for scenarios and trade-offs.
- If the plan names chronology, timing anchors, milestones, monthly checkpoints, quarterly events, or phase order as the section's central material, prefer a Mermaid timeline as the orientation surface before adding prose around it.
- Use the visual to orient the reader, then explain the takeaway in prose. Keep the visual close to the source's own resolution and names.
- This is a writing heuristic, not a visible checklist. The final report should read naturally and should not narrate the selection process.`
}
