package mcp

import (
	"context"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) callReportPartEditSubmit(ctx context.Context, call ToolCall) ToolResult {
	var input reportPartEditSubmitInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "Part edit submit arguments are invalid", false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.requirePartEditBinding(common)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return errorResult(call.Name, common.MissionID, "binding", "Part edit submit does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rpe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	server.mu.Lock()
	draft, ok := server.partEditDrafts[draftID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", "Part edit draft was not found", false, []string{draftID})
	}
	if err := validatePartEditAccess(draft, common.MissionID, common.SessionID); err != nil {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted {
		copyDraft := *draft
		server.mu.Unlock()
		return ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{copyDraft.EventID}, Content: partEditFromState(copyDraft)}
	}
	if draft.Finalizing {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "conflict", "Part edit draft is already finalizing", true, []string{draftID})
	}
	draft.Finalizing = true
	markdown, operationCount := draft.Content, len(draft.Operations)
	server.mu.Unlock()
	result, err := reporting.FinalizePartEdit(ctx, server.service, binding, newMCPID("evt"), markdown, operationCount)
	if err != nil {
		server.mu.Lock()
		if current, exists := server.partEditDrafts[draftID]; exists {
			current.Finalizing = false
			current.UpdatedAt = nowUTC()
		}
		server.mu.Unlock()
		return errorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	server.mu.Lock()
	current := server.partEditDrafts[draftID]
	current.Finalizing, current.Submitted = false, true
	current.ArtifactID, current.EventID, current.UpdatedAt = result.Artifact.ArtifactID, result.Event.EventID, nowUTC()
	copyDraft := *current
	server.mu.Unlock()
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: partEditFromState(copyDraft)}
}

func reportingLoadPartEdit(ctx context.Context, server *Server, binding reporting.PartEditBinding) (reporting.PartEditResult, bool, error) {
	return reporting.LoadPartEdit(ctx, server.service, binding)
}
