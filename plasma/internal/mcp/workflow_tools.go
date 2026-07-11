package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (server *Server) callWorkflowStart(ctx context.Context, call ToolCall) ToolResult {
	var input workflowStartInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	toolSessionID := firstNonEmpty(strings.TrimSpace(input.RequestedByToolSessionID), strings.TrimSpace(server.binding.AgentSessionID))
	explicitStartAfterEventID := strings.TrimSpace(input.StartAfterEventID)
	boundCurrentUserEventID := strings.TrimSpace(server.binding.CurrentUserEventID)
	if boundCurrentUserEventID != "" && explicitStartAfterEventID != "" && explicitStartAfterEventID != boundCurrentUserEventID {
		return errorResult(call.Name, missionID, "validation", fmt.Sprintf("workflow start start_after_event_id %q does not match bound current user event %q", explicitStartAfterEventID, boundCurrentUserEventID), false, nil)
	}
	startAfterEventID := firstNonEmpty(boundCurrentUserEventID, explicitStartAfterEventID)
	if startAfterEventID == "" {
		return errorResult(call.Name, missionID, "validation", "workflow start from MCP requires a current user event binding or explicit start_after_event_id", false, nil)
	}
	boundExecutor := strings.TrimSpace(server.binding.AgentExecutor)
	if boundExecutor == "" {
		return errorResult(call.Name, missionID, "validation", "workflow start from MCP requires an agent executor binding", false, nil)
	}
	agentExecutor := firstNonEmpty(strings.TrimSpace(strings.ToLower(input.AgentExecutor)), boundExecutor)
	if agentExecutor != boundExecutor {
		return errorResult(call.Name, missionID, "validation", fmt.Sprintf("workflow start agent_executor %q does not match bound executor %q", agentExecutor, boundExecutor), false, nil)
	}
	view, err := server.service.RequestWorkflowRun(ctx, app.RequestWorkflowRunRequest{
		WorkflowRunID:            strings.TrimSpace(input.WorkflowRunID),
		MissionID:                missionID,
		RequestedBySurface:       app.WorkflowSurfaceMCP,
		RequestedByToolSessionID: toolSessionID,
		AgentExecutor:            agentExecutor,
		MCPMode:                  firstNonEmpty(strings.TrimSpace(input.MCPMode), "auto"),
		StepInstructionMode:      strings.TrimSpace(input.StepInstructionMode),
		UserInstructionRaw:       strings.TrimSpace(input.UserInstructionRaw),
		RunGoal:                  strings.TrimSpace(input.RunGoal),
		Instruction:              strings.TrimSpace(input.Instruction),
		MaxSteps:                 input.MaxSteps,
		MaxDurationMS:            input.MaxDurationMS,
		StopCondition:            strings.TrimSpace(input.StopCondition),
		StartAfterEventID:        startAfterEventID,
		ArgumentSummary:          strings.TrimSpace(input.Instruction),
	})
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{"workflow_run": view, "provider_invoked": false}}
}

func (server *Server) callWorkflowStatus(ctx context.Context, call ToolCall) ToolResult {
	var input workflowStatusInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if strings.TrimSpace(input.WorkflowRunID) != "" {
		view, err := server.service.GetWorkflowRun(ctx, missionID, input.WorkflowRunID)
		if err != nil {
			return errorFromErr(call.Name, missionID, err, []string{input.WorkflowRunID})
		}
		return ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{"workflow_run": view}}
	}
	runs, err := server.service.ListWorkflowRuns(ctx, missionID)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{"workflow_runs": runs}}
}

func (server *Server) callWorkflowStop(ctx context.Context, call ToolCall) ToolResult {
	var input workflowStopInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	view, err := server.service.RequestWorkflowStop(ctx, app.RequestWorkflowStopRequest{
		WorkflowRunID:            strings.TrimSpace(input.WorkflowRunID),
		MissionID:                missionID,
		RequestedBySurface:       app.WorkflowSurfaceMCP,
		RequestedByToolSessionID: strings.TrimSpace(server.binding.AgentSessionID),
		Reason:                   firstNonEmpty(strings.TrimSpace(input.Reason), "MCP stop requested"),
	})
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{input.WorkflowRunID})
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{"workflow_run": view}}
}
