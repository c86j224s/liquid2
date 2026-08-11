package legacyfinalize

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

// PromptWithRequirements는 legacy finalizer prompt bytes를 기존 Web 구현과 동일하게 만든다.
func PromptWithRequirements(input Input, binding reporting.LongFormFinalizeBinding, attempt int, canonical bool, hint reporting.LongFormFinalizationHint) string {
	if reportprompt.IsNarrativeContract(input.GenerationGuidanceProfile) {
		return finalEditPrompt(input, binding, attempt, canonical)
	}
	guidance := strings.TrimSpace(reportprompt.LongFormReportGenerationGuidance(input.GenerationGuidanceProfile))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	retry := ""
	if attempt > 1 {
		retry = "\nThis is the one allowed final-stage retry."
		if canonical {
			retry += " The canonical report already exists: replay the same tool call with identical opening_markdown and closing_markdown, then return the sentinel."
		}
		if hint.Available {
			retry += fmt.Sprintf("\nUse these recovered values only as writing hints when making the tool call:\nopening_markdown hint: %s\nclosing_markdown hint: %s", finaledit.AgentReportAnyJSON(hint.OpeningMarkdown), finaledit.AgentReportAnyJSON(hint.ClosingMarkdown))
		}
	}
	return fmt.Sprintf(`Finalize a Korean long-form Plasma report through the dedicated MCP command.

Report title: %s
Mission ID: %s

Bound tool inputs:
- session_id: %s
- pending_event_id: %s
- plan_event_id: %s
- idempotency_key: %s
- producer: {"type":"agent_session","id":%s}

The Part manuscripts are already written and will be mechanically preserved by Plasma. Do not rewrite them.

Part inventory:
%s

Overall plan:
%s

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s%s

Rules:
- Use Korean.
- Call plasma.report.long_form.finalize with exactly the bound identities and producer above plus opening_markdown and closing_markdown that you write.
- opening_markdown contains the Markdown title, introduction, reading guide, and compact table of contents.
- closing_markdown contains the conclusion that synthesizes tensions, supported conclusions, remaining uncertainty, and useful next checks.
- %s
- The server owns ordered Part assembly. Do not submit Part bodies, artifact IDs, title, full Markdown, or metadata.
- After the tool succeeds or durably replays, return exactly REPORT_FINALIZED as the entire response. Do not add text or fences.
- Do not mention prompts, experiments, internal run labels, tool session IDs, or temporary implementation details.`, input.Title, input.MissionID, binding.ToolSessionID, binding.PendingEventID, binding.PlanEventID, binding.IdempotencyKey, finaledit.AgentReportAnyJSON(binding.ToolSessionID), partInventoryJSON(input.Parts), finaledit.AgentReportAnyJSON(input.Plan), input.Rigor.Level, input.Rigor.Label, input.Rigor.Description, input.Rigor.Instructions, guidance, retry, reportprompt.MermaidValidationRuleText)
}

func finalEditPrompt(input Input, binding reporting.LongFormFinalizeBinding, attempt int, canonical bool) string {
	if canonical {
		return `The canonical long-form report already exists. Return exactly REPORT_FINALIZED as the entire response. Do not call a tool and do not add text or fences.`
	}
	guidance := strings.TrimSpace(reportprompt.LongFormReportGenerationGuidance(input.GenerationGuidanceProfile))
	retry := ""
	if attempt > 1 {
		retry = "\nThis is the one allowed final-stage retry. Start a new bound draft and repeat the editorial pass from the durable Part artifacts."
	}
	ownerBoundRequirements := reporting.ReportOwnerBoundRequirements(input.RequirementMap)
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

Global requirement preservation checks:
%s

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s

%s%s

Workflow:
1. Call plasma.report.long_form.final_edit.start. Use the bound identities above and a unique idempotency_key ending in ":start". Keep the returned draft_id.
2. Read the entire manuscript with plasma.report.long_form.final_edit.read. Continue only from the returned next_offset until truncated is false. Do not edit from a partial read.
3. Act as an editor, not a new researcher. Call plasma.report.long_form.final_edit.patch with exact replace, insert_after, or append operations only when a material manuscript edit is justified. Give every patch a different idempotency_key ending in ":patch-N".
4. If you patched, read the affected passages again. Edits can shift UTF-8 byte offsets, so restart at offset 0 and follow only returned next_offset values instead of guessing an offset. If no material edit is justified after the full read, submit the unchanged draft. Then call plasma.report.long_form.final_edit.submit with the bound pending_event_id and plan_event_id and a unique idempotency_key ending in ":submit".
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
- Return no report body in the response.`, input.Title, input.MissionID, binding.ToolSessionID, binding.PendingEventID, binding.PlanEventID, finaledit.AgentReportAnyJSON(binding.ToolSessionID), finaledit.AgentReportAnyJSON(binding.IdempotencyKey), finaledit.AgentReportAnyJSON(input.Plan), finaledit.AgentReportAnyJSON(ownerBoundRequirements), input.Rigor.Level, input.Rigor.Label, input.Rigor.Description, input.Rigor.Instructions, guidance, retry, reportprompt.MermaidValidationRuleText)
}

func partInventoryJSON(parts []Part) string {
	items := make([]map[string]any, 0, len(parts))
	for index, part := range parts {
		items = append(items, map[string]any{
			"part_index":  index + 1,
			"title":       part.Title,
			"artifact_id": part.ArtifactID,
			"word_count":  part.WordCount,
		})
	}
	return finaledit.AgentReportAnyJSON(items)
}
