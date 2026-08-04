package agentexec

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// AgentExecutor는 Plasma 실행 경로가 provider별 agent process를 호출하기 위한 port다.
//
// 구현체는 Codex, Claude 등 provider 차이를 숨기고 AgentResult로 정규화한다.
type AgentExecutor interface {
	Run(context.Context, AgentRequest) (AgentResult, error)
}

// AgentRequest는 한 번의 agent 실행에 필요한 prompt, model, MCP binding을 담는다.
//
// ReportPlan/PartEdit/FinalEditStage 같은 포인터 필드는 해당 실행이 특정 report
// workflow stage에 묶였음을 뜻한다. 동시에 여러 stage를 켜는 조합은 caller가
// 명시적으로 구성해야 하며 executor가 임의로 추론하지 않는다.
type AgentRequest struct {
	UserText           string
	Prompt             string
	Model              string
	ReasoningEffort    string
	MissionID          string
	ToolSessionID      string
	UserEventID        string
	PreviousSessionID  string
	AgentExecutor      string
	MCPMode            string
	Compaction         bool
	DisableTools       bool
	ExtraMCPTools      []string
	ReplaceMCPTools    bool
	ReportPatch        *AgentReportPatchContext
	ReportPlan         *AgentReportPlanContext
	ReportRequirements *reporting.ReportRequirementMapBinding
	PartAssembly       *reporting.PartAssemblyBinding
	PartEdit           *reporting.PartEditBinding
	LongFormFinalize   *reporting.LongFormFinalizeBinding
	FinalEditStage     *reporting.FinalEditStageBinding
}

// AgentReportPlanContext는 report planning MCP tool 제출에 필요한 session 계약이다.
type AgentReportPlanContext struct {
	PendingEventID            string
	ReportMode                string
	IdempotencyKey            string
	PreviousProviderSessionID string
	AgentModel                string
	AgentReasoningEffort      string
	RequireWritingContract    bool
}

// AgentReportPatchContext는 report patch agent가 기존 report artifact를 수정할 때의
// session lineage와 pending event 경계다.
type AgentReportPatchContext struct {
	BaseArtifactID               string
	PendingEventID               string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	MCPMode                      string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ReturnedAgentSessionID       string
	ReportSessionID              string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	SessionChainKind             string
}

// AgentResult는 provider 실행 결과를 Plasma 공통 형태로 정리한 값이다.
//
// Log는 usage/session 추출용 원시 실행 로그일 수 있으므로 사용자 응답이나 장부에
// 무분별하게 저장하지 않는다.
type AgentResult struct {
	Text      string
	SessionID string
	Resumed   bool
	Log       string
	Usage     agentusage.AgentUsage
}

// AgentSessionForkResult는 provider session fork 작업의 lineage와 크기 검증 결과다.
type AgentSessionForkResult struct {
	SessionID       string
	SourceSessionID string
	SourceHash      string
	CloneHash       string
	SourceSizeBytes int64
	CloneSizeBytes  int64
}

// AgentSessionForker는 기존 provider session을 report patch용 session으로 복제하는 port다.
type AgentSessionForker interface {
	ForkSession(context.Context, string) (AgentSessionForkResult, error)
}

// AgentSessionForkReadiness는 provider session fork가 가능한지 사전 점검하는 port다.
type AgentSessionForkReadiness interface {
	CheckForkSession(context.Context, string) error
}

// CodexExecutor는 Codex CLI를 Plasma agent executor port로 연결하는 adapter다.
//
// WorkDir, Timeout, MCPServer 설정은 process 실행에만 적용된다. 제품 state transition은
// MCP tool과 app/reporting service가 맡는다.
type CodexExecutor struct {
	Command   string
	WorkDir   string
	Timeout   time.Duration
	Env       []string
	MCPServer CodexMCPServer
}

// CodexMCPServer는 Codex CLI에 연결할 Plasma MCP server process 설정이다.
type CodexMCPServer struct {
	Name              string
	Command           string
	Args              []string
	Required          bool
	StartupTimeoutSec int
	ToolTimeoutSec    int
	EnabledTools      []string
}
