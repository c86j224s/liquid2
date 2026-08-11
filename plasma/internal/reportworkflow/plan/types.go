package plan

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// Service는 planned plan 단계가 durable replay와 canonical promotion에 쓰는 저장 포트다.
type Service interface {
	ListEvents(context.Context, string) ([]ledger.Event, error)
}

// LongFormSectionPlanRepairService validates replacement references before it
// records the single durable plan-repair outcome.
type LongFormSectionPlanRepairService interface {
	reporting.LongFormSectionPlanRepairStore
	ValidateReportPlanRefs(context.Context, string, []reporting.ReportPlanSourceRefs) error
}

// LatestSessionFunc는 현재 미션의 최신 research/provider session을 조회하는 adapter다.
type LatestSessionFunc func(context.Context, string, string) string

// Runner는 planned Markdown plan stage의 실행 의존성을 묶는다.
type Runner struct {
	Service         Service
	RepairStore     LongFormSectionPlanRepairService
	Lifecycle       reporting.Runner
	Executor        agentexec.AgentExecutor
	NewID           func(string) string
	LatestSessionID LatestSessionFunc
}

// Input은 planned plan stage가 pending 요청에서 받아야 하는 typed 값이다.
type Input struct {
	MissionID, PendingEventID        string
	Title, DirectionHint             string
	AgentExecutor                    string
	AgentModel, AgentReasoningEffort string
	AgentSelectionSource, MCPMode    string
	Rigor                            reportprompt.RigorProfile
	ReportSessionPolicy              string
	ReportSessionPolicySelection     string
	PostReportHumanize               string
	GenerationGuidanceProfile        string
	GenerationGuidanceSHA256         string
}

// Output은 canonical planned plan과 다음 directdraft 단계가 이어야 할 session lineage다.
type Output struct {
	Plan                         reporting.ReportPlan
	Event                        ledger.Event
	ArtifactID                   string
	PlanToolSessionID            string
	ReportPlanSessionID          string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	SessionChainKind             string
	PreReportResearchSessionID   string
	ForkSourceSessionID          string
	Recovered                    bool
	StartedAt                    time.Time
}

// LongFormInput은 section-first long-form plan stage가 pending 요청에서 받는 typed 값이다.
type LongFormInput struct {
	Input
	SectionFanout bool
}

// LongFormOutput은 canonical long-form plan과 prefix graph 선택에 필요한 frozen flag다.
type LongFormOutput struct {
	Plan                         reporting.SectionalReportPlan
	Event                        ledger.Event
	ArtifactID                   string
	ReportPlanSessionID          string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSelectionSource         string
	MCPMode                      string
	SessionChainKind             string
	PreReportResearchSessionID   string
	ForkSourceSessionID          string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	PartEditEnabled              bool
	PartPlanningEnabled          bool
	FinalEditPipeline            string
	Recovered                    bool
	StartedAt                    time.Time
}

// LongFormSectionRepairInput limits one plan correction to the terminal gap
// coordinates supplied by the root long-form runner.
type LongFormSectionRepairInput struct {
	Request        LongFormInput
	Plan           LongFormOutput
	RequirementMap reporting.ReportRequirementMap
	Gaps           []reporting.ReportSectionCoordinate
}

// LongFormSectionRepairOutput returns the effective immutable-plan amendment.
type LongFormSectionRepairOutput struct {
	Plan         reporting.SectionalReportPlan
	Event        ledger.Event
	Replacements []reporting.ReportSectionPlanReplacement
	Recovered    bool
}
