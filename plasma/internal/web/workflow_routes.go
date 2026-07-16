package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	workflowruntime "github.com/c86j224s/liquid2/plasma/internal/workflow"
)

func (server *Server) handleMissionWorkflows(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			runs, err := server.service.ListWorkflowRuns(r.Context(), missionID)
			if err != nil {
				writeAppError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"workflow_runs": runs})
		case http.MethodPost:
			var req workflowStartRequest
			if !decodeJSON(w, r, &req) {
				return
			}
			view, err := server.startWorkflow(r.Context(), missionID, req)
			if err != nil {
				writeAppError(w, err)
				return
			}
			writeJSON(w, http.StatusAccepted, map[string]any{"workflow_run": view})
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	if len(rest) == 1 && rest[0] == "goal_draft" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req workflowGoalDraftRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		draft, err := server.draftWorkflowGoal(r.Context(), missionID, req)
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"workflow_goal_draft": draft})
		return
	}
	if len(rest) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		view, err := server.service.GetWorkflowRun(r.Context(), missionID, rest[0])
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"workflow_run": view})
		return
	}
	if len(rest) == 2 && rest[1] == "stop" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		reason := "사용자가 웹에서 워크플로우 정지를 요청했습니다."
		view, err := server.service.RequestWorkflowStop(r.Context(), app.RequestWorkflowStopRequest{
			WorkflowRunID:      rest[0],
			MissionID:          missionID,
			RequestedBySurface: app.WorkflowSurfaceWeb,
			Reason:             reason,
		})
		if err != nil {
			writeAppError(w, err)
			return
		}
		if view.Status == app.WorkflowStatusStopping {
			server.runningWorkflow.cancel(view.WorkflowRunID)
			if _, err := server.stopWorkflowRunNow(r.Context(), missionID, view.WorkflowRunID, reason); err != nil {
				writeAppError(w, err)
				return
			}
			view, err = server.service.GetWorkflowRun(r.Context(), missionID, view.WorkflowRunID)
			if err != nil {
				writeAppError(w, err)
				return
			}
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"workflow_run": view})
		return
	}
	http.NotFound(w, r)
}

func (server *Server) draftWorkflowGoal(ctx context.Context, missionID string, req workflowGoalDraftRequest) (workflowGoalDraftResponse, error) {
	userInstructionRaw := strings.TrimSpace(req.UserInstructionRaw)
	if userInstructionRaw == "" {
		return workflowGoalDraftResponse{}, fmt.Errorf("%w: user instruction is required", app.ErrInvalidInput)
	}
	if len(userInstructionRaw) > app.WorkflowInstructionLimit {
		return workflowGoalDraftResponse{}, fmt.Errorf("%w: user instruction is longer than %d bytes", app.ErrInvalidInput, app.WorkflowInstructionLimit)
	}
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return workflowGoalDraftResponse{}, err
	}
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return workflowGoalDraftResponse{}, fmt.Errorf("%w: workflow goal draft requires a configured agent executor", app.ErrInvalidInput)
	}
	if err := server.validateMissionAgentExecutor(ctx, missionID, executorName); err != nil {
		return workflowGoalDraftResponse{}, err
	}
	projection, err := server.service.GetProjection(ctx, missionID)
	if err != nil {
		return workflowGoalDraftResponse{}, err
	}
	defaults, err := server.workflowGoalDefaults(ctx)
	if err != nil {
		return workflowGoalDraftResponse{}, err
	}
	prompt := workflowGoalDraftPrompt(projection, userInstructionRaw)
	result, err := executor.Run(ctx, AgentRequest{
		UserText:        userInstructionRaw,
		Prompt:          prompt,
		Model:           defaults.WorkflowGoalModel,
		ReasoningEffort: defaults.WorkflowGoalReasoningEffort,
		MissionID:       missionID,
		ToolSessionID:   newID("ses"),
		AgentExecutor:   executorName,
		MCPMode:         "explicit",
	})
	if err != nil {
		return workflowGoalDraftResponse{}, err
	}
	rawJSON, err := extractAgentJSONObject(result.Text)
	if err != nil {
		return workflowGoalDraftResponse{}, err
	}
	var draft workflowGoalDraftResponse
	if err := json.Unmarshal([]byte(rawJSON), &draft); err != nil {
		return workflowGoalDraftResponse{}, fmt.Errorf("%w: workflow goal draft JSON could not be parsed", app.ErrInvalidInput)
	}
	draft.UserInstructionRaw = userInstructionRaw
	draft.RunGoal = strings.TrimSpace(draft.RunGoal)
	draft.StepInstruction = strings.TrimSpace(draft.StepInstruction)
	draft.Model = strings.TrimSpace(defaults.WorkflowGoalModel)
	draft.ReasoningEffort = strings.TrimSpace(defaults.WorkflowGoalReasoningEffort)
	if draft.RunGoal == "" || draft.StepInstruction == "" {
		return workflowGoalDraftResponse{}, fmt.Errorf("%w: workflow goal draft must include run_goal and step_instruction", app.ErrInvalidInput)
	}
	return draft, nil
}

