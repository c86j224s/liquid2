package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

const proactiveContextCompactionThresholdPercent = 55

type contextCompactionTrigger struct {
	EventID     string
	UserEventID string
	Metrics     agentusage.ContextWindowMetrics
}

// compactBeforeNextStep proactively compacts a Codex research session before
// a new durable workflow step is opened. The trigger event makes the attempt
// idempotent across runner restarts.
func (runner Runner) compactBeforeNextStep(ctx context.Context, view workflowstate.WorkflowRunView) (bool, error) {
	events, err := runner.Service.ListEvents(ctx, view.MissionID)
	if err != nil {
		return false, err
	}
	previousSessionID := LatestAgentSessionID(events, view.AgentExecutor)
	trigger, ok := latestContextCompactionTrigger(events, view.AgentExecutor, previousSessionID)
	if !ok || !trigger.Metrics.AtOrAbovePercent(proactiveContextCompactionThresholdPercent) {
		return false, nil
	}
	if proactiveCompactionRecorded(events, trigger.EventID) {
		return false, nil
	}
	if !strings.EqualFold(strings.TrimSpace(view.AgentExecutor), "codex") {
		return false, nil
	}

	toolSessionID := runner.newID("ses")
	started := runner.now()
	agentCtx, cancel := context.WithTimeout(ctx, runner.stepTimeout())
	defer cancel()
	result, err := runner.Agent.Run(agentCtx, AgentRequest{
		UserText:          "compact session context",
		Prompt:            workflowCompactPrompt(),
		Model:             runner.AgentModel,
		ReasoningEffort:   runner.ReasoningEffort,
		MissionID:         view.MissionID,
		ToolSessionID:     toolSessionID,
		UserEventID:       trigger.UserEventID,
		PreviousSessionID: previousSessionID,
		AgentExecutor:     view.AgentExecutor,
		MCPMode:           view.MCPMode,
		Compaction:        true,
	})
	err = agentExecutionError(agentCtx, err)
	if err != nil {
		return false, fmt.Errorf("proactive context compaction failed: %w", err)
	}
	result, err = validatedSameSessionResult(result, previousSessionID)
	if err != nil {
		return false, fmt.Errorf("proactive context compaction changed session: %w", err)
	}
	durationMS := runner.now().Sub(started).Milliseconds()
	_, err = runner.Service.AppendEvent(ctx, conversation.BuildTurnAgentCompactedAppendRequest(conversation.TurnAgentCompactedEventRequest{
		EventID:                 runner.newID("evt"),
		MissionID:               view.MissionID,
		AgentExecutor:           view.AgentExecutor,
		AgentModel:              strings.TrimSpace(runner.AgentModel),
		AgentReasoningEffort:    strings.TrimSpace(runner.ReasoningEffort),
		MCPMode:                 view.MCPMode,
		AgentSessionID:          result.SessionID,
		PreviousAgentSessionID:  previousSessionID,
		WorkflowRunID:           view.WorkflowRunID,
		ToolSessionID:           toolSessionID,
		Summary:                 result.Text,
		DurationMS:              durationMS,
		UserEventID:             trigger.UserEventID,
		Manual:                  false,
		Reason:                  "context_window_threshold",
		ContextTriggerEventID:   trigger.EventID,
		ContextUsedTokens:       trigger.Metrics.UsedTokens,
		ContextWindowTokens:     trigger.Metrics.WindowTokens,
		ContextThresholdPercent: proactiveContextCompactionThresholdPercent,
		Usage:                   result.Usage,
		Resumed:                 result.Resumed,
		Producer:                ledger.Producer{Type: "agent", ID: view.AgentExecutor},
	}))
	if err != nil {
		return false, err
	}
	return true, nil
}

func latestContextCompactionTrigger(events []ledger.Event, executorName string, sessionID string) (contextCompactionTrigger, bool) {
	sessionID = strings.TrimSpace(sessionID)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "turn.agent.response" {
			continue
		}
		var payload struct {
			Kind           string                `json:"kind"`
			AgentExecutor  string                `json:"agent_executor"`
			AgentSessionID string                `json:"agent_session_id"`
			UserEventID    string                `json:"user_event_id"`
			AgentUsage     agentusage.AgentUsage `json:"agent_usage"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil || payload.Kind != "agent_response" {
			continue
		}
		if !conversation.AgentEventMatchesExecutor(payload.AgentExecutor, executorName) || strings.TrimSpace(payload.AgentSessionID) != sessionID {
			continue
		}
		if payload.AgentUsage.ContextWindow == nil || !payload.AgentUsage.ContextWindow.Valid() {
			return contextCompactionTrigger{}, false
		}
		return contextCompactionTrigger{
			EventID:     event.EventID,
			UserEventID: strings.TrimSpace(payload.UserEventID),
			Metrics:     *payload.AgentUsage.ContextWindow,
		}, true
	}
	return contextCompactionTrigger{}, false
}

func proactiveCompactionRecorded(events []ledger.Event, triggerEventID string) bool {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].EventType != "turn.agent.compacted" {
			continue
		}
		var payload struct {
			Reason                string `json:"reason"`
			ContextTriggerEventID string `json:"context_trigger_event_id"`
		}
		if json.Unmarshal(events[i].Payload, &payload) == nil &&
			payload.Reason == "context_window_threshold" &&
			payload.ContextTriggerEventID == triggerEventID {
			return true
		}
	}
	return false
}
