package web

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	workflowruntime "github.com/c86j224s/liquid2/plasma/internal/workflow"
)

// Run은 웹 및 에이전트 어댑터의 실행 진입점이다. 호출자는 취소, 실패, 외부 부작용 범위를 해당 패키지 계약에 맞게 보존해야 한다.
func (adapter workflowAgentAdapter) Run(ctx context.Context, req workflowruntime.AgentRequest) (workflowruntime.AgentResult, error) {
	result, err := adapter.executor.Run(ctx, AgentRequest{
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
		Compaction:        req.Compaction,
	})
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
	previousSessionID := server.latestAgentSessionID(ctx, missionID, executorName)
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
		Agent:                 workflowAgentAdapter{executor: executor},
		AgentModel:            model,
		ReasoningEffort:       effort,
		NewID:                 newID,
		SourceCandidateStager: server.stageSourceCandidateProposalEvent,
	}, nil
}

func activeWorkflowRun(runs []app.WorkflowRunView) *app.WorkflowRunView {
	for i := range runs {
		switch runs[i].Status {
		case app.WorkflowStatusQueued, app.WorkflowStatusRunning, app.WorkflowStatusStopping:
			return &runs[i]
		}
	}
	return nil
}
