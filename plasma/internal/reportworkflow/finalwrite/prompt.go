package finalwrite

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

func prompt(input finaledit.Input, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	text := fmt.Sprintf(`Write the final long-form manuscript from the deterministic assembly through the dedicated final-write MCP tools.

Report title: %s
Mission ID: %s
Overall writing contract:
%s

Bound stage metadata:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full deterministic assembly with %s until truncated is false.
3. Use %s for final writing edits that improve whole-report opening, conclusion, Part transitions, global logic, and cross-Part duplicate paragraphs without changing the report's evidence boundary.
4. Submit with %s using the same draft_id, pending_event_id, and plan_event_id.
5. Return exactly %s and nothing else.

Final-writer responsibilities:
- You may create or improve a whole-report opening, a conclusion, Part-to-Part transitions, and global connective logic.
- You may merge or move duplicate paragraphs across Parts when they repeat the same function, but preserve Part order and do not perform a full Part or Section reorder.
- Preserve every unique fact, number, condition, citation, uncertainty, owner requirement, caveat meaning, code identifier, technical identifier, and cited relationship.
- Do not add research, external facts, new sources, new citations, unsupported claims, or new policy.
- Keep source-boundary and uncertainty language where it changes claim scope; do not erase model/session-memory-supported connective reasoning merely because it is synthesis.
- Submit unchanged only after a full read finds no justified final-writing edit.

Do not call research or source tools. Do not expose IDs in the manuscript.%s`,
		input.Title, input.MissionID, finaledit.AgentReportAnyJSON(map[string]any{"writing_contract": input.Plan.WritingContract}), finaledit.AgentReportAnyJSON(binding),
		mcptools.ToolReportLongFormFinalWriteStart, draftID,
		mcptools.ToolReportLongFormFinalWriteRead, mcptools.ToolReportLongFormFinalWritePatch,
		mcptools.ToolReportLongFormFinalWriteSubmit, finaledit.StageSubmittedSentinel, finaledit.RetryNote(attempt))
	return reportprompt.WithLongFormDownstreamDirection(text, input.DirectionHint)
}
