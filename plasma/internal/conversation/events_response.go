package conversation

import (
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// TurnAgentResponseEventRequest는 대화 이벤트 경계에 전달되는 요청 값이다.
type TurnAgentResponseEventRequest struct {
	EventID                string
	MissionID              string
	Kind                   string
	AgentExecutor          string
	AgentModel             string
	AgentReasoningEffort   string
	IncludeAgentConfig     bool
	MCPMode                string
	IncludeMCPMode         bool
	Text                   string
	Error                  string
	IncludeError           bool
	LogExcerpt             string
	IncludeLogExcerpt      bool
	AgentSessionID         string
	IncludeAgentSessionID  bool
	Resumed                bool
	IncludeResumed         bool
	DurationMS             int64
	IncludeDuration        bool
	UserEventID            string
	Extra                  map[string]any
	Usage                  agentusage.AgentUsage
	UsageSurface           string
	UsagePreviousSessionID string
	UsageCompaction        bool
	Producer               app.Producer
}

// TurnAgentCompactedEventRequest는 대화 이벤트 경계에 전달되는 요청 값이다.
type TurnAgentCompactedEventRequest struct {
	EventID                 string
	MissionID               string
	AgentExecutor           string
	AgentModel              string
	AgentReasoningEffort    string
	MCPMode                 string
	AgentSessionID          string
	PreviousAgentSessionID  string
	WorkflowRunID           string
	WorkflowStepID          string
	ToolSessionID           string
	Summary                 string
	DurationMS              int64
	UserEventID             string
	Manual                  bool
	Reason                  string
	ContextTriggerEventID   string
	ContextUsedTokens       int
	ContextWindowTokens     int
	ContextThresholdPercent int
	Usage                   agentusage.AgentUsage
	Resumed                 bool
	Producer                app.Producer
}

// BuildTurnAgentResponseAppendRequest는 대화 이벤트 경계에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildTurnAgentResponseAppendRequest(req TurnAgentResponseEventRequest) app.AppendEventRequest {
	payload := map[string]any{
		"kind":           req.Kind,
		"agent_executor": req.AgentExecutor,
		"text":           req.Text,
		"user_event_id":  req.UserEventID,
	}
	putOptionalString(payload, "mcp_mode", req.MCPMode, req.IncludeMCPMode)
	putOptionalString(payload, "agent_model", req.AgentModel, req.IncludeAgentConfig)
	putOptionalString(payload, "agent_reasoning_effort", req.AgentReasoningEffort, req.IncludeAgentConfig)
	putOptionalString(payload, "agent_session_id", req.AgentSessionID, req.IncludeAgentSessionID)
	if req.IncludeResumed {
		payload["resumed"] = req.Resumed
	}
	if req.IncludeDuration {
		payload["duration_ms"] = req.DurationMS
	}
	putOptionalString(payload, "error", req.Error, req.IncludeError)
	putOptionalString(payload, "log_excerpt", req.LogExcerpt, req.IncludeLogExcerpt)
	for key, value := range req.Extra {
		payload[key] = value
	}
	addUsagePayload(payload, req.Usage, req.UsageSurface, req.DurationMS, req.UsagePreviousSessionID, req.AgentSessionID, req.Resumed, req.UsageCompaction)
	return app.AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "turn.agent.response",
		Producer:  req.Producer,
		Payload:   mustMarshalJSON(payload),
	}
}

// BuildTurnAgentCompactedAppendRequest는 대화 이벤트 경계에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildTurnAgentCompactedAppendRequest(req TurnAgentCompactedEventRequest) app.AppendEventRequest {
	payload := map[string]any{
		"kind":                      "agent_session_compacted",
		"agent_executor":            req.AgentExecutor,
		"agent_model":               req.AgentModel,
		"agent_reasoning_effort":    req.AgentReasoningEffort,
		"mcp_mode":                  req.MCPMode,
		"agent_session_id":          req.AgentSessionID,
		"previous_agent_session_id": req.PreviousAgentSessionID,
		"tool_session_id":           req.ToolSessionID,
		"summary":                   req.Summary,
		"duration_ms":               req.DurationMS,
		"user_event_id":             req.UserEventID,
		"manual":                    req.Manual,
	}
	putNonEmpty(payload, "workflow_run_id", req.WorkflowRunID)
	putNonEmpty(payload, "workflow_step_id", req.WorkflowStepID)
	putNonEmpty(payload, "reason", req.Reason)
	putNonEmpty(payload, "context_trigger_event_id", req.ContextTriggerEventID)
	if req.ContextUsedTokens > 0 {
		payload["context_used_tokens"] = req.ContextUsedTokens
	}
	if req.ContextWindowTokens > 0 {
		payload["context_window_tokens"] = req.ContextWindowTokens
	}
	if req.ContextThresholdPercent > 0 {
		payload["context_threshold_percent"] = req.ContextThresholdPercent
	}
	addUsagePayload(payload, req.Usage, "compaction", req.DurationMS, req.PreviousAgentSessionID, req.AgentSessionID, req.Resumed, true)
	return app.AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "turn.agent.compacted",
		Producer:  req.Producer,
		Payload:   mustMarshalJSON(payload),
	}
}

func addUsagePayload(payload map[string]any, usage agentusage.AgentUsage, surface string, durationMS int64, previousSessionID string, sessionID string, resumed bool, compaction bool) {
	if eventUsage, ok := usage.ForEvent(surface, durationMS, previousSessionID, sessionID, resumed, compaction); ok {
		payload["agent_usage"] = eventUsage
	}
}
