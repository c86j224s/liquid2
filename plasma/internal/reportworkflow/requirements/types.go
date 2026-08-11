package requirements

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// Service는 requirements stage가 recovery와 requirement map lifecycle에 쓰는 저장 포트다.
type Service interface {
	ListEvents(context.Context, string) ([]ledger.Event, error)
}

// Runner는 requirements stage의 provider와 durable lifecycle 의존성을 묶는다.
type Runner struct {
	Service   Service
	Lifecycle reporting.Runner
	Executor  agentexec.AgentExecutor
}

// Input은 canonical plan 이후 요구사항 mapping stage가 필요로 하는 typed 값이다.
type Input struct {
	MissionID      string
	PendingEventID string
	PlanEventID    string
	PlanSessionID  string
	// ValidatedDownstream은 root가 section/part artifact와 payload lineage를 이미 검증했을 때만
	// requirements legacy mapping을 건너뛸 수 있음을 표시하는 typed 신호다.
	ValidatedDownstream bool
	Title               string
	DirectionHint       string
	AgentExecutor       string
	AgentModel          string
	ReasoningEffort     string
	MCPMode             string
	Plan                reporting.SectionalReportPlan
}

// Output은 다음 writing stage들이 조회하는 요구사항 map과 durable event다.
type Output struct {
	RequirementMap reporting.ReportRequirementMap
	Event          ledger.Event
	Recovered      bool
}
