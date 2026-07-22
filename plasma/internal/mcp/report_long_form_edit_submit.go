package mcp

import (
	"context"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) callReportLongFormEditSubmit(ctx context.Context, call ToolCall) ToolResult {
	var input reportLongFormEditSubmitInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "long-form edit submit arguments are invalid", false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.requireLongFormEditBinding(common)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return errorResult(call.Name, common.MissionID, "binding", "long-form edit submit does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	server.mu.Lock()
	draft, ok := server.longFormEditDrafts[draftID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", "long-form edit draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validateLongFormEditAccess(draft, common.MissionID, common.SessionID); err != nil {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted {
		copyDraft := *draft
		server.mu.Unlock()
		return ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{copyDraft.EventID}, Content: longFormEditFromState(copyDraft)}
	}
	if draft.Finalizing {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "conflict", "long-form edit draft is already finalizing", true, []string{draftID})
	}
	if len(draft.Operations) == 0 {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", "long-form final editor must make at least one explicit edit", false, []string{draftID})
	}
	draft.Finalizing = true
	manuscript := draft.Content
	server.mu.Unlock()
	result, err := reporting.FinalizeLongForm(ctx, server.service, reporting.LongFormFinalizeRequest{
		Binding: binding, EventID: newMCPID("evt"), ManuscriptMarkdown: manuscript,
	})
	if err != nil {
		server.mu.Lock()
		if current, exists := server.longFormEditDrafts[draftID]; exists {
			current.Finalizing = false
			current.UpdatedAt = nowUTC()
		}
		server.mu.Unlock()
		return errorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	server.mu.Lock()
	current, exists := server.longFormEditDrafts[draftID]
	if exists {
		current.Finalizing = false
		current.Submitted = true
		current.ArtifactID = result.Artifact.ArtifactID
		current.EventID = result.Event.EventID
		current.UpdatedAt = nowUTC()
		copyDraft := *current
		server.mu.Unlock()
		return ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: longFormEditFromState(copyDraft)}
	}
	server.mu.Unlock()
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: map[string]any{
		"draft_id": draftID, "submitted": true, "artifact_id": result.Artifact.ArtifactID, "event_id": result.Event.EventID,
	}}
}
