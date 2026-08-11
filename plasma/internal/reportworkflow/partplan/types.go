package partplan

import (
	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// Service는 part planning이 conditional durable event를 기록할 때 쓰는 저장 포트다.
type Service interface {
	reporting.PartPlanStore
}

// Runner는 Part planning provider와 ID 생성을 묶는다.
type Runner struct {
	Service  Service
	Executor agentexec.AgentExecutor
	NewID    func(string) string
}

// BaseInput은 모든 Part planning task가 공유하는 long-form plan 계약이다.
type BaseInput struct {
	MissionID                    string
	PendingEventID               string
	Title                        string
	DirectionHint                string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSelectionSource         string
	MCPMode                      string
	Rigor                        reportprompt.RigorProfile
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	PostReportHumanize           string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	Plan                         reporting.SectionalReportPlan
	PlanEvent                    ledger.Event
	ReportPlanSessionID          string
	SessionChainKind             string
	PreReportResearchSessionID   string
	RequirementMap               reporting.ReportRequirementMap
}

// Input은 root가 session fork 후 단일 Part planning 실행에 넘기는 값이다.
type Input struct {
	Base              BaseInput
	Part              reporting.ReportPlanPart
	PartIndex         int
	ProviderSessionID string
	ForkSourceSession string
}

// Output은 durable part_plan event에서 복원 가능한 Part brief와 provider session이다.
type Output struct {
	PartIndex         int
	Brief             string
	ProviderSessionID string
	Event             ledger.Event
	Recovered         bool
}
