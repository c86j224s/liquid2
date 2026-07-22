package web

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	reportGenerationGuidanceProfileNarrativeContract                    = "narrative-contract"
	reportGenerationGuidanceProfileSectionBriefNarrativeContract        = "section-brief-narrative-contract"
	reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract = "section-brief-cluster-memory-narrative-contract"
)

func isReportGenerationGuidanceProfileNarrativeContract(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileNarrativeContract, "narrative_contract", "reader-first-editor", "reader_first_editor",
		reportGenerationGuidanceProfileSectionBriefNarrativeContract, "section_brief_narrative_contract",
		reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract, "section_brief_cluster_memory_narrative_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionBriefNarrativeContract(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionBriefNarrativeContract, "section_brief_narrative_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileSectionBriefClusterNarrativeContract(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract, "section_brief_cluster_memory_narrative_contract":
		return true
	default:
		return false
	}
}

func requireReportWritingContract(profile string) bool {
	return isReportGenerationGuidanceProfileNarrativeContract(profile)
}

func longFormCompositionStrategy(profile string) string {
	if isReportGenerationGuidanceProfileNarrativeContract(profile) {
		return reporting.LongFormCompositionNarrativeEdit
	}
	return reporting.LongFormCompositionPreserveMarkdown
}

func reportNarrativeContractPlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileNarrativeContract(profile) {
		return ""
	}
	return `Reader-facing writing-contract guidance:
- First understand the original sources and the user's requested outcome. Then plan a report for a reader who will rely on the report instead of reading every source.
- Add writing_contract to the submitted plan. This is editorial direction, not evidence and not a source summary.
- central_question states the one question the report must answer. reader_takeaway states what the reader should understand or be able to decide after reading.
- reading_path lists the few reasoning moves that make the answer easy to follow. must_keep lists concrete facts, caveats, distinctions, examples, and unresolved tensions that later editing must not erase.
- can_summarize identifies background that may be compressed. move_to_supporting_layer identifies useful detail that should remain available without interrupting the main explanation.
- visual_role explains the reading job, if any, for a table or diagram; use "none needed" when prose is clearer. tone_and_shape describes how the explanation should feel and unfold.
- Keep the contract short and actionable. Do not turn it into a second outline, a source inventory, or a list of disclaimers.

Use this plan field:
"writing_contract": {
  "central_question": "the question this report answers",
  "reader_takeaway": "what the reader should understand or decide",
  "reading_path": ["first reasoning move", "next reasoning move"],
  "must_keep": ["source-backed detail or caveat that must survive editing"],
  "can_summarize": ["background that may be compressed"],
  "move_to_supporting_layer": ["detail that may move out of the main flow"],
  "visual_role": "the reading job for a visual, or none needed",
  "tone_and_shape": "the intended explanatory stance and shape"
}`
}

func reportNarrativeContractWritingGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileNarrativeContract(profile) {
		return ""
	}
	return `Reader-facing explanation guidance:
- Read and digest the original sources before writing. Then explain the subject directly to a reader who may read only this report.
- The report should sound like a knowledgeable person teaching another person, not like an operator describing how sources were inspected or asking the reader to interpret the sources for themselves.
- Use source details inside the explanation: state the point, explain the mechanism or reasoning, and make clear which source fact, example, comparison, or caveat supports it.
- Follow writing_contract as editorial direction. Preserve must_keep items, compress can_summarize material when useful, and move supporting detail only when the main explanation remains complete.
- Synthesis and practical implications are welcome when they follow from the sources. Mark interpretation, inference, uncertainty, and genuinely missing information at the point where they matter.
- When evidence is limited, say only what the reader needs to understand the boundary, then continue. Do not pad the report with repeated apologies about source scarcity.
- Prefer a coherent answer and natural transitions over a source-by-source tour, while preserving concrete facts, citations, distinctions, and unresolved tensions.`
}
