package workflowstate

import "time"

const (
	// WorkflowRunRequestedEvent부터 WorkflowRunInterruptedEvent까지는 workflow runner와
	// UI projection이 공유하는 장부 event type이다. 기존 장부 해석을 위해 문자열은
	// stable contract로 취급한다.
	WorkflowRunRequestedEvent     = "workflow.run.requested"
	WorkflowRunStartedEvent       = "workflow.run.started"
	WorkflowRunStopRequestedEvent = "workflow.run.stop_requested"
	WorkflowSourceSkippedEvent    = "workflow.source.skipped"
	WorkflowStepStartedEvent      = "workflow.step.started"
	WorkflowStepCompletedEvent    = "workflow.step.completed"
	WorkflowRunCompletedEvent     = "workflow.run.completed"
	WorkflowRunPausedEvent        = "workflow.run.paused"
	WorkflowRunStoppedEvent       = "workflow.run.stopped"
	WorkflowRunFailedEvent        = "workflow.run.failed"
	WorkflowRunInterruptedEvent   = "workflow.run.interrupted"
)

const (
	// WorkflowStatus* 값은 WorkflowRunView.Status의 닫힌 집합이다.
	WorkflowStatusQueued      = "queued"
	WorkflowStatusRunning     = "running"
	WorkflowStatusStopping    = "stopping"
	WorkflowStatusCompleted   = "completed"
	WorkflowStatusPaused      = "paused"
	WorkflowStatusStopped     = "stopped"
	WorkflowStatusFailed      = "failed"
	WorkflowStatusInterrupted = "interrupted"
)

const (
	// WorkflowSurface* 값은 workflow 요청이 들어온 제품 표면을 나타낸다.
	WorkflowSurfaceWeb          = "web"
	WorkflowSurfaceCLI          = "cli"
	WorkflowSurfaceMCP          = "mcp"
	WorkflowSurfaceAgentSession = "agent_session"
)

const (
	// WorkflowStepInstructionMode* 값은 workflow step prompt를 구성하는 방식이다.
	WorkflowStepInstructionModeCurrent = "current"
	WorkflowStepInstructionModeLayered = "layered"
)

// RequestWorkflowRunRequest는 새 workflow run을 장부에 요청할 때의 application 입력이다.
//
// 이 타입은 실행 자체를 시작하지 않는다. workflowruns service가 조건부 append로
// requested event를 기록하고, 별도 runner가 그 event를 보고 실행한다.
type RequestWorkflowRunRequest struct {
	WorkflowRunID             string
	MissionID                 string
	RequestedBySurface        string
	RequestedByToolSessionID  string
	AgentExecutor             string
	MCPMode                   string
	StepInstructionMode       string
	UserInstructionRaw        string
	RunGoal                   string
	Instruction               string
	MaxSteps                  int
	MaxDurationMS             int64
	StopCondition             string
	StartAfterEventID         string
	ArgumentSummary           string
	ContinueFromWorkflowRunID string
}

// RequestWorkflowStopRequest는 실행 중이거나 queued 상태인 workflow run의 중지 요청이다.
type RequestWorkflowStopRequest struct {
	WorkflowRunID            string
	MissionID                string
	RequestedBySurface       string
	RequestedByToolSessionID string
	Reason                   string
}

// WorkflowRunTerminalEventRequest는 runner가 run을 terminal 상태로 닫을 때 쓰는 입력이다.
//
// 같은 run에 terminal event가 이미 있으면 builder는 새 event를 만들지 않아야 한다.
type WorkflowRunTerminalEventRequest struct {
	WorkflowRunID string
	MissionID     string
	EventType     string
	Reason        string
	Error         string
}

