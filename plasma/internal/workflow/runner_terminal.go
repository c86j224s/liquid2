package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

func (runner Runner) appendAgentError(ctx context.Context, view workflowstate.WorkflowRunView, userEventID string, stepID string, toolSessionID string, previousSessionID string, result AgentResult, durationMS int64, cause error, extra ...map[string]any) (ledger.Event, error) {
	extraPayload := map[string]any{
		"workflow_run_id":           view.WorkflowRunID,
		"workflow_step_id":          stepID,
		"tool_session_id":           toolSessionID,
		"previous_agent_session_id": previousSessionID,
	}
	for _, fields := range extra {
		for key, value := range fields {
			if value != nil {
				extraPayload[key] = value
			}
		}
	}
	compactionAttempted, _ := extraPayload["compaction_attempted"].(bool)
	return runner.Service.AppendEvent(ctx, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
		EventID:                runner.newID("evt"),
		MissionID:              view.MissionID,
		Kind:                   "agent_error",
		AgentExecutor:          view.AgentExecutor,
		AgentModel:             strings.TrimSpace(runner.AgentModel),
		AgentReasoningEffort:   strings.TrimSpace(runner.ReasoningEffort),
		IncludeAgentConfig:     true,
		MCPMode:                view.MCPMode,
		IncludeMCPMode:         true,
		Text:                   "워크플로우 단계의 에이전트 실행이 실패했습니다.",
		Error:                  cause.Error(),
		IncludeError:           true,
		LogExcerpt:             headTailExcerpt(result.Log, 4000),
		IncludeLogExcerpt:      true,
		AgentSessionID:         strings.TrimSpace(result.SessionID),
		IncludeAgentSessionID:  strings.TrimSpace(result.SessionID) != "",
		DurationMS:             durationMS,
		IncludeDuration:        true,
		UserEventID:            userEventID,
		Extra:                  extraPayload,
		Usage:                  result.Usage,
		UsageSurface:           "workflow_step",
		UsagePreviousSessionID: previousSessionID,
		UsageCompaction:        compactionAttempted,
		Producer:               ledger.Producer{Type: "agent", ID: view.AgentExecutor},
	}))
}

func (runner Runner) terminal(ctx context.Context, missionID string, workflowRunID string, eventType string, reason string, errorText string) (workflowstate.WorkflowRunView, error) {
	return runner.terminalWithNextInstruction(ctx, missionID, workflowRunID, eventType, reason, errorText, "")
}

func (runner Runner) terminalWithNextInstruction(ctx context.Context, missionID string, workflowRunID string, eventType string, reason string, errorText string, nextInstruction string) (workflowstate.WorkflowRunView, error) {
	view, _ := runner.Service.GetWorkflowRun(ctx, missionID, workflowRunID)
	payload := workflowstate.WorkflowRunTerminalPayload{
		WorkflowRunID:      workflowRunID,
		MissionID:          missionID,
		Reason:             reason,
		Error:              errorText,
		NextInstruction:    strings.TrimSpace(nextInstruction),
		CompletedStepCount: view.CompletedStepCount,
		TerminalAt:         runner.now().Format(time.RFC3339Nano),
	}
	if eventType == workflowstate.WorkflowRunStoppedEvent {
		payload.StopReason = reason
	}
	if _, err := runner.appendWorkflowEvent(ctx, missionID, eventType, payload, ledger.Producer{Type: "workflow", ID: workflowRunID}); err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	return runner.Service.GetWorkflowRun(ctx, missionID, workflowRunID)
}

func (runner Runner) limitReached(ctx context.Context, missionID string, workflowRunID string, view workflowstate.WorkflowRunView, reason string) (workflowstate.WorkflowRunView, error) {
	nextInstruction, ok := latestContinuationInstruction(view)
	if ok {
		return runner.terminalWithNextInstruction(ctx, missionID, workflowRunID, workflowstate.WorkflowRunPausedEvent, reason, "", nextInstruction)
	}
	return runner.terminal(ctx, missionID, workflowRunID, workflowstate.WorkflowRunCompletedEvent, reason, "")
}

// ParseControlDecision separates the visible response from the final bounded
// workflow control marker.
func ParseControlDecision(text string) (string, ControlDecision, bool) {
	index := strings.LastIndex(text, controlMarker)
	if index < 0 {
		return strings.TrimSpace(text), ControlDecision{}, false
	}
	visible := strings.TrimSpace(text[:index])
	controlText := strings.TrimSpace(text[index+len(controlMarker):])
	var decision ControlDecision
	if err := json.Unmarshal([]byte(controlText), &decision); err != nil {
		return visible, ControlDecision{}, false
	}
	decision.Decision = strings.TrimSpace(strings.ToLower(decision.Decision))
	decision.Reason = strings.TrimSpace(decision.Reason)
	decision.NextInstruction = strings.TrimSpace(decision.NextInstruction)
	switch decision.Decision {
	case "continue", "stop":
	default:
		return visible, ControlDecision{}, false
	}
	if visible == "" {
		visible = "워크플로우 단계가 사용자에게 보여줄 별도 결과 없이 control decision만 반환했습니다."
	}
	return visible, decision, true
}

// VisibleTextBeforeControl returns the user-facing prefix without exposing a
// complete or partial workflow control envelope.
func VisibleTextBeforeControl(text string) string {
	index := strings.Index(text, controlMarker)
	if index >= 0 {
		return strings.TrimSpace(text[:index])
	}
	for prefixLength := len(controlMarker) - 1; prefixLength > 0; prefixLength-- {
		prefix := controlMarker[:prefixLength]
		if strings.HasSuffix(text, prefix) {
			return strings.TrimSpace(text[:len(text)-prefixLength])
		}
	}
	return strings.TrimSpace(text)
}
