package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// Handler owns workflow MCP decoding, validation, and canonical port calls.
type Handler struct {
	service      Service
	binding      Binding
	boundMission EnforceBoundMission
	errorResult  ErrorResult
	errorFromErr ErrorFromErr
}

// NewHandler binds the narrow workflow port and root-owned policy callbacks.
func NewHandler(service Service, binding Binding, boundMission EnforceBoundMission, errorResult ErrorResult, errorFromErr ErrorFromErr) *Handler {
	return &Handler{service: service, binding: normalizeBinding(binding), boundMission: boundMission, errorResult: errorResult, errorFromErr: errorFromErr}
}

// CallStart handles plasma.workflow.start without invoking a provider.
func (handler *Handler) CallStart(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input StartInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return handler.validation(call.Name, input.MissionID, err)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return handler.validation(call.Name, missionID, err)
	}
	if err := handler.boundMission(missionID); err != nil {
		return handler.validation(call.Name, missionID, err)
	}
	toolSessionID := firstNonEmpty(strings.TrimSpace(input.RequestedByToolSessionID), handler.binding.AgentSessionID)
	explicitStartAfterEventID := strings.TrimSpace(input.StartAfterEventID)
	boundCurrentUserEventID := handler.binding.CurrentUserEventID
	if boundCurrentUserEventID != "" && explicitStartAfterEventID != "" && explicitStartAfterEventID != boundCurrentUserEventID {
		return handler.validation(call.Name, missionID, fmt.Errorf("workflow start start_after_event_id %q does not match bound current user event %q", explicitStartAfterEventID, boundCurrentUserEventID))
	}
	startAfterEventID := firstNonEmpty(boundCurrentUserEventID, explicitStartAfterEventID)
	if startAfterEventID == "" {
		return handler.validation(call.Name, missionID, errors.New("workflow start from MCP requires a current user event binding or explicit start_after_event_id"))
	}
	boundExecutor := handler.binding.AgentExecutor
	if boundExecutor == "" {
		return handler.validation(call.Name, missionID, errors.New("workflow start from MCP requires an agent executor binding"))
	}
	agentExecutor := firstNonEmpty(strings.TrimSpace(strings.ToLower(input.AgentExecutor)), boundExecutor)
	if agentExecutor != boundExecutor {
		return handler.validation(call.Name, missionID, fmt.Errorf("workflow start agent_executor %q does not match bound executor %q", agentExecutor, boundExecutor))
	}
	view, err := handler.service.RequestWorkflowRun(ctx, workflowstate.RequestWorkflowRunRequest{
		WorkflowRunID:            strings.TrimSpace(input.WorkflowRunID),
		MissionID:                missionID,
		RequestedBySurface:       workflowstate.WorkflowSurfaceMCP,
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
		return handler.errorFrom(call.Name, missionID, err, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{"workflow_run": view, "provider_invoked": false}}
}

// CallStatus handles plasma.workflow.status and preserves raw nonblank IDs for Get.
func (handler *Handler) CallStatus(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input StatusInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return handler.validation(call.Name, input.MissionID, err)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return handler.validation(call.Name, missionID, err)
	}
	if err := handler.boundMission(missionID); err != nil {
		return handler.validation(call.Name, missionID, err)
	}
	if strings.TrimSpace(input.WorkflowRunID) != "" {
		view, err := handler.service.GetWorkflowRun(ctx, missionID, input.WorkflowRunID)
		if err != nil {
			return handler.errorFrom(call.Name, missionID, err, []string{input.WorkflowRunID})
		}
		return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{"workflow_run": view}}
	}
	runs, err := handler.service.ListWorkflowRuns(ctx, missionID)
	if err != nil {
		return handler.errorFrom(call.Name, missionID, err, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{"workflow_runs": runs}}
}

// CallStop handles plasma.workflow.stop and preserves existing output mapping.
func (handler *Handler) CallStop(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input StopInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return handler.validation(call.Name, input.MissionID, err)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return handler.validation(call.Name, missionID, err)
	}
	if err := handler.boundMission(missionID); err != nil {
		return handler.validation(call.Name, missionID, err)
	}
	view, err := handler.service.RequestWorkflowStop(ctx, workflowstate.RequestWorkflowStopRequest{
		WorkflowRunID:            strings.TrimSpace(input.WorkflowRunID),
		MissionID:                missionID,
		RequestedBySurface:       workflowstate.WorkflowSurfaceMCP,
		RequestedByToolSessionID: handler.binding.AgentSessionID,
		Reason:                   firstNonEmpty(strings.TrimSpace(input.Reason), "MCP stop requested"),
	})
	if err != nil {
		return handler.errorFrom(call.Name, missionID, err, []string{input.WorkflowRunID})
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{"workflow_run": view}}
}

func (handler *Handler) validation(toolName, missionID string, err error) wire.ToolResult {
	return handler.errorResult(toolName, missionID, "validation", err.Error(), false, nil)
}

func (handler *Handler) errorFrom(toolName, missionID string, err error, related []string) wire.ToolResult {
	return handler.errorFromErr(toolName, missionID, err, related)
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}

func decodeArgs(args json.RawMessage, target any) error {
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	if !json.Valid(args) {
		return fmt.Errorf("%w: tool arguments must be valid JSON", producterror.ErrInvalidInput)
	}
	if err := json.Unmarshal(args, target); err != nil {
		return fmt.Errorf("%w: decode tool arguments: %v", producterror.ErrInvalidInput, err)
	}
	return nil
}

func normalizeBinding(binding Binding) Binding {
	return Binding{
		MissionID:          strings.TrimSpace(binding.MissionID),
		AgentSessionID:     strings.TrimSpace(binding.AgentSessionID),
		CurrentUserEventID: strings.TrimSpace(binding.CurrentUserEventID),
		AgentExecutor:      strings.TrimSpace(strings.ToLower(binding.AgentExecutor)),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
