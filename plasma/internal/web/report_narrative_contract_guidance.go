package web

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	reportGenerationGuidanceProfileNarrativeContract                    = "narrative-contract"
	reportGenerationGuidanceProfileReaderParagraphContract              = "reader-paragraph-contract"
	reportGenerationGuidanceProfileCuriosityLedExplanation              = "curiosity-led-explanation"
	reportGenerationGuidanceProfileCuriosityNaturalVoice                = "curiosity-natural-voice"
	reportGenerationGuidanceProfileCuriosityTightVoice                  = "curiosity-tight-voice"
	reportGenerationGuidanceProfileEditedReadingVoice                   = "edited-reading-voice"
	reportGenerationGuidanceProfileSectionBriefNarrativeContract        = "section-brief-narrative-contract"
	reportGenerationGuidanceProfileSectionBriefClusterNarrativeContract = "section-brief-cluster-memory-narrative-contract"
)

func isReportGenerationGuidanceProfileNarrativeContract(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileNarrativeContract, "narrative_contract", "reader-first-editor", "reader_first_editor",
		reportGenerationGuidanceProfileReaderParagraphContract, "reader_paragraph_contract", "direct-explanation-contract", "direct_explanation_contract",
		reportGenerationGuidanceProfileCuriosityLedExplanation, "curiosity_led_explanation", "curiosity-explanation", "curiosity_explanation", "processed-reading-artifact", "processed_reading_artifact",
		reportGenerationGuidanceProfileCuriosityNaturalVoice, "curiosity_natural_voice", "natural-curiosity", "natural_curiosity",
		reportGenerationGuidanceProfileCuriosityTightVoice, "curiosity_tight_voice", "tight-curiosity", "tight_curiosity", "compact-curiosity", "compact_curiosity",
		reportGenerationGuidanceProfileEditedReadingVoice, "edited_reading_voice", "edited-reading", "edited_reading", "reading-editor", "reading_editor",
		reportGenerationGuidanceProfileSectionDirectReadingVoice, "section_direct_reading_voice", "section-direct", "section_direct",
		reportGenerationGuidanceProfilePartConnectiveEconomyVoice, "part_connective_economy_voice", "part-connective-economy", "part_connective_economy",
		reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice,
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

func isReportGenerationGuidanceProfileReaderParagraphContract(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileReaderParagraphContract, "reader_paragraph_contract", "direct-explanation-contract", "direct_explanation_contract":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileCuriosityLedExplanation(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileCuriosityLedExplanation, "curiosity_led_explanation", "curiosity-explanation", "curiosity_explanation", "processed-reading-artifact", "processed_reading_artifact",
		reportGenerationGuidanceProfileCuriosityNaturalVoice, "curiosity_natural_voice", "natural-curiosity", "natural_curiosity",
		reportGenerationGuidanceProfileCuriosityTightVoice, "curiosity_tight_voice", "tight-curiosity", "tight_curiosity", "compact-curiosity", "compact_curiosity",
		reportGenerationGuidanceProfileEditedReadingVoice, "edited_reading_voice", "edited-reading", "edited_reading", "reading-editor", "reading_editor",
		reportGenerationGuidanceProfileSectionDirectReadingVoice, "section_direct_reading_voice", "section-direct", "section_direct",
		reportGenerationGuidanceProfilePartConnectiveEconomyVoice, "part_connective_economy_voice", "part-connective-economy", "part_connective_economy",
		reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice:
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileCuriosityNaturalVoice(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileCuriosityNaturalVoice, "curiosity_natural_voice", "natural-curiosity", "natural_curiosity",
		reportGenerationGuidanceProfileCuriosityTightVoice, "curiosity_tight_voice", "tight-curiosity", "tight_curiosity", "compact-curiosity", "compact_curiosity",
		reportGenerationGuidanceProfileEditedReadingVoice, "edited_reading_voice", "edited-reading", "edited_reading", "reading-editor", "reading_editor",
		reportGenerationGuidanceProfileSectionDirectReadingVoice, "section_direct_reading_voice", "section-direct", "section_direct",
		reportGenerationGuidanceProfilePartConnectiveEconomyVoice, "part_connective_economy_voice", "part-connective-economy", "part_connective_economy",
		reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice:
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileCuriosityTightVoice(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileCuriosityTightVoice, "curiosity_tight_voice", "tight-curiosity", "tight_curiosity", "compact-curiosity", "compact_curiosity":
		return true
	default:
		return false
	}
}

func isReportGenerationGuidanceProfileEditedReadingVoice(profile string) bool {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case reportGenerationGuidanceProfileEditedReadingVoice, "edited_reading_voice", "edited-reading", "edited_reading", "reading-editor", "reading_editor",
		reportGenerationGuidanceProfileSectionDirectReadingVoice, "section_direct_reading_voice", "section-direct", "section_direct",
		reportGenerationGuidanceProfilePartConnectiveEconomyVoice, "part_connective_economy_voice", "part-connective-economy", "part_connective_economy",
		reportGenerationGuidanceProfilePartConnectiveSubjectDirectSynthesisVoice:
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
	guidance := `Reader-facing writing-contract guidance:
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
	if extra := strings.TrimSpace(reportReaderParagraphContractPlanningGuidance(profile)); extra != "" {
		guidance += "\n\n" + extra
	}
	if extra := strings.TrimSpace(reportCuriosityLedExplanationPlanningGuidance(profile)); extra != "" {
		guidance += "\n\n" + extra
	}
	if extra := strings.TrimSpace(reportCuriosityNaturalVoicePlanningGuidance(profile)); extra != "" {
		guidance += "\n\n" + extra
	}
	if extra := strings.TrimSpace(reportCuriosityTightVoicePlanningGuidance(profile)); extra != "" {
		guidance += "\n\n" + extra
	}
	if extra := strings.TrimSpace(reportEditedReadingVoicePlanningGuidance(profile)); extra != "" {
		guidance += "\n\n" + extra
	}
	return guidance
}

func reportNarrativeContractWritingGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileNarrativeContract(profile) {
		return ""
	}
	guidance := `Reader-facing explanation guidance:
- Read and digest the original sources before writing. Then explain the subject directly to a reader who may read only this report.
- The report should sound like a knowledgeable person teaching another person, not like an operator describing how sources were inspected or asking the reader to interpret the sources for themselves.
- Use source details inside the explanation: state the point, explain the mechanism or reasoning, and make clear which source fact, example, comparison, or caveat supports it.
- Follow writing_contract as editorial direction. Preserve must_keep items, compress can_summarize material when useful, and move supporting detail only when the main explanation remains complete.
- Synthesis and practical implications are welcome when they follow from the sources. Mark interpretation, inference, uncertainty, and genuinely missing information at the point where they matter.
- When evidence is limited, say only what the reader needs to understand the boundary, then continue. Do not pad the report with repeated apologies about source scarcity.
- Prefer a coherent answer and natural transitions over a source-by-source tour, while preserving concrete facts, citations, distinctions, and unresolved tensions.`
	if extra := strings.TrimSpace(reportReaderParagraphContractWritingGuidance(profile)); extra != "" {
		guidance += "\n\n" + extra
	}
	if extra := strings.TrimSpace(reportCuriosityLedExplanationWritingGuidance(profile)); extra != "" {
		guidance += "\n\n" + extra
	}
	if extra := strings.TrimSpace(reportCuriosityNaturalVoiceWritingGuidance(profile)); extra != "" {
		guidance += "\n\n" + extra
	}
	if extra := strings.TrimSpace(reportCuriosityTightVoiceWritingGuidance(profile)); extra != "" {
		guidance += "\n\n" + extra
	}
	if extra := strings.TrimSpace(reportEditedReadingVoiceWritingGuidance(profile)); extra != "" {
		guidance += "\n\n" + extra
	}
	return guidance
}

func reportReaderParagraphContractPlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileReaderParagraphContract(profile) {
		return ""
	}
	return `Reader paragraph-contract planning guidance:
- Keep the submitted plan schema unchanged. Do not add reader_brief, curiosity_map, paragraph_plan, claim_source_map, paragraph_quality_pass, or similar custom fields.
- Put the report-level reader brief into writing_contract: the reader's actual question, the decision or understanding they should reach, and the few reasoning moves that make the explanation readable.
- Put curiosity-path notes into reading_path and tone_and_shape: what should become clearer first, what tension keeps the reader moving, and where the report should slow down for evidence.
- For long-form reports, write each Part purpose as the reader movement it owns. Write each Section purpose as a compact paragraph plan in natural prose: opening promise, controlling idea, evidence path, necessary caveat, and transition role.
- Use coverage_notes only as compact claim-source memory: source-backed clusters, unresolved boundaries, and details that must stay available without interrupting the main explanation.
- Do not turn purposes into labels or checklist prose. They should help the later writer decide what each paragraph is for.`
}

func reportReaderParagraphContractWritingGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileReaderParagraphContract(profile) {
		return ""
	}
	return `Reader paragraph-contract writing guidance:
- Before drafting each section, treat the Section purpose as a paragraph plan rather than a topic label.
- Each paragraph should make an early promise to the reader, then connect the point, source-backed evidence or mechanism, caveat if needed, and implication for the report's main question.
- Avoid source-by-source tours, generic background detours, and repeated meta-openers. Use sources inside the explanation instead of making the reader assemble the explanation from source inventory.
- After drafting, silently run a paragraph_quality_pass: the early promise matches the paragraph, details support one controlling idea, caveats sit near the claims they limit, and the transition moves the reader forward.
- Do not mention reader_brief, curiosity_map, paragraph_plan, claim_source_map, paragraph_quality_pass, hidden guidance, or experiment labels in the report.`
}

func reportCuriosityLedExplanationPlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileCuriosityLedExplanation(profile) {
		return ""
	}
	return `Curiosity-led explanation planning guidance:
- Treat the report as a processed reading artifact, not a source annotation layer, source inventory, or investigation log.
- Keep the submitted plan schema unchanged. Do not add curiosity_map, information_gap, tension_map, payoff_map, reading_hook, or similar custom fields.
- Use writing_contract.central_question and reader_takeaway to name why the reader should care and what understanding, insight, or decision payoff the artifact should deliver.
- Use writing_contract.reading_path to plan the curiosity path: initial information gap or tension, partial resolution, next question, surprising connection, and final payoff.
- Use Part and Section purpose strings to state each local reading job: what question, contrast, uncertainty, market signal, or insight path makes this part worth continuing, and what it resolves.
- Use coverage_notes as source-detail memory for facts, caveats, numbers, examples, and counterpoints that must support the reading path without taking over the surface structure.
- Adapt the rhythm to the user's purpose: learning, insight synthesis, gathered reading, market research, or decision support.`
}

func reportCuriosityLedExplanationWritingGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileCuriosityLedExplanation(profile) {
		return ""
	}
	return `Curiosity-led explanation writing guidance:
- Write the artifact from the reader's reason to care, not from the source list or the investigation order.
- Create useful information gaps, tensions, contrasts, or insight paths, then resolve them enough that the reader understands why the next part matters.
- Use sources as material inside the explanation. Do not organize the surface as "source A says, source B says" unless the source disagreement itself is the point.
- Long is acceptable only when each paragraph moves understanding, curiosity, synthesis, or decision support forward.
- Put caveats, weak evidence, and source boundaries exactly where they change the claim. Do not let repeated evidence disclaimers become the main rhythm of the prose.
- Prefer concrete mechanisms, examples, comparisons, and implications over abstract meta-signposting. Make the writing feel edited for a reader, not assembled from research notes.
- Do not mention curiosity-led explanation, curiosity_map, information_gap, tension_map, payoff_map, processed reading artifact, hidden guidance, or experiment labels in the report.`
}

func reportCuriosityNaturalVoicePlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileCuriosityNaturalVoice(profile) {
		return ""
	}
	return `Natural curiosity-voice planning guidance:
- Keep the curiosity path, but plan fewer visible signposts. In writing_contract.tone_and_shape, name the report's natural stance without prescribing repeated transition formulas.
- In reading_path, mark only the few places where a caveat materially changes the reader's understanding. Do not plan a disclaimer rhythm for every section.
- In Part and Section purposes, prefer the concrete question, contrast, scene, or mechanism over abstract labels such as importance, complexity, or key implication.
- Keep the submitted plan schema unchanged. Do not add voice_pass, style_pass, caveat_budget, signpost_map, or similar custom fields.`
}

func reportCuriosityNaturalVoiceWritingGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileCuriosityNaturalVoice(profile) {
		return ""
	}
	return `Natural curiosity-voice writing guidance:
- Keep the curiosity-led explanation structure, but remove prose that sounds like the report is announcing its own method.
- Avoid repeated stock emphasis frames such as "핵심은", "중요한 점은", "주목할 점은", "결론적으로", or "따라서". Use them only when they fit the sentence, not as paragraph machinery.
- Do not begin many paragraphs by telling the reader what the section will do. Start with the actual claim, question, contrast, scene, mechanism, or consequence.
- State a caveat once where it changes a claim, then continue with the substance. Do not repeat "the available sources do not show" or similar source-boundary language after the boundary is already clear.
- Vary paragraph endings. Not every paragraph needs a neat lesson, takeaway, implication sentence, or miniature conclusion.
- Prefer specific nouns and verbs from the material over vague labels such as interesting, important, complex, multi-layered, or meaningful unless the next words show why.
- Do not include horizontal-rule separators unless the user explicitly requested that format.
- Do not mention natural curiosity voice, voice_pass, style_pass, caveat_budget, signpost_map, hidden guidance, or experiment labels in the report.`
}

func reportCuriosityTightVoicePlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileCuriosityTightVoice(profile) {
		return ""
	}
	return `Tight curiosity-voice planning guidance:
- Keep the curiosity path and natural voice, but do not plan extra prose just to make the artifact feel warmer or more complete.
- Use writing_contract.can_summarize and move_to_supporting_layer aggressively for background, repeated mechanisms, repeated source-boundary notes, and examples that support the same point.
- In reading_path, keep only reasoning moves that change the reader's understanding. Remove planned moves that merely restate the question, repeat the caveat, or summarize the previous paragraph.
- In Part and Section purposes, state the one job that earns the section's space. Split a section only when different claims, mechanisms, evidence clusters, or caveats would otherwise be blurred together.
- Keep the submitted plan schema unchanged. Do not add compactness_pass, paragraph_budget, caveat_ledger, compression_pass, or similar custom fields.`
}

func reportCuriosityTightVoiceWritingGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileCuriosityTightVoice(profile) {
		return ""
	}
	return `Tight curiosity-voice writing guidance:
- Natural voice means less framing, not more prose. Do not add setup, reassurance, or recap sentences merely to sound conversational.
- A paragraph earns its space only when it adds a new claim, mechanism, source-backed detail, contrast, caveat, or transition that changes the reader's understanding. Merge or delete paragraphs that only echo the previous point.
- Keep one clear caveat where it limits the claim. When the same source boundary affects later claims, refer back briefly or let the earlier boundary stand instead of restating it.
- Prefer one specific example over several examples that prove the same point. Keep additional examples only when they change the mechanism, scale, exception, or decision implication.
- Keep the finished report near the curiosity-led candidate's density unless the sources genuinely require more coverage. Do not become shorter by dropping must_keep facts, citations, counterpoints, or unresolved tensions.
- After drafting or final editing, silently compress repeated signposts, repeated caveats, repeated section previews, and tidy mini-conclusions before submitting.
- Do not mention tight curiosity voice, compactness_pass, paragraph_budget, caveat_ledger, compression_pass, hidden guidance, or experiment labels in the report.`
}

func reportEditedReadingVoicePlanningGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileEditedReadingVoice(profile) {
		return ""
	}
	return `Edited reading-voice planning guidance:
- Keep the curiosity path and natural voice, but plan the artifact as edited reading material rather than a shorter report.
- In writing_contract.tone_and_shape, name the intended reader experience in plain language: what should feel worth reading, where the prose should slow down, and where it should move quickly.
- Plan titles and headings in the user's report language unless the source title is a proper noun, product name, or code identifier. Do not let English source headings leak into a Korean report title by default.
- Use writing_contract.reading_path to plan the first real subject move, not a sentence about what the report will do.
- Use can_summarize for repetitive background, but keep source-backed examples, numbers, mechanisms, and caveats when they make the reading more concrete.
- Keep the submitted plan schema unchanged. Do not add editor_pass, title_language, prose_rhythm, self_framing_check, or similar custom fields.`
}

func reportEditedReadingVoiceWritingGuidance(profile string) string {
	if !isReportGenerationGuidanceProfileEditedReadingVoice(profile) {
		return ""
	}
	return `Edited reading-voice writing guidance:
- Write as if an editor turned research into a readable article for the user. Do not write as if the artifact is introducing, explaining, or justifying itself.
- Avoid self-framing sentences such as "이 보고서는...", "이 글은...", or "자료는..." when the sentence can begin with the subject itself. Use them only when the report scope would otherwise be unclear.
- Keep headings in one deliberate language and depth pattern. In Korean reports, translate ordinary topic headings to Korean unless the term is a proper noun, product name, or source title that should remain visible.
- Keep the opening close to the subject: start with the concrete tension, mechanism, scene, or consequence, then bring in source scope only when it limits a claim.
- Do not make compactness the goal. A shorter paragraph is better only when it preserves the detail that makes the point understandable and worth reading.
- Vary paragraph rhythm without adding chatter. Cut repeated previews, repeated caveats, and tidy mini-conclusions, but keep explanations that create understanding or insight.
- Do not mention edited reading voice, editor_pass, title_language, prose_rhythm, self_framing_check, hidden guidance, or experiment labels in the report.`
}
