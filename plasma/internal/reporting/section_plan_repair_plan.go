package reporting

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func applySectionPlanReplacements(original SectionalReportPlan, values []ReportSectionPlanReplacement) (SectionalReportPlan, []ReportSectionPlanReplacement, error) {
	plan, err := NormalizeSectionalReportPlan(original)
	if err != nil || len(values) == 0 {
		return SectionalReportPlan{}, nil, fmt.Errorf("%w: Section plan repair needs replacements", producterror.ErrInvalidInput)
	}
	replacements := append([]ReportSectionPlanReplacement(nil), values...)
	sort.Slice(replacements, func(i, j int) bool {
		if replacements[i].PartIndex == replacements[j].PartIndex {
			return replacements[i].SectionIndex < replacements[j].SectionIndex
		}
		return replacements[i].PartIndex < replacements[j].PartIndex
	})
	seen := map[ReportSectionCoordinate]bool{}
	for index := range replacements {
		replacement := &replacements[index]
		coordinate := replacement.ReportSectionCoordinate
		if seen[coordinate] || coordinate.PartIndex < 1 || coordinate.PartIndex > len(plan.Parts) || coordinate.SectionIndex < 1 || coordinate.SectionIndex > len(plan.Parts[coordinate.PartIndex-1].Sections) {
			return SectionalReportPlan{}, nil, fmt.Errorf("%w: Section plan repair coordinate is invalid", producterror.ErrInvalidInput)
		}
		seen[coordinate] = true
		replacement.Section.Title = strings.TrimSpace(replacement.Section.Title)
		replacement.Section.Purpose = strings.TrimSpace(replacement.Section.Purpose)
		if replacement.Section.Title == "" || replacement.Section.Purpose == "" {
			return SectionalReportPlan{}, nil, fmt.Errorf("%w: replacement Section title and purpose are required", producterror.ErrInvalidInput)
		}
		current := plan.Parts[coordinate.PartIndex-1].Sections[coordinate.SectionIndex-1]
		if reflect.DeepEqual(current, replacement.Section) {
			return SectionalReportPlan{}, nil, fmt.Errorf("%w: replacement Section did not change", producterror.ErrInvalidInput)
		}
		plan.Parts[coordinate.PartIndex-1].Sections[coordinate.SectionIndex-1] = replacement.Section
	}
	plan, err = NormalizeSectionalReportPlan(plan)
	if err != nil {
		return SectionalReportPlan{}, nil, err
	}
	return plan, replacements, nil
}

func normalizeSectionPlanRepairCoordinates(plan SectionalReportPlan, values []ReportSectionCoordinate) ([]ReportSectionCoordinate, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("%w: Section plan repair needs coordinates", producterror.ErrInvalidInput)
	}
	coordinates := append([]ReportSectionCoordinate(nil), values...)
	sort.Slice(coordinates, func(i, j int) bool {
		if coordinates[i].PartIndex == coordinates[j].PartIndex {
			return coordinates[i].SectionIndex < coordinates[j].SectionIndex
		}
		return coordinates[i].PartIndex < coordinates[j].PartIndex
	})
	for index, coordinate := range coordinates {
		if coordinate.PartIndex < 1 || coordinate.PartIndex > len(plan.Parts) ||
			coordinate.SectionIndex < 1 || coordinate.SectionIndex > len(plan.Parts[coordinate.PartIndex-1].Sections) ||
			index > 0 && coordinate == coordinates[index-1] {
			return nil, fmt.Errorf("%w: Section plan repair coordinate is invalid", producterror.ErrInvalidInput)
		}
	}
	return coordinates, nil
}

func coordinatesForReplacements(values []ReportSectionPlanReplacement) []ReportSectionCoordinate {
	coordinates := make([]ReportSectionCoordinate, len(values))
	for index, value := range values {
		coordinates[index] = value.ReportSectionCoordinate
	}
	return coordinates
}

func sameRepairCoordinates(left, right []ReportSectionCoordinate) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
