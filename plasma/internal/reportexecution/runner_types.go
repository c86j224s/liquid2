package reportexecution

import (
	"context"
	"log"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

// Service는 report runner가 필요한 artifact, source, 장부 기능만 사용하는 애플리케이션 포트다.
type Service interface {
	AppendEvent(context.Context, ledger.AppendRequest) (ledger.Event, error)
	AppendEvents(context.Context, string, []ledger.AppendRequest) ([]ledger.Event, error)
	AppendReportTerminalIfOpen(context.Context, string, string, []ledger.AppendRequest) ([]ledger.Event, bool, error)
	AppendEventsIfNoActiveAgentWork(context.Context, string, []ledger.AppendRequest) ([]ledger.Event, error)
	ListEvents(context.Context, string) ([]ledger.Event, error)
	ListSourceSnapshotsWithState(context.Context, source.ListRequest) ([]source.Snapshot, error)
}

// DraftRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type DraftRequest struct {
	Title                        string
	DirectionHint                string
	ExecutionStrategy            string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSelectionSource         string
	MCPMode                      string
	RigorLevel                   string
	RigorLabel                   string
	ReportMode                   string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	PostReportHumanize           string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
}

// SessionPolicySelectionInput는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type SessionPolicySelectionInput struct {
	RequestedPolicy             string
	ReportMode                  string
	CanForkSession              bool
	HasPreReportResearchSession bool
	ForkReady                   bool
}

// DesignRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type DesignRequest struct {
	SourceArtifactID     string
	SourceMediaType      string
	Title                string
	AgentExecutor        string
	AgentModel           string
	AgentReasoningEffort string
	RendererVersion      string
}

// HumanizeRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type HumanizeRequest struct {
	SourceArtifactID       string
	SourceArtifactSHA256   string
	SourceMediaType        string
	Title                  string
	AgentExecutor          string
	AgentModel             string
	AgentReasoningEffort   string
	MCPMode                string
	PreviousAgentSessionID string
	ToolSessionID          string
	ReportMode             string
	ReportPendingEventID   string
}

// PatchRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type PatchRequest struct {
	BaseArtifactID               string
	Instruction                  string
	Title                        string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	MCPMode                      string
	ReportSessionID              string
	PreviousAgentSessionID       string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	SessionChainKind             string
}

// PatchFinalizedEventRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type PatchFinalizedEventRequest struct {
	EventID                      string
	MissionID                    string
	CorrelationID                string
	PendingEventID               string
	Title                        string
	Artifact                     artifact.Raw
	BaseArtifactID               string
	PatchID                      string
	PatchInstruction             string
	PatchSummary                 string
	OperationCount               int
	Operations                   any
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ReturnedAgentSessionID       string
	ReportSessionID              string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	ToolSessionID                string
	MCPMode                      string
	ProducerToolName             string
	SessionChainKind             string
	Producer                     ledger.Producer
}

// SelfContainedHTMLExportEventRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type SelfContainedHTMLExportEventRequest struct {
	EventID          string
	MissionID        string
	SourceArtifactID string
	Artifact         artifact.Raw
	RendererVersion  string
	Producer         ledger.Producer
}

// DesignedHTMLExportEventRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type DesignedHTMLExportEventRequest struct {
	EventID                string
	MissionID              string
	PendingEventID         string
	SourceArtifactID       string
	ContentModelArtifactID string
	Artifact               artifact.Raw
	RendererVersion        string
	ImageSetFingerprint    string
	AgentExecutor          string
	AgentModel             string
	AgentReasoningEffort   string
	AgentSessionID         string
	ToolSessionID          string
	DurationMS             int64
	AgentDurationMS        int64
	AgentUsage             agentusage.AgentUsage
	AgentResumed           bool
	Producer               ledger.Producer
}

// Runner는 report 요청을 pending event에서 terminal event까지 실행하는 오케스트레이터다.
type Runner struct {
	Service          Service
	InFlight         *InFlight
	NewID            func(string) string
	GenerateDraft    func(context.Context, string, DraftRequest, string) error
	GenerateDesign   func(context.Context, string, DesignRequest, string) error
	GenerateHumanize func(context.Context, string, HumanizeRequest, string) error
	GeneratePatch    func(context.Context, string, PatchRequest, string) error
}

func logTerminalWriteFailure(missionID, pendingEventID, reportType, intendedEventType string, err error) {
	if err == nil {
		return
	}
	log.Printf("report_terminal_write_failed mission_id=%q pending_event_id=%q report_type=%q intended_event_type=%q err=%q", missionID, pendingEventID, reportType, intendedEventType, err)
}
