package partedit

import (
	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// Service는 Part edit durable start/outcome replay에 필요한 저장 포트다.
type Service interface {
	reporting.PartEditStore
}

// Runner는 Part edit stage 실행 의존성을 묶는다.
type Runner struct {
	Service  Service
	Executor agentexec.AgentExecutor
	NewID    func(string) string
}

// PartDraft는 Part edit이 읽고 결과로 반환하는 Part manuscript artifact다.
type PartDraft struct {
	Title      string
	Markdown   string
	ArtifactID string
	WordCount  int
}

// BaseInput은 모든 Part edit task가 공유하는 long-form 계약이다.
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
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	Plan                         reporting.SectionalReportPlan
	PlanEvent                    ledger.Event
	RequirementMap               reporting.ReportRequirementMap
	RequirementMapEvent          ledger.Event
	ReportPlanSessionID          string
	SessionChainKind             string
}

// Input은 root가 session lineage를 정한 뒤 단일 Part editor/author에게 넘기는 값이다.
type Input struct {
	Base                     BaseInput
	Part                     reporting.ReportPlanPart
	PartIndex                int
	Source                   PartDraft
	ToolSessionID            string
	PreviousSessionID        string
	EditedArtifactID         string
	Filename                 string
	ForkSourceAgentSessionID string
	PartPlanningBrief        string
	AuthorMode               bool
}

// Output은 Part edit 도구가 durably 제출한 Part manuscript artifact다.
type Output struct {
	Draft  PartDraft
	Result agentexec.AgentResult
}
