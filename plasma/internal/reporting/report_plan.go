package reporting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const ReportPlanSchemaVersion = "plasma.report_plan.v1"

type ReportPlan struct {
	Summary          string              `json:"summary"`
	Sections         []ReportPlanSection `json:"sections"`
	CoverageNotes    []string            `json:"coverage_notes,omitempty"`
	PlannedOmissions []string            `json:"planned_omissions,omitempty"`
}

type SectionalReportPlan struct {
	Summary          string           `json:"summary"`
	Parts            []ReportPlanPart `json:"parts"`
	CoverageNotes    []string         `json:"coverage_notes,omitempty"`
	PlannedOmissions []string         `json:"planned_omissions,omitempty"`
}

type ReportPlanPart struct {
	Title    string              `json:"title"`
	Purpose  string              `json:"purpose"`
	Sections []ReportPlanSection `json:"sections"`
}

type ReportPlanSection struct {
	Title      string                    `json:"title"`
	Purpose    string                    `json:"purpose"`
	TargetRefs app.ReportBlockSourceRefs `json:"target_refs,omitempty"`
}

func NormalizeReportPlan(plan ReportPlan) (ReportPlan, error) {
	if strings.TrimSpace(plan.Summary) == "" && len(plan.Sections) == 0 {
		return ReportPlan{}, fmt.Errorf("%w: report plan is empty", app.ErrInvalidInput)
	}
	return plan, nil
}

func NormalizeSectionalReportPlan(plan SectionalReportPlan) (SectionalReportPlan, error) {
	plan.Summary = strings.TrimSpace(plan.Summary)
	plan.CoverageNotes = limitNonEmptyPlanStrings(plan.CoverageNotes, 24)
	plan.PlannedOmissions = limitNonEmptyPlanStrings(plan.PlannedOmissions, 24)
	normalized := make([]ReportPlanPart, 0, len(plan.Parts))
	for _, part := range plan.Parts {
		part.Title = strings.TrimSpace(part.Title)
		part.Purpose = strings.TrimSpace(part.Purpose)
		sections := make([]ReportPlanSection, 0, len(part.Sections))
		for _, section := range part.Sections {
			section.Title = strings.TrimSpace(section.Title)
			section.Purpose = strings.TrimSpace(section.Purpose)
			if section.Title == "" && section.Purpose == "" && emptyReportPlanRefs(section.TargetRefs) {
				continue
			}
			if section.Title == "" {
				return SectionalReportPlan{}, fmt.Errorf("%w: long-form report section title is required", app.ErrInvalidInput)
			}
			sections = append(sections, section)
		}
		if part.Title == "" && part.Purpose == "" && len(sections) == 0 {
			continue
		}
		if part.Title == "" {
			return SectionalReportPlan{}, fmt.Errorf("%w: long-form report part title is required", app.ErrInvalidInput)
		}
		if len(sections) == 0 {
			return SectionalReportPlan{}, fmt.Errorf("%w: long-form report part requires a section", app.ErrInvalidInput)
		}
		part.Sections = sections
		normalized = append(normalized, part)
	}
	if len(normalized) == 0 {
		return SectionalReportPlan{}, fmt.Errorf("%w: long-form report plan requires a part", app.ErrInvalidInput)
	}
	plan.Parts = normalized
	return plan, nil
}

func ReportPlanHash(plan any) (string, json.RawMessage, error) {
	encoded, err := json.Marshal(plan)
	if err != nil {
		return "", nil, fmt.Errorf("%w: report plan cannot be encoded", app.ErrInvalidInput)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), encoded, nil
}

func ReportPlanRefs(plan any) []app.ReportBlockSourceRefs {
	refs := []app.ReportBlockSourceRefs{}
	switch value := plan.(type) {
	case ReportPlan:
		for _, section := range value.Sections {
			refs = append(refs, section.TargetRefs)
		}
	case SectionalReportPlan:
		for _, part := range value.Parts {
			for _, section := range part.Sections {
				refs = append(refs, section.TargetRefs)
			}
		}
	}
	return refs
}

func limitNonEmptyPlanStrings(values []string, limit int) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
			if len(result) == limit {
				break
			}
		}
	}
	return result
}

func emptyReportPlanRefs(refs app.ReportBlockSourceRefs) bool {
	return len(refs.ClaimIDs)+len(refs.EvidenceIDs)+len(refs.SnapshotIDs)+len(refs.QuestionIDs)+len(refs.OptionIDs) == 0
}
