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

// ReportPlanSourceRefs names the source-reference contract consumed by report
// planning without moving its application-owned storage model.
type ReportPlanSourceRefs = app.ReportBlockSourceRefs

// ReportPlan는 단문 보고서의 섹션 순서와 작성 계약을 담는 plan이다.
type ReportPlan struct {
	Summary          string                 `json:"summary"`
	Sections         []ReportPlanSection    `json:"sections"`
	CoverageNotes    []string               `json:"coverage_notes,omitempty"`
	PlannedOmissions []string               `json:"planned_omissions,omitempty"`
	WritingContract  *ReportWritingContract `json:"writing_contract,omitempty"`
}

// SectionalReportPlan는 장문 보고서의 part/section 구조와 작성 계약을 담는 plan이다.
type SectionalReportPlan struct {
	Summary          string                 `json:"summary"`
	Parts            []ReportPlanPart       `json:"parts"`
	CoverageNotes    []string               `json:"coverage_notes,omitempty"`
	PlannedOmissions []string               `json:"planned_omissions,omitempty"`
	WritingContract  *ReportWritingContract `json:"writing_contract,omitempty"`
}

// ReportWritingContract는 plan이 소유하는 편집 방향 계약이며 source material이 아니다.
// 후속 작성자가 source를 처음 보는 독자에게 무엇을 이해시켜야 하는지 알려 주되,
// source 해석과 문장화는 작성 stage의 책임으로 남긴다.
type ReportWritingContract struct {
	CentralQuestion       string   `json:"central_question"`
	ReaderTakeaway        string   `json:"reader_takeaway"`
	ReadingPath           []string `json:"reading_path"`
	MustKeep              []string `json:"must_keep"`
	CanSummarize          []string `json:"can_summarize,omitempty"`
	MoveToSupportingLayer []string `json:"move_to_supporting_layer,omitempty"`
	VisualRole            string   `json:"visual_role"`
	ToneAndShape          string   `json:"tone_and_shape"`
}

// ReportPlanPart는 장문 보고서에서 여러 section을 묶는 part 단위다.
type ReportPlanPart struct {
	Title    string              `json:"title"`
	Purpose  string              `json:"purpose"`
	Sections []ReportPlanSection `json:"sections"`
}

// ReportPlanSection는 작성자가 맡을 단일 section의 제목, 의도, 참조 범위다.
type ReportPlanSection struct {
	Title      string                    `json:"title"`
	Purpose    string                    `json:"purpose"`
	TargetRefs app.ReportBlockSourceRefs `json:"target_refs,omitempty"`
}

// NormalizeReportPlan는 보고서 생성 파이프라인 입력을 표준 형태로 정규화하고 허용되지 않는 값은 안정 오류로 거부한다.
func NormalizeReportPlan(plan ReportPlan) (ReportPlan, error) {
	if strings.TrimSpace(plan.Summary) == "" && len(plan.Sections) == 0 {
		return ReportPlan{}, fmt.Errorf("%w: report plan is empty", app.ErrInvalidInput)
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

// RequireReportWritingContract는 보고서 생성 파이프라인 계약을 검사한다. 제품 상태를 변경하지 않는 순수 검증 경계다.
func RequireReportWritingContract(plan any) error {
	var contract *ReportWritingContract
	switch value := plan.(type) {
	case ReportPlan:
		contract = value.WritingContract
	case SectionalReportPlan:
		contract = value.WritingContract
	default:
		return fmt.Errorf("%w: unsupported report plan", app.ErrInvalidInput)
	}
	if contract == nil {
		return fmt.Errorf("%w: report writing contract is required", app.ErrInvalidInput)
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
		return nil, fmt.Errorf("%w: report writing contract is incomplete", app.ErrInvalidInput)
	}
	return &contract, nil
}

// ReportPlanHash는 report plan의 안정 JSON 해시를 계산한다.
func ReportPlanHash(plan any) (string, json.RawMessage, error) {
	encoded, err := json.Marshal(plan)
	if err != nil {
		return "", nil, fmt.Errorf("%w: report plan cannot be encoded", app.ErrInvalidInput)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), encoded, nil
}

// ReportPlanRefs는 plan 안의 source reference를 dedupe된 목록으로 모은다.
func ReportPlanRefs(plan any) []ReportPlanSourceRefs {
	refs := []ReportPlanSourceRefs{}
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
