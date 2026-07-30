package mcp

import (
	"context"
	"strings"
	"unicode/utf8"
)

func (server *Server) callReportPartEditPatch(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input reportPartEditPatchInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "Part edit patch arguments are invalid", false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if _, err := server.requirePartEditBinding(common); err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rpe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if !utf8.ValidString(input.Replacement) || len([]byte(input.Replacement)) > reportPatchMaxApplyBytes {
		return errorResult(call.Name, common.MissionID, "validation", "Part edit replacement is not bounded UTF-8 text", false, []string{draftID})
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	draft, ok := server.partEditDrafts[draftID]
	if !ok {
		return errorResult(call.Name, common.MissionID, "validation", "Part edit draft was not found", false, []string{draftID})
	}
	if err := validatePartEditAccess(draft, common.MissionID, common.SessionID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted || draft.Finalizing {
		return errorResult(call.Name, common.MissionID, "conflict", "Part edit draft is no longer editable", false, []string{draftID})
	}
	if len(draft.Operations) >= reportPartEditMaxOperations {
		return errorResult(call.Name, common.MissionID, "validation", "Part edit draft has too many operations", false, []string{draftID})
	}
	next, err := applyReportPatchOperation(draft.Content, reportPatchApplyInput{Operation: input.Operation, MatchText: input.MatchText, Replacement: input.Replacement, Occurrence: input.Occurrence, ReplaceAll: input.ReplaceAll})
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if strings.TrimSpace(next) == "" || len([]byte(next)) > reportPatchMaxBytes || !utf8.ValidString(next) {
		return errorResult(call.Name, common.MissionID, "validation", "Part edit would produce invalid Markdown", false, []string{draftID})
	}
	draft.Content = next
	draft.Operations = append(draft.Operations, reportPatchOperation{Operation: strings.TrimSpace(input.Operation), Summary: strings.TrimSpace(input.Summary), Bytes: len([]byte(input.Replacement))})
	draft.UpdatedAt = nowUTC()
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: partEditFromState(*draft)}
}
