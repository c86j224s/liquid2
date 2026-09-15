package reportfinaledit

import (
	"context"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Handler) CallReportLongFormEditSubmit(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportLongFormEditSubmitInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "long-form edit submit arguments are invalid", false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.RequireLongFormBinding(common)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return server.ErrorResult(call.Name, common.MissionID, "binding", "long-form edit submit does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("rfe_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	server.Mu.Lock()
	draft, ok := server.State.LegacyDrafts[draftID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", "long-form edit draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validateLongFormEditAccess(draft, common.MissionID, common.SessionID); err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted {
		copyDraft := *draft
		server.Mu.Unlock()
		return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{copyDraft.EventID}, Content: longFormEditFromState(copyDraft)}
	}
	if draft.Finalizing {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "long-form edit draft is already finalizing", true, []string{draftID})
	}
	draft.Finalizing = true
	manuscript := draft.Content
	server.Mu.Unlock()
	result, err := reporting.FinalizeLongForm(ctx, server.Service, reporting.LongFormFinalizeRequest{
		Binding: binding, EventID: server.NewID("evt"), ManuscriptMarkdown: manuscript,
	})
	if err != nil {
		server.Mu.Lock()
		if current, exists := server.State.LegacyDrafts[draftID]; exists {
			current.Finalizing = false
			current.UpdatedAt = nowUTC()
		}
		server.Mu.Unlock()
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	server.Mu.Lock()
	current, exists := server.State.LegacyDrafts[draftID]
	if exists {
		current.Finalizing = false
		current.Submitted = true
		current.ArtifactID = result.Artifact.ArtifactID
		current.EventID = result.Event.EventID
		current.UpdatedAt = nowUTC()
		copyDraft := *current
		server.Mu.Unlock()
		return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: longFormEditFromState(copyDraft)}
	}
	server.Mu.Unlock()
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: map[string]any{
		"draft_id": draftID, "submitted": true, "artifact_id": result.Artifact.ArtifactID, "event_id": result.Event.EventID,
	}}
}

func (server *Handler) submitLongFormDurableStageEdit(ctx context.Context, call wire.ToolCall, common wire.CommonMutatingInput, input ReportLongFormStageEditSubmitInput, binding reporting.FinalEditStageBinding) wire.ToolResult {
	draftID := strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("rfe_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	server.Mu.Lock()
	draft, ok := server.State.StageDrafts[draftID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", "long-form stage edit draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validateLongFormStageEditAccess(draft, common.MissionID, common.SessionID, binding.Stage); err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted {
		copyDraft := *draft
		server.Mu.Unlock()
		return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{copyDraft.EventID}, Content: longFormStageEditFromState(copyDraft)}
	}
	if draft.Finalizing {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "long-form stage edit draft is already finalizing", true, []string{draftID})
	}
	draft.Finalizing = true
	manuscript := draft.Content
	operationCount := len(draft.Operations)
	styleDiagnoses := styleOperationDiagnosesFromOperations(draft.Operations)
	server.Mu.Unlock()
	if binding.Stage == reporting.FinalEditStageStyle {
		source, err := server.Service.GetRawArtifact(ctx, binding.SourceArtifactID)
		if err != nil {
			server.clearStageFinalizing(draftID)
			return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID})
		}
		if err := reporting.ValidateFinalEditStyleMarkdown(string(source.Content), manuscript); err != nil {
			manuscript = string(source.Content)
			operationCount = 0
			styleDiagnoses = nil
		}
	}
	var result reporting.FinalEditStageResult
	var err error
	if binding.Stage == reporting.FinalEditStageStyle {
		result, err = reporting.SubmitFinalEditStyleStage(ctx, server.Service, binding, server.NewID("evt"), manuscript, operationCount, styleDiagnoses)
	} else {
		result, err = reporting.SubmitFinalEditStage(ctx, server.Service, binding, server.NewID("evt"), manuscript, operationCount)
	}
	if err != nil {
		server.clearStageFinalizing(draftID)
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	return server.stageSubmitResult(call, common.MissionID, draftID, binding.Stage, result.Artifact.ArtifactID, result.Event.EventID, string(result.Artifact.Content), result.OperationCount)
}

