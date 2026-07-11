package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentmodels"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidates"
	"github.com/c86j224s/liquid2/plasma/internal/sourceingest"
)

func (server *Server) handleMissionTurns(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) == 1 && rest[0] == "cancel" {
		server.handleCancelMissionTurn(w, r, missionID)
		return
	}
	if len(rest) != 0 {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req turnRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "turn text is required")
		return
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		writeAppError(w, err)
		return
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		writeAppError(w, err)
		return
	}
	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	unlockTurns := server.turns.lock(missionID)
	defer unlockTurns()
	if err := server.validateMissionAgentExecutor(r.Context(), missionID, executorName); err != nil {
		writeAppError(w, err)
		return
	}
	if err := server.reconcileStaleAgentTurn(r.Context(), missionID); err != nil {
		writeAppError(w, err)
		return
	}
	if server.hasOpenAgentTurn(r.Context(), missionID) {
		writeError(w, http.StatusConflict, "agent turn is already running for this mission")
		return
	}
	if server.hasOpenReportDraft(r.Context(), missionID) {
		writeError(w, http.StatusConflict, "report draft is already running for this mission")
		return
	}
	if active := server.activeWorkflowRun(r.Context(), missionID); active != nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("workflow %s is %s for this mission; stop it before sending a normal turn", active.WorkflowRunID, active.Status))
		return
	}
	toolSessionID := newID("ses")

	turnProducer := app.Producer{Type: "user", ID: "plasma-ui"}
	turnKind := "user_turn"
	if req.Controller {
		turnProducer = app.Producer{Type: "steering_chat", ID: "plasma-controller"}
		turnKind = "controller_steering"
	}
	userEventReq := conversation.BuildTurnUserAppendRequest(conversation.TurnUserEventRequest{
		EventID:       newID("evt"),
		MissionID:     missionID,
		Kind:          turnKind,
		Text:          req.Text,
		AgentExecutor: executorName,
		MCPMode:       mcpMode,
		ToolSessionID: toolSessionID,
		Producer:      turnProducer,
	})
	recall, err := server.buildRecall(r.Context(), missionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	controllerDecision := controllerStrategyDecision{}
	previousSessionID := ""
	if !isManualCompactCommand(req.Text) {
		previousSessionID = server.latestAgentSessionID(r.Context(), missionID, executorName)
		controllerDecision = selectControllerStrategy(req.ControllerStrategy, req.Text, recall, previousSessionID != "")
	}
	eventReqs := []app.AppendEventRequest{userEventReq}
	if !isManualCompactCommand(req.Text) {
		eventReqs = append(eventReqs, conversation.BuildControllerStrategySelectedAppendRequest(conversation.ControllerStrategySelectedEventRequest{
			EventID:           newID("evt"),
			MissionID:         missionID,
			StrategyID:        controllerDecision.ID,
			StrategyLabel:     controllerDecision.Label,
			Reason:            controllerDecision.Reason,
			Guidance:          controllerDecision.Guidance,
			RequestedStrategy: normalizeControllerStrategy(req.ControllerStrategy),
			AgentExecutor:     executorName,
			MCPMode:           mcpMode,
			UserEventID:       userEventReq.EventID,
			ToolSessionID:     toolSessionID,
			PreviousSessionID: previousSessionID,
			Producer:          app.Producer{Type: "steering_chat", ID: "plasma-controller"},
		}))
	}
	eventReqs = append(eventReqs, conversation.BuildTurnAgentPendingAppendRequest(conversation.TurnAgentPendingEventRequest{
		EventID:           newID("evt"),
		MissionID:         missionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
		StrategyID:        controllerDecision.ID,
		IncludeStrategyID: true,
		Text:              "에이전트 응답을 기다리는 중입니다.",
		UserEventID:       userEventReq.EventID,
		ToolSessionID:     toolSessionID,
		StartedAt:         time.Now().UTC().Format(time.RFC3339Nano),
		Producer:          app.Producer{Type: "agent", ID: executorName},
	}))
	appendedEvents, err := server.service.AppendEventsIfNoActiveAgentWork(r.Context(), missionID, eventReqs)
	if err != nil {
		writeAppError(w, err)
		return
	}
	userEvent := appendedEvents[0]
	pendingEvent := appendedEvents[len(appendedEvents)-1]
	agentCtx, cancel := context.WithCancel(context.Background())
	runID := server.runningTurns.start(missionID, executorName, cancel)
	go func() {
		server.completeAgentTurn(agentCtx, missionID, req.Text, userEvent.EventID, recall, executorName, mcpMode, toolSessionID, controllerDecision)
		server.runningTurns.finish(missionID, runID)
		server.drainQueuedWorkflows(context.Background(), missionID)
	}()
	response := map[string]any{
		"user_event":    userEvent,
		"pending_event": pendingEvent,
		"recall":        recall,
	}
	writeJSON(w, http.StatusAccepted, response)
}

func (server *Server) handleCancelMissionTurn(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req cancelTurnRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	unlockTurns := server.turns.lock(missionID)
	defer unlockTurns()
	pending, ok := server.latestOpenAgentPending(r.Context(), missionID)
	if !ok {
		writeError(w, http.StatusConflict, "agent turn is not running for this mission")
		return
	}
	if strings.TrimSpace(pending.WorkflowRunID) != "" {
		canceled := server.runningWorkflow.cancel(pending.WorkflowRunID)
		event, err := server.appendAgentCanceledWithWorkflowTerminal(
			r.Context(),
			missionID,
			pending.UserEventID,
			pending.AgentExecutor,
			"사용자가 웹에서 워크플로우 단계의 에이전트 응답을 취소했습니다.",
			app.WorkflowRunStoppedEvent,
		)
		if err != nil {
			writeAppError(w, err)
			return
		}
		status := http.StatusOK
		stale := true
		if canceled {
			status = http.StatusAccepted
			stale = false
		}
		writeJSON(w, status, map[string]any{"canceled": true, "stale": stale, "event": event})
		return
	}
	canceled := server.runningTurns.cancel(missionID, pending.AgentExecutor)
	if !canceled {
		event, err := server.appendAgentCanceled(r.Context(), missionID, pending.UserEventID, pending.AgentExecutor, "브라우저에서 끊긴 오래된 대기 상태를 취소했습니다.")
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"canceled": true, "stale": true, "event": event})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"canceled": true, "stale": false})
}

