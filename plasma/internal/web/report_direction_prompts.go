package web

import "github.com/c86j224s/liquid2/plasma/internal/reportprompt"

const (
	longFormDirectionAdvisory = reportprompt.LongFormDirectionAdvisory

	longFormPlanningDirectionGuidance = reportprompt.LongFormPlanningDirectionGuidance

	longFormDownstreamDirectionGuidance = reportprompt.LongFormDownstreamDirectionGuidance

	longFormDirectionCoverageDepthRule = reportprompt.LongFormDirectionCoverageDepthRule

	longFormDirectionPriorityRule = reportprompt.LongFormDirectionPriorityRule
)

func withReportDirection(prompt, hint string) string {
	return reportprompt.WithReportDirection(prompt, hint)
}

// formatLongFormDirectionHint는 장문 report에서만 쓰는 원문 direction 블록이다.
// 원문은 durable plan이나 binding으로 복사하지 않고 prompt 경계에서만 전달한다.
func formatLongFormDirectionHint(value string) string {
	return reportprompt.FormatLongFormDirectionHint(value)
}

// withLongFormPlanningDirection은 개요가 고정되기 전에만 원문 방향과 계획 해석 규칙을 붙인다.
// 방향이 비어 있으면 기존 계획 프롬프트를 그대로 반환한다.
func withLongFormPlanningDirection(prompt, hint string) string {
	return reportprompt.WithLongFormPlanningDirection(prompt, hint)
}

// withLongFormDownstreamDirection은 장문 내용 작성 단계에 원문과 계획 계약의 우선순위를 붙인다.
// 검증·말투·근거 확인 단계에서는 호출하지 않으며, 방향이 비어 있으면 기존 프롬프트를 보존한다.
func withLongFormDownstreamDirection(prompt, hint string) string {
	return reportprompt.WithLongFormDownstreamDirection(prompt, hint)
}
