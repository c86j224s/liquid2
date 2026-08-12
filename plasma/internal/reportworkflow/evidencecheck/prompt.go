package evidencecheck

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

func evidencePrompt(input finaledit.Input, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Run read-only evidence connection judgment and canonicalize the long-form report through MCP.

Report title: %s
Mission ID: %s
Rigor: %s (%s)
Bound stage metadata:
%s

Use exactly this workflow:
1. Use draft_id %q for every evidence-gate read and submit. Use session_id %q, which is the bound tool_session_id; provider session IDs are not MCP session IDs. Do not create or switch drafts.
2. Start %s at offset 0. While truncated is true, call the same tool with the same draft_id and session_id and copy the returned next_offset exactly. Stop reading only after truncated is false.
3. If a contract error returns continuation content, follow its draft_id, session_id, next_offset, and next_action instead of starting another draft.
4. Use only server-provided statement_sha256 values from the completed packet. Use approved read tools to verify report-to-evidence connections when support is unclear.
5. After the packet is complete, call %s exactly once with the same draft_id and session_id and gate_findings. Each finding may contain only statement_sha256, classification, and approved evidence_ids. Use only these classifications: mission_source_grounded, session_grounded, derived_synthesis, rhetorical_construction, unverified_external_fact.
6. Return exactly %s and nothing else after submit succeeds.

Evidence gate responsibilities:
- Judge report-to-evidence connections only.
- Do not judge owner requirements, prose quality, style, or structure.
- Do not calculate statement hashes; copy statement_sha256 exactly from the read packet.
- Do not submit prose, patches, repair actions, manuscript Markdown, semantic acceptance, or operation counts.
- Evidence judgments do not trigger automatic repair; the server canonicalizes the exact bound source artifact with zero operations.%s`,
		input.Title, input.MissionID, input.Rigor.Level, input.Rigor.Label, finaledit.AgentReportAnyJSON(binding),
		draftID, binding.ToolSessionID,
		mcptools.ToolReportLongFormEvidenceGateRead,
		mcptools.ToolReportLongFormEvidenceGateSubmit,
		finaledit.GateSubmittedSentinel, finaledit.RetryNote(attempt))
}

func gatePrompt(input finaledit.Input, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Run the corrective provenance gate and canonicalize the long-form report through MCP.

Report title: %s
Mission ID: %s
Rigor: %s (%s)
Bound gate metadata:
%s

Global requirement preservation checks:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full manuscript with %s until truncated is false.
3. Use approved read tools to verify claims when support is unclear. Do not mutate sources or create new policy.
4. Apply required corrections with %s before submit.
5. Submit with %s and gate_findings. If there are no findings, pass an empty array. Use only these classifications: mission_source_grounded, session_grounded, derived_synthesis, rhetorical_construction, unverified_external_fact. Use only these repair actions: attach_approved_evidence, qualify_inference_or_uncertainty, retain_with_footnote, remove.
6. Return exactly %s and nothing else after submit succeeds.

Gate responsibilities:
- Read the complete manuscript before judging it.
- Enforce source/evidence boundaries and every owner-bound output requirement according to the rigor level.
- Order repairs before canonicalization; the gate is the only canonical producer.
- Do not include raw statement text anywhere except the transient gate_findings tool input.%s`,
		input.Title, input.MissionID, input.Rigor.Level, input.Rigor.Label, finaledit.AgentReportAnyJSON(binding),
		finaledit.AgentReportAnyJSON(reporting.ReportOwnerBoundRequirements(input.RequirementMap)),
		mcptools.ToolReportLongFormEditStart, draftID,
		mcptools.ToolReportLongFormEditRead, mcptools.ToolReportLongFormEditPatch,
		mcptools.ToolReportLongFormEditSubmit, finaledit.GateSubmittedSentinel, finaledit.RetryNote(attempt))
}

func semanticGatePrompt(input finaledit.Input, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Run the corrective provenance gate and canonicalize the long-form report through MCP.

Report title: %s
Mission ID: %s
Rigor: %s (%s)
Bound gate metadata:
%s

Global requirement preservation checks:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full manuscript with %s until truncated is false.
3. Use approved read tools to verify claims when support is unclear. Do not mutate sources or create new policy.
4. Read changed style paragraphs with %s until truncated is false. Follow next_offset exactly; the gate cannot submit until this bounded review read returns truncated=false. For each changed source paragraph, submit paragraph_ordinal, final_paragraph_ordinal, and exactly one semantic verdict: accepted_equivalent when style and final meaning are equivalent, reverted_to_reader when final text returns to reader meaning, or repaired_by_gate when you repaired unsafe style drift.
5. Apply required corrections with %s before submit. Semantic review is not an invitation to rewrite; fail closed on uncertainty by reverting the affected paragraph to reader meaning or repairing only the unsafe local drift.
6. Submit with %s, gate_findings, and semantic_acceptance. If there are no findings, pass an empty array. If there were no changed style paragraphs, omit semantic_acceptance or pass an empty array. Use only these classifications: mission_source_grounded, session_grounded, derived_synthesis, rhetorical_construction, unverified_external_fact. Use only these repair actions: attach_approved_evidence, qualify_inference_or_uncertainty, retain_with_footnote, remove.
7. Return exactly %s and nothing else after submit succeeds.

Gate responsibilities:
- Read the complete manuscript before judging it.
- Enforce source/evidence boundaries and every owner-bound output requirement according to the rigor level.
- Order repairs before canonicalization; the gate is the only canonical producer.
- Do not include raw statement text anywhere except the transient gate_findings tool input.%s`,
		input.Title, input.MissionID, input.Rigor.Level, input.Rigor.Label, finaledit.AgentReportAnyJSON(binding),
		finaledit.AgentReportAnyJSON(reporting.ReportOwnerBoundRequirements(input.RequirementMap)),
		mcptools.ToolReportLongFormEditStart, draftID,
		mcptools.ToolReportLongFormEditRead, mcptools.ToolReportLongFormStyleReviewRead, mcptools.ToolReportLongFormEditPatch,
		mcptools.ToolReportLongFormEditSubmit, finaledit.GateSubmittedSentinel, finaledit.RetryNote(attempt))
}

func promptForHumanize(humanize string) finaledit.StagePrompt {
	if humanize == reporting.FinalEditHumanizeEnabled {
		return semanticGatePrompt
	}
	return gatePrompt
}