func workflowGoalDraftPrompt(projection app.MissionProjection, userInstructionRaw string) string {
	return fmt.Sprintf(`You draft the kickoff for a Plasma autonomous investigation run.

Use Korean. Do not research the topic. Do not answer the user's substantive request. Do not call tools unless the runtime forces tool discovery.

Return only one JSON object with this exact shape:
{"run_goal":"...","step_instruction":"..."}

Mission title: %s
Mission objective: %s

User's original autonomous-run request:
%s

Rules:
- Preserve the user's original wording, breadth, ambiguity, and intentionally open possibilities.
- The run_goal is a working interpretation shown to the user before the run starts. It must not replace or narrow the original request.
- If the original request says things like "다각도로", "넓게", "가능성을 열어두고", keep that breadth in the run_goal.
- The step_instruction is only the first concrete investigation move. It may be narrower than the whole run, but must not close off the broader request.
- Do not insert facts, citations, conclusions, or source claims. This is planning text only.
- Keep both fields concise enough to be editable in a browser form.`, strings.TrimSpace(projection.Title), strings.TrimSpace(projection.Objective), userInstructionRaw)
}

func (server *Server) startWorkflow(ctx context.Context, missionID string, req workflowStartRequest) (app.WorkflowRunView, error) {
	instruction := strings.TrimSpace(req.Instruction)
	if instruction == "" {
		return app.WorkflowRunView{}, fmt.Errorf("%w: workflow instruction is required", app.ErrInvalidInput)
	}
	server.reconcileWorkflowState(ctx, missionID)
	executorName, err := normalizeAgentExecutorName(req.AgentExecutor)
	if err != nil {
		return app.WorkflowRunView{}, err
	}
	if server.agentExecutor(executorName) == nil {
		return app.WorkflowRunView{}, fmt.Errorf("%w: workflow start requires a configured agent executor", app.ErrInvalidInput)
	}
	mcpMode, err := normalizeMCPMode(req.MCPMode)
	if err != nil {
		return app.WorkflowRunView{}, err
	}
	maxSteps := req.MaxSteps
	maxDurationMS := req.MaxDurationMS

	unlockReports := server.reports.lock(missionID)
	defer unlockReports()
	unlockTurns := server.turns.lock(missionID)
	defer unlockTurns()
	if err := server.validateMissionAgentExecutor(ctx, missionID, executorName); err != nil {
		return app.WorkflowRunView{}, err
	}
	if err := server.reconcileStaleAgentTurn(ctx, missionID); err != nil {
		return app.WorkflowRunView{}, err
	}
	if server.hasOpenReportDraft(ctx, missionID) {
		return app.WorkflowRunView{}, fmt.Errorf("%w: report draft is already running for this mission", app.ErrInvalidInput)
	}
	runs, err := server.service.ListWorkflowRuns(ctx, missionID)
	if err != nil {
		return app.WorkflowRunView{}, err
	}
	if active := activeWorkflowRun(runs); active != nil {
		return app.WorkflowRunView{}, fmt.Errorf("%w: workflow %s is already %s", app.ErrInvalidInput, active.WorkflowRunID, active.Status)
	}
	startAfterEventID := ""
	if pending, ok := server.latestOpenAgentPending(ctx, missionID); ok {
		startAfterEventID = pending.UserEventID
	}
	view, err := server.service.RequestWorkflowRun(ctx, app.RequestWorkflowRunRequest{
		WorkflowRunID:             strings.TrimSpace(req.WorkflowRunID),
		MissionID:                 missionID,
		RequestedBySurface:        app.WorkflowSurfaceWeb,
		AgentExecutor:             executorName,
		MCPMode:                   mcpMode,
		StepInstructionMode:       strings.TrimSpace(req.StepInstructionMode),
		UserInstructionRaw:        strings.TrimSpace(req.UserInstructionRaw),
		RunGoal:                   strings.TrimSpace(req.RunGoal),
		Instruction:               instruction,
		MaxSteps:                  maxSteps,
		MaxDurationMS:             maxDurationMS,
		StopCondition:             strings.TrimSpace(req.StopCondition),
		StartAfterEventID:         startAfterEventID,
		ArgumentSummary:           firstNonEmpty(strings.TrimSpace(req.RunGoal), instruction),
		ContinueFromWorkflowRunID: strings.TrimSpace(req.ContinueFromWorkflowRunID),
	})
	if err != nil {
		return app.WorkflowRunView{}, err
	}
	if startAfterEventID == "" {
		server.startWorkflowRunner(missionID, view.WorkflowRunID, executorName)
	}
	return view, nil
}

