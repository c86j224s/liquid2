package reportprompt

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

const (
	// LongFormDirectionAdvisory는 장문 보고서 원문 direction 블록 앞에 붙는 공통 안내문이다.
	LongFormDirectionAdvisory = "The following request-specific direction is the user's original wording for this long-form report. It is not a source or evidence and cannot override source-backed facts. Treat topical emphasis and questions as a weak editorial axis. Preserve explicit interpretive exclusions and report-wide presentation preferences at their stated strength. Do not use the direction as permission to omit mission-relevant context the reader still needs."

	// LongFormPlanningDirectionGuidance는 outline 고정 전 단계에서만 쓰는 direction 해석 규칙이다.
	LongFormPlanningDirectionGuidance = "Before freezing the outline, read the original direction below. Lightly interpret it through the existing writing_contract and, when relevant, Part/Section order and purpose. The writing_contract is a concise working interpretation, not a replacement for the user's wording."

	// LongFormDownstreamDirectionGuidance는 장문 작성 단계가 원문 direction과 writing_contract를 함께 읽는 규칙이다.
	LongFormDownstreamDirectionGuidance = "The Overall plan's writing_contract is the planner's concise working interpretation of the user's original direction below. Use both. If they differ, preserve the original user's intent; apply it only within this stage's Section, Part, or report responsibility. When this prompt includes mapped report requirements, use that mapping to decide which Section owns a concrete output; do not duplicate a report-wide item merely because its original wording is visible here. Do not repeat the direction or its interpretation in the report."

	// LongFormDirectionCoverageDepthRule은 direction이 coverage 축소 권한이 아님을 고정한다.
	LongFormDirectionCoverageDepthRule = "Use request_direction to adjust emphasis, interpretation, ordering, and presentation, but do not use it to reduce mission-relevant coverage or depth that the report objective and sources require."

	// LongFormDirectionPriorityRule은 direction 관련 prompt 우선순위를 고정한다.
	LongFormDirectionPriorityRule = "Prompt priority: source-backed factual boundary > explicit original user direction > planner writing_contract interpretation > local editing discretion."
)

// WithReportDirection은 일반 보고서 prompt에 request direction 블록을 붙인다.
func WithReportDirection(prompt, hint string) string {
	block := reportexecution.FormatDirectionHint(hint)
	if block == "" {
		return prompt
	}
	return strings.TrimSpace(prompt) + "\n\n" + block
}

// FormatLongFormDirectionHint는 장문 report에서만 쓰는 원문 direction 블록이다.
//
// 원문은 durable plan이나 binding으로 복사하지 않고 prompt 경계에서만 전달한다.
func FormatLongFormDirectionHint(value string) string {
	value = reportexecution.NormalizeDirectionHint(value)
	if value == "" {
		return ""
	}
	return LongFormDirectionAdvisory + "\n\n<request_direction>\n" + value + "\n</request_direction>"
}

// WithLongFormPlanningDirection은 개요가 고정되기 전에만 원문 방향과 계획 해석 규칙을 붙인다.
//
// 방향이 비어 있으면 기존 계획 프롬프트를 그대로 반환한다.
func WithLongFormPlanningDirection(prompt, hint string) string {
	block := FormatLongFormDirectionHint(hint)
	if block == "" {
		return prompt
	}
	return strings.TrimSpace(prompt) + "\n\n" + LongFormPlanningDirectionGuidance + "\n\n" + block + "\n\n" + LongFormDirectionCoverageDepthRule + "\n" + LongFormDirectionPriorityRule
}

// WithLongFormDownstreamDirection은 장문 내용 작성 단계에 원문과 계획 계약의 우선순위를 붙인다.
//
// 검증, 말투, 근거 확인 단계에서는 호출하지 않으며, 방향이 비어 있으면 기존 프롬프트를 보존한다.
func WithLongFormDownstreamDirection(prompt, hint string) string {
	block := FormatLongFormDirectionHint(hint)
	if block == "" {
		return prompt
	}
	return strings.TrimSpace(prompt) + "\n\n" + LongFormDownstreamDirectionGuidance + "\n\n" + block + "\n\n" + LongFormDirectionCoverageDepthRule + "\n" + LongFormDirectionPriorityRule
}