// WorkflowRunRequestedPayload는 workflow.run.requested event의 stable JSON payload다.
type WorkflowRunRequestedPayload struct {
	WorkflowRunID             string `json:"workflow_run_id"`
	MissionID                 string `json:"mission_id"`
	RequestedBySurface        string `json:"requested_by_surface"`
	RequestedByToolSessionID  string `json:"requested_by_tool_session_id,omitempty"`
	AgentExecutor             string `json:"agent_executor"`
	MCPMode                   string `json:"mcp_mode"`
	StepInstructionMode       string `json:"step_instruction_mode"`
	UserInstructionRaw        string `json:"user_instruction_raw,omitempty"`
	RunGoal                   string `json:"run_goal,omitempty"`
	Instruction               string `json:"instruction"`
	MaxSteps                  int    `json:"max_steps"`
	MaxDurationMS             int64  `json:"max_duration_ms"`
	StopCondition             string `json:"stop_condition"`
	StartAfterEventID         string `json:"start_after_event_id,omitempty"`
	CreatedAt                 string `json:"created_at"`
	ArgumentSummary           string `json:"argument_summary"`
	ContinueFromWorkflowRunID string `json:"continue_from_workflow_run_id,omitempty"`
}

// WorkflowRunStopRequestedPayload는 workflow.run.stop_requested event payload다.
type WorkflowRunStopRequestedPayload struct {
	WorkflowRunID            string `json:"workflow_run_id"`
	MissionID                string `json:"mission_id"`
	RequestedBySurface       string `json:"requested_by_surface"`
	RequestedByToolSessionID string `json:"requested_by_tool_session_id,omitempty"`
	Reason                   string `json:"reason,omitempty"`
	RequestedAt              string `json:"requested_at"`
}

// WorkflowRunStartedPayload는 runner가 run ownership을 claim한 시점의 payload다.
type WorkflowRunStartedPayload struct {
	WorkflowRunID string `json:"workflow_run_id"`
	MissionID     string `json:"mission_id"`
	StartedAt     string `json:"started_at,omitempty"`
}

