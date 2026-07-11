package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidates"
)

type Service interface {
	AppendEvent(context.Context, app.AppendEventRequest) (app.LedgerEvent, error)
	ListEvents(context.Context, string) ([]app.LedgerEvent, error)
	ListSourceSnapshotsWithState(context.Context, app.ListSourceSnapshotsRequest) ([]app.SourceSnapshot, error)
	GetWorkflowRun(context.Context, string, string) (app.WorkflowRunView, error)
	ClaimWorkflowRunStart(context.Context, string, string, time.Time) (app.WorkflowRunView, bool, error)
}

type AgentExecutor interface {
	Run(context.Context, AgentRequest) (AgentResult, error)
}

type AgentRequest struct {
	UserText          string
	Prompt            string
	Model             string
	ReasoningEffort   string
	MissionID         string
	ToolSessionID     string
	UserEventID       string
	PreviousSessionID string
	AgentExecutor     string
	MCPMode           string
	Compaction        bool
}

type AgentResult struct {
	Text      string
	SessionID string
	Resumed   bool
	Log       string
	Usage     agentusage.AgentUsage
}

type Runner struct {
	Service               Service
	Agent                 AgentExecutor
	AgentModel            string
	ReasoningEffort       string
	Now                   func() time.Time
	NewID                 func(string) string
	SourceCandidateStager func(context.Context, app.LedgerEvent)
}

type ControlDecision struct {
	Decision        string `json:"decision"`
	Reason          string `json:"reason"`
	NextInstruction string `json:"next_instruction"`
}

func (runner Runner) Run(ctx context.Context, missionID string, workflowRunID string) (app.WorkflowRunView, error) {
	if runner.Service == nil {
		return app.WorkflowRunView{}, fmt.Errorf("%w: workflow service is required", app.ErrInvalidInput)
	}
	if runner.Agent == nil {
		return app.WorkflowRunView{}, fmt.Errorf("%w: workflow agent executor is required", app.ErrInvalidInput)
	}
	view, err := runner.Service.GetWorkflowRun(ctx, missionID, workflowRunID)
	if err != nil {
		return app.WorkflowRunView{}, err
	}
	if workflowTerminalStatus(view.Status) {
		return view, nil
	}
	events, err := runner.Service.ListEvents(ctx, missionID)
	if err != nil {
		return app.WorkflowRunView{}, err
	}
	if view.StartAfterEventID != "" && !hasAgentTerminalEventForUser(events, view.StartAfterEventID) {
		return view, nil
	}
	if hasOpenAgentPending(events) {
		return view, fmt.Errorf("%w: agent turn is already running for this mission", app.ErrInvalidInput)
	}
	if view.StartedEventID == "" {
		claimedView, claimed, err := runner.Service.ClaimWorkflowRunStart(ctx, missionID, workflowRunID, runner.now())
		if err != nil {
			return app.WorkflowRunView{}, err
		}
		if !claimed {
			return claimedView, nil
		}
		view = claimedView
	}

	startedAt := runner.now()
	for {
		if err := ctx.Err(); err != nil {
			return runner.terminal(ctx, missionID, workflowRunID, app.WorkflowRunInterruptedEvent, "context canceled", err.Error())
		}
		view, err = runner.Service.GetWorkflowRun(ctx, missionID, workflowRunID)
		if err != nil {
			return app.WorkflowRunView{}, err
		}
		if workflowTerminalStatus(view.Status) {
			return view, nil
		}
		if view.StopRequestedEventID != "" {
			return runner.terminal(ctx, missionID, workflowRunID, app.WorkflowRunStoppedEvent, firstNonEmpty(view.StopReason, "stop requested"), "")
		}
		if view.MaxSteps > 0 && view.CompletedStepCount >= view.MaxSteps {
			return runner.limitReached(ctx, missionID, workflowRunID, view, "max_steps reached")
		}
		if view.MaxDurationMS > 0 && runner.now().Sub(startedAt).Milliseconds() >= view.MaxDurationMS {
			return runner.limitReached(ctx, missionID, workflowRunID, view, "max_duration reached")
		}
		if _, err := runner.runStep(ctx, view); err != nil {
			return runner.terminal(ctx, missionID, workflowRunID, app.WorkflowRunFailedEvent, "workflow step failed", err.Error())
		}
	}
}

