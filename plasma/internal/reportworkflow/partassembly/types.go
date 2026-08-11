package partassembly

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// Service는 Part assembly replay와 Part artifact 저장에 필요한 저장 포트다.
type Service interface {
	ListEvents(context.Context, string) ([]ledger.Event, error)
	AppendEvent(context.Context, ledger.AppendRequest) (ledger.Event, error)
	CreateRawArtifact(context.Context, artifact.CreateRequest) (artifact.Raw, error)
}

// Runner는 Part assembly stage 실행 의존성을 묶는다.
type Runner struct {
	Service  Service
	Executor agentexec.AgentExecutor
	NewID    func(string) string
}

// SectionDraft는 Section writer 출력 중 Part assembly가 소비하는 불변 manuscript 조각이다.
type SectionDraft struct {
	Title      string
	Markdown   string
	ArtifactID string
	WordCount  int
}

// PartDraft는 다음 Part edit 또는 final tail이 소비하는 assembled Part artifact다.
type PartDraft struct {
	Title      string
	Markdown   string
	ArtifactID string
	WordCount  int
	SessionID  string
}

// BaseInput은 모든 Part assembly task가 공유하는 long-form 계약이다.
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
}

// Input은 root가 session lineage와 Section 목록을 확정한 뒤 단일 Part assembly에 넘기는 값이다.
type Input struct {
	Base                BaseInput
	Part                reporting.ReportPlanPart
	PartIndex           int
	Sections            []SectionDraft
	ToolSessionID       string
	PreviousSessionID   string
	ForkSourceSessionID string
}

// Output은 durable Part artifact와 provider session metadata다.
type Output struct {
	Draft             PartDraft
	ReturnedSessionID string
	DurationMS        int64
}
