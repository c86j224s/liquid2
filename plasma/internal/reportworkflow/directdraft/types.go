package directdraft

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// LatestSessionFunc는 미션의 최신 research/provider session을 조회하는 좁은 adapter다.
//
// 반환값은 durable product state가 아니며 one_take 시작 session 또는 explicit same-session
// planned 시작 session을 결정하는 입력으로만 쓰인다.
type LatestSessionFunc func(context.Context, string, string) string

// Runner는 one_take와 planned 본문 작성 stage의 실행 의존성을 묶는다.
type Runner struct {
	Executor        agentexec.AgentExecutor
	NewID           func(string) string
	LatestSessionID LatestSessionFunc
}

// BaseInput은 직접 본문 작성 단계가 공통으로 보존해야 하는 pending 요청 값이다.
type BaseInput struct {
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

// PlannedInput은 canonical plan 이후 planned 본문 작성 단계의 typed 입력이다.
type PlannedInput struct {
	BaseInput
	Plan                       reporting.ReportPlan
	PlanEventID                string
	PlanToolSessionID          string
	ArtifactID                 string
	ReportPlanSessionID        string
	SessionChainKind           string
	PreReportResearchSessionID string
	ForkSourceSessionID        string
	WorkflowStartedAt          time.Time
}

// OneTakeCandidate는 provider가 만든 one_take Markdown과 저장 단계가 보존해야 할 session metadata다.
//
// 이 값은 artifact나 ledger event를 포함하지 않는다. finalstore가 media type, filename,
// producer, artifact request와 terminal event payload를 고정 정책으로 조립한다.
type OneTakeCandidate struct {
	ArtifactID          string
	ToolSessionID       string
	PreviousSessionID   string
	ReturnedSessionID   string
	ReportSessionID     string
	ReportSessionPolicy string
	Markdown            string
	StartedAt           time.Time
	AgentDurationMS     int64
	AgentUsage          agentusage.AgentUsage
	AgentResumed        bool
}

// PlannedCandidate는 canonical plan 이후 provider가 만든 planned Markdown 후보와 plan binding이다.
//
// Planned 저장은 반드시 finalstore.CommitPlanned가 수행하며, 이 후보는 provider 결과와
// canonical plan artifact identity를 전달할 뿐 저장 방식을 선택하지 않는다.
type PlannedCandidate struct {
	ArtifactID                 string
	ToolSessionID              string
	PlanEventID                string
	PlanToolSessionID          string
	ReportPlanSessionID        string
	SessionChainKind           string
	PreReportResearchSessionID string
	ForkSourceSessionID        string
	ReturnedSessionID          string
	ReportSessionID            string
	Markdown                   string
	WorkflowStartedAt          time.Time
	AgentDurationMS            int64
	AgentUsage                 agentusage.AgentUsage
	AgentResumed               bool
}
