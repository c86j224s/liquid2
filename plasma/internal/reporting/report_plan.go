package reporting

import (
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting/reportdocument"
)

const ReportPlanSchemaVersion = "plasma.report_plan.v1"

// ReportPlanSourceRefs names the source-reference contract consumed by report
// planning without moving its application-owned storage model.
type ReportPlanSourceRefs = reportdocument.ReportBlockSourceRefs

// ReportPlanSubmissionQuery는 이미 제출된 report plan을 정확히 하나 선택하기 위한 binding query다.
type ReportPlanSubmissionQuery = app.ReportPlanSubmissionQuery

// ReportPlanSubmissionSelection은 선택된 report plan event와 hash metadata다.
type ReportPlanSubmissionSelection = app.ReportPlanSubmissionSelection

// PromoteReportPlanRequest는 제출된 report plan을 canonical pending event로 승격할 때의 입력이다.
type PromoteReportPlanRequest = app.PromoteReportPlanRequest

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
	Title      string                               `json:"title"`
	Purpose    string                               `json:"purpose"`
	TargetRefs reportdocument.ReportBlockSourceRefs `json:"target_refs,omitempty"`
}
