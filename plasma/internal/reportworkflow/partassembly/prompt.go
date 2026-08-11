package partassembly

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// Prompt는 JSON connective assembly 경로의 기존 prompt bytes다.
func Prompt(input Input) string {
	guidance := strings.TrimSpace(strings.Join([]string{
		reportprompt.LongFormReportGenerationGuidance(input.Base.GenerationGuidanceProfile),
		reportprompt.PartConnectiveEconomyGuidance(input.Base.GenerationGuidanceProfile),
	}, "\n\n"))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	return fmt.Sprintf(`Prepare connective tissue for one Part of a Korean long-form Plasma report.

Report title: %s
Mission ID: %s
Part %d: %s

This is not a rewrite task. The Section bodies are immutable and will be mechanically inserted by Plasma. You must not return rewritten Section bodies.

Section inventory:
%s

Overall plan:
%s

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s

Return JSON only:
{
  "intro": "short Markdown introduction for this Part",
  "transitions": [
    {"after_section_index": 1, "markdown": "short transition after section 1"}
  ],
  "closing": "short Markdown closing for this Part"
}

Rules:
- Use Korean.
- Do not include the immutable Section bodies.
- Do not summarize the Section bodies into a replacement overview.
- %s
- Transitions are optional, but when useful they should connect adjacent Sections without compressing them.
- Do not mention prompts, experiments, internal run labels, tool session IDs, or temporary implementation details.`, input.Base.Title, input.Base.MissionID, input.PartIndex+1, input.Part.Title, sectionInventoryJSON(input.Sections), longformutil.AnyJSON(input.Base.Plan), input.Base.Rigor.Level, input.Base.Rigor.Label, input.Base.Rigor.Description, input.Base.Rigor.Instructions, guidance, reportprompt.MermaidValidationRuleText)
}

// EditToolsPrompt는 MCP Part assembly tools 경로의 기존 prompt bytes다.
func EditToolsPrompt(input Input, binding reporting.PartAssemblyBinding, draftID string) string {
	partConnectiveGuidance := strings.TrimSpace(reportprompt.PartConnectiveEconomyGuidance(input.Base.GenerationGuidanceProfile))
	guidance := strings.TrimSpace(strings.Join([]string{
		reportprompt.LongFormReportGenerationGuidance(input.Base.GenerationGuidanceProfile),
		partConnectiveGuidance,
	}, "\n\n"))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	connectiveRule := "- Prefer one good intro and one good closing over many filler transitions."
	if partConnectiveGuidance != "" {
		connectiveRule = "- Intro, transitions, and closing are optional. Add them only when actual Section relationships justify connective text."
	}
	sectionReading := ""
	sectionInventory := sectionInventoryJSON(input.Sections)
	promptBinding := binding
	if reportprompt.IsNarrativeContract(input.Base.GenerationGuidanceProfile) {
		sectionReading = fmt.Sprintf("\nRequired manuscript reading:\n- Call %s for every Section in this Part, following next_offset until truncated is false. Read the actual Section bodies before writing connective text.\n", mcptools.ToolReportPartSectionRead)
		sectionInventory = narrativeSectionInventoryJSON(input.Sections)
		promptBinding.SectionArtifactIDs = nil
	}
	return fmt.Sprintf(`Prepare connective tissue for one Part of a Korean long-form Plasma report using MCP edit tools.

Report title: %s
Mission ID: %s
Part %d: %s

This is not a rewrite task. The Section bodies are immutable and will be mechanically inserted by Plasma. You must not submit rewritten Section bodies.

Section inventory:
%s

Overall plan:
%s

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s

Bound MCP part assembly metadata:
%s

Required tool sequence:
1. Call %s once with draft_id %q, the bound mission/session/pending/plan IDs, part_index, section_count, producer, and a start idempotency_key.
%s
2. Use %s when you need to inspect the current connective draft.
3. Use %s to set only intro, transition, or closing. For a transition, after_section_index is the section number after which the transition appears; it must be before another section.
4. Call %s once with the same draft_id and bound pending/plan IDs.
5. After the submit tool succeeds, return exactly %s and nothing else.

Rules:
- Use Korean for the connective Markdown.
- Do not include immutable Section bodies in any patch.
- Do not summarize the Section bodies into a replacement overview.
%s
- When useful, transitions should connect adjacent Sections without compressing them.
- Do not mention prompts, experiments, internal run labels, tool session IDs, or temporary implementation details.`, input.Base.Title, input.Base.MissionID, input.PartIndex+1, input.Part.Title, sectionInventory, longformutil.AnyJSON(input.Base.Plan), input.Base.Rigor.Level, input.Base.Rigor.Label, input.Base.Rigor.Description, input.Base.Rigor.Instructions, guidance, longformutil.AnyJSON(promptBinding), mcptools.ToolReportPartAssemblyStart, draftID, sectionReading, mcptools.ToolReportPartAssemblyRead, mcptools.ToolReportPartAssemblyPatch, mcptools.ToolReportPartAssemblySubmit, reporting.PartAssemblySubmittedSentinel, connectiveRule)
}

func sectionInventoryJSON(drafts []SectionDraft) string {
	items := make([]map[string]any, 0, len(drafts))
	for index, draft := range drafts {
		items = append(items, map[string]any{"section_index": index + 1, "title": draft.Title, "artifact_id": draft.ArtifactID, "word_count": draft.WordCount})
	}
	return longformutil.AnyJSON(items)
}

func narrativeSectionInventoryJSON(drafts []SectionDraft) string {
	items := make([]map[string]any, len(drafts))
	for index, draft := range drafts {
		items[index] = map[string]any{"section_index": index + 1, "title": draft.Title, "word_count": draft.WordCount}
	}
	return longformutil.AnyJSON(items)
}
