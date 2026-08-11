package finalstore

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// Service는 finalstore가 최종 report artifact와 terminal event를 durable하게 기록할 때 쓰는 저장 포트다.
type Service interface {
	AppendEvent(context.Context, ledger.AppendRequest) (ledger.Event, error)
	CreateRawArtifact(context.Context, artifact.CreateRequest) (artifact.Raw, error)
	CreateRawArtifactWithEvent(context.Context, artifact.CreateRequest, func(artifact.Raw) ledger.AppendRequest) (artifact.Raw, ledger.Event, error)
}

// GateReader는 이미 canonical으로 저장된 long-form final gate 결과를 재조회하는 소비자 측 포트다.
//
// finalstore는 terminal event를 쓰지 않고, gate stage가 만든 durable binding으로 결과를
// 다시 읽어 cross-validation한 뒤 workflow output으로만 채택한다.
type GateReader interface {
	ReadFinalGate(context.Context, GateReadRequest) (GateRecord, error)
}

// Runner는 one_take/planned 최종 저장과 gate 결과 채택을 담당하는 stage runner다.
type Runner struct {
	Service    Service
	GateReader GateReader
	NewID      func(string) string
}

// BaseInput은 최종 report artifact event payload가 pending 요청에서 보존해야 하는 값이다.
type BaseInput struct {
	MissionID       string
	PendingEventID  string
	Title           string
	AgentExecutor   string
	AgentModel      string
	ReasoningEffort string
	SelectionSource string
	MCPMode         string
	Rigor           reportprompt.RigorProfile
	SessionPolicy   string
	PolicySelection string
	PostHumanize    string
	GuidanceProfile string
	GuidanceSHA256  string
}

// OneTakeCandidate는 directdraft가 검증한 provider 결과와 one_take identity 예약 값이다.
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

// PlannedCandidate는 directdraft가 검증한 planned provider 결과와 canonical plan binding이다.
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

// OneTakeInput은 one_take 비원자 저장 정책을 실행하기 위한 typed 입력이다.
type OneTakeInput struct {
	Base      BaseInput
	Candidate OneTakeCandidate
}

// PlannedInput은 planned 원자 저장 정책을 실행하기 위한 typed 입력이다.
type PlannedInput struct {
	Base      BaseInput
	Candidate PlannedCandidate
}

// Output은 검증을 통과해 workflow의 최종 결과로 채택된 artifact와 event다.
type Output struct {
	Artifact        artifact.Raw
	Event           ledger.Event
	Markdown        string
	ReportSessionID string
}

// GateReadRequest는 finalstore가 long-form gate canonical 결과를 durable store에서 다시 읽을 때 쓰는 key다.
type GateReadRequest struct {
	MissionID      string
	PendingEventID string
	PlanEventID    string
	ArtifactID     string
	Binding        reporting.LongFormFinalizeBinding
}

// GateInput은 이미 저장된 long-form gate 결과를 finalstore 출력으로 채택하기 위한 typed 입력이다.
type GateInput struct {
	GateReadRequest
}

// GateRecord는 durable reader가 복원한 canonical artifact/event와 lineage cardinality다.
type GateRecord struct {
	Artifact              artifact.Raw
	Event                 ledger.Event
	Markdown              string
	ReportSessionID       string
	CanonicalLineageCount int
}
