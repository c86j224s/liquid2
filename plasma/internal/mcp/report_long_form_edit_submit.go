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

func (server *Server) submitLongFormDurableStageEdit(ctx context.Context, call ToolCall, common commonMutatingInput, input reportLongFormStageEditSubmitInput, binding reporting.FinalEditStageBinding) ToolResult {
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	server.mu.Lock()
	draft, ok := server.longFormStageEditDrafts[draftID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", "long-form stage edit draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validateLongFormStageEditAccess(draft, common.MissionID, common.SessionID, binding.Stage); err != nil {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted {
		copyDraft := *draft
		server.mu.Unlock()
		return ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{copyDraft.EventID}, Content: longFormStageEditFromState(copyDraft)}
	}
	if draft.Finalizing {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "conflict", "long-form stage edit draft is already finalizing", true, []string{draftID})
	}
	draft.Finalizing = true
	manuscript := draft.Content
	operationCount := len(draft.Operations)
	server.mu.Unlock()
	if binding.Stage == reporting.FinalEditStageStyle {
		source, err := server.service.GetRawArtifact(ctx, binding.SourceArtifactID)
		if err != nil {
			server.clearStageFinalizing(draftID)
			return errorFromErr(call.Name, common.MissionID, err, []string{draftID})
		}
		if err := reporting.ValidateFinalEditStyleMarkdown(string(source.Content), manuscript); err != nil {
			manuscript = string(source.Content)
			operationCount = 0
		}
	}
	result, err := reporting.SubmitFinalEditStage(ctx, server.service, binding, newMCPID("evt"), manuscript, operationCount)
	if err != nil {
		server.clearStageFinalizing(draftID)
		return errorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	return server.stageSubmitResult(call, common.MissionID, draftID, binding.Stage, result.Artifact.ArtifactID, result.Event.EventID, string(result.Artifact.Content), result.OperationCount)
}

func (server *Server) submitLongFormGateEdit(ctx context.Context, call ToolCall, common commonMutatingInput, input reportLongFormStageEditSubmitInput, binding reporting.FinalEditStageBinding) ToolResult {
	findings, err := gateFindingsFromInput(input.GateFindings)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", "final edit gate findings are invalid", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	server.mu.Lock()
	draft, ok := server.longFormStageEditDrafts[draftID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", "long-form stage edit draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validateLongFormStageEditAccess(draft, common.MissionID, common.SessionID, binding.Stage); err != nil {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Finalizing {
		server.mu.Unlock()
		return errorResult(call.Name, common.MissionID, "conflict", "long-form stage edit draft is already finalizing", true, []string{draftID})
	}
	draft.Finalizing = true
	manuscript := draft.Content
	operationCount := len(draft.Operations)
	if draft.StageSubmitted {
		operationCount = draft.StageOperationCount
	}
	server.mu.Unlock()
	result, err := reporting.SubmitFinalEditGate(ctx, server.service, reporting.FinalEditGateSubmitRequest{
		StageBinding: binding, FinalBinding: server.longFormFinalizeBinding,
		StageEventID: newMCPID("evt"), CanonicalEventID: newMCPID("evt"),
		ManuscriptMarkdown: manuscript, OperationCount: operationCount, Findings: findings,
	})
	if err != nil {
		server.clearStageFinalizing(draftID)
		return errorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	return server.stageSubmitResult(call, common.MissionID, draftID, binding.Stage, result.Artifact.ArtifactID, result.Event.EventID, string(result.Artifact.Content), operationCount)
}

func (server *Server) stageSubmitResult(call ToolCall, missionID string, draftID string, stage string, artifactID string, eventID string, content string, operationCount int) ToolResult {
	server.mu.Lock()
	current, exists := server.longFormStageEditDrafts[draftID]
	if exists {
		current.Finalizing = false
		current.Submitted = true
		current.ArtifactID = artifactID
		current.EventID = eventID
		current.Content = content
		current.Operations = nil
		current.StageSubmitted = true
		current.StageOperationCount = operationCount
		current.UpdatedAt = nowUTC()
		copyDraft := *current
		server.mu.Unlock()
		return ToolResult{ToolName: call.Name, MissionID: missionID, CreatedEventIDs: []string{eventID}, Content: longFormStageEditFromState(copyDraft)}
	}
	server.mu.Unlock()
	return ToolResult{ToolName: call.Name, MissionID: missionID, CreatedEventIDs: []string{eventID}, Content: map[string]any{
		"draft_id": draftID, "stage": stage, "submitted": true, "artifact_id": artifactID, "event_id": eventID,
	}}
}

func (server *Server) clearStageFinalizing(draftID string) {
	server.mu.Lock()
	defer server.mu.Unlock()
	if current, exists := server.longFormStageEditDrafts[draftID]; exists {
		current.Finalizing = false
		current.UpdatedAt = nowUTC()
	}
}
