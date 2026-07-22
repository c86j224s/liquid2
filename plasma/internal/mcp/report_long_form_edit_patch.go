package mcp

import (
	"context"
	"strings"
	"unicode/utf8"
)

func (server *Server) callReportLongFormEditPatch(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input reportLongFormEditPatchInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "long-form edit patch arguments are invalid", false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if _, err := server.requireLongFormEditBinding(common); err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if !utf8.ValidString(input.Replacement) || len([]byte(input.Replacement)) > reportPatchMaxApplyBytes {
		return errorResult(call.Name, common.MissionID, "validation", "long-form edit replacement is not bounded UTF-8 text", false, []string{draftID})
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	draft, ok := server.longFormEditDrafts[draftID]
	if !ok {
		return errorResult(call.Name, common.MissionID, "validation", "long-form edit draft was not found in this MCP process", false, []string{draftID})
	}
	if err := validateLongFormEditAccess(draft, common.MissionID, common.SessionID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted || draft.Finalizing {
		return errorResult(call.Name, common.MissionID, "conflict", "long-form edit draft is no longer editable", false, []string{draftID})
	}
	if len(draft.Operations) >= reportLongFormEditMaxOperations {
		return errorResult(call.Name, common.MissionID, "validation", "long-form edit draft has too many operations", false, []string{draftID})
	}
	next, err := applyReportPatchOperation(draft.Content, reportPatchApplyInput{
		Operation: input.Operation, MatchText: input.MatchText, Replacement: input.Replacement,
		Occurrence: input.Occurrence, ReplaceAll: input.ReplaceAll,
	})
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if strings.TrimSpace(next) == "" || len([]byte(next)) > reportPatchMaxBytes || !utf8.ValidString(next) {
		return errorResult(call.Name, common.MissionID, "validation", "long-form edit would produce an invalid manuscript", false, []string{draftID})
	}
	draft.Content = next
	draft.Operations = append(draft.Operations, reportPatchOperation{Operation: strings.TrimSpace(input.Operation), Summary: strings.TrimSpace(input.Summary), Bytes: len([]byte(input.Replacement))})
	draft.UpdatedAt = nowUTC()
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: longFormEditFromState(*draft)}
}