func (server *Server) handleAgentSessions(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) != 1 || rest[0] != "reset" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req resetAgentSessionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		writeAppError(w, err)
		return
	}
	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	unlockTurns := server.turns.lock(missionID)
	defer unlockTurns()
	if err := server.reconcileStaleAgentTurn(r.Context(), missionID); err != nil {
		writeAppError(w, err)
		return
	}
	if server.hasOpenAgentTurn(r.Context(), missionID) {
		writeError(w, http.StatusConflict, "agent turn is already running for this mission")
		return
	}
	if server.hasOpenReportDraft(r.Context(), missionID) {
		writeError(w, http.StatusConflict, "report draft is already running for this mission")
		return
	}
	if active := server.activeWorkflowRun(r.Context(), missionID); active != nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("workflow %s is %s for this mission; stop it before resetting the agent session", active.WorkflowRunID, active.Status))
		return
	}
	if err := server.validateMissionAgentExecutor(r.Context(), missionID, executorName); err != nil {
		writeAppError(w, err)
		return
	}
	agentModel, err := normalizeAgentModelName(req.AgentModel)
	if err != nil {
		writeAppError(w, err)
		return
	}
	agentReasoningEffort, err := normalizeAgentReasoningEffort(req.AgentReasoningEffort)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if executorName != "codex" {
		agentReasoningEffort = ""
	} else {
		_, _, err = agentmodels.Resolve(agentModel, agentReasoningEffort)
		if err != nil {
			writeAppError(w, fmt.Errorf("%w: %v", app.ErrInvalidInput, err))
			return
		}
	}
	previousSessionID := server.latestAgentSessionID(r.Context(), missionID, executorName)
	appendedEvents, err := server.service.AppendEventsIfNoActiveAgentWork(r.Context(), missionID, []app.AppendEventRequest{
		conversation.BuildAgentSessionResetAppendRequest(conversation.AgentSessionResetEventRequest{
			EventID:                newID("evt"),
			MissionID:              missionID,
			AgentExecutor:          executorName,
			AgentModel:             agentModel,
			AgentReasoningEffort:   agentReasoningEffort,
			PreviousAgentSessionID: previousSessionID,
			Producer:               app.Producer{Type: "user", ID: "plasma-ui"},
		}),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	event := appendedEvents[0]
	writeJSON(w, http.StatusCreated, map[string]any{"event": event, "previous_agent_session_id": previousSessionID, "agent_model": agentModel, "agent_reasoning_effort": agentReasoningEffort})
}

func (server *Server) completeAgentTurn(
	ctx context.Context,
	missionID string,
	userText string,
	userEventID string,
	recall recallPreview,
	executorName string,
	mcpMode string,
	toolSessionID string,
	controller controllerStrategyDecision,
) {
	if _, err := server.runAgentTurn(ctx, missionID, userText, userEventID, recall, executorName, mcpMode, toolSessionID, controller); err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			_, _ = server.appendAgentCanceled(context.Background(), missionID, userEventID, executorName, "에이전트 응답을 취소했습니다.")
			return
		}
		_, _ = server.appendAgentError(context.Background(), missionID, userEventID, executorName, err, AgentResult{}, 0, nil)
	}
}

func (server *Server) runAgentTurn(
	ctx context.Context,
	missionID string,
	userText string,
	userEventID string,
	recall recallPreview,
	executorName string,
	mcpMode string,
	toolSessionID string,
	controller controllerStrategyDecision,
) (app.LedgerEvent, error) {
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return server.service.AppendEvent(ctx, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
			EventID:        newID("evt"),
			MissionID:      missionID,
			Kind:           "placeholder",
			AgentExecutor:  executorName,
			MCPMode:        mcpMode,
			IncludeMCPMode: true,
			Text:           "에이전트 실행기가 아직 연결되지 않았습니다. 사용자 턴은 장부에 기록했습니다.",
			UserEventID:    userEventID,
			Extra: map[string]any{
				"strategy_id": controller.ID,
			},
			Producer: app.Producer{Type: "agent", ID: executorName},
		}))
	}
	if isManualCompactCommand(userText) {
		return server.runManualAgentCompaction(ctx, missionID, userEventID, recall, executorName, mcpMode, toolSessionID)
	}
	previousSessionID := server.latestAgentSessionID(ctx, missionID, executorName)
	agentModel := server.latestAgentSessionModel(ctx, missionID, executorName)
	agentReasoningEffort := server.latestAgentReasoningEffort(ctx, missionID, executorName)
	agentModel, agentReasoningEffort, err := resolveAgentSettings(executorName, agentModel, agentReasoningEffort, previousSessionID)
	if err != nil {
		return app.LedgerEvent{}, err
	}
	prompt := agentPrompt(userText, recall, mcpMode, previousSessionID != "", toolSessionID, controller)
	started := time.Now()
	result, err := executor.Run(ctx, AgentRequest{
		UserText:          userText,
		Prompt:            prompt,
		Model:             agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		UserEventID:       userEventID,
		PreviousSessionID: previousSessionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
	})
	durationMS := time.Since(started).Milliseconds()
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return app.LedgerEvent{}, err
		}
		if shouldAutoCompactAfterAgentError(previousSessionID, err, result) {
			return server.retryAgentTurnAfterAutoCompaction(ctx, missionID, userText, userEventID, recall, executor, executorName, agentModel, agentReasoningEffort, mcpMode, toolSessionID, previousSessionID, prompt, err, result, durationMS, controller)
		}
		return server.appendAgentError(ctx, missionID, userEventID, executorName, err, result, durationMS, map[string]any{
			"previous_agent_session_id": previousSessionID,
			"tool_session_id":           toolSessionID,
			"strategy_id":               controller.ID,
			"agent_model":               agentModel,
			"agent_reasoning_effort":    agentReasoningEffort,
		})
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validatedSameSessionResult(result, previousSessionID)
	if err != nil {
		return server.appendAgentError(ctx, missionID, userEventID, executorName, err, result, durationMS, map[string]any{
			"previous_agent_session_id": previousSessionID,
			"returned_agent_session_id": returnedSessionID,
			"tool_session_id":           toolSessionID,
			"strategy_id":               controller.ID,
			"agent_model":               agentModel,
			"agent_reasoning_effort":    agentReasoningEffort,
			"text":                      sameSessionValidationUserText(err),
		})
	}
	return server.appendAgentSuccess(ctx, missionID, userEventID, executorName, mcpMode, result, durationMS, map[string]any{
		"tool_session_id":           toolSessionID,
		"strategy_id":               controller.ID,
		"previous_agent_session_id": previousSessionID,
		"agent_model":               agentModel,
		"agent_reasoning_effort":    agentReasoningEffort,
	})
}

func sameSessionValidationUserText(err error) string {
	message := ""
	if err != nil {
		message = strings.ToLower(err.Error())
	}
	if strings.Contains(message, "did not return a session id") {
		return "에이전트가 재개 요청에 대한 세션 ID를 반환하지 않았습니다. 같은 세션으로 이어졌는지 확인할 수 없어 새 세션으로 자동 전환하지 않았습니다."
	}
	return "에이전트가 재개 요청과 다른 세션 ID를 반환했습니다. 새 세션으로 자동 전환하지 않았습니다."
}

