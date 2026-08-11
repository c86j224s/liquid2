package partedit

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// Prompt는 Part editor 또는 final author 역할에 맞는 MCP edit prompt bytes를 반환한다.
func Prompt(input Input, binding reporting.PartEditBinding, draftID string) string {
	if input.AuthorMode {
		return authorPrompt(input, binding, draftID)
	}
	return editorPrompt(input, binding, draftID)
}

func editorPrompt(input Input, binding reporting.PartEditBinding, draftID string) string {
	requirements := reporting.ReportRequirementsForPart(input.Base.RequirementMap, input.PartIndex+1)
	adjacentBoundaryGuidance := strings.TrimSpace(reportprompt.PartAdjacentBoundaryEditGuidance(input.Base.GenerationGuidanceProfile))
	if adjacentBoundaryGuidance != "" {
		adjacentBoundaryGuidance = "\n" + adjacentBoundaryGuidance
	}
	return fmt.Sprintf(`Edit one assembled Part of a Korean long-form Plasma report through its dedicated MCP tools.

Report title: %s
Mission ID: %s
Part %d: %s

This is a separate Part editor role. The source Part artifact is immutable. A real edit creates a separate artifact; an unchanged review records completion while reusing the source artifact.

Part-level requirements to preserve:
%s

Overall plan and writing contract:
%s

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s

Bound MCP Part edit metadata:
%s

Mutating call identity:
- Use mission_id %q and session_id %q on every start, patch, and submit call.
- Use producer {"type":"agent_session","id":%q} on every mutating call.
- Use idempotency_key %q for start, %q, %q, ... for successive patches, and %q for submit. Never reuse one call's key for another call.

Required tool sequence:
1. Call %s once with draft_id %q and the bound mission, session, pending, plan, Part, and source artifact values.
2. Read the entire Part with %s. Continue only from returned next_offset values until truncated is false.
3. Act as an editor, not as a researcher or a new Section author. Use %s only for exact edits that improve this Part.
4. Read each affected passage again. If no material edit is justified after the full read, leave the draft unchanged.
5. Call %s once with the same draft_id and bound pending and plan IDs.
6. After submission succeeds, return exactly %s and nothing else.

Editing responsibility:
- Fix repetition, abrupt Section order, weak transitions, and logical gaps inside this Part only.
- Preserve every concrete fact, number, example, code identifier, caveat, citation, uncertainty boundary, and assigned requirement.
- Prefer the smallest edit that improves a real reading problem. Do not rewrite merely to demonstrate activity.
- Do not add researched facts, use research or source tools, change other Parts, or pre-write the report opening or conclusion.
- Do not mention prompts, experiments, internal run labels, tool session IDs, or artifact IDs in the manuscript.%s`,
		input.Base.Title, input.Base.MissionID, input.PartIndex+1, input.Part.Title,
		longformutil.AnyJSON(requirements), longformutil.AnyJSON(input.Base.Plan),
		input.Base.Rigor.Level, input.Base.Rigor.Label, input.Base.Rigor.Description, input.Base.Rigor.Instructions,
		longformutil.AnyJSON(binding), binding.MissionID, binding.ToolSessionID, binding.ToolSessionID,
		binding.IdempotencyKey+":start", binding.IdempotencyKey+":patch-1", binding.IdempotencyKey+":patch-2", binding.IdempotencyKey+":submit",
		mcptools.ToolReportPartEditStart, draftID,
		mcptools.ToolReportPartEditRead, mcptools.ToolReportPartEditPatch,
		mcptools.ToolReportPartEditSubmit, reporting.PartEditSubmittedSentinel,
		adjacentBoundaryGuidance)
}

func authorPrompt(input Input, binding reporting.PartEditBinding, draftID string) string {
	requirements := reporting.ReportRequirementsForPart(input.Base.RequirementMap, input.PartIndex+1)
	return fmt.Sprintf(`Write one final Part of a Korean long-form Plasma report through its dedicated MCP Part edit tools.

You are the final author of this Part. The assembled Sections are drafting material, not immutable manuscript prose.

Report title: %s
Mission ID: %s
Part %d: %s

Part-level requirements to preserve:
%s

Overall plan and writing contract:
%s

Part planning brief:
%s

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s

Bound MCP Part edit metadata:
%s

Mutating call identity:
- Use mission_id %q and session_id %q on every start, patch, and submit call.
- Use producer {"type":"agent_session","id":%q} on every mutating call.
- Use idempotency_key %q for start, %q, %q, ... for successive patches, and %q for submit. Never reuse one call's key for another call.

Required tool sequence:
1. Call %s once with draft_id %q and the bound mission, session, pending, plan, Part, and source artifact values.
2. Read the entire Part with %s. Continue only from returned next_offset values until truncated is false.
3. Use %s for exact edits. A purposeful whole-document exact replacement is allowed when it produces a more coherent final Part within the existing Part bounds.
4. Reread the affected passages, and reread the whole Part before submitting.
5. Call %s once with the same draft_id and bound pending and plan IDs.
6. After submission succeeds, return exactly %s and nothing else.

Authorship responsibility:
- Read all input before writing: the current Part, the Part brief, the overall plan, the writing contract, rigor, and assigned requirements.
- Use the Part brief to recover the intended reader movement.
- Write one coherent standalone Part, not a stitched inventory of Sections.
- Keep the Part title and planned Section headings/order.
- Treat Section prose as material that may be substantially rewritten, merged, shortened, or moved within the planned order.
- Preserve every fact, number, example, code identifier, caveat, citation, uncertainty boundary, and assigned requirement.
- Purposeful spaced recall is allowed only when it recontextualizes, applies, or decides something for the reader.
- Merge or remove restatement that gives the reader no new job.
- Explain the subject directly, with sources backstage as support for claims rather than recurring sentence subjects.
- Localize uncertainty beside the claim it qualifies.
- Add no researched facts, use no research or evidence tools, and do not change other Parts.
- Do not mention prompts, experiments, internal run labels, tool session IDs, or artifact IDs in the manuscript.`,
		input.Base.Title, input.Base.MissionID, input.PartIndex+1, input.Part.Title,
		longformutil.AnyJSON(requirements), longformutil.AnyJSON(input.Base.Plan), strings.TrimSpace(input.PartPlanningBrief),
		input.Base.Rigor.Level, input.Base.Rigor.Label, input.Base.Rigor.Description, input.Base.Rigor.Instructions,
		longformutil.AnyJSON(binding), binding.MissionID, binding.ToolSessionID, binding.ToolSessionID,
		binding.IdempotencyKey+":start", binding.IdempotencyKey+":patch-1", binding.IdempotencyKey+":patch-2", binding.IdempotencyKey+":submit",
		mcptools.ToolReportPartEditStart, draftID,
		mcptools.ToolReportPartEditRead, mcptools.ToolReportPartEditPatch,
		mcptools.ToolReportPartEditSubmit, reporting.PartEditSubmittedSentinel)
}
