package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	workflowruntime "github.com/c86j224s/liquid2/plasma/internal/workflow"
)

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

func activeWorkflowRun(runs []app.WorkflowRunView) *app.WorkflowRunView {
	for i := range runs {
		switch runs[i].Status {
		case app.WorkflowStatusQueued, app.WorkflowStatusRunning, app.WorkflowStatusStopping:
			return &runs[i]
		}
	}
	return nil
}