func (server *Server) retryAgentTurnAfterAutoCompaction(
	ctx context.Context,
	missionID string,
	userText string,
	userEventID string,
	recall recallPreview,
	executor AgentExecutor,
	executorName string,
	agentModel string,
	agentReasoningEffort string,
	mcpMode string,
	toolSessionID string,
	previousSessionID string,
	prompt string,
	initialErr error,
	initialResult AgentResult,
	initialDurationMS int64,
	controller controllerStrategyDecision,
) (app.LedgerEvent, error) {
	compactStarted := time.Now()
	compactResult, err := executor.Run(ctx, AgentRequest{
		UserText:          "compact session context",
		Prompt:            agentCompactPrompt(recall),
		Model:             agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		UserEventID:       userEventID,
		PreviousSessionID: previousSessionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
		Compaction:        true,
	})
	compactDurationMS := time.Since(compactStarted).Milliseconds()
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return app.LedgerEvent{}, err
		}
		return server.appendAgentError(ctx, missionID, userEventID, executorName, err, compactResult, initialDurationMS+compactDurationMS, map[string]any{
			"previous_agent_session_id": previousSessionID,
			"tool_session_id":           toolSessionID,
			"compaction_attempted":      true,
			"original_error":            initialErr.Error(),
			"original_log_excerpt":      headTailExcerpt(initialResult.Log, 2000),
			"strategy_id":               controller.ID,
			"agent_model":               agentModel,
			"agent_reasoning_effort":    agentReasoningEffort,
			"text":                      "에이전트 컨텍스트가 가득 차 자동 압축을 시도했지만 실패했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
	}
	returnedCompactSessionID := strings.TrimSpace(compactResult.SessionID)
	compactResult, err = validatedSameSessionResult(compactResult, previousSessionID)
	if err != nil {
		return server.appendAgentError(ctx, missionID, userEventID, executorName, err, compactResult, initialDurationMS+compactDurationMS, map[string]any{
			"previous_agent_session_id": previousSessionID,
			"returned_agent_session_id": returnedCompactSessionID,
			"tool_session_id":           toolSessionID,
			"compaction_attempted":      true,
			"original_error":            initialErr.Error(),
			"strategy_id":               controller.ID,
			"agent_model":               agentModel,
			"agent_reasoning_effort":    agentReasoningEffort,
			"text":                      "에이전트가 자동 압축 요청에서 다른 세션 ID를 반환했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
	}
	compactEvent, err := server.service.AppendEvent(ctx, conversation.BuildTurnAgentCompactedAppendRequest(conversation.TurnAgentCompactedEventRequest{
		EventID:                newID("evt"),
		MissionID:              missionID,
		AgentExecutor:          executorName,
		AgentModel:             agentModel,
		AgentReasoningEffort:   agentReasoningEffort,
		MCPMode:                mcpMode,
		AgentSessionID:         compactResult.SessionID,
		PreviousAgentSessionID: previousSessionID,
		ToolSessionID:          toolSessionID,
		Summary:                compactResult.Text,
		DurationMS:             compactDurationMS,
		UserEventID:            userEventID,
		Manual:                 false,
		Reason:                 "context_window_exhausted",
		Usage:                  compactResult.Usage,
		Resumed:                compactResult.Resumed,
		Producer:               app.Producer{Type: "agent", ID: executorName},
	}))
	if err != nil {
		return app.LedgerEvent{}, err
	}

	retryStarted := time.Now()
	result, err := executor.Run(ctx, AgentRequest{
		UserText:          userText,
		Prompt:            prompt,
		Model:             agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		UserEventID:       userEventID,
		PreviousSessionID: previousSessionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
	})
	retryDurationMS := time.Since(retryStarted).Milliseconds()
	durationMS := initialDurationMS + compactDurationMS + retryDurationMS
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return app.LedgerEvent{}, err
		}
		return server.appendAgentError(ctx, missionID, userEventID, executorName, err, result, retryDurationMS, map[string]any{
			"previous_agent_session_id": previousSessionID,
			"tool_session_id":           toolSessionID,
			"compaction_attempted":      true,
			"compaction_event_id":       compactEvent.EventID,
			"original_error":            initialErr.Error(),
			"strategy_id":               controller.ID,
			"total_duration_ms":         durationMS,
			"agent_usage_surface":       "turn",
			"agent_model":               agentModel,
			"agent_reasoning_effort":    agentReasoningEffort,
			"text":                      "에이전트 컨텍스트가 가득 차 같은 세션을 자동 압축한 뒤 재시도했지만 실패했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validatedSameSessionResult(result, previousSessionID)
	if err != nil {
		return server.appendAgentError(ctx, missionID, userEventID, executorName, err, result, retryDurationMS, map[string]any{
			"previous_agent_session_id": previousSessionID,
			"returned_agent_session_id": returnedSessionID,
			"tool_session_id":           toolSessionID,
			"compaction_attempted":      true,
			"compaction_event_id":       compactEvent.EventID,
			"strategy_id":               controller.ID,
			"total_duration_ms":         durationMS,
			"agent_usage_surface":       "turn",
			"agent_model":               agentModel,
			"agent_reasoning_effort":    agentReasoningEffort,
			"text":                      "에이전트가 자동 압축 후 재개 요청과 다른 세션 ID를 반환했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
	}
	return server.appendAgentSuccess(ctx, missionID, userEventID, executorName, mcpMode, result, retryDurationMS, map[string]any{
		"compaction_attempted":      true,
		"compaction_event_id":       compactEvent.EventID,
		"previous_agent_session_id": previousSessionID,
		"previous_agent_error":      initialErr.Error(),
		"retry_after_compacted":     true,
		"strategy_id":               controller.ID,
		"total_duration_ms":         durationMS,
		"agent_model":               agentModel,
		"agent_reasoning_effort":    agentReasoningEffort,
	})
}

func (server *Server) ensureAgentProposals(
	ctx context.Context,
	missionID string,
	userEventID string,
	recall recallPreview,
	executor AgentExecutor,
	executorName string,
	mcpMode string,
	toolSessionID string,
	result AgentResult,
) map[string]any {
	status := map[string]any{"attempted": false}
	if mcpMode != "auto" {
		status["reason"] = "explicit_mode"
		return status
	}
	if strings.TrimSpace(result.SessionID) == "" {
		status["reason"] = "no_agent_session"
		return status
	}
	existingProposalEvents := server.countAgentKnowledgeProposalEvents(ctx, missionID, toolSessionID)
	status["existing_proposal_events"] = existingProposalEvents
	if existingProposalEvents > 0 {
		status["main_turn_created_proposals"] = true
	}
	sources, err := server.service.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		status["error"] = err.Error()
		return status
	}
	if len(sources) == 0 {
		status["reason"] = "no_sources"
		return status
	}
	status["attempted"] = true
	started := time.Now()
	extraction, err := executor.Run(ctx, AgentRequest{
		UserText:          "create source-backed review proposals",
		Prompt:            agentProposalPrompt(recall, result.Text, toolSessionID),
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		UserEventID:       userEventID,
		PreviousSessionID: result.SessionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
	})
	status["duration_ms"] = time.Since(started).Milliseconds()
	recordFinalProposalEvents := func() int {
		finalProposalEvents := server.countAgentKnowledgeProposalEvents(ctx, missionID, toolSessionID)
		status["final_proposal_events"] = finalProposalEvents
		status["created_proposals"] = finalProposalEvents > existingProposalEvents
		return finalProposalEvents
	}
	if err != nil {
		finalProposalEvents := recordFinalProposalEvents()
		status["log_excerpt"] = headTailExcerpt(extraction.Log, 2000)
		if finalProposalEvents > existingProposalEvents {
			status["warning"] = err.Error()
			return status
		}
		status["error"] = err.Error()
		return status
	}
	if _, err := validatedSameSessionResult(extraction, result.SessionID); err != nil {
		status["error"] = err.Error()
		status["returned_agent_session_id"] = strings.TrimSpace(extraction.SessionID)
		return status
	}
	status["agent_session_id"] = strings.TrimSpace(extraction.SessionID)
	recordFinalProposalEvents()
	return status
}

func (server *Server) hasAgentKnowledgeProposalEvents(ctx context.Context, missionID string, toolSessionID string) bool {
	return server.countAgentKnowledgeProposalEvents(ctx, missionID, toolSessionID) > 0
}

func (server *Server) countAgentKnowledgeProposalEvents(ctx context.Context, missionID string, toolSessionID string) int {
	toolSessionID = strings.TrimSpace(toolSessionID)
	if toolSessionID == "" {
		return 0
	}
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return 0
	}
	count := 0
	for _, event := range events {
		if event.Producer.Type != "agent_session" || event.Producer.ID != toolSessionID {
			continue
		}
		switch event.EventType {
		case "evidence.proposed", "claim.proposed":
			count++
		}
	}
	return count
}

func (server *Server) runManualAgentCompaction(
	ctx context.Context,
	missionID string,
	userEventID string,
	recall recallPreview,
	executorName string,
	mcpMode string,
	toolSessionID string,
) (app.LedgerEvent, error) {
	previousSessionID := server.latestAgentSessionID(ctx, missionID, executorName)
	if previousSessionID == "" {
		return server.service.AppendEvent(ctx, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
			EventID:        newID("evt"),
			MissionID:      missionID,
			Kind:           "agent_compaction_skipped",
			AgentExecutor:  executorName,
			MCPMode:        mcpMode,
			IncludeMCPMode: true,
			Text:           "압축할 기존 에이전트 세션이 없습니다.",
			UserEventID:    userEventID,
			Producer:       app.Producer{Type: "agent", ID: executorName},
		}))
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return server.service.AppendEvent(ctx, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
			EventID:        newID("evt"),
			MissionID:      missionID,
			Kind:           "placeholder",
			AgentExecutor:  executorName,
			MCPMode:        mcpMode,
			IncludeMCPMode: true,
			Text:           "에이전트 실행기가 아직 연결되지 않았습니다. 수동 압축 요청은 장부에 기록했습니다.",
			UserEventID:    userEventID,
			Producer:       app.Producer{Type: "agent", ID: executorName},
		}))
	}
	agentModel := server.latestAgentSessionModel(ctx, missionID, executorName)
	agentReasoningEffort := server.latestAgentReasoningEffort(ctx, missionID, executorName)
	agentModel, agentReasoningEffort, err := resolveAgentSettings(executorName, agentModel, agentReasoningEffort, previousSessionID)
	if err != nil {
		return app.LedgerEvent{}, err
	}
	started := time.Now()
	result, err := executor.Run(ctx, AgentRequest{
		UserText:          "compact session context",
		Prompt:            agentCompactPrompt(recall),
		Model:             agentModel,
		ReasoningEffort:   agentReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		UserEventID:       userEventID,
		PreviousSessionID: previousSessionID,
		AgentExecutor:     executorName,
		MCPMode:           mcpMode,
		Compaction:        true,
	})
	durationMS := time.Since(started).Milliseconds()
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return app.LedgerEvent{}, err
		}
		return server.appendAgentError(ctx, missionID, userEventID, executorName, err, result, durationMS, map[string]any{
			"previous_agent_session_id": previousSessionID,
			"manual_compaction":         true,
			"tool_session_id":           toolSessionID,
			"agent_model":               agentModel,
			"agent_reasoning_effort":    agentReasoningEffort,
			"text":                      "에이전트 세션 압축 요청이 실패했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validatedSameSessionResult(result, previousSessionID)
	if err != nil {
		return server.appendAgentError(ctx, missionID, userEventID, executorName, err, result, durationMS, map[string]any{
			"previous_agent_session_id": previousSessionID,
			"returned_agent_session_id": returnedSessionID,
			"manual_compaction":         true,
			"tool_session_id":           toolSessionID,
			"agent_model":               agentModel,
			"agent_reasoning_effort":    agentReasoningEffort,
			"text":                      "에이전트 세션 압축 요청이 다른 세션 ID를 반환했습니다. 새 세션으로 자동 전환하지 않았습니다.",
		})
	}
	compactEvent, err := server.service.AppendEvent(ctx, conversation.BuildTurnAgentCompactedAppendRequest(conversation.TurnAgentCompactedEventRequest{
		EventID:                newID("evt"),
		MissionID:              missionID,
		AgentExecutor:          executorName,
		AgentModel:             agentModel,
		AgentReasoningEffort:   agentReasoningEffort,
		MCPMode:                mcpMode,
		AgentSessionID:         result.SessionID,
		PreviousAgentSessionID: previousSessionID,
		ToolSessionID:          toolSessionID,
		Summary:                result.Text,
		DurationMS:             durationMS,
		UserEventID:            userEventID,
		Manual:                 true,
		Usage:                  result.Usage,
		Resumed:                result.Resumed,
		Producer:               app.Producer{Type: "agent", ID: executorName},
	}))
	if err != nil {
		return app.LedgerEvent{}, err
	}
	return server.service.AppendEvent(ctx, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
		EventID:               newID("evt"),
		MissionID:             missionID,
		Kind:                  "agent_compacted",
		AgentExecutor:         executorName,
		AgentModel:            agentModel,
		AgentReasoningEffort:  agentReasoningEffort,
		IncludeAgentConfig:    true,
		MCPMode:               mcpMode,
		IncludeMCPMode:        true,
		Text:                  "에이전트 세션 압축 요청을 완료했습니다. 같은 세션에서 다음 턴을 이어갈 수 있습니다.",
		AgentSessionID:        result.SessionID,
		IncludeAgentSessionID: true,
		DurationMS:            durationMS,
		IncludeDuration:       true,
		UserEventID:           userEventID,
		Extra: map[string]any{
			"previous_agent_session_id": previousSessionID,
			"compaction_event_id":       compactEvent.EventID,
			"tool_session_id":           toolSessionID,
			"summary":                   result.Text,
		},
		Usage:                  result.Usage,
		UsageSurface:           "compaction",
		UsagePreviousSessionID: previousSessionID,
		UsageCompaction:        true,
		Producer:               app.Producer{Type: "agent", ID: executorName},
	}))
}

