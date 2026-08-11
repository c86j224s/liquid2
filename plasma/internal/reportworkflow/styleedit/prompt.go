package styleedit

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

func prompt(input finaledit.Input, binding reporting.FinalEditStageBinding, draftID string, attempt int) string {
	return fmt.Sprintf(`Apply the pre-canonical Korean natural-voice style pass through the dedicated style-edit MCP tools.

Report title: %s
Mission ID: %s
Bound stage metadata:
%s

Use exactly this workflow:
1. Call %s with draft_id %q and the bound identities.
2. Read the full manuscript with %s until truncated is false.
3. Privately diagnose every paragraph before patching. Use only these categories and meanings:
   - opaque_or_strained_mapping = relationship between domains is not quickly recoverable, collocation feels invented/strained, or image adds interpretation cost without explanatory gain; do not prohibit metaphor and preserve conventional/clarifying metaphor.
   - unnatural_collocation = grammatical words that do not sound like normal Korean report prose in context.
   - vague_reference = unclear/cheap pointer instead of naming the referent.
   - nominalized_or_bureaucratic = noun/process-heavy phrase hiding a simple action.
   - compressed_abstraction = several ideas packed into an abstract phrase that costs effort to unpack.
   - report_process_meta = report/section narration where subject matter should be foregrounded.
   - formulaic_transition = stock movement announcement with no useful logic.
4. No edit quota/minimum.
5. Use %s only with exact replace operations. Never use insert_after, append, or replace_all. Empty replacement is allowed only for a diagnosed local deletion that leaves its Markdown block non-empty.
6. Each patch summary must use exactly this format: category: <one-known-token>; <concrete issue>. The category token must be one of the seven diagnosis categories above.
7. Preserve structure, claims, citations, and paragraph boundaries. Never change heading lines, table rows, list markers, blockquote lines, code fences or fenced code, or source/reference lines.
8. In every replacement, keep the exact ordered sequence of numbers, Latin technical tokens, inline-code spans, quoted spans, links, footnotes, and citation markers. If a repair overlaps one, copy the protected token or span verbatim and edit only the surrounding Korean; otherwise skip the repair.
9. For report_process_meta, skip the repair when the navigation phrase contains a number or Latin token. Forbidden examples: changing "1부에서" to "앞에서", changing "이 Part에서는" to "여기서는", or rewriting any table row.
10. Treat each successful replacement as the current draft. Before a later repair that overlaps earlier text, read the affected range again and use the current exact match.
11. Before submit, reread every changed range and verify that its protected sequences are unchanged. Repair any mismatch before submitting.
12. Submit unchanged if no safe local repair is justified.
13. Submit with %s. After submit succeeds, make no further tool calls and return exactly %s.

Do not summarize, add facts, call research/source tools, or expose IDs in the manuscript.%s`,
		input.Title, input.MissionID, finaledit.AgentReportAnyJSON(binding),
		mcptools.ToolReportLongFormStyleEditStart, draftID,
		mcptools.ToolReportLongFormStyleEditRead, mcptools.ToolReportLongFormStyleEditPatch,
		mcptools.ToolReportLongFormStyleEditSubmit, finaledit.StageSubmittedSentinel, finaledit.RetryNote(attempt))
}