func (runner Runner) runStep(ctx context.Context, view app.WorkflowRunView) (app.WorkflowRunView, error) {
	stepID := runner.newID("wfs")
	toolSessionID := runner.newID("ses")
	instruction := nextInstruction(view)
	stepIndex := view.CompletedStepCount + 1
	stepStartedAt := runner.now()
	if err := runner.appendRemovedSourceSkips(ctx, view, stepID, stepIndex); err != nil {
		return app.WorkflowRunView{}, err
	}
	stepEvent, err := runner.appendWorkflowEvent(ctx, view.MissionID, app.WorkflowStepStartedEvent, app.WorkflowStepStartedPayload{
		WorkflowRunID:  view.WorkflowRunID,
		MissionID:      view.MissionID,
		WorkflowStepID: stepID,
		Instruction:    instruction,
		StepIndex:      stepIndex,
		StartedAt:      stepStartedAt.Format(time.RFC3339Nano),
		ToolSessionID:  toolSessionID,
	}, app.Producer{Type: "workflow", ID: view.WorkflowRunID})
	if err != nil {
		return app.WorkflowRunView{}, err
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
		Producer:             app.Producer{Type: "workflow", ID: view.WorkflowRunID},
	})
	userEvent, err := runner.Service.AppendEvent(ctx, userEventReq)
	if err != nil {
		return app.WorkflowRunView{}, err
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
		Producer:             app.Producer{Type: "agent", ID: view.AgentExecutor},
	})); err != nil {
		return app.WorkflowRunView{}, err
	}

	events, err := runner.Service.ListEvents(ctx, view.MissionID)
	if err != nil {
		return app.WorkflowRunView{}, err
	}
	previousSessionID := LatestAgentSessionID(events, view.AgentExecutor)
	started := runner.now()
	prompt := StepPrompt(view, instruction, toolSessionID, previousSessionID != "")
	result, err := runner.Agent.Run(ctx, AgentRequest{
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
	durationMS := runner.now().Sub(started).Milliseconds()
	compactionAttempted := false
	compactionEventID := ""
	totalDurationMS := int64(0)
	if err != nil {
		if ctx.Err() == nil && !errors.Is(err, context.Canceled) && shouldAutoCompactAfterAgentError(previousSessionID, err, result) {
			var retryErr error
			result, durationMS, totalDurationMS, compactionEventID, retryErr = runner.retryStepAfterAutoCompaction(ctx, view, userEvent.EventID, stepID, toolSessionID, instruction, prompt, previousSessionID, err, result, durationMS)
			if retryErr == nil {
				compactionAttempted = true
				err = nil
			} else {
				return app.WorkflowRunView{}, retryErr
			}
		}
	}
	if err != nil {
		_, _ = runner.appendAgentError(ctx, view, userEvent.EventID, stepID, toolSessionID, previousSessionID, result, durationMS, err)
		return app.WorkflowRunView{}, err
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validatedSameSessionResult(result, previousSessionID)
	if err != nil {
		_, _ = runner.appendAgentError(ctx, view, userEvent.EventID, stepID, toolSessionID, previousSessionID, result, durationMS, err)
		return app.WorkflowRunView{}, fmt.Errorf("%w: returned session %q", err, returnedSessionID)
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
		Producer:               app.Producer{Type: "agent", ID: view.AgentExecutor},
	}))
	if appendErr != nil {
		return app.WorkflowRunView{}, appendErr
	}
	if !ok {
		return app.WorkflowRunView{}, fmt.Errorf("%w: workflow control decision is missing or invalid", app.ErrInvalidInput)
	}
	if err := runner.appendSourceCandidateEvent(ctx, view, userEvent.EventID, responseEvent.EventID, stepID, visibleText); err != nil {
		return app.WorkflowRunView{}, err
	}
	if _, err := runner.appendWorkflowEvent(ctx, view.MissionID, app.WorkflowStepCompletedEvent, app.WorkflowStepCompletedPayload{
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
	}, app.Producer{Type: "workflow", ID: view.WorkflowRunID}); err != nil {
		return app.WorkflowRunView{}, err
	}
	if decision.Decision == "stop" {
		return runner.terminal(ctx, view.MissionID, view.WorkflowRunID, app.WorkflowRunCompletedEvent, firstNonEmpty(decision.Reason, "agent declared complete"), "")
	}
	_ = stepEvent
	return runner.Service.GetWorkflowRun(ctx, view.MissionID, view.WorkflowRunID)
}