func (server *Server) appendAgentSuccess(
	ctx context.Context,
	missionID string,
	userEventID string,
	executorName string,
	mcpMode string,
	result AgentResult,
	durationMS int64,
	extra map[string]any,
) (app.LedgerEvent, error) {
	previousSessionID, _ := extra["previous_agent_session_id"].(string)
	compactionAttempted, _ := extra["compaction_attempted"].(bool)
	surface := "turn"
	agentEventID := newID("evt")
	eventReqs := []app.AppendEventRequest{conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
		EventID:                agentEventID,
		MissionID:              missionID,
		Kind:                   "agent_response",
		AgentExecutor:          executorName,
		MCPMode:                mcpMode,
		IncludeMCPMode:         true,
		Text:                   result.Text,
		AgentSessionID:         result.SessionID,
		IncludeAgentSessionID:  true,
		Resumed:                result.Resumed,
		IncludeResumed:         true,
		DurationMS:             durationMS,
		IncludeDuration:        true,
		UserEventID:            userEventID,
		Extra:                  extra,
		Usage:                  result.Usage,
		UsageSurface:           surface,
		UsagePreviousSessionID: previousSessionID,
		UsageCompaction:        compactionAttempted,
		Producer:               app.Producer{Type: "agent", ID: executorName},
	})}
	candidateReq := sourceCandidateEventRequestFromAgentResult(missionID, userEventID, agentEventID, executorName, mcpMode, result.Text, extra)
	if candidateReq != nil {
		eventReqs = append(eventReqs, *candidateReq)
	}
	agentEvents, err := server.service.AppendEvents(ctx, missionID, eventReqs)
	if err != nil {
		return app.LedgerEvent{}, err
	}
	if candidateReq != nil && len(agentEvents) > 1 {
		server.stageSourceCandidateProposalEvent(context.Background(), agentEvents[1])
	}
	return agentEvents[0], nil
}

