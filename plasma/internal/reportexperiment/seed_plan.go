package reportexperiment

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func planFromFixture(loaded LoadedFixture) (reporting.SectionalReportPlan, error) {
	parts := make([]reporting.ReportPlanPart, 0, len(loaded.Parts))
	for _, part := range loaded.Parts {
		sectionTitle := strings.TrimSpace(part.Spec.SectionTitle)
		if sectionTitle == "" {
			sectionTitle = strings.TrimSpace(part.Spec.Title)
		}
		parts = append(parts, reporting.ReportPlanPart{
			Title:   part.Spec.Title,
			Purpose: "Use the fixed reviewed Part as the ordered material for this Part of the report.",
			Sections: []reporting.ReportPlanSection{{
				Title:   sectionTitle,
				Purpose: "Preserve reviewed facts, caveats, evidence markers, and the fixture writing contract while editing only the final report.",
			}},
		})
	}
	return reporting.NormalizeSectionalReportPlan(reporting.SectionalReportPlan{
		Summary:         "Fixed reviewed-Part finalization-only experiment for " + loaded.Spec.ReportTitle + ".",
		Parts:           parts,
		WritingContract: loaded.Spec.WritingContract,
	})
}

func requirementsFromFixture(loaded LoadedFixture, plan reporting.SectionalReportPlan, pendingID, stem string) (reporting.ReportRequirementMap, error) {
	requirements := make([]reporting.ReportRequirement, 0, len(loaded.Parts))
	for _, part := range loaded.Parts {
		requirements = append(requirements, reporting.ReportRequirement{
			RequirementID:  fmt.Sprintf("req_reportexperiment_%s_%02d", stem, part.Spec.Index),
			Instruction:    "Preserve the reviewed Part's facts, caveats, evidence markers, and original order in the final report.",
			SourceEventIDs: []string{pendingID},
			Owner:          &reporting.ReportRequirementOwner{PartIndex: part.Spec.Index, SectionIndex: 1},
		})
	}
	return reporting.NormalizeReportRequirementMap(reporting.ReportRequirementMap{ReviewedEventIDs: []string{pendingID}, Requirements: requirements}, plan)
}
