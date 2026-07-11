package conversation

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const isolatedForkReportSessionPolicy = "isolated_fork"

type OpenAgentPending struct {
	UserEventID    string
	AgentExecutor  string
	WorkflowRunID  string
	WorkflowStepID string
}

func LatestAgentSessionID(events []app.LedgerEvent, executorName string) string {
	latestOrder := int64(-1)
	latestSessionID := ""
	for i, event := range events {
		if event.EventType != "turn.agent.response" && event.EventType != "report.artifact.created" && event.EventType != "agent.session.reset" {
			continue
		}
		var payload struct {
			AgentSessionID             string `json:"agent_session_id"`
			AgentExecutor              string `json:"agent_executor"`
			Kind                       string `json:"kind"`
			ReportSessionPolicy        string `json:"report_session_policy"`
			PreReportResearchSessionID string `json:"pre_report_research_session_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if !AgentEventMatchesExecutor(payload.AgentExecutor, executorName) {
			continue
		}
		if event.EventType == "turn.agent.response" && strings.TrimSpace(payload.Kind) != "agent_response" {
			continue
		}
		order := event.Sequence
		if order == 0 {
			order = int64(i + 1)
		}
		if order < latestOrder {
			continue
		}
		if event.EventType == "agent.session.reset" {
			latestOrder = order
			latestSessionID = ""
			continue
		}
		if event.EventType == "report.artifact.created" && isIsolatedForkReportSessionPolicy(payload.ReportSessionPolicy) {
			preReportSessionID := strings.TrimSpace(payload.PreReportResearchSessionID)
			if preReportSessionID == "" {
				continue
			}
			latestOrder = order
			latestSessionID = preReportSessionID
			continue
		}
		if strings.TrimSpace(payload.AgentSessionID) != "" {
			latestOrder = order
			latestSessionID = strings.TrimSpace(payload.AgentSessionID)
		}
	}
	return latestSessionID
}

func LatestAgentModel(events []app.LedgerEvent, executorName string) string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].EventType != "agent.session.reset" && events[i].EventType != "turn.agent.response" {
			continue
		}
		var payload struct {
			AgentExecutor string `json:"agent_executor"`
			AgentModel    string `json:"agent_model"`
			Kind          string `json:"kind"`
		}
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			continue
		}
		if !AgentEventMatchesExecutor(payload.AgentExecutor, executorName) {
			continue
		}
		if events[i].EventType == "agent.session.reset" {
			return strings.TrimSpace(payload.AgentModel)
		}
		if strings.TrimSpace(payload.Kind) == "agent_response" && strings.TrimSpace(payload.AgentModel) != "" {
			return strings.TrimSpace(payload.AgentModel)
		}
	}
	return ""
}

func LatestAgentReasoningEffort(events []app.LedgerEvent, executorName string) string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].EventType != "agent.session.reset" && events[i].EventType != "turn.agent.response" {
			continue
		}
		var payload struct {
			AgentExecutor        string `json:"agent_executor"`
			AgentReasoningEffort string `json:"agent_reasoning_effort"`
			Kind                 string `json:"kind"`
		}
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			continue
		}
		if !AgentEventMatchesExecutor(payload.AgentExecutor, executorName) {
			continue
		}
		if events[i].EventType == "agent.session.reset" {
			return strings.TrimSpace(payload.AgentReasoningEffort)
		}
		if strings.TrimSpace(payload.Kind) == "agent_response" && strings.TrimSpace(payload.AgentReasoningEffort) != "" {
			return strings.TrimSpace(payload.AgentReasoningEffort)
		}
	}
	return ""
}

func LatestOpenAgentPending(events []app.LedgerEvent, workflowRunID string) (OpenAgentPending, bool) {
	completed := CompletedUserEventIDs(events)
	workflowRunID = strings.TrimSpace(workflowRunID)
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].EventType != "turn.agent.pending" {
			continue
		}
		var payload struct {
			UserEventID    string `json:"user_event_id"`
			AgentExecutor  string `json:"agent_executor"`
			WorkflowRunID  string `json:"workflow_run_id"`
			WorkflowStepID string `json:"workflow_step_id"`
		}
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			continue
		}
		pendingWorkflowRunID := strings.TrimSpace(payload.WorkflowRunID)
		if workflowRunID != "" && pendingWorkflowRunID != workflowRunID {
			continue
		}
		userEventID := strings.TrimSpace(payload.UserEventID)
		if userEventID == "" {
			continue
		}
		if _, ok := completed[userEventID]; ok {
			continue
		}
		return OpenAgentPending{
			UserEventID:    userEventID,
			AgentExecutor:  defaultAgentExecutor(payload.AgentExecutor),
			WorkflowRunID:  pendingWorkflowRunID,
			WorkflowStepID: strings.TrimSpace(payload.WorkflowStepID),
		}, true
	}
	return OpenAgentPending{}, false
}

func AgentPendingForUserEvent(events []app.LedgerEvent, userEventID string) (OpenAgentPending, bool) {
	userEventID = strings.TrimSpace(userEventID)
	if userEventID == "" {
		return OpenAgentPending{}, false
	}
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].EventType != "turn.agent.pending" {
			continue
		}
		var payload struct {
			UserEventID    string `json:"user_event_id"`
			AgentExecutor  string `json:"agent_executor"`
			WorkflowRunID  string `json:"workflow_run_id"`
			WorkflowStepID string `json:"workflow_step_id"`
		}
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.UserEventID) != userEventID {
			continue
		}
		return OpenAgentPending{
			UserEventID:    userEventID,
			AgentExecutor:  defaultAgentExecutor(payload.AgentExecutor),
			WorkflowRunID:  strings.TrimSpace(payload.WorkflowRunID),
			WorkflowStepID: strings.TrimSpace(payload.WorkflowStepID),
		}, true
	}
	return OpenAgentPending{}, false
}

func HasOpenAgentPending(events []app.LedgerEvent) bool {
	_, ok := LatestOpenAgentPending(events, "")
	return ok
}

func HasAgentTerminalEventForUser(events []app.LedgerEvent, userEventID string) bool {
	userEventID = strings.TrimSpace(userEventID)
	if userEventID == "" {
		return false
	}
	_, ok := CompletedUserEventIDs(events)[userEventID]
	return ok
}

func CompletedUserEventIDs(events []app.LedgerEvent) map[string]struct{} {
	completed := map[string]struct{}{}
	for _, event := range events {
		if event.EventType != "turn.agent.response" {
			continue
		}
		var payload struct {
			UserEventID string `json:"user_event_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		userEventID := strings.TrimSpace(payload.UserEventID)
		if userEventID != "" {
			completed[userEventID] = struct{}{}
		}
	}
	return completed
}

func AgentEventMatchesExecutor(eventExecutor string, executorName string) bool {
	eventExecutor = strings.TrimSpace(eventExecutor)
	executorName = strings.TrimSpace(executorName)
	if eventExecutor == "" {
		return executorName == "codex"
	}
	return eventExecutor == executorName
}

func defaultAgentExecutor(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "codex"
	}
	return value
}

func isIsolatedForkReportSessionPolicy(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case isolatedForkReportSessionPolicy, "isolated-fork", "fork":
		return true
	default:
		return false
	}
}