// WorkflowStepStartedPayload는 단일 workflow step 실행 시작을 나타낸다.
type WorkflowStepStartedPayload struct {
	WorkflowRunID  string `json:"workflow_run_id"`
	MissionID      string `json:"mission_id"`
	WorkflowStepID string `json:"workflow_step_id"`
	Instruction    string `json:"instruction,omitempty"`
	StepIndex      int    `json:"step_index,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	ToolSessionID  string `json:"tool_session_id,omitempty"`
}

// WorkflowSourceSkippedPayload는 workflow가 source를 읽지 않고 넘어간 이유를 기록한다.
type WorkflowSourceSkippedPayload struct {
	WorkflowRunID   string `json:"workflow_run_id"`
	MissionID       string `json:"mission_id"`
	WorkflowStepID  string `json:"workflow_step_id,omitempty"`
	StepIndex       int    `json:"step_index,omitempty"`
	SnapshotID      string `json:"snapshot_id"`
	Reason          string `json:"reason"`
	RemovedEventID  string `json:"removed_event_id,omitempty"`
	SkippedAt       string `json:"skipped_at,omitempty"`
	RetrievalPolicy string `json:"retrieval_policy,omitempty"`
	ConnectorType   string `json:"connector_type,omitempty"`
}

// WorkflowStepCompletedPayload는 한 step의 agent 판단과 다음 지시를 기록한다.
type WorkflowStepCompletedPayload struct {
	WorkflowRunID   string `json:"workflow_run_id"`
	MissionID       string `json:"mission_id"`
	WorkflowStepID  string `json:"workflow_step_id"`
	Decision        string `json:"decision"`
	NextInstruction string `json:"next_instruction,omitempty"`
	Reason          string `json:"reason,omitempty"`
	DurationMS      int64  `json:"duration_ms,omitempty"`
	AgentSessionID  string `json:"agent_session_id,omitempty"`
	ToolSessionID   string `json:"tool_session_id,omitempty"`
	ResultEventID   string `json:"result_event_id,omitempty"`
}

// WorkflowRunTerminalPayload는 completed/paused/stopped/failed/interrupted event가
// 공유하는 terminal payload다.
type WorkflowRunTerminalPayload struct {
	WorkflowRunID      string `json:"workflow_run_id"`
	MissionID          string `json:"mission_id"`
	Reason             string `json:"reason,omitempty"`
	StopReason         string `json:"stop_reason,omitempty"`
	Error              string `json:"error,omitempty"`
	NextInstruction    string `json:"next_instruction,omitempty"`
	CompletedStepCount int    `json:"completed_step_count,omitempty"`
	TerminalAt         string `json:"terminal_at,omitempty"`
}

// WorkflowRunView는 장부 event 열에서 계산한 workflow run의 읽기 전용 projection이다.
//
// View는 저장소의 source of truth가 아니며, ProjectRuns가 매번 event 순서대로 다시
// 계산할 수 있어야 한다.
type WorkflowRunView struct {
	WorkflowRunID             string             `json:"workflow_run_id"`
	MissionID                 string             `json:"mission_id"`
	Status                    string             `json:"status"`
	RequestedBySurface        string             `json:"requested_by_surface,omitempty"`
	AgentExecutor             string             `json:"agent_executor,omitempty"`
	MCPMode                   string             `json:"mcp_mode,omitempty"`
	StepInstructionMode       string             `json:"step_instruction_mode,omitempty"`
	UserInstructionRaw        string             `json:"user_instruction_raw,omitempty"`
	RunGoal                   string             `json:"run_goal,omitempty"`
	Instruction               string             `json:"instruction,omitempty"`
	MaxSteps                  int                `json:"max_steps,omitempty"`
	MaxDurationMS             int64              `json:"max_duration_ms,omitempty"`
	StopCondition             string             `json:"stop_condition,omitempty"`
	StartAfterEventID         string             `json:"start_after_event_id,omitempty"`
	CurrentStep               *WorkflowStepView  `json:"current_step,omitempty"`
	Steps                     []WorkflowStepView `json:"steps,omitempty"`
	CompletedStepCount        int                `json:"completed_step_count"`
	StopReason                string             `json:"stop_reason,omitempty"`
	ContinuationInstruction   string             `json:"continuation_instruction,omitempty"`
	StatusText                string             `json:"status_text"`
	RequestedEventID          string             `json:"requested_event_id,omitempty"`
	StartedEventID            string             `json:"started_event_id,omitempty"`
	StopRequestedEventID      string             `json:"stop_requested_event_id,omitempty"`
	TerminalEventID           string             `json:"terminal_event_id,omitempty"`
	ContinueFromWorkflowRunID string             `json:"continue_from_workflow_run_id,omitempty"`
	LatestEventID             string             `json:"latest_event_id,omitempty"`
	LatestSequence            int64              `json:"latest_sequence,omitempty"`
	RequestedAt               time.Time          `json:"requested_at,omitempty"`
	UpdatedAt                 time.Time          `json:"updated_at,omitempty"`
}

// WorkflowStepView는 WorkflowRunView 안에서 한 step의 현재 또는 완료 상태를 나타낸다.
type WorkflowStepView struct {
	WorkflowStepID  string   `json:"workflow_step_id"`
	StepIndex       int      `json:"step_index,omitempty"`
	Status          string   `json:"status"`
	Instruction     string   `json:"instruction,omitempty"`
	Decision        string   `json:"decision,omitempty"`
	NextInstruction string   `json:"next_instruction,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	DurationMS      int64    `json:"duration_ms,omitempty"`
	AgentSessionID  string   `json:"agent_session_id,omitempty"`
	ToolSessionID   string   `json:"tool_session_id,omitempty"`
	ResultEventID   string   `json:"result_event_id,omitempty"`
	StartedEventID  string   `json:"started_event_id,omitempty"`
	ResultEventIDs  []string `json:"result_event_ids,omitempty"`
}