func sourceCandidateEventRequestFromAgentResult(
	missionID string,
	userEventID string,
	agentEventID string,
	executorName string,
	mcpMode string,
	text string,
	extra map[string]any,
) *app.AppendEventRequest {
	candidates := sourceCandidatesFromText(text)
	if len(candidates) == 0 {
		return nil
	}
	appCandidates := make([]sourcecandidates.SourceCandidateProposal, 0, len(candidates))
	for _, candidate := range candidates {
		appCandidates = append(appCandidates, sourcecandidates.SourceCandidateProposal{
			URL:    candidate.URL,
			Title:  candidate.Title,
			Reason: candidate.Reason,
			State:  candidate.State,
		})
	}
	req := sourcecandidates.SourceCandidateProposalEventRequest{
		EventID:      newID("evt"),
		MissionID:    missionID,
		UserEventID:  userEventID,
		AgentEventID: agentEventID,
		ExecutorName: executorName,
		MCPMode:      mcpMode,
		Producer:     app.Producer{Type: "agent", ID: executorName},
		Candidates:   appCandidates,
	}
	if toolSessionID, ok := extra["tool_session_id"].(string); ok && strings.TrimSpace(toolSessionID) != "" {
		req.ToolSessionID = strings.TrimSpace(toolSessionID)
	}
	if strategyID, ok := extra["strategy_id"].(string); ok && strings.TrimSpace(strategyID) != "" {
		req.StrategyID = strings.TrimSpace(strategyID)
	}
	eventReq, ok, err := sourcecandidates.BuildSourceCandidateProposalEventRequest(req)
	if err != nil || !ok {
		return nil
	}
	return &eventReq
}

