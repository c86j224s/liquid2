package web

import (
	"context"
	"errors"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	workflowruntime "github.com/c86j224s/liquid2/plasma/internal/workflow"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// Run은 웹 및 에이전트 어댑터의 실행 진입점이다. 호출자는 취소, 실패, 외부 부작용 범위를 해당 패키지 계약에 맞게 보존해야 한다.
func (adapter workflowAgentAdapter) Run(ctx context.Context, req workflowruntime.AgentRequest) (workflowruntime.AgentResult, error) {
	agentRequest := AgentRequest{
		UserText:          req.UserText,
		Prompt:            req.Prompt,
		Model:             req.Model,
		ReasoningEffort:   req.ReasoningEffort,
		MissionID:         req.MissionID,
		ToolSessionID:     req.ToolSessionID,
		UserEventID:       req.UserEventID,
		PreviousSessionID: req.PreviousSessionID,
		AgentExecutor:     req.AgentExecutor,
		MCPMode:           req.MCPMode,
		CapabilityProfile: req.CapabilityProfile,
		ProfileRevision:   req.ProfileRevision,
		Compaction:        req.Compaction,
	}
	var result AgentResult
	var err error
	if adapter.server != nil {
		result, err = adapter.server.runAgentWithObserver(ctx, adapter.executor, agentRequest, func(event AgentObservation) {
			if event.Type == AgentObservationAnswer {
				event.Text = workflowruntime.VisibleTextBeforeControl(event.Text)
				if event.Text == "" {
					return
				}
			}
			adapter.server.liveTurns.applyObservation(req.MissionID, req.UserEventID, event)
		})
	} else {
		result, err = adapter.executor.Run(ctx, agentRequest)
	}
	return workflowruntime.AgentResult{
		Text:      result.Text,
		SessionID: result.SessionID,
		Resumed:   result.Resumed,
		Log:       result.Log,
		Usage:     result.Usage,
	}, err
}

func (server *Server) workflowRunner(ctx context.Context, missionID string, executorName string) (workflowruntime.RunExecutor, error) {
	executor := server.agentExecutor(executorName)
	if executor == nil {
		return nil, fmt.Errorf("%w: workflow start requires a configured agent executor", app.ErrInvalidInput)
	}
	session := server.latestAgentSession(ctx, missionID, executorName)
	profile := agentcapability.Research()
	if session.SessionID != "" {
		var err error
		profile, err = agentcapability.Resolve(session.ProfileID, session.ProfileRevision)
		if err != nil {
			return nil, fmt.Errorf("%w: persisted agent capability profile is invalid: %v", app.ErrConflict, err)
		}
	}
	previousSessionID := session.SessionID
	model, effort, err := resolveAgentSettings(
		executorName,
		server.latestAgentSessionModel(ctx, missionID, executorName),
		server.latestAgentReasoningEffort(ctx, missionID, executorName),
		previousSessionID,
	)
	if err != nil {
		return nil, err
	}
	return workflowruntime.Runner{
		Service:               server.service,
		Agent:                 workflowAgentAdapter{server: server, executor: executor},
		AgentModel:            model,
		ReasoningEffort:       effort,
		CapabilityProfile:     profile.ID,
		ProfileRevision:       profile.Revision,
		NewID:                 newID,
		SourceCandidateStager: server.stageSourceCandidateProposalEvent,
		AgentTurnStarted:      server.liveTurns.start,
		AgentTurnFinished: func(missionID, userEventID string, err error) {
			server.liveTurns.finish(missionID, userEventID, workflowLiveTerminalState(err))
		},
	}, nil
}

func workflowLiveTerminalState(err error) string {
	if err == nil {
		return "completed"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "error"
}

func activeWorkflowRun(runs []workflowstate.WorkflowRunView) *workflowstate.WorkflowRunView {
	for i := range runs {
		switch runs[i].Status {
		case app.WorkflowStatusQueued, app.WorkflowStatusRunning, app.WorkflowStatusStopping:
			return &runs[i]
		}
	}
	return nil
}
