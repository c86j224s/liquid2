package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// runStep은 workflow 한 단계를 원장에 시작으로 기록하고 에이전트 호출, control
// decision 검증, 단계 완료 이벤트 기록까지 처리한다. 단계 실패는 caller가 run
// terminal 실패로 승격한다.
func (runner Runner) runStep(ctx context.Context, view workflowstate.WorkflowRunView) (workflowstate.WorkflowRunView, error) {
	stepID := runner.newID("wfs")
	toolSessionID := runner.newID("ses")
	instruction := nextInstruction(view)
	stepIndex := view.CompletedStepCount + 1
	stepStartedAt := runner.now()
	if err := runner.appendRemovedSourceSkips(ctx, view, stepID, stepIndex); err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	stepEvent, err := runner.appendWorkflowEvent(ctx, view.MissionID, workflowstate.WorkflowStepStartedEvent, workflowstate.WorkflowStepStartedPayload{
		WorkflowRunID:  view.WorkflowRunID,
		MissionID:      view.MissionID,
		WorkflowStepID: stepID,
		Instruction:    instruction,
		StepIndex:      stepIndex,
		StartedAt:      stepStartedAt.Format(time.RFC3339Nano),
		ToolSessionID:  toolSessionID,
	}, ledger.Producer{Type: "workflow", ID: view.WorkflowRunID})
	if err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	userEventReq := conversation.BuildTurnUserAppendRequest(conversation.TurnUserEventRequest{
		EventID:              runner.newID("evt"),
		MissionID:            view.MissionID,
		Kind:                 "workflow_steering",
		Text:                 instruction,
		AgentExecutor:        view.AgentExecutor,
		AgentModel:           strings.TrimSpace(runner.AgentModel),
		AgentReasoningEffort: strings.TrimSpace(runner.ReasoningEffort),
		IncludeAgentConfig:   true,
		MCPMode:              view.MCPMode,
		ToolSessionID:        toolSessionID,
		WorkflowRunID:        view.WorkflowRunID,
		WorkflowStepID:       stepID,
		StepInstructionMode:  view.StepInstructionMode,
		Producer:             ledger.Producer{Type: "workflow", ID: view.WorkflowRunID},
	})
	userEvent, err := runner.Service.AppendEvent(ctx, userEventReq)
	if err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	if _, err := runner.Service.AppendEvent(ctx, conversation.BuildTurnAgentPendingAppendRequest(conversation.TurnAgentPendingEventRequest{
		EventID:              runner.newID("evt"),
		MissionID:            view.MissionID,
		AgentExecutor:        view.AgentExecutor,
		AgentModel:           strings.TrimSpace(runner.AgentModel),
		AgentReasoningEffort: strings.TrimSpace(runner.ReasoningEffort),
		IncludeAgentConfig:   true,
		MCPMode:              view.MCPMode,
		Text:                 "워크플로우 단계의 에이전트 응답을 기다리는 중입니다.",
		UserEventID:          userEvent.EventID,
		WorkflowRunID:        view.WorkflowRunID,
		WorkflowStepID:       stepID,
		StepInstructionMode:  view.StepInstructionMode,
		ToolSessionID:        toolSessionID,
		StartedAt:            runner.now().Format(time.RFC3339Nano),
		Producer:             ledger.Producer{Type: "agent", ID: view.AgentExecutor},
	})); err != nil {
		return workflowstate.WorkflowRunView{}, err
	}

	events, err := runner.Service.ListEvents(ctx, view.MissionID)
	if err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	previousSessionID := LatestAgentSessionID(events, view.AgentExecutor)
	started := runner.now()
	prompt := StepPrompt(view, instruction, toolSessionID, previousSessionID != "")
	agentCtx, cancelAgent := context.WithTimeout(ctx, runner.stepTimeout())
	defer cancelAgent()
	result, err := runner.Agent.Run(agentCtx, AgentRequest{
		UserText:          instruction,
		Prompt:            prompt,
		Model:             runner.AgentModel,
		ReasoningEffort:   runner.ReasoningEffort,
		MissionID:         view.MissionID,
		ToolSessionID:     toolSessionID,
		UserEventID:       userEvent.EventID,
		PreviousSessionID: previousSessionID,
		AgentExecutor:     view.AgentExecutor,
		MCPMode:           view.MCPMode,
	})
	err = agentExecutionError(agentCtx, err)
	durationMS := runner.now().Sub(started).Milliseconds()
	compactionAttempted := false
	compactionEventID := ""
	totalDurationMS := int64(0)
	if err != nil {
		if ctx.Err() == nil && agentCtx.Err() == nil && !errors.Is(err, context.Canceled) && shouldAutoCompactAfterAgentError(previousSessionID, err, result) {
			var retryErr error
			var agentErrorRecorded bool
			result, durationMS, totalDurationMS, compactionEventID, agentErrorRecorded, retryErr = runner.retryStepAfterAutoCompaction(agentCtx, ctx, view, userEvent.EventID, stepID, toolSessionID, instruction, prompt, previousSessionID, err, result, durationMS)
			if retryErr == nil {
				compactionAttempted = true
				err = nil
			} else {
				if !agentErrorRecorded {
					_, _ = runner.appendAgentError(ctx, view, userEvent.EventID, stepID, toolSessionID, previousSessionID, result, durationMS, retryErr)
				}
				return workflowstate.WorkflowRunView{}, retryErr
			}
		}
	}
	if err != nil {
		cause := err
		if errors.Is(agentCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			cause = context.DeadlineExceeded
		}
		_, _ = runner.appendAgentError(ctx, view, userEvent.EventID, stepID, toolSessionID, previousSessionID, result, durationMS, cause)
		return workflowstate.WorkflowRunView{}, cause
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validatedSameSessionResult(result, previousSessionID)
	if err != nil {
		_, _ = runner.appendAgentError(ctx, view, userEvent.EventID, stepID, toolSessionID, previousSessionID, result, durationMS, err)
		return workflowstate.WorkflowRunView{}, fmt.Errorf("%w: returned session %q", err, returnedSessionID)
	}
	visibleText, decision, ok := ParseControlDecision(result.Text)
	responseExtra := map[string]any{
		"previous_agent_session_id": previousSessionID,
		"workflow_run_id":           view.WorkflowRunID,
		"workflow_step_id":          stepID,
		"tool_session_id":           toolSessionID,
	}
	if compactionAttempted {
		responseExtra["compaction_attempted"] = true
		responseExtra["compaction_event_id"] = compactionEventID
		responseExtra["retry_after_compacted"] = true
		responseExtra["total_duration_ms"] = totalDurationMS
	}
	responseEvent, appendErr := runner.Service.AppendEvent(ctx, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
		EventID:                runner.newID("evt"),
		MissionID:              view.MissionID,
		Kind:                   "agent_response",
		AgentExecutor:          view.AgentExecutor,
		AgentModel:             strings.TrimSpace(runner.AgentModel),
		AgentReasoningEffort:   strings.TrimSpace(runner.ReasoningEffort),
		IncludeAgentConfig:     true,
		MCPMode:                view.MCPMode,
		IncludeMCPMode:         true,
		Text:                   visibleText,
		AgentSessionID:         result.SessionID,
		IncludeAgentSessionID:  true,
		Resumed:                result.Resumed,
		IncludeResumed:         true,
		DurationMS:             durationMS,
		IncludeDuration:        true,
		UserEventID:            userEvent.EventID,
		Extra:                  responseExtra,
		Usage:                  result.Usage,
		UsageSurface:           "workflow_step",
		UsagePreviousSessionID: previousSessionID,
		UsageCompaction:        false,
		Producer:               ledger.Producer{Type: "agent", ID: view.AgentExecutor},
	}))
	if appendErr != nil {
		return workflowstate.WorkflowRunView{}, appendErr
	}
	if !ok {
		return workflowstate.WorkflowRunView{}, fmt.Errorf("%w: workflow control decision is missing or invalid", producterror.ErrInvalidInput)
	}
	if err := runner.appendSourceCandidateEvent(ctx, view, userEvent.EventID, responseEvent.EventID, stepID, visibleText); err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	if _, err := runner.appendWorkflowEvent(ctx, view.MissionID, workflowstate.WorkflowStepCompletedEvent, workflowstate.WorkflowStepCompletedPayload{
		WorkflowRunID:   view.WorkflowRunID,
		MissionID:       view.MissionID,
		WorkflowStepID:  stepID,
		Decision:        decision.Decision,
		NextInstruction: decision.NextInstruction,
		Reason:          decision.Reason,
		DurationMS:      durationMS,
		AgentSessionID:  result.SessionID,
		ToolSessionID:   toolSessionID,
		ResultEventID:   responseEvent.EventID,
	}, ledger.Producer{Type: "workflow", ID: view.WorkflowRunID}); err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	if decision.Decision == "stop" {
		return runner.terminal(ctx, view.MissionID, view.WorkflowRunID, workflowstate.WorkflowRunCompletedEvent, firstNonEmpty(decision.Reason, "agent declared complete"), "")
	}
	_ = stepEvent
	return runner.Service.GetWorkflowRun(ctx, view.MissionID, view.WorkflowRunID)
}
