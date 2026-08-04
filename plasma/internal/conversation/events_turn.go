package conversation

import (
	"encoding/json"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// TurnUserEventRequest는 대화 이벤트 경계에 전달되는 요청 값이다.
type TurnUserEventRequest struct {
	EventID              string
	MissionID            string
	Kind                 string
	Text                 string
	AgentExecutor        string
	AgentModel           string
	AgentReasoningEffort string
	IncludeAgentConfig   bool
	MCPMode              string
	ToolSessionID        string
	WorkflowRunID        string
	WorkflowStepID       string
	StepInstructionMode  string
	Producer             app.Producer
}

// ControllerStrategySelectedEventRequest는 대화 이벤트 경계에 전달되는 요청 값이다.
type ControllerStrategySelectedEventRequest struct {
	EventID           string
	MissionID         string
	StrategyID        string
	StrategyLabel     string
	Reason            string
	Guidance          string
	RequestedStrategy string
	AgentExecutor     string
	MCPMode           string
	UserEventID       string
	ToolSessionID     string
	PreviousSessionID string
	Producer          app.Producer
}

// TurnAgentPendingEventRequest는 대화 이벤트 경계에 전달되는 요청 값이다.
type TurnAgentPendingEventRequest struct {
	EventID              string
	MissionID            string
	AgentExecutor        string
	AgentModel           string
	AgentReasoningEffort string
	IncludeAgentConfig   bool
	MCPMode              string
	StrategyID           string
	IncludeStrategyID    bool
	Text                 string
	UserEventID          string
	ToolSessionID        string
	StartedAt            string
	WorkflowRunID        string
	WorkflowStepID       string
	StepInstructionMode  string
	Producer             app.Producer
}

// AgentSessionResetEventRequest는 대화 이벤트 경계에 전달되는 요청 값이다.
type AgentSessionResetEventRequest struct {
	EventID                string
	MissionID              string
	AgentExecutor          string
	AgentModel             string
	AgentReasoningEffort   string
	PreviousAgentSessionID string
	Producer               app.Producer
}

// BuildTurnUserAppendRequest는 대화 이벤트 경계에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildTurnUserAppendRequest(req TurnUserEventRequest) app.AppendEventRequest {
	payload := map[string]any{
		"kind":            req.Kind,
		"text":            req.Text,
		"agent_executor":  req.AgentExecutor,
		"mcp_mode":        req.MCPMode,
		"tool_session_id": req.ToolSessionID,
	}
	putOptionalString(payload, "agent_model", req.AgentModel, req.IncludeAgentConfig)
	putOptionalString(payload, "agent_reasoning_effort", req.AgentReasoningEffort, req.IncludeAgentConfig)
	putNonEmpty(payload, "workflow_run_id", req.WorkflowRunID)
	putNonEmpty(payload, "workflow_step_id", req.WorkflowStepID)
	putNonEmpty(payload, "step_instruction_mode", req.StepInstructionMode)
	return app.AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "turn.user",
		Producer:  req.Producer,
		Payload:   mustMarshalJSON(payload),
	}
}

// BuildControllerStrategySelectedAppendRequest는 대화 이벤트 경계에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildControllerStrategySelectedAppendRequest(req ControllerStrategySelectedEventRequest) app.AppendEventRequest {
	return app.AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "controller.strategy.selected",
		Producer:  req.Producer,
		Payload: mustMarshalJSON(map[string]any{
			"kind":                "controller_strategy_selected",
			"strategy_id":         req.StrategyID,
			"strategy_label":      req.StrategyLabel,
			"reason":              req.Reason,
			"guidance":            req.Guidance,
			"requested_strategy":  req.RequestedStrategy,
			"agent_executor":      req.AgentExecutor,
			"mcp_mode":            req.MCPMode,
			"user_event_id":       req.UserEventID,
			"tool_session_id":     req.ToolSessionID,
			"previous_session_id": req.PreviousSessionID,
		}),
	}
}

// BuildTurnAgentPendingAppendRequest는 대화 이벤트 경계에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildTurnAgentPendingAppendRequest(req TurnAgentPendingEventRequest) app.AppendEventRequest {
	payload := map[string]any{
		"kind":            "agent_pending",
		"agent_executor":  req.AgentExecutor,
		"mcp_mode":        req.MCPMode,
		"text":            req.Text,
		"user_event_id":   req.UserEventID,
		"tool_session_id": req.ToolSessionID,
		"started_at":      req.StartedAt,
	}
	putOptionalString(payload, "agent_model", req.AgentModel, req.IncludeAgentConfig)
	putOptionalString(payload, "agent_reasoning_effort", req.AgentReasoningEffort, req.IncludeAgentConfig)
	putOptionalString(payload, "strategy_id", req.StrategyID, req.IncludeStrategyID)
	putNonEmpty(payload, "workflow_run_id", req.WorkflowRunID)
	putNonEmpty(payload, "workflow_step_id", req.WorkflowStepID)
	putNonEmpty(payload, "step_instruction_mode", req.StepInstructionMode)
	return app.AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "turn.agent.pending",
		Producer:  req.Producer,
		Payload:   mustMarshalJSON(payload),
	}
}

// BuildAgentSessionResetAppendRequest는 대화 이벤트 경계에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildAgentSessionResetAppendRequest(req AgentSessionResetEventRequest) app.AppendEventRequest {
	return app.AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "agent.session.reset",
		Producer:  req.Producer,
		Payload: mustMarshalJSON(map[string]any{
			"kind":                      "agent_session_reset",
			"agent_executor":            req.AgentExecutor,
			"agent_model":               req.AgentModel,
			"agent_reasoning_effort":    req.AgentReasoningEffort,
			"previous_agent_session_id": req.PreviousAgentSessionID,
			"text":                      "사용자가 에이전트 세션을 새로 시작하도록 요청했습니다.",
		}),
	}
}

func putNonEmpty(payload map[string]any, key string, value string) {
	if value != "" {
		payload[key] = value
	}
}

func putOptionalString(payload map[string]any, key string, value string, includeEmpty bool) {
	if includeEmpty || value != "" {
		payload[key] = value
	}
}

func mustMarshalJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("marshal conversation event payload: %v", err))
	}
	return data
}