func (runner Runner) retryStepAfterAutoCompaction(ctx context.Context, view app.WorkflowRunView, userEventID string, stepID string, toolSessionID string, instruction string, prompt string, previousSessionID string, initialErr error, initialResult AgentResult, initialDurationMS int64) (AgentResult, int64, int64, string, error) {
	compactStarted := runner.now()
	compactResult, err := runner.Agent.Run(ctx, AgentRequest{
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
	compactDurationMS := runner.now().Sub(compactStarted).Milliseconds()
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return AgentResult{}, 0, 0, "", err
		}
		_, _ = runner.appendAgentError(ctx, view, userEventID, stepID, toolSessionID, previousSessionID, compactResult, initialDurationMS+compactDurationMS, err, map[string]any{
			"compaction_attempted": true,
			"original_error":       initialErr.Error(),
			"original_log_excerpt": headTailExcerpt(initialResult.Log, 2000),
			"text":                 "워크플로우 단계에서 에이전트 컨텍스트가 가득 차 자동 압축을 시도했지만 실패했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
		return AgentResult{}, 0, 0, "", err
	}
	returnedCompactSessionID := strings.TrimSpace(compactResult.SessionID)
	compactResult, err = validatedSameSessionResult(compactResult, previousSessionID)
	if err != nil {
		_, _ = runner.appendAgentError(ctx, view, userEventID, stepID, toolSessionID, previousSessionID, compactResult, initialDurationMS+compactDurationMS, err, map[string]any{
			"compaction_attempted":      true,
			"original_error":            initialErr.Error(),
			"returned_agent_session_id": returnedCompactSessionID,
			"text":                      "워크플로우 단계의 자동 압축 요청에서 에이전트가 다른 세션 ID를 반환했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
		return AgentResult{}, 0, 0, "", err
	}
	compactEvent, err := runner.Service.AppendEvent(ctx, conversation.BuildTurnAgentCompactedAppendRequest(conversation.TurnAgentCompactedEventRequest{
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
		Producer:               app.Producer{Type: "agent", ID: view.AgentExecutor},
	}))
	if err != nil {
		return AgentResult{}, 0, 0, "", err
	}

	retryStarted := runner.now()
	result, err := runner.Agent.Run(ctx, AgentRequest{
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
	retryDurationMS := runner.now().Sub(retryStarted).Milliseconds()
	totalDurationMS := initialDurationMS + compactDurationMS + retryDurationMS
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return AgentResult{}, 0, 0, "", err
		}
		_, _ = runner.appendAgentError(ctx, view, userEventID, stepID, toolSessionID, previousSessionID, result, retryDurationMS, err, map[string]any{
			"compaction_attempted": true,
			"compaction_event_id":  compactEvent.EventID,
			"original_error":       initialErr.Error(),
			"total_duration_ms":    totalDurationMS,
			"text":                 "워크플로우 단계에서 같은 세션을 자동 압축한 뒤 재시도했지만 실패했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
		return AgentResult{}, 0, 0, "", err
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validatedSameSessionResult(result, previousSessionID)
	if err != nil {
		_, _ = runner.appendAgentError(ctx, view, userEventID, stepID, toolSessionID, previousSessionID, result, retryDurationMS, err, map[string]any{
			"compaction_attempted":      true,
			"compaction_event_id":       compactEvent.EventID,
			"returned_agent_session_id": returnedSessionID,
			"total_duration_ms":         totalDurationMS,
			"text":                      "워크플로우 단계의 자동 압축 후 재시도에서 에이전트가 다른 세션 ID를 반환했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
		return AgentResult{}, 0, 0, "", err
	}
	return result, retryDurationMS, totalDurationMS, compactEvent.EventID, nil
}

func (runner Runner) appendRemovedSourceSkips(ctx context.Context, view app.WorkflowRunView, stepID string, stepIndex int) error {
	sources, err := runner.Service.ListSourceSnapshotsWithState(ctx, app.ListSourceSnapshotsRequest{MissionID: view.MissionID, IncludeRemoved: true})
	if err != nil {
		return err
	}
	events, err := runner.Service.ListEvents(ctx, view.MissionID)
	if err != nil {
		return err
	}
	byID := eventsByID(events)
	boundarySequence := workflowRunStartBoundarySequence(view, byID)
	alreadySkipped := workflowSourceSkipKeys(events, view.WorkflowRunID)
	for _, source := range sources {
		if !source.State.Removed {
			continue
		}
		removedEventID := strings.TrimSpace(source.State.RemovedEventID)
		if removedEventID == "" {
			continue
		}
		removedEvent, ok := byID[removedEventID]
		if ok && boundarySequence > 0 && removedEvent.Sequence <= boundarySequence {
			continue
		}
		key := source.SnapshotID + "|" + removedEventID
		if _, ok := alreadySkipped[key]; ok {
			continue
		}
		if _, err := runner.appendWorkflowEvent(ctx, view.MissionID, app.WorkflowSourceSkippedEvent, app.WorkflowSourceSkippedPayload{
			WorkflowRunID:   view.WorkflowRunID,
			MissionID:       view.MissionID,
			WorkflowStepID:  stepID,
			StepIndex:       stepIndex,
			SnapshotID:      source.SnapshotID,
			Reason:          "source_removed",
			RemovedEventID:  removedEventID,
			SkippedAt:       runner.now().Format(time.RFC3339Nano),
			RetrievalPolicy: source.Access.RetrievalPolicy,
			ConnectorType:   source.Connector.ConnectorType,
		}, app.Producer{Type: "workflow", ID: view.WorkflowRunID}); err != nil {
			return err
		}
		alreadySkipped[key] = struct{}{}
	}
	return nil
}

func eventsByID(events []app.LedgerEvent) map[string]app.LedgerEvent {
	byID := make(map[string]app.LedgerEvent, len(events))
	for _, event := range events {
		byID[event.EventID] = event
	}
	return byID
}

func workflowRunStartBoundarySequence(view app.WorkflowRunView, byID map[string]app.LedgerEvent) int64 {
	for _, eventID := range []string{view.StartedEventID, view.RequestedEventID} {
		event, ok := byID[strings.TrimSpace(eventID)]
		if ok {
			return event.Sequence
		}
	}
	return 0
}

func workflowSourceSkipKeys(events []app.LedgerEvent, workflowRunID string) map[string]struct{} {
	keys := map[string]struct{}{}
	for _, event := range events {
		if event.EventType != app.WorkflowSourceSkippedEvent {
			continue
		}
		var payload app.WorkflowSourceSkippedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.WorkflowRunID) != strings.TrimSpace(workflowRunID) {
			continue
		}
		snapshotID := strings.TrimSpace(payload.SnapshotID)
		removedEventID := strings.TrimSpace(payload.RemovedEventID)
		if snapshotID == "" || removedEventID == "" {
			continue
		}
		keys[snapshotID+"|"+removedEventID] = struct{}{}
	}
	return keys
}

func (runner Runner) appendAgentError(ctx context.Context, view app.WorkflowRunView, userEventID string, stepID string, toolSessionID string, previousSessionID string, result AgentResult, durationMS int64, cause error, extra ...map[string]any) (app.LedgerEvent, error) {
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
		Producer:               app.Producer{Type: "agent", ID: view.AgentExecutor},
	}))
}

func (runner Runner) terminal(ctx context.Context, missionID string, workflowRunID string, eventType string, reason string, errorText string) (app.WorkflowRunView, error) {
	return runner.terminalWithNextInstruction(ctx, missionID, workflowRunID, eventType, reason, errorText, "")
}

func (runner Runner) terminalWithNextInstruction(ctx context.Context, missionID string, workflowRunID string, eventType string, reason string, errorText string, nextInstruction string) (app.WorkflowRunView, error) {
	view, _ := runner.Service.GetWorkflowRun(ctx, missionID, workflowRunID)
	payload := app.WorkflowRunTerminalPayload{
		WorkflowRunID:      workflowRunID,
		MissionID:          missionID,
		Reason:             reason,
		Error:              errorText,
		NextInstruction:    strings.TrimSpace(nextInstruction),
		CompletedStepCount: view.CompletedStepCount,
		TerminalAt:         runner.now().Format(time.RFC3339Nano),
	}
	if eventType == app.WorkflowRunStoppedEvent {
		payload.StopReason = reason
	}
	if _, err := runner.appendWorkflowEvent(ctx, missionID, eventType, payload, app.Producer{Type: "workflow", ID: workflowRunID}); err != nil {
		return app.WorkflowRunView{}, err
	}
	return runner.Service.GetWorkflowRun(ctx, missionID, workflowRunID)
}

func (runner Runner) limitReached(ctx context.Context, missionID string, workflowRunID string, view app.WorkflowRunView, reason string) (app.WorkflowRunView, error) {
	nextInstruction, ok := latestContinuationInstruction(view)
	if ok {
		return runner.terminalWithNextInstruction(ctx, missionID, workflowRunID, app.WorkflowRunPausedEvent, reason, "", nextInstruction)
	}
	return runner.terminal(ctx, missionID, workflowRunID, app.WorkflowRunCompletedEvent, reason, "")
}

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

func LatestAgentSessionID(events []app.LedgerEvent, executorName string) string {
	return conversation.LatestAgentSessionID(events, executorName)
}

func validatedSameSessionResult(result AgentResult, previousSessionID string) (AgentResult, error) {
	previousSessionID = strings.TrimSpace(previousSessionID)
	result.SessionID = strings.TrimSpace(result.SessionID)
	if previousSessionID == "" {
		if result.SessionID == "" {
			return result, fmt.Errorf("%w: agent did not return a session id", app.ErrInvalidInput)
		}
		return result, nil
	}
	if result.SessionID == "" {
		result.SessionID = previousSessionID
		return result, nil
	}
	if result.SessionID != previousSessionID {
		result.SessionID = ""
		return result, fmt.Errorf("%w: agent returned a different session id", app.ErrInvalidInput)
	}
	return result, nil
}

func shouldAutoCompactAfterAgentError(previousSessionID string, err error, result AgentResult) bool {
	if strings.TrimSpace(previousSessionID) == "" || err == nil {
		return false
	}
	text := strings.ToLower(err.Error() + "\n" + result.Log)
	return strings.Contains(text, "ran out of room in the model's context window")
}

func workflowCompactPrompt() string {
	return strings.TrimSpace(`Compact the useful session context for future Plasma workflow steps. Do not answer the user's research question in this turn.

Preserve:
- the mission objective and any steering that changed it
- important sources, source candidates, and why they matter
- useful findings, unresolved questions, and next investigation paths
- constraints about using Plasma MCP tools and not treating agent results as sources`)
}

func hasOpenAgentPending(events []app.LedgerEvent) bool {
	return conversation.HasOpenAgentPending(events)
}

func hasAgentTerminalEventForUser(events []app.LedgerEvent, userEventID string) bool {
	if strings.TrimSpace(userEventID) == "" {
		return true
	}
	return conversation.HasAgentTerminalEventForUser(events, userEventID)
}

func nextInstruction(view app.WorkflowRunView) string {
	for i := len(view.Steps) - 1; i >= 0; i-- {
		if strings.TrimSpace(view.Steps[i].NextInstruction) != "" {
			return strings.TrimSpace(view.Steps[i].NextInstruction)
		}
	}
	return strings.TrimSpace(view.Instruction)
}

func latestContinuationInstruction(view app.WorkflowRunView) (string, bool) {
	for i := len(view.Steps) - 1; i >= 0; i-- {
		step := view.Steps[i]
		if strings.TrimSpace(step.Decision) != "continue" {
			return "", false
		}
		return strings.TrimSpace(firstNonEmpty(step.NextInstruction, step.Reason, view.Instruction)), true
	}
	return "", false
}

func (runner Runner) appendSourceCandidateEvent(ctx context.Context, view app.WorkflowRunView, userEventID string, agentEventID string, stepID string, text string) error {
	candidates := sourceCandidatesFromText(text)
	if len(candidates) == 0 {
		return nil
	}
	appCandidates := make([]sourcecandidates.WorkflowSourceCandidateProposal, 0, len(candidates))
	for _, candidate := range candidates {
		appCandidates = append(appCandidates, sourcecandidates.WorkflowSourceCandidateProposal{
			URL:    candidate.URL,
			Title:  candidate.Title,
			Reason: candidate.Reason,
			State:  candidate.State,
		})
	}
	eventReq, ok, err := sourcecandidates.BuildWorkflowSourceCandidateProposalEventRequest(sourcecandidates.WorkflowSourceCandidateProposalEventRequest{
		EventID:        runner.newID("evt"),
		MissionID:      view.MissionID,
		WorkflowRunID:  view.WorkflowRunID,
		WorkflowStepID: stepID,
		UserEventID:    userEventID,
		AgentEventID:   agentEventID,
		Producer:       app.Producer{Type: "agent", ID: view.AgentExecutor},
		Candidates:     appCandidates,
	})
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	event, err := runner.Service.AppendEvent(ctx, eventReq)
	if err != nil {
		return err
	}
	if runner.SourceCandidateStager != nil {
		runner.SourceCandidateStager(context.Background(), event)
	}
	return nil
}

type sourceCandidate struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
	State  string `json:"state"`
}

func sourceCandidatesFromText(text string) []sourceCandidate {
	parsed := sourcecandidates.Parse(text)
	if len(parsed) == 0 {
		return nil
	}
	candidates := make([]sourceCandidate, 0, len(parsed))
	for _, candidate := range parsed {
		candidates = append(candidates, sourceCandidate{
			URL:    candidate.URL,
			Title:  candidate.Title,
			Reason: candidate.Reason,
			State:  candidate.State,
		})
	}
	return candidates
}

func (runner Runner) appendWorkflowEvent(ctx context.Context, missionID string, eventType string, payload any, producer app.Producer) (app.LedgerEvent, error) {
	return runner.appendEvent(ctx, missionID, eventType, payload, producer)
}

func (runner Runner) appendEvent(ctx context.Context, missionID string, eventType string, payload any, producer app.Producer) (app.LedgerEvent, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return app.LedgerEvent{}, err
	}
	return runner.Service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   runner.newID("evt"),
		MissionID: missionID,
		EventType: eventType,
		Producer:  producer,
		Payload:   encoded,
	})
}

func (runner Runner) now() time.Time {
	if runner.Now != nil {
		return runner.Now().UTC()
	}
	return time.Now().UTC()
}

func (runner Runner) newID(prefix string) string {
	if runner.NewID != nil {
		return runner.NewID(prefix)
	}
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s_%s", strings.TrimSuffix(prefix, "_"), time.Now().UTC().Format("20060102150405"), hex.EncodeToString(b[:]))
}

func workflowTerminalStatus(status string) bool {
	switch status {
	case app.WorkflowStatusCompleted, app.WorkflowStatusPaused, app.WorkflowStatusStopped, app.WorkflowStatusFailed, app.WorkflowStatusInterrupted:
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func headTailExcerpt(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	headLimit := limit / 2
	tailLimit := limit - headLimit
	return value[:headLimit] + "\n[truncated middle]\n" + value[len(value)-tailLimit:]
}
