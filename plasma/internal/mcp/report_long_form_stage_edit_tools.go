package mcp

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) finalEditStageMode() string {
	if server.finalEditConfigErr != nil || !server.finalEditStageBindingSet {
		return ""
	}
	return strings.TrimSpace(server.finalEditStageBinding.Stage)
}

func (server *Server) finalEditStageToolEnabled(name string) bool {
	if server.finalEditConfigErr != nil || !server.toolEnabled(name) {
		return false
	}
	stage := server.finalEditStageMode()
	switch name {
	case ToolReportLongFormFinalWriteStart, ToolReportLongFormFinalWriteRead, ToolReportLongFormFinalWritePatch, ToolReportLongFormFinalWriteSubmit:
		return stage == reporting.FinalEditStageWriter
	case ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditRead, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit:
		return stage == reporting.FinalEditStageReader
	case ToolReportLongFormStyleEditStart, ToolReportLongFormStyleEditRead, ToolReportLongFormStyleEditPatch, ToolReportLongFormStyleEditSubmit:
		return stage == reporting.FinalEditStageStyle
	case ToolReportLongFormEditStart, ToolReportLongFormEditRead, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit:
		return stage == reporting.FinalEditStageGate
	default:
		return false
	}
}

func (server *Server) requireFinalEditStageBinding(common commonMutatingInput, expectedStage string) (reporting.FinalEditStageBinding, error) {
	if server.finalEditConfigErr != nil {
		return reporting.FinalEditStageBinding{}, fmt.Errorf("%w: final edit MCP binding configuration is closed: %v", app.ErrInvalidInput, server.finalEditConfigErr)
	}
	if err := server.requireBoundWriteSession(common); err != nil {
		return reporting.FinalEditStageBinding{}, err
	}
	binding := server.finalEditStageBinding
	if err := ValidateFinalEditStageBinding(server.binding, binding); err != nil {
		return reporting.FinalEditStageBinding{}, err
	}
	if strings.TrimSpace(binding.Stage) != strings.TrimSpace(expectedStage) {
		return reporting.FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage tool does not match the runner binding", app.ErrInvalidInput)
	}
	if binding.Stage == reporting.FinalEditStageGate {
		if err := ValidateLongFormFinalizeBinding(server.binding, server.longFormFinalizeBinding); err != nil {
			return reporting.FinalEditStageBinding{}, err
		}
	}
	return binding, nil
}

func finalEditStageDisabledResult(call ToolCall) ToolResult {
	return errorResult(call.Name, missionIDFromArguments(call.Arguments), "binding", "final edit stage tools are only enabled for a matching bound stage session; invalid stage/final binding configurations are closed", false, nil)
}

func validateLongFormStageEditAccess(draft *longFormStageEditDraft, missionID string, sessionID string, stage string) error {
	if draft == nil || draft.MissionID != strings.TrimSpace(missionID) || draft.SessionID != strings.TrimSpace(sessionID) || draft.Stage != strings.TrimSpace(stage) {
		return fmt.Errorf("%w: long-form stage edit draft is outside this MCP session", app.ErrInvalidInput)
	}
	return nil
}

func longFormStageEditFromState(draft longFormStageEditDraft) map[string]any {
	state := "open"
	if draft.Submitted || draft.StageSubmitted {
		state = "submitted"
	} else if draft.Finalizing {
		state = "finalizing"
	}
	operationCount := len(draft.Operations)
	if draft.StageSubmitted {
		operationCount = draft.StageOperationCount
	}
	return map[string]any{
		"draft_id": draft.DraftID, "stage": draft.Stage, "mission_id": draft.MissionID, "session_id": draft.SessionID,
		"pending_event_id": draft.PendingID, "plan_event_id": draft.PlanEventID,
		"state": state, "content_length": len([]byte(draft.Content)), "operation_count": operationCount,
		"submitted": draft.Submitted || draft.StageSubmitted, "artifact_id": draft.ArtifactID, "event_id": draft.EventID,
	}
}

func (server *Server) patchLongFormStageEditDraft(call ToolCall, common commonMutatingInput, input reportLongFormEditPatchInput, expectedStage string) ToolResult {
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if !utf8.ValidString(input.Replacement) || len([]byte(input.Replacement)) > reportPatchMaxApplyBytes {
		return errorResult(call.Name, common.MissionID, "validation", "long-form stage edit replacement is not bounded UTF-8 text", false, []string{draftID})
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	draft, ok := server.longFormStageEditDrafts[draftID]
	if !ok {
		return errorResult(call.Name, common.MissionID, "validation", "long-form stage edit draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validateLongFormStageEditAccess(draft, common.MissionID, common.SessionID, expectedStage); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted || draft.StageSubmitted || draft.Finalizing {
		return errorResult(call.Name, common.MissionID, "conflict", "long-form stage edit draft is no longer editable", false, []string{draftID})
	}
	if len(draft.Operations) >= reportLongFormEditMaxOperations {
		return errorResult(call.Name, common.MissionID, "validation", "long-form stage edit draft has too many operations", false, []string{draftID})
	}
	next, err := applyReportPatchOperation(draft.Content, reportPatchApplyInput{
		Operation: input.Operation, MatchText: input.MatchText, Replacement: input.Replacement,
		Occurrence: input.Occurrence, ReplaceAll: input.ReplaceAll,
	})
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if strings.TrimSpace(next) == "" || len([]byte(next)) > reportPatchMaxBytes || !utf8.ValidString(next) {
		return errorResult(call.Name, common.MissionID, "validation", "long-form stage edit would produce an invalid manuscript", false, []string{draftID})
	}
	draft.Content = next
	draft.Operations = append(draft.Operations, reportPatchOperation{Operation: strings.TrimSpace(input.Operation), Summary: strings.TrimSpace(input.Summary), Bytes: len([]byte(input.Replacement))})
	draft.UpdatedAt = nowUTC()
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: longFormStageEditFromState(*draft)}
}

func gateFindingsFromInput(input *[]reportLongFormGateFindingInput) ([]reporting.FinalEditGateFinding, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: final edit gate findings are required", app.ErrInvalidInput)
	}
	findings := make([]reporting.FinalEditGateFinding, 0, len(*input))
	for _, item := range *input {
		statement := strings.TrimSpace(item.Statement)
		classification := strings.TrimSpace(item.Classification)
		action := strings.TrimSpace(item.RepairAction)
		if statement == "" || classification == "" {
			return nil, fmt.Errorf("%w: final edit gate finding is incomplete", app.ErrInvalidInput)
		}
		evidenceIDs := make([]string, 0, len(item.EvidenceIDs))
		for _, evidenceID := range item.EvidenceIDs {
			if trimmed := strings.TrimSpace(evidenceID); trimmed != "" {
				evidenceIDs = append(evidenceIDs, trimmed)
			}
		}
		findings = append(findings, reporting.FinalEditGateFinding{
			Statement: statement, Classification: classification, RepairAction: action, EvidenceIDs: evidenceIDs,
		})
	}
	return findings, nil
}

func (server *Server) callReportLongFormStageEditSubmit(ctx context.Context, call ToolCall, expectedStage string) ToolResult {
	var input reportLongFormStageEditSubmitInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "long-form stage edit submit arguments are invalid", false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.requireFinalEditStageBinding(common, expectedStage)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return errorResult(call.Name, common.MissionID, "binding", "long-form stage edit submit does not match the runner binding", false, nil)
	}
	if binding.Stage == reporting.FinalEditStageGate {
		return server.submitLongFormGateEdit(ctx, call, common, input, binding)
	}
	return server.submitLongFormDurableStageEdit(ctx, call, common, input, binding)
}
