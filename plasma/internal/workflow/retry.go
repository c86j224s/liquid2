package workflow

import (
	"context"
	"errors"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

func (runner Runner) retryStepAfterAutoCompaction(agentCtx context.Context, ledgerCtx context.Context, view workflowstate.WorkflowRunView, userEventID string, stepID string, toolSessionID string, instruction string, prompt string, previousSessionID string, initialErr error, initialResult AgentResult, initialDurationMS int64) (AgentResult, int64, int64, string, bool, error) {
	compactStarted := runner.now()
	compactResult, err := runner.Agent.Run(agentCtx, AgentRequest{
		UserText:          "compact session context",
		Prompt:            workflowCompactPrompt(),
		Model:             runner.AgentModel,
		ReasoningEffort:   runner.ReasoningEffort,
		MissionID:         view.MissionID,
		ToolSessionID:     toolSessionID,
		UserEventID:       userEventID,
		PreviousSessionID: previousSessionID,
		AgentExecutor:     view.AgentExecutor,
		MCPMode:           view.MCPMode,
		Compaction:        true,
	})
	err = agentExecutionError(agentCtx, err)
	compactDurationMS := runner.now().Sub(compactStarted).Milliseconds()
	if err != nil {
		if agentCtx.Err() != nil || errors.Is(err, context.Canceled) {
			return AgentResult{}, 0, 0, "", false, err
		}
		_, appendErr := runner.appendAgentError(ledgerCtx, view, userEventID, stepID, toolSessionID, previousSessionID, compactResult, initialDurationMS+compactDurationMS, err, map[string]any{
			"compaction_attempted": true,
			"original_error":       initialErr.Error(),
			"original_log_excerpt": headTailExcerpt(initialResult.Log, 2000),
			"text":                 "워크플로우 단계에서 에이전트 컨텍스트가 가득 차 자동 압축을 시도했지만 실패했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
		return AgentResult{}, 0, 0, "", appendErr == nil, err
	}
	returnedCompactSessionID := strings.TrimSpace(compactResult.SessionID)
	compactResult, err = validatedSameSessionResult(compactResult, previousSessionID)
	if err != nil {
		_, appendErr := runner.appendAgentError(ledgerCtx, view, userEventID, stepID, toolSessionID, previousSessionID, compactResult, initialDurationMS+compactDurationMS, err, map[string]any{
			"compaction_attempted":      true,
			"original_error":            initialErr.Error(),
			"returned_agent_session_id": returnedCompactSessionID,
			"text":                      "워크플로우 단계의 자동 압축 요청에서 에이전트가 다른 세션 ID를 반환했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
		return AgentResult{}, 0, 0, "", appendErr == nil, err
	}
	compactEvent, err := runner.Service.AppendEvent(ledgerCtx, conversation.BuildTurnAgentCompactedAppendRequest(conversation.TurnAgentCompactedEventRequest{
		EventID:                runner.newID("evt"),
		MissionID:              view.MissionID,
		AgentExecutor:          view.AgentExecutor,
		AgentModel:             strings.TrimSpace(runner.AgentModel),
		AgentReasoningEffort:   strings.TrimSpace(runner.ReasoningEffort),
		MCPMode:                view.MCPMode,
		AgentSessionID:         compactResult.SessionID,
		PreviousAgentSessionID: previousSessionID,
		WorkflowRunID:          view.WorkflowRunID,
		WorkflowStepID:         stepID,
		ToolSessionID:          toolSessionID,
		Summary:                compactResult.Text,
		DurationMS:             compactDurationMS,
		UserEventID:            userEventID,
		Manual:                 false,
		Reason:                 "context_window_exhausted",
		Usage:                  compactResult.Usage,
		Resumed:                compactResult.Resumed,
		Producer:               ledger.Producer{Type: "agent", ID: view.AgentExecutor},
	}))
	if err != nil {
		return AgentResult{}, 0, 0, "", false, err
	}

	retryStarted := runner.now()
	result, err := runner.Agent.Run(agentCtx, AgentRequest{
		UserText:          instruction,
		Prompt:            prompt,
		Model:             runner.AgentModel,
		ReasoningEffort:   runner.ReasoningEffort,
		MissionID:         view.MissionID,
		ToolSessionID:     toolSessionID,
		UserEventID:       userEventID,
		PreviousSessionID: previousSessionID,
		AgentExecutor:     view.AgentExecutor,
		MCPMode:           view.MCPMode,
	})
	err = agentExecutionError(agentCtx, err)
	retryDurationMS := runner.now().Sub(retryStarted).Milliseconds()
	totalDurationMS := initialDurationMS + compactDurationMS + retryDurationMS
	if err != nil {
		if agentCtx.Err() != nil || errors.Is(err, context.Canceled) {
			return AgentResult{}, 0, 0, "", false, err
		}
		_, appendErr := runner.appendAgentError(ledgerCtx, view, userEventID, stepID, toolSessionID, previousSessionID, result, retryDurationMS, err, map[string]any{
			"compaction_attempted": true,
			"compaction_event_id":  compactEvent.EventID,
			"original_error":       initialErr.Error(),
			"total_duration_ms":    totalDurationMS,
			"text":                 "워크플로우 단계에서 같은 세션을 자동 압축한 뒤 재시도했지만 실패했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
		return AgentResult{}, 0, 0, "", appendErr == nil, err
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validatedSameSessionResult(result, previousSessionID)
	if err != nil {
		_, appendErr := runner.appendAgentError(ledgerCtx, view, userEventID, stepID, toolSessionID, previousSessionID, result, retryDurationMS, err, map[string]any{
			"compaction_attempted":      true,
			"compaction_event_id":       compactEvent.EventID,
			"returned_agent_session_id": returnedSessionID,
			"total_duration_ms":         totalDurationMS,
			"text":                      "워크플로우 단계의 자동 압축 후 재시도에서 에이전트가 다른 세션 ID를 반환했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
		return AgentResult{}, 0, 0, "", appendErr == nil, err
	}
	return result, retryDurationMS, totalDurationMS, compactEvent.EventID, false, nil
}
