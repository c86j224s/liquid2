package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/web"
	workflowruntime "github.com/c86j224s/liquid2/plasma/internal/workflow"
)

func drainCLIQueuedWorkflows(ctx context.Context, svc *app.Service, missionID string, executor web.AgentExecutor, executorName string) error {
	runs, err := svc.ListWorkflowRuns(ctx, missionID)
	if err != nil {
		return err
	}
	for _, run := range runs {
		if run.Status != app.WorkflowStatusQueued || strings.TrimSpace(run.StartAfterEventID) == "" {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(run.AgentExecutor), strings.TrimSpace(executorName)) {
			return fmt.Errorf("queued workflow %s requires agent_executor %q but current CLI executor is %q", run.WorkflowRunID, run.AgentExecutor, executorName)
		}
		runner := workflowruntime.Runner{
			Service:               svc,
			Agent:                 cliWorkflowAgentAdapter{executor: executor},
			NewID:                 cliNewID,
			SourceCandidateStager: cliSourceCandidateStager(svc),
		}
		_, err := runner.Run(ctx, missionID, run.WorkflowRunID)
		return err
	}
	return nil
}

type cliWorkflowAgentAdapter struct {
	executor web.AgentExecutor
}

func (adapter cliWorkflowAgentAdapter) Run(ctx context.Context, req workflowruntime.AgentRequest) (workflowruntime.AgentResult, error) {
	result, err := adapter.executor.Run(ctx, web.AgentRequest{
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