func styleOperationDiagnosesFromOperations(operations []patchhandler.Operation) []reporting.FinalEditStyleOperationDiagnosis {
	diagnoses := make([]reporting.FinalEditStyleOperationDiagnosis, 0, len(operations))
	for index, operation := range operations {
		diagnoses = append(diagnoses, reporting.FinalEditStyleOperationDiagnosis{
			OperationOrdinal: index + 1,
			Category:         strings.TrimSpace(operation.Category),
			Reason:           strings.TrimSpace(operation.Reason),
			MatchText:        operation.MatchText,
			Replacement:      operation.Replacement,
			Occurrence:       operation.Occurrence,
		})
	}
	return diagnoses
}

func (server *Handler) submitLongFormGateEdit(ctx context.Context, call wire.ToolCall, common wire.CommonMutatingInput, input ReportLongFormStageEditSubmitInput, binding reporting.FinalEditStageBinding) wire.ToolResult {
	findings, err := gateFindingsFromInput(input.GateFindings)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "final edit gate findings are invalid", false, nil)
	}
	semanticAcceptance, err := semanticAcceptanceFromInput(input.SemanticAcceptance)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "final edit semantic acceptance is invalid", false, nil)
	}
	semanticReviewEnabled := binding.PostReportHumanize == reporting.FinalEditHumanizeEnabled
	if !semanticReviewEnabled && len(semanticAcceptance) != 0 {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "semantic acceptance is only valid when post_report_humanize is enabled", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("rfe_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	server.Mu.Lock()
	draft, ok := server.State.StageDrafts[draftID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", "long-form stage edit draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validateLongFormStageEditAccess(draft, common.MissionID, common.SessionID, binding.Stage); err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Finalizing {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "long-form stage edit draft is already finalizing", true, []string{draftID})
	}
	draft.Finalizing = true
	manuscript := draft.Content
	operationCount := len(draft.Operations)
	if draft.StageSubmitted {
		operationCount = draft.StageOperationCount
	}
	styleReviewComplete := draft.StyleReviewComplete
	server.Mu.Unlock()
	if semanticReviewEnabled {
		comparison, err := reporting.FinalEditSemanticComparison(ctx, server.Service, binding, manuscript)
		if err != nil {
			server.clearStageFinalizing(draftID)
			return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID})
		}
		if len(comparison) > 0 && !styleReviewComplete {
			server.clearStageFinalizing(draftID)
			return server.ErrorResult(call.Name, common.MissionID, "conflict", "changed style paragraphs must be read to completion before corrective gate submit", false, []string{draftID})
		}
	}
	result, err := reporting.SubmitFinalEditGate(ctx, server.Service, reporting.FinalEditGateSubmitRequest{
		StageBinding: binding, FinalBinding: server.FinalizeBinding(),
		StageEventID: server.NewID("evt"), CanonicalEventID: server.NewID("evt"),
		ManuscriptMarkdown: manuscript, OperationCount: operationCount, Findings: findings, SemanticAcceptance: semanticAcceptance,
	})
	if err != nil {
		server.clearStageFinalizing(draftID)
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	return server.stageSubmitResult(call, common.MissionID, draftID, binding.Stage, result.Artifact.ArtifactID, result.Event.EventID, string(result.Artifact.Content), operationCount)
}

func (server *Handler) stageSubmitResult(call wire.ToolCall, missionID string, draftID string, stage string, artifactID string, eventID string, content string, operationCount int) wire.ToolResult {
	server.Mu.Lock()
	current, exists := server.State.StageDrafts[draftID]
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
		server.Mu.Unlock()
		return wire.ToolResult{ToolName: call.Name, MissionID: missionID, CreatedEventIDs: []string{eventID}, Content: longFormStageEditFromState(copyDraft)}
	}
	server.Mu.Unlock()
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, CreatedEventIDs: []string{eventID}, Content: map[string]any{
		"draft_id": draftID, "stage": stage, "submitted": true, "artifact_id": artifactID, "event_id": eventID,
	}}
}

func (server *Handler) clearStageFinalizing(draftID string) {
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if current, exists := server.State.StageDrafts[draftID]; exists {
		current.Finalizing = false
		current.UpdatedAt = nowUTC()
	}
}
