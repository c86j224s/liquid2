package reportparts

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Handler) CallReportPartEditSubmit(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportPartEditSubmitInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "Part edit submit arguments are invalid", false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.EditBinding(common)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return server.ErrorResult(call.Name, common.MissionID, "binding", "Part edit submit does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("rpe_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	server.Mu.Lock()
	draft, ok := server.State.EditDrafts[draftID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", "Part edit draft was not found", false, []string{draftID})
	}
	if err := validatePartEditAccess(draft, common.MissionID, common.SessionID); err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted {
		copyDraft := *draft
		server.Mu.Unlock()
		return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{copyDraft.EventID}, Content: partEditFromState(copyDraft)}
	}
	if draft.Finalizing {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "Part edit draft is already finalizing", true, []string{draftID})
	}
	draft.Finalizing = true
	markdown, operationCount := draft.Content, len(draft.Operations)
	server.Mu.Unlock()
	result, err := reporting.FinalizePartEdit(ctx, server.Service, binding, server.NewID("evt"), markdown, operationCount)
	if err != nil {
		server.Mu.Lock()
		if current, exists := server.State.EditDrafts[draftID]; exists {
			current.Finalizing = false
			current.UpdatedAt = nowUTC()
		}
		server.Mu.Unlock()
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	server.Mu.Lock()
	current := server.State.EditDrafts[draftID]
	current.Finalizing, current.Submitted = false, true
	current.ArtifactID, current.EventID, current.UpdatedAt = result.Artifact.ArtifactID, result.Event.EventID, nowUTC()
	copyDraft := *current
	server.Mu.Unlock()
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: partEditFromState(copyDraft)}
}

func reportingLoadPartEdit(ctx context.Context, server *Handler, binding reporting.PartEditBinding) (reporting.PartEditResult, bool, error) {
	return reporting.LoadPartEdit(ctx, server.Service, binding)
}
