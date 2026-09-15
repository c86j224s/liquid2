package workflow

// StartInput is the complete plasma.workflow.start wire payload. Root MCP
// validation must decode this exact shape before applying shared transport
// policy so optional fields and their zero values retain their wire meaning.
type StartInput struct {
	MissionID                string `json:"mission_id"`
	WorkflowRunID            string `json:"workflow_run_id"`
	StepInstructionMode      string `json:"step_instruction_mode"`
	UserInstructionRaw       string `json:"user_instruction_raw"`
	RunGoal                  string `json:"run_goal"`
	Instruction              string `json:"instruction"`
	AgentExecutor            string `json:"agent_executor"`
	MCPMode                  string `json:"mcp_mode"`
	MaxSteps                 int    `json:"max_steps"`
	MaxDurationMS            int64  `json:"max_duration_ms"`
	StopCondition            string `json:"stop_condition"`
	StartAfterEventID        string `json:"start_after_event_id"`
	RequestedByToolSessionID string `json:"requested_by_tool_session_id"`
}

// StatusInput is the complete plasma.workflow.status wire payload.
type StatusInput struct {
	MissionID     string `json:"mission_id"`
	WorkflowRunID string `json:"workflow_run_id"`
}

// StopInput is the complete plasma.workflow.stop wire payload.
type StopInput struct {
	MissionID     string `json:"mission_id"`
	WorkflowRunID string `json:"workflow_run_id"`
	Reason        string `json:"reason"`
}
