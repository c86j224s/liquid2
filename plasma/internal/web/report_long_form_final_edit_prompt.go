package web

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func agentLongFormFinalEditPrompt(title string, missionID string, rigor reportRigorProfile, plan agentSectionalReportPlan, generationGuidanceProfile string, binding reporting.LongFormFinalizeBinding, attempt int, canonical bool) string {
	if canonical {
		return `The canonical long-form report already exists. Return exactly REPORT_FINALIZED as the entire response. Do not call a tool and do not add text or fences.`
	}
	guidance := strings.TrimSpace(LongFormReportGenerationGuidance(generationGuidanceProfile))
	retry := ""
	if attempt > 1 {
		retry = "\nThis is the one allowed final-stage retry. Start a new bound draft and repeat the editorial pass from the durable Part artifacts."
	}
	return fmt.Sprintf(`Edit and atomically finalize a Korean long-form Plasma report through the dedicated MCP tools.

Report title: %s
Mission ID: %s

Bound tool inputs:
- session_id: %s
- pending_event_id: %s
- plan_event_id: %s
- producer: {"type":"agent_session","id":%s}
- idempotency key prefix: %s

Overall plan and writing contract:
%s

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s

%s%s

Workflow:
1. Call plasma.report.long_form.final_edit.start. Use the bound identities above and a unique idempotency_key ending in ":start". Keep the returned draft_id.
2. Read the entire manuscript with plasma.report.long_form.final_edit.read. Continue only from the returned next_offset until truncated is false. Do not edit from a partial read.
3. Act as an editor, not a new researcher. Call plasma.report.long_form.final_edit.patch with exact replace, insert_after, or append operations to improve the manuscript. Give every patch a different idempotency_key ending in ":patch-N".
4. Read the affected passages again. Edits can shift UTF-8 byte offsets, so restart at offset 0 and follow only returned next_offset values instead of guessing an offset. Then call plasma.report.long_form.final_edit.submit with the bound pending_event_id and plan_event_id and a unique idempotency_key ending in ":submit".
5. After submit succeeds or durably replays, return exactly REPORT_FINALIZED as the entire response.

Editorial responsibilities:
- Read the full Part manuscript and turn it into one report that directly explains the subject to a reader who may not read the original sources.
- Add or improve the opening, reading path, cross-Part transitions, and conclusion. Merge avoidable repetition and repair abrupt ordering when that makes the explanation easier to follow.
- Preserve the plan writing_contract, especially every must_keep fact, caveat, distinction, example, number, citation, code identifier, and unresolved tension.
- Preserve source attribution and evidence boundaries. Synthesis and practical implications are welcome only when they follow from the existing manuscript; label inference or uncertainty where it matters.
- When evidence is limited, state the boundary once where relevant and continue. Do not pad the report with repeated apologies about source scarcity.
- Do not add new researched facts, call source/research tools, mutate Part or Section artifacts, or expose artifact IDs in the report.
- Keep valid Mermaid blocks intact unless an exact edit is necessary; any edited or added Mermaid block must follow this rule: %s
- Do not mention prompts, experiments, internal run labels, tool session IDs, or temporary implementation details.
- Return no report body in the response.`, title, missionID, binding.ToolSessionID, binding.PendingEventID, binding.PlanEventID, agentReportAnyJSON(binding.ToolSessionID), agentReportAnyJSON(binding.IdempotencyKey), agentReportAnyJSON(plan), rigor.level, rigor.label, rigor.description, rigor.instructions, guidance, retry, reportMermaidValidationRule)
}
