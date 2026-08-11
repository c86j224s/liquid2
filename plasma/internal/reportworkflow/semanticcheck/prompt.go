package semanticcheck

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

func prompt(input finaledit.Input, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Run read-only style semantic validation for the long-form report through MCP.

Report title: %s
Mission ID: %s
Bound stage metadata:
%s

Use exactly this workflow:
1. Read changed reader/style paragraph comparisons with %s until truncated is false.
2. Submit with %s using semantic_acceptance. For each changed paragraph submit paragraph_ordinal and exactly one verdict: accepted_equivalent or rejected_revert_to_reader.
3. Return exactly %s and nothing else after submit succeeds.

Validation responsibilities:
- Judge only whether the style paragraph preserves the reader paragraph's meaning.
- Do not submit prose, patches, final paragraph ordinals, repaired_by_gate, manuscript Markdown, or repair instructions.
- When uncertain, use rejected_revert_to_reader.%s`,
		input.Title, input.MissionID, finaledit.AgentReportAnyJSON(binding),
		mcptools.ToolReportLongFormStyleSemanticValidationRead,
		mcptools.ToolReportLongFormStyleSemanticValidationSubmit,
		finaledit.StageSubmittedSentinel, finaledit.RetryNote(attempt))
}
