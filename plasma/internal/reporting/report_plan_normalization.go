package reporting

import (
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

// NormalizeReportPlan는 보고서 생성 파이프라인 입력을 표준 형태로 정규화하고 허용되지 않는 값은 안정 오류로 거부한다.
func NormalizeReportPlan(plan ReportPlan) (ReportPlan, error) {
	if strings.TrimSpace(plan.Summary) == "" && len(plan.Sections) == 0 {
		return ReportPlan{}, fmt.Errorf("%w: report plan is empty", producterror.ErrInvalidInput)
	}
	contract, err := normalizeReportWritingContract(plan.WritingContract)
	if err != nil {
		return ReportPlan{}, err
	}
	plan.WritingContract = contract
	return plan, nil
}

// NormalizeSectionalReportPlan는 보고서 생성 파이프라인 입력을 표준 형태로 정규화하고 허용되지 않는 값은 안정 오류로 거부한다.
func NormalizeSectionalReportPlan(plan SectionalReportPlan) (SectionalReportPlan, error) {
	plan.Summary = strings.TrimSpace(plan.Summary)
	plan.CoverageNotes = limitNonEmptyPlanStrings(plan.CoverageNotes, 24)
	plan.PlannedOmissions = limitNonEmptyPlanStrings(plan.PlannedOmissions, 24)
	contract, err := normalizeReportWritingContract(plan.WritingContract)
	if err != nil {
		return SectionalReportPlan{}, err
	}
	plan.WritingContract = contract
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
				return SectionalReportPlan{}, fmt.Errorf("%w: long-form report section title is required", producterror.ErrInvalidInput)
			}
			sections = append(sections, section)
		}
		if part.Title == "" && part.Purpose == "" && len(sections) == 0 {
			continue
		}
		if part.Title == "" {
			return SectionalReportPlan{}, fmt.Errorf("%w: long-form report part title is required", producterror.ErrInvalidInput)
		}
		if len(sections) == 0 {
			return SectionalReportPlan{}, fmt.Errorf("%w: long-form report part requires a section", producterror.ErrInvalidInput)
		}
		part.Sections = sections
		normalized = append(normalized, part)
	}
	if len(normalized) == 0 {
		return SectionalReportPlan{}, fmt.Errorf("%w: long-form report plan requires a part", producterror.ErrInvalidInput)
	}
	plan.Parts = normalized
	return plan, nil
}

// RequireReportWritingContract는 보고서 생성 파이프라인 계약을 검사한다. 제품 상태를 변경하지 않는 순수 검증 경계다.
func RequireReportWritingContract(plan any) error {
	var contract *ReportWritingContract
	switch value := plan.(type) {
	case ReportPlan:
		contract = value.WritingContract
	case SectionalReportPlan:
		contract = value.WritingContract
	default:
		return fmt.Errorf("%w: unsupported report plan", producterror.ErrInvalidInput)
	}
	if contract == nil {
		return fmt.Errorf("%w: report writing contract is required", producterror.ErrInvalidInput)
	}
	return nil
}

func normalizeReportWritingContract(value *ReportWritingContract) (*ReportWritingContract, error) {
	if value == nil {
		return nil, nil
	}
	contract := *value
	contract.CentralQuestion = strings.TrimSpace(contract.CentralQuestion)
	contract.ReaderTakeaway = strings.TrimSpace(contract.ReaderTakeaway)
	contract.ReadingPath = limitNonEmptyPlanStrings(contract.ReadingPath, 12)
	contract.MustKeep = limitNonEmptyPlanStrings(contract.MustKeep, 24)
	contract.CanSummarize = limitNonEmptyPlanStrings(contract.CanSummarize, 24)
	contract.MoveToSupportingLayer = limitNonEmptyPlanStrings(contract.MoveToSupportingLayer, 24)
	contract.VisualRole = strings.TrimSpace(contract.VisualRole)
	contract.ToneAndShape = strings.TrimSpace(contract.ToneAndShape)
	if contract.CentralQuestion == "" || contract.ReaderTakeaway == "" || len(contract.ReadingPath) == 0 || len(contract.MustKeep) == 0 || contract.VisualRole == "" || contract.ToneAndShape == "" {
		return nil, fmt.Errorf("%w: report writing contract is incomplete", producterror.ErrInvalidInput)
	}
	return &contract, nil
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
