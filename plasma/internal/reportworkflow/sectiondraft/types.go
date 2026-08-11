package sectiondraft

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// Service는 Section draft 실행 결과를 durable artifact/event로 저장하는 포트다.
type Service interface {
	AppendEvent(context.Context, ledger.AppendRequest) (ledger.Event, error)
	CreateRawArtifact(context.Context, artifact.CreateRequest) (artifact.Raw, error)
}

// Runner는 단일 Section draft stage 실행 의존성을 묶는다.
type Runner struct {
	Service  Service
	Executor agentexec.AgentExecutor
	NewID    func(string) string
}

// BaseInput은 모든 Section draft가 공유하는 canonical long-form plan 계약이다.
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
	ForkSourceSessionID          string
	RequirementMap               reporting.ReportRequirementMap
}

// Input은 root가 session lineage를 확정한 뒤 단일 Section writer에게 넘기는 값이다.
type Input struct {
	Base              BaseInput
	Part              reporting.ReportPlanPart
	Section           reporting.ReportPlanSection
	PartIndex         int
	SectionIndex      int
	Attempt           int
	ToolSessionID     string
	PreviousSessionID string
	SourceSessionID   string
	UserText          string
	StartedEvent      bool
	CreatedText       string
}

// Draft는 durable Section artifact와 Markdown body를 다음 stage로 넘기는 typed 결과다.
type Draft struct {
	Title      string
	Markdown   string
	ArtifactID string
	WordCount  int
	SessionID  string
}

// EvidenceGap is the typed non-Markdown outcome for a Section writer attempt.
// It never carries source excerpts or free-form diagnosis; durable recovery
// relies only on the fixed reason code, section coordinate, and session lineage.
type EvidenceGap struct {
	PartIndex         int
	SectionIndex      int
	Attempt           int
	ReasonCode        string
	SessionID         string
	ReturnedSessionID string
	PreviousSessionID string
	ToolSessionID     string
	SourceSessionID   string
	DurationMS        int64
}

// Output은 provider 결과와 durable Section artifact metadata다.
type Output struct {
	Draft             Draft
	EvidenceGap       *EvidenceGap
	ReturnedSessionID string
	DurationMS        int64
}