func (server *Server) stageSourceCandidateProposalEvent(ctx context.Context, event app.LedgerEvent) {
	var payload struct {
		ToolSessionID string            `json:"tool_session_id"`
		AgentExecutor string            `json:"agent_executor"`
		Candidates    []sourceCandidate `json:"candidates"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil || len(payload.Candidates) == 0 {
		return
	}
	for _, candidate := range payload.Candidates {
		candidate := candidate
		normalizedURL, err := normalizedHTTPURL(candidate.URL)
		if err != nil {
			continue
		}
		candidate.URL = normalizedURL
		candidateKind := sourceCandidateKindForURL(normalizedURL)
		started, err := sourcecandidates.StartStaging(ctx, server.service, sourcecandidates.SourceCandidateStagingStartRequest{
			EventID:          newID("evt"),
			MissionID:        event.MissionID,
			SessionID:        strings.TrimSpace(payload.ToolSessionID),
			ProposalEventID:  event.EventID,
			CausationEventID: event.EventID,
			CandidateKind:    candidateKind,
			Candidate: sourcecandidates.SourceCandidateProposal{
				URL:    candidate.URL,
				Title:  candidate.Title,
				Reason: candidate.Reason,
				State:  candidate.State,
			},
			Producer:      event.Producer,
			AgentExecutor: strings.TrimSpace(payload.AgentExecutor),
		})
		if err != nil {
			continue
		}
		go server.stageSourceCandidateBody(context.Background(), sourcecandidates.SourceCandidateStagingJob{
			MissionID:       event.MissionID,
			SessionID:       strings.TrimSpace(payload.ToolSessionID),
			ProposalEventID: event.EventID,
			CandidateKind:   candidateKind,
			Candidate: sourcecandidates.SourceCandidateProposal{
				URL:    candidate.URL,
				Title:  candidate.Title,
				Reason: candidate.Reason,
				State:  candidate.State,
			},
			Producer:       event.Producer,
			StartedEventID: started.Event.EventID,
			AgentExecutor:  strings.TrimSpace(payload.AgentExecutor),
		})
	}
}

func (server *Server) stageSourceCandidateBody(ctx context.Context, job sourcecandidates.SourceCandidateStagingJob) {
	_ = sourcecandidates.Stage(ctx, server.service, sourcecandidates.SourceCandidateStageRequest{
		Job:              job,
		Fetcher:          server.sourceCandidateFetcher(job.MissionID),
		NewArtifactID:    newID,
		NewEventID:       newID,
		FilenameFallback: "source",
	})
}

func (server *Server) sourceCandidateFetcher(missionID string) sourcecandidates.SourceCandidateFetcher {
	return func(ctx context.Context, rawURL string) (sourcecandidates.SourceCandidateFetched, error) {
		if fetched, ok, err := server.fetchConfluenceSourceCandidate(ctx, missionID, rawURL); ok {
			return fetched, err
		}
		fetched, err := server.fetchURLSource(ctx, rawURL)
		if err != nil {
			return sourcecandidates.SourceCandidateFetched{}, err
		}
		return sourcecandidates.SourceCandidateFetched{
			Content:           fetched.Content,
			MediaType:         fetched.MediaType,
			Title:             fetched.Title,
			ExternalVersion:   fetched.ExternalVersion,
			ExternalUpdatedAt: fetched.ExternalUpdatedAt,
			ByteSize:          fetched.ByteSize,
			PageCount:         fetched.PageCount,
			TextLength:        fetched.TextLength,
			TextLengthKnown:   fetched.TextLengthKnown,
		}, nil
	}
}

func sourceCandidateKindForURL(rawURL string) string {
	if _, ok, _ := parseConfluencePageURL(rawURL); ok {
		return "confluence_url"
	}
	return "url"
}

func (server *Server) fetchConfluenceSourceCandidate(ctx context.Context, missionID string, rawURL string) (sourcecandidates.SourceCandidateFetched, bool, error) {
	target, ok, err := parseConfluencePageURL(rawURL)
	if !ok {
		return sourcecandidates.SourceCandidateFetched{}, false, nil
	}
	if err != nil {
		return sourcecandidates.SourceCandidateFetched{}, true, err
	}
	access, err := server.service.GetMissionConnectorAccess(ctx, strings.TrimSpace(missionID), app.ConfluenceConnectorID)
	if err != nil {
		return sourcecandidates.SourceCandidateFetched{}, true, err
	}
	if !access.Enabled {
		return sourcecandidates.SourceCandidateFetched{}, true, fmt.Errorf("%w: 이 미션에서 Confluence 접근이 켜져 있지 않습니다. Sources의 Confluence 접근 허용을 먼저 설정하세요.", app.ErrInvalidInput)
	}
	if access.Status != app.ConnectorAccessStatusEnabled {
		return sourcecandidates.SourceCandidateFetched{}, true, fmt.Errorf("%w: 이 미션의 Confluence 접근 설정이 유효하지 않습니다: %s", app.ErrInvalidInput, access.InvalidReason)
	}
	if strings.TrimSpace(access.CloudID) != target.CloudID {
		return sourcecandidates.SourceCandidateFetched{}, true, fmt.Errorf("%w: 이 미션에 허용된 Confluence site와 후보 URL의 site가 일치하지 않습니다.", app.ErrInvalidInput)
	}
	connection, err := server.confluenceConnectionForPageURL(ctx, target, access.ConnectionID, access.CloudID)
	if err != nil {
		return sourcecandidates.SourceCandidateFetched{}, true, err
	}
	connector, err := server.confluenceClientForConnection(connection, target.CloudID)
	if err != nil {
		return sourcecandidates.SourceCandidateFetched{}, true, err
	}
	page, err := connector.ReadConfluenceSource(ctx, app.ConfluenceSourceReadRequest{CloudID: target.CloudID, PageID: target.PageID})
	if err != nil {
		return sourcecandidates.SourceCandidateFetched{}, true, err
	}
	if strings.TrimSpace(page.PageID) != "" && strings.TrimSpace(page.PageID) != target.PageID {
		return sourcecandidates.SourceCandidateFetched{}, true, fmt.Errorf("%w: Confluence page id가 후보 URL과 일치하지 않습니다.", app.ErrInvalidInput)
	}
	if page.SiteURL != "" && webConfluenceURLHost(page.SiteURL) != "" && webConfluenceURLHost(page.SiteURL) != webConfluenceURLHost(target.SiteURL) {
		return sourcecandidates.SourceCandidateFetched{}, true, fmt.Errorf("%w: Confluence page site가 후보 URL의 site와 일치하지 않습니다.", app.ErrInvalidInput)
	}
	content := strings.TrimSpace(page.PlainText)
	if content == "" {
		content = strings.TrimSpace(page.BodyStorage)
	}
	if content == "" {
		return sourcecandidates.SourceCandidateFetched{}, true, fmt.Errorf("%w: Confluence page content is empty", app.ErrInvalidInput)
	}
	title := strings.TrimSpace(page.Title)
	if title == "" {
		title = target.RawURL
	}
	return sourcecandidates.SourceCandidateFetched{
		CandidateKind:     "confluence_url",
		Content:           []byte(content),
		MediaType:         "text/plain; charset=utf-8",
		Title:             title,
		ExternalVersion:   page.Connector.ExternalVersion,
		ExternalUpdatedAt: page.UpdatedAt,
		ByteSize:          int64(len([]byte(content))),
		TextLength:        len([]rune(content)),
		TextLengthKnown:   true,
	}, true, nil
}

func appFetchedURLSource(fetched fetchedURLSource) sourceingest.FetchedURLSource {
	return sourceingest.FetchedURLSource{
		Content:           fetched.Content,
		MediaType:         fetched.MediaType,
		Title:             fetched.Title,
		ExternalVersion:   fetched.ExternalVersion,
		ExternalUpdatedAt: fetched.ExternalUpdatedAt,
		ByteSize:          fetched.ByteSize,
		PageCount:         fetched.PageCount,
		TextLength:        fetched.TextLength,
		TextLengthKnown:   fetched.TextLengthKnown,
	}
}

func appFetchedPDFSource(fetched fetchedPDFSource) sourceingest.FetchedURLSource {
	return sourceingest.FetchedURLSource{
		Content:           fetched.Content,
		MediaType:         fetched.MediaType,
		Title:             fetched.Title,
		ExternalVersion:   fetched.ExternalVersion,
		ExternalUpdatedAt: fetched.ExternalUpdatedAt,
		ByteSize:          fetched.ByteSize,
		PageCount:         fetched.PageCount,
		TextLength:        fetched.TextLength,
		TextLengthKnown:   fetched.TextLengthKnown,
	}
}

func appFetchedMediaSource(fetched fetchedMediaSource) sourceingest.FetchedMediaSource {
	return sourceingest.FetchedMediaSource{
		Content:           fetched.Content,
		MediaType:         fetched.MediaType,
		MediaKind:         fetched.MediaKind,
		Title:             fetched.Title,
		ExternalVersion:   fetched.ExternalVersion,
		ExternalUpdatedAt: fetched.ExternalUpdatedAt,
		ByteSize:          fetched.ByteSize,
		Width:             fetched.Width,
		Height:            fetched.Height,
	}
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
		return result, fmt.Errorf("%w: agent did not return a session id for resumed session", app.ErrInvalidInput)
	}
	if result.SessionID != previousSessionID {
		result.SessionID = ""
		return result, fmt.Errorf("%w: agent returned a different session id", app.ErrInvalidInput)
	}
	return result, nil
}

func isManualCompactCommand(text string) bool {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "/compact", "compact":
		return true
	default:
		return false
	}
}

func shouldAutoCompactAfterAgentError(previousSessionID string, err error, result AgentResult) bool {
	if strings.TrimSpace(previousSessionID) == "" || err == nil {
		return false
	}
	text := strings.ToLower(err.Error() + "\n" + result.Log)
	return strings.Contains(text, "ran out of room in the model's context window")
}

func (server *Server) latestAgentSessionID(ctx context.Context, missionID string, executorName string) string {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return ""
	}
	return conversation.LatestAgentSessionID(events, executorName)
}

func (server *Server) latestAgentSessionModel(ctx context.Context, missionID string, executorName string) string {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return ""
	}
	return conversation.LatestAgentModel(events, executorName)
}

func (server *Server) latestAgentReasoningEffort(ctx context.Context, missionID string, executorName string) string {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return ""
	}
	return conversation.LatestAgentReasoningEffort(events, executorName)
}

func resolveAgentSettings(executorName, model, effort, previousSessionID string) (string, string, error) {
	if executorName != "codex" {
		return strings.TrimSpace(model), "", nil
	}
	model, effort, err := agentmodels.ResolveForSession(model, effort, previousSessionID)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", app.ErrInvalidInput, err)
	}
	return model, effort, nil
}

func (server *Server) hasOpenAgentTurn(ctx context.Context, missionID string) bool {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return false
	}
	return hasOpenAgentPending(events)
}

func (server *Server) latestOpenAgentPending(ctx context.Context, missionID string) (openAgentPending, bool) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return openAgentPending{}, false
	}
	return latestOpenAgentPendingInEvents(events, "")
}

func latestOpenAgentPendingInEvents(events []app.LedgerEvent, workflowRunID string) (openAgentPending, bool) {
	return conversation.LatestOpenAgentPending(events, workflowRunID)
}

func agentPendingForUserEventInEvents(events []app.LedgerEvent, userEventID string) (openAgentPending, bool) {
	return conversation.AgentPendingForUserEvent(events, userEventID)
}

func (server *Server) reconcileStaleAgentTurn(ctx context.Context, missionID string) error {
	pending, ok := server.latestOpenAgentPending(ctx, missionID)
	if !ok {
		return nil
	}
	if server.runningTurns.has(missionID) {
		return nil
	}
	if strings.TrimSpace(pending.WorkflowRunID) != "" && server.runningWorkflow.has(pending.WorkflowRunID) {
		return nil
	}
	workflowTerminalEventType := ""
	if strings.TrimSpace(pending.WorkflowRunID) != "" {
		workflowTerminalEventType = app.WorkflowRunInterruptedEvent
		server.runningWorkflow.cancel(pending.WorkflowRunID)
	}
	_, err := server.appendAgentCanceledWithWorkflowTerminal(ctx, missionID, pending.UserEventID, pending.AgentExecutor, "서버 재시작 또는 연결 중단 뒤 장부에 남은 오래된 에이전트 대기 상태를 자동 정리했습니다.", workflowTerminalEventType)
	if err != nil {
		return err
	}
	return nil
}

func (server *Server) appendAgentError(
	ctx context.Context,
	missionID string,
	userEventID string,
	executor string,
	cause error,
	result AgentResult,
	durationMS int64,
	extra map[string]any,
) (app.LedgerEvent, error) {
	if server.hasAgentTerminalEvent(ctx, missionID, userEventID) {
		return app.LedgerEvent{}, nil
	}
	text := "Agent failed: " + cause.Error()
	if override, ok := extra["text"].(string); ok && strings.TrimSpace(override) != "" {
		text = override
		delete(extra, "text")
	}
	explicitUsageSurface, _ := extra["agent_usage_surface"].(string)
	delete(extra, "agent_usage_surface")
	previousSessionID, _ := extra["previous_agent_session_id"].(string)
	compactionAttempted, _ := extra["compaction_attempted"].(bool)
	manualCompaction, _ := extra["manual_compaction"].(bool)
	surface := "turn"
	if compactionAttempted || manualCompaction {
		surface = "compaction"
	}
	if strings.TrimSpace(explicitUsageSurface) != "" {
		surface = strings.TrimSpace(explicitUsageSurface)
	}
	return server.service.AppendEvent(ctx, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
		EventID:                newID("evt"),
		MissionID:              missionID,
		Kind:                   "agent_error",
		AgentExecutor:          executor,
		Text:                   text,
		Error:                  cause.Error(),
		IncludeError:           true,
		LogExcerpt:             headTailExcerpt(result.Log, 4000),
		IncludeLogExcerpt:      true,
		AgentSessionID:         result.SessionID,
		IncludeAgentSessionID:  result.SessionID != "",
		Resumed:                result.Resumed,
		IncludeResumed:         true,
		DurationMS:             durationMS,
		IncludeDuration:        true,
		UserEventID:            userEventID,
		Extra:                  extra,
		Usage:                  result.Usage,
		UsageSurface:           surface,
		UsagePreviousSessionID: previousSessionID,
		UsageCompaction:        compactionAttempted || manualCompaction,
		Producer:               app.Producer{Type: "agent", ID: executor},
	}))
}

func addAgentUsagePayload(payload map[string]any, usage agentusage.AgentUsage, surface string, durationMS int64, previousSessionID string, sessionID string, resumed bool, compaction bool) {
	if eventUsage, ok := usage.ForEvent(surface, durationMS, previousSessionID, sessionID, resumed, compaction); ok {
		payload["agent_usage"] = eventUsage
	}
}

type reportFailureWithPayload struct {
	cause   error
	payload map[string]any
}

func reportAgentFailure(cause error, result AgentResult, surface string, durationMS int64, previousSessionID string) error {
	if cause == nil {
		return nil
	}
	payload := map[string]any{
		"failed_surface": surface,
	}
	if strings.TrimSpace(result.SessionID) != "" {
		payload["agent_session_id"] = result.SessionID
	}
	payload["resumed"] = result.Resumed
	addAgentUsagePayload(payload, result.Usage, surface, durationMS, previousSessionID, result.SessionID, result.Resumed, false)
	return reportFailureWithPayload{cause: cause, payload: payload}
}

func (server *Server) stopWorkflowRunNow(ctx context.Context, missionID string, workflowRunID string, reason string) (app.LedgerEvent, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return app.LedgerEvent{}, err
	}
	if pending, ok := latestOpenAgentPendingInEvents(events, workflowRunID); ok {
		return server.appendAgentCanceledWithWorkflowTerminal(ctx, missionID, pending.UserEventID, pending.AgentExecutor, reason, app.WorkflowRunStoppedEvent)
	}
	req, ok, err := app.BuildWorkflowRunTerminalAppendRequest(events, app.WorkflowRunTerminalEventRequest{
		WorkflowRunID: workflowRunID,
		MissionID:     missionID,
		EventType:     app.WorkflowRunStoppedEvent,
		Reason:        reason,
	})
	if err != nil || !ok {
		return app.LedgerEvent{}, err
	}
	appended, err := server.service.AppendEvents(ctx, missionID, []app.AppendEventRequest{req})
	if err != nil {
		return app.LedgerEvent{}, err
	}
	if len(appended) == 0 {
		return app.LedgerEvent{}, nil
	}
	return appended[0], nil
}

func (server *Server) appendAgentCanceled(ctx context.Context, missionID string, userEventID string, executor string, text string) (app.LedgerEvent, error) {
	return server.appendAgentCanceledWithWorkflowTerminal(ctx, missionID, userEventID, executor, text, "")
}

func (server *Server) appendAgentCanceledWithWorkflowTerminal(ctx context.Context, missionID string, userEventID string, executor string, text string, workflowTerminalEventType string) (app.LedgerEvent, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return app.LedgerEvent{}, err
	}
	pending, _ := agentPendingForUserEventInEvents(events, userEventID)
	executor = firstNonEmpty(strings.TrimSpace(executor), pending.AgentExecutor, "codex")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	reqs := make([]app.AppendEventRequest, 0, 2)
	if !hasAgentTerminalEventInEvents(events, userEventID) {
		extra := map[string]any{
			"canceled_at": now,
		}
		if strings.TrimSpace(pending.WorkflowRunID) != "" {
			extra["workflow_run_id"] = pending.WorkflowRunID
		}
		if strings.TrimSpace(pending.WorkflowStepID) != "" {
			extra["workflow_step_id"] = pending.WorkflowStepID
		}
		reqs = append(reqs, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
			EventID:       newID("evt"),
			MissionID:     missionID,
			Kind:          "agent_canceled",
			AgentExecutor: executor,
			Text:          text,
			UserEventID:   userEventID,
			Extra:         extra,
			Producer:      app.Producer{Type: "agent", ID: executor},
		}))
	}
	if strings.TrimSpace(workflowTerminalEventType) != "" && strings.TrimSpace(pending.WorkflowRunID) != "" {
		req, ok, err := app.BuildWorkflowRunTerminalAppendRequest(events, app.WorkflowRunTerminalEventRequest{
			WorkflowRunID: pending.WorkflowRunID,
			MissionID:     missionID,
			EventType:     workflowTerminalEventType,
			Reason:        text,
		})
		if err != nil {
			return app.LedgerEvent{}, err
		}
		if ok {
			reqs = append(reqs, req)
		}
	}
	if len(reqs) == 0 {
		return app.LedgerEvent{}, nil
	}
	appended, err := server.service.AppendEvents(ctx, missionID, reqs)
	if err != nil {
		return app.LedgerEvent{}, err
	}
	if len(appended) == 0 {
		return app.LedgerEvent{}, nil
	}
	return appended[0], nil
}

func (server *Server) hasAgentTerminalEvent(ctx context.Context, missionID string, userEventID string) bool {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return false
	}
	return hasAgentTerminalEventInEvents(events, userEventID)
}

func hasAgentTerminalEventInEvents(events []app.LedgerEvent, userEventID string) bool {
	return conversation.HasAgentTerminalEventForUser(events, userEventID)
}

func hasOpenAgentPending(events []app.LedgerEvent) bool {
	return conversation.HasOpenAgentPending(events)
}

func hasOpenReportDraftPending(events []app.LedgerEvent) bool {
	_, ok := latestOpenReportDraftPendingEvent(events)
	return ok
}

func latestOpenReportDraftPendingEvent(events []app.LedgerEvent) (app.LedgerEvent, bool) {
	completed := reporting.CompletedPendingEventIDs(events)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.draft.pending" && event.EventType != "report.design.pending" && event.EventType != "report.humanize.pending" && event.EventType != "report.patch.pending" {
			continue
		}
		if _, ok := completed[event.EventID]; !ok {
			return event, true
		}
	}
	return app.LedgerEvent{}, false
}

func reportDraftPendingEventID(event app.LedgerEvent) string {
	var payload struct {
		PendingEventID string         `json:"pending_event_id"`
		Generation     map[string]any `json:"generation"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return ""
	}
	if pendingEventID := strings.TrimSpace(payload.PendingEventID); pendingEventID != "" {
		return pendingEventID
	}
	if payload.Generation == nil {
		return ""
	}
	pendingEventID, _ := payload.Generation["pending_event_id"].(string)
	return strings.TrimSpace(pendingEventID)
}