func (server *Server) startWorkflowRunner(missionID string, workflowRunID string, executorName string) {
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return
	}
	workerCtx, cancel := context.WithCancel(context.Background())
	runID, ok := server.runningWorkflow.start(workflowRunID, cancel)
	if !ok {
		cancel()
		return
	}
	go func() {
		defer cancel()
		defer server.runningWorkflow.finish(workflowRunID, runID)
		previousSessionID := server.latestAgentSessionID(context.Background(), missionID, executorName)
		model, effort, err := resolveAgentSettings(executorName,
			server.latestAgentSessionModel(context.Background(), missionID, executorName),
			server.latestAgentReasoningEffort(context.Background(), missionID, executorName), previousSessionID)
		if err != nil {
			return
		}
		runner := workflowruntime.Runner{
			Service:               server.service,
			Agent:                 workflowAgentAdapter{executor: executor},
			AgentModel:            model,
			ReasoningEffort:       effort,
			NewID:                 newID,
			SourceCandidateStager: server.stageSourceCandidateProposalEvent,
		}
		_, _ = runner.Run(workerCtx, missionID, workflowRunID)
	}()
}

func (server *Server) drainQueuedWorkflows(ctx context.Context, missionID string) {
	server.reconcileWorkflowState(ctx, missionID)
}

func (server *Server) reconcileWorkflowState(ctx context.Context, missionID string) {
	unlock := server.workflows.lock(missionID)
	defer unlock()
	runs, err := server.service.ListWorkflowRuns(ctx, missionID)
	if err != nil {
		return
	}
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return
	}
	for _, run := range runs {
		switch run.Status {
		case app.WorkflowStatusInterrupted, app.WorkflowStatusFailed, app.WorkflowStatusStopped:
			server.closeWorkflowOpenPending(ctx, run, events)
		case app.WorkflowStatusStopping:
			server.runningWorkflow.cancel(run.WorkflowRunID)
			if _, err := server.stopWorkflowRunNow(ctx, missionID, run.WorkflowRunID, firstNonEmpty(run.StopReason, "workflow stop requested")); err == nil {
				return
			}
		case app.WorkflowStatusQueued:
			if strings.TrimSpace(run.StartAfterEventID) == "" || !hasAgentTerminalEventInEvents(events, run.StartAfterEventID) {
				continue
			}
			if server.agentExecutor(run.AgentExecutor) == nil {
				continue
			}
			server.startWorkflowRunner(missionID, run.WorkflowRunID, run.AgentExecutor)
			return
		}
	}
}

func (server *Server) closeWorkflowOpenPending(ctx context.Context, run app.WorkflowRunView, events []app.LedgerEvent) {
	pending, ok := latestOpenAgentPendingInEvents(events, run.WorkflowRunID)
	if !ok {
		return
	}
	executor := firstNonEmpty(pending.AgentExecutor, run.AgentExecutor, "codex")
	text := fmt.Sprintf("자동조사 실행자가 사라져 열린 대기 상태를 정리했습니다. workflow=%s status=%s", run.WorkflowRunID, run.Status)
	_, _ = server.appendAgentCanceled(ctx, run.MissionID, pending.UserEventID, executor, text)
}

type workflowAgentAdapter struct {
	executor AgentExecutor
}

func (server *Server) activeWorkflowRun(ctx context.Context, missionID string) *app.WorkflowRunView {
	runs, err := server.service.ListWorkflowRuns(ctx, missionID)
	if err != nil {
		return nil
	}
	return activeWorkflowRun(runs)
}
