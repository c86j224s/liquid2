package app

import "github.com/c86j224s/liquid2/plasma/internal/agentpolicy"

// NormalizeAgentExecutorName는 애플리케이션 서비스 계층 입력을 표준 형태로 정규화하고 허용되지 않는 값은 안정 오류로 거부한다.
func NormalizeAgentExecutorName(value string) (string, error) {
	return agentpolicy.NormalizeExecutorName(value)
}

// LockedAgentExecutorFromEvents는 과거 이벤트가 이미 고정한 agent executor를 찾는다.
func LockedAgentExecutorFromEvents(events []LedgerEvent) string {
	return agentpolicy.LockedExecutorFromEvents(events)
}

// ValidateMissionAgentExecutorForEvents는 애플리케이션 서비스 계층 계약을 검사한다. 제품 상태를 변경하지 않는 순수 검증 경계다.
func ValidateMissionAgentExecutorForEvents(events []LedgerEvent, requested string) error {
	return agentpolicy.ValidateMissionExecutor(events, requested)
}

// ValidateAgentExecutorAppend는 애플리케이션 서비스 계층 계약을 검사한다. 제품 상태를 변경하지 않는 순수 검증 경계다.
func ValidateAgentExecutorAppend(events []LedgerEvent, appended []LedgerEvent) error {
	return agentpolicy.ValidateAppend(events, appended)
}

// ExplicitLockingAgentExecutor는 이벤트 payload가 명시한 executor lock을 안전하게 읽는다.
func ExplicitLockingAgentExecutor(event LedgerEvent) (string, bool) {
	return agentpolicy.ExplicitLockingExecutor(event)
}

// EventLocksAgentExecutor는 이벤트 타입이 미션의 executor 선택을 고정하는지 판정한다.
func EventLocksAgentExecutor(eventType string) bool {
	return agentpolicy.EventLocksExecutor(eventType)
}