func reportDraftPendingExecutor(event app.LedgerEvent) string {
	var payload struct {
		AgentExecutor string `json:"agent_executor"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return "plasma"
	}
	executor := strings.TrimSpace(payload.AgentExecutor)
	if executor == "" {
		return "plasma"
	}
	return executor
}

func reportDraftPendingMode(event app.LedgerEvent) string {
	var payload struct {
		ReportMode string `json:"report_mode"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return defaultReportMode
	}
	mode, err := normalizeReportMode(payload.ReportMode)
	if err != nil {
		return defaultReportMode
	}
	return mode
}

func (server *Server) agentExecutor(name string) AgentExecutor {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		name = "codex"
	}
	if executor, ok := server.agents[name]; ok {
		return executor
	}
	if name == "codex" {
		return server.agent
	}
	return nil
}

func (server *Server) agentStatuses() []agentExecutorStatus {
	statuses := []agentExecutorStatus{
		agentExecutorStatusFor("codex", "Codex", server.agentExecutor("codex")),
		agentExecutorStatusFor("claude", "Claude", server.agentExecutor("claude")),
	}
	return statuses
}

func agentExecutorStatusFor(name string, label string, executor AgentExecutor) agentExecutorStatus {
	defaultModel, defaultModelLabel, defaultModelVersion := agentDefaultModelMetadata(name, executor)
	status := agentExecutorStatus{
		Name:                name,
		Label:               label,
		Configured:          executor != nil,
		DefaultModel:        defaultModel,
		DefaultModelLabel:   defaultModelLabel,
		DefaultModelVersion: defaultModelVersion,
	}
	switch name {
	case "codex":
		status.ReasoningEffortSupported = true
		status.DefaultReasoningEffort = agentmodels.DefaultReasoningEffort
		for _, model := range agentmodels.Catalog() {
			status.Models = append(status.Models, agentModelCapability{Name: model.Name, Label: model.Label, ReasoningEfforts: model.ReasoningEfforts})
		}
	case "claude":
		status.ReasoningEffortSupported = false
		status.ReasoningEffortNote = "Claude 실행기는 아직 추론 강도 지정을 지원하지 않습니다."
	}
	return status
}

