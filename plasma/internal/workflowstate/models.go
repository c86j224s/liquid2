package workflowstate

import "time"

const (
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
	WorkflowSurfaceWeb          = "web"
	WorkflowSurfaceCLI          = "cli"
	WorkflowSurfaceMCP          = "mcp"
	WorkflowSurfaceAgentSession = "agent_session"
)

const (
	WorkflowStepInstructionModeCurrent = "current"
	WorkflowStepInstructionModeLayered = "layered"
)

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

type RequestWorkflowStopRequest struct {
	WorkflowRunID            string
	MissionID                string
	RequestedBySurface       string
	RequestedByToolSessionID string
	Reason                   string
}

type WorkflowRunTerminalEventRequest struct {
	WorkflowRunID string
	MissionID     string
	EventType     string
	Reason        string
	Error         string
}

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

type WorkflowRunStopRequestedPayload struct {
	WorkflowRunID            string `json:"workflow_run_id"`
	MissionID                string `json:"mission_id"`
	RequestedBySurface       string `json:"requested_by_surface"`
	RequestedByToolSessionID string `json:"requested_by_tool_session_id,omitempty"`
	Reason                   string `json:"reason,omitempty"`
	RequestedAt              string `json:"requested_at"`
}

type WorkflowRunStartedPayload struct {
	WorkflowRunID string `json:"workflow_run_id"`
	MissionID     string `json:"mission_id"`
	StartedAt     string `json:"started_at,omitempty"`
}

type WorkflowStepStartedPayload struct {
	WorkflowRunID  string `json:"workflow_run_id"`
	MissionID      string `json:"mission_id"`
	WorkflowStepID string `json:"workflow_step_id"`
	Instruction    string `json:"instruction,omitempty"`
	StepIndex      int    `json:"step_index,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	ToolSessionID  string `json:"tool_session_id,omitempty"`
}

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
