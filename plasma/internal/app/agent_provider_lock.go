package app

import (
	"github.com/c86j224s/liquid2/plasma/internal/agentpolicy"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

// NormalizeAgentExecutorName는 애플리케이션 서비스 계층 입력을 표준 형태로 정규화하고 허용되지 않는 값은 안정 오류로 거부한다.
func NormalizeAgentExecutorName(value string) (string, error) {
	return agentpolicy.NormalizeExecutorName(value)
}

// LockedAgentExecutorFromEvents는 현재 진행 중인 작업이 있어 실질적으로 고정된
// agent executor를 찾는다. 진행 중인 작업이 없으면 빈 문자열을 반환하며, 이 경우
// 다음 요청은 다른 executor를 자유롭게 선택할 수 있다.
func LockedAgentExecutorFromEvents(events []ledger.Event) string {
	return agentpolicy.ActiveExecutorFromEvents(events)
}

// ValidateMissionAgentExecutorForEvents는 애플리케이션 서비스 계층 계약을 검사한다. 제품 상태를 변경하지 않는 순수 검증 경계다.
func ValidateMissionAgentExecutorForEvents(events []ledger.Event, requested string) error {
	return agentpolicy.ValidateMissionExecutor(events, requested)
}

// ValidateAgentExecutorAppend는 애플리케이션 서비스 계층 계약을 검사한다. 제품 상태를 변경하지 않는 순수 검증 경계다.
func ValidateAgentExecutorAppend(events []ledger.Event, appended []ledger.Event) error {
	return agentpolicy.ValidateAppend(events, appended)
}

// ExplicitLockingAgentExecutor는 이벤트 payload가 명시한 executor lock을 안전하게 읽는다.
func ExplicitLockingAgentExecutor(event ledger.Event) (string, bool) {
	return agentpolicy.ExplicitLockingExecutor(event)
}

// EventLocksAgentExecutor는 이벤트 타입이 미션의 executor 선택을 고정하는지 판정한다.
func EventLocksAgentExecutor(eventType string) bool {
	return agentpolicy.EventLocksExecutor(eventType)
}
