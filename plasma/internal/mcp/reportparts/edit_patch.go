package reportparts

import (
	"context"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"unicode/utf8"
)

func (server *Handler) CallReportPartEditPatch(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	var input ReportPartEditPatchInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "Part edit patch arguments are invalid", false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if _, err := server.EditBinding(common); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("rpe_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if !utf8.ValidString(input.Replacement) || len([]byte(input.Replacement)) > patchhandler.ReportPatchMaxApplyBytes {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "Part edit replacement is not bounded UTF-8 text", false, []string{draftID})
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	draft, ok := server.State.EditDrafts[draftID]
	if !ok {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "Part edit draft was not found", false, []string{draftID})
	}
	if err := validatePartEditAccess(draft, common.MissionID, common.SessionID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if draft.Submitted || draft.Finalizing {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "Part edit draft is no longer editable", false, []string{draftID})
	}
	if len(draft.Operations) >= reportPartEditMaxOperations {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "Part edit draft has too many operations", false, []string{draftID})
	}
	next, err := patchhandler.ApplyOperation(draft.Content, patchhandler.ReportPatchApplyInput{Operation: input.Operation, MatchText: input.MatchText, Replacement: input.Replacement, Occurrence: input.Occurrence, ReplaceAll: input.ReplaceAll})
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	if strings.TrimSpace(next) == "" || len([]byte(next)) > patchhandler.ReportPatchMaxBytes || !utf8.ValidString(next) {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "Part edit would produce invalid Markdown", false, []string{draftID})
	}
	draft.Content = next
	draft.Operations = append(draft.Operations, patchhandler.Operation{Operation: strings.TrimSpace(input.Operation), Summary: strings.TrimSpace(input.Summary), Bytes: len([]byte(input.Replacement))})
	draft.UpdatedAt = nowUTC()
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: partEditFromState(*draft)}
}