func agentDefaultModelMetadata(name string, executor AgentExecutor) (string, string, string) {
	switch name {
	case "codex":
		model := agentmodels.Default()
		return model.Name, model.Label, model.Name
	case "claude":
		model := "haiku"
		switch typed := executor.(type) {
		case ClaudeExecutor:
			if strings.TrimSpace(typed.Model) != "" {
				model = strings.TrimSpace(typed.Model)
			}
		case *ClaudeExecutor:
			if typed != nil && strings.TrimSpace(typed.Model) != "" {
				model = strings.TrimSpace(typed.Model)
			}
		}
		return model, claudeModelDisplayName(model), model
	default:
		return "", "", ""
	}
}

func claudeModelDisplayName(model string) string {
	switch strings.TrimSpace(strings.ToLower(model)) {
	case "haiku":
		return "Claude Haiku"
	case "sonnet":
		return "Claude Sonnet"
	case "opus":
		return "Claude Opus"
	default:
		if strings.TrimSpace(model) == "" {
			return ""
		}
		return strings.TrimSpace(model)
	}
}

func (server *Server) validateMissionAgentExecutor(ctx context.Context, missionID string, requested string) error {
	requested, err := normalizeAgentExecutorName(requested)
	if err != nil {
		return err
	}
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return err
	}
	return app.ValidateMissionAgentExecutorForEvents(events, requested)
}

func normalizeAgentExecutorName(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "codex", nil
	}
	switch value {
	case "codex", "claude":
		return value, nil
	default:
		return "", fmt.Errorf("%w: unsupported agent executor %q", app.ErrInvalidInput, value)
	}
}

func normalizeAgentModelName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) > 128 {
		return "", fmt.Errorf("%w: agent model is too long", app.ErrInvalidInput)
	}
	if strings.ContainsAny(value, "\r\n\t") {
		return "", fmt.Errorf("%w: agent model must be a single-line value", app.ErrInvalidInput)
	}
	return value, nil
}

func normalizeAgentReasoningEffort(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) > 128 {
		return "", fmt.Errorf("%w: agent reasoning effort is too long", app.ErrInvalidInput)
	}
	if strings.ContainsAny(value, "\r\n\t") {
		return "", fmt.Errorf("%w: agent reasoning effort must be a single-line value", app.ErrInvalidInput)
	}
	return value, nil
}

func normalizeMCPMode(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "auto", nil
	}
	switch value {
	case "explicit", "auto":
		return value, nil
	default:
		return "", fmt.Errorf("%w: unsupported MCP mode %q", app.ErrInvalidInput, value)
	}
}
