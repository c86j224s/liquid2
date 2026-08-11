package plan

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

const SectionPlanUnrepairableControlToken = "SECTION_PLAN_UNREPAIRABLE"

type sectionRepairPromptItem struct {
	PartIndex    int                           `json:"part_index"`
	SectionIndex int                           `json:"section_index"`
	Part         reporting.ReportPlanPart      `json:"part"`
	Section      reporting.ReportPlanSection   `json:"section"`
	Requirements []reporting.ReportRequirement `json:"requirements"`
}

// LongFormSectionRepairPrompt asks the original report planner for one bounded,
// same-coordinate amendment after Section writers exhaust their evidence search.
func LongFormSectionRepairPrompt(input LongFormSectionRepairInput) string {
	items := make([]sectionRepairPromptItem, 0, len(input.Gaps))
	for _, gap := range input.Gaps {
		part := input.Plan.Plan.Parts[gap.PartIndex-1]
		items = append(items, sectionRepairPromptItem{
			PartIndex: gap.PartIndex, SectionIndex: gap.SectionIndex,
			Part: part, Section: part.Sections[gap.SectionIndex-1],
			Requirements: reporting.ReportRequirementsForSection(input.RequirementMap, gap.PartIndex, gap.SectionIndex),
		})
	}
	prompt := fmt.Sprintf(`Repair only the unsupported Section slots in an existing Korean long-form Plasma report plan.

Do not write report prose. This is the single allowed plan-repair round for the report lineage.

Report title: %s
Mission ID: %s

Canonical plan before repair:
%s

Section slots whose writers returned a final evidence gap:
%s

Use the read-only Plasma research and source tools to inspect original material for every listed slot. A replacement must give the reader a substantive historical, technical, or conceptual explanation that the mission's original sources can support.

Rules:
- Return exactly one replacement for every listed coordinate, or return exactly %s if any listed slot has no supportable replacement.
- Keep each 1-based part_index and section_index unchanged. Do not add, remove, merge, split, reorder, or move Sections.
- Do not change any unlisted Section, Part title or purpose, report summary, writing contract, coverage note, or planned omission.
- Preserve every assigned user requirement at its current coordinate. Requirements guide output but are not factual evidence.
- Replace the unsupported explanatory job rather than merely weakening its wording. The new title and purpose must be specific enough for an independent Section writer.
- target_refs are starting points, not proof. Use only mission IDs that the tools expose; include substantive source, evidence, or saved-claim refs when available.
- Do not substitute bibliography, catalog metadata, provenance, source comparison, or instructions for reading material unless that is genuinely the supported subject of the replacement Section.
- Do not invent facts or promise coverage that the inspected material cannot support.

Return only this JSON shape, with no fence or commentary:
{
  "replacements": [
    {
      "part_index": 1,
      "section_index": 2,
      "section": {
        "title": "replacement Section title",
        "purpose": "the supportable explanatory job",
        "target_refs": {"claim_ids": ["clm_..."], "evidence_ids": ["evd_..."], "snapshot_ids": ["src_..."]}
      }
    }
  ]
}`,
		input.Request.Title, input.Request.MissionID, longformutil.AnyJSON(input.Plan.Plan),
		longformutil.AnyJSON(items), SectionPlanUnrepairableControlToken)
	return reportprompt.WithLongFormPlanningDirection(prompt, input.Request.DirectionHint)
}
