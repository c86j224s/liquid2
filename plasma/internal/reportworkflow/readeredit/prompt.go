package readeredit

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

func prompt(input finaledit.Input, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	if pipeline := input.FinalEditPipeline(); pipeline == reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 || pipeline == reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		return v2Prompt(input, binding, draftID, attempt)
	}
	text := fmt.Sprintf(`Read and edit the durable long-form Part manuscript through the dedicated reader-edit MCP tools.

Report title: %s
Mission ID: %s
Overall writing contract:
%s

Bound stage metadata:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full manuscript with %s until truncated is false.
3. Use %s for reader-facing edits whenever the manuscript can be made clearer for a report-only reader without changing meaning: improve opening, transitions, ordering, repetition, conclusion, clarity, and direct explanation of the subject.
4. Submit with %s using the same draft_id, pending_event_id, and plan_event_id.
5. Return exactly %s and nothing else.

Reader-edit responsibilities:
- Explain the subject as the report's author to a reader who will only see this report. Digest the material and present the explanation instead of telling the reader how to interpret the sources. Use source-boundary language only where it changes claim scope or certainty.
- Do not optimize for brevity by itself. Keep or add explanation when it makes a supported concept, causal link, context, condition, example, or technical detail easier to understand; remove prose only when it does not advance the reader's understanding.
- Keep or create a brief report-level opening that states the subject, central question, and main answer or evidence boundary. Treat this orientation as useful content, not removable meta-signposting.
- Let later transitions follow the subject and the reader's next question. Remove repeated section roadmaps or writing-process narration, but keep transitions that add context, logic, or stakes. Clean obviously duplicated headings when their intended form is clear.
- Preserve every unique fact, citation, caveat meaning, number, code identifier, technical identifier, uncertainty boundary, and assigned requirement.
- Consolidate redundant caveats and source-process narration without losing unique information; keep the remaining limit near the claim it qualifies rather than repeating investigation-log phrasing.
- Judge repetition by function: keep a brief reminder when a long-form reader or a new context needs it; remove adjacent restatements and section-level duplication, keeping the strongest occurrence and merging unique detail into it.
- Submit unchanged only after a full read finds none of these responsibilities applicable.

Do not call research or source tools. Do not expose IDs in the manuscript.%s`,
		input.Title, input.MissionID, finaledit.AgentReportAnyJSON(map[string]any{"writing_contract": input.Plan.WritingContract}), finaledit.AgentReportAnyJSON(binding),
		mcptools.ToolReportLongFormReaderEditStart, draftID,
		mcptools.ToolReportLongFormReaderEditRead, mcptools.ToolReportLongFormReaderEditPatch,
		mcptools.ToolReportLongFormReaderEditSubmit, finaledit.StageSubmittedSentinel, finaledit.RetryNote(attempt))
	return reportprompt.WithLongFormDownstreamDirection(text, input.DirectionHint)
}

func v2Prompt(input finaledit.Input, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	text := fmt.Sprintf(`Read and edit the final-writer manuscript through the dedicated reader-edit MCP tools.

Report title: %s
Mission ID: %s
Overall writing contract:
%s

Bound stage metadata:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full manuscript with %s until truncated is false.
3. Use %s for reader-facing edits that improve direct explanation, paragraph flow, comprehension order, memory-supporting accumulation, and awkward wording without changing meaning.
4. Submit with %s using the same draft_id, pending_event_id, and plan_event_id.
5. Return exactly %s and nothing else.

Reader-edit responsibilities:
- Explain the subject as the report's author to a reader who will only see this report. Improve local clarity, sequence, and paragraph-level comprehension.
- Do not create a new opening, new conclusion, global redesign, full Part or Section reorder, or cross-Part restructure; those are final-writer responsibilities already completed.
- Do not optimize for brevity by itself. Keep or add explanation when it makes a supported concept, causal link, condition, example, or technical detail easier to understand.
- Preserve every unique fact, citation, caveat meaning, number, code identifier, technical identifier, uncertainty boundary, and assigned requirement.
- Consolidate adjacent repetition only when no unique information is lost; keep memory-supporting reminders when a long-form reader needs them.
- Submit unchanged only after a full read finds none of these responsibilities applicable.

Do not call research or source tools. Do not expose IDs in the manuscript.%s`,
		input.Title, input.MissionID, finaledit.AgentReportAnyJSON(map[string]any{"writing_contract": input.Plan.WritingContract}), finaledit.AgentReportAnyJSON(binding),
		mcptools.ToolReportLongFormReaderEditStart, draftID,
		mcptools.ToolReportLongFormReaderEditRead, mcptools.ToolReportLongFormReaderEditPatch,
		mcptools.ToolReportLongFormReaderEditSubmit, finaledit.StageSubmittedSentinel, finaledit.RetryNote(attempt))
	return reportprompt.WithLongFormDownstreamDirection(text, input.DirectionHint)
}
