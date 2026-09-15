package reportfinaledit

import (
	"context"
	"encoding/json"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Handler) CallReportLongFormStyleReviewRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportLongFormEditReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "long-form style review read arguments are invalid", false, nil)
	}
	missionID, sessionID, draftID := strings.TrimSpace(input.MissionID), strings.TrimSpace(input.SessionID), strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.ValidateID("ses_", sessionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.ValidateID("rfe_", draftID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	binding, err := server.RequireStageBinding(wire.CommonMutatingInput{MissionID: missionID, SessionID: sessionID}, reporting.FinalEditStageGate)
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "binding", err.Error(), false, nil)
	}
	server.Mu.Lock()
	draft, ok := server.State.StageDrafts[draftID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, missionID, "validation", "long-form stage edit draft was not found in this MCP process", false, []string{draftID})
	}
	copyDraft := *draft
	server.Mu.Unlock()
	if err := validateLongFormStageEditAccess(&copyDraft, missionID, sessionID, reporting.FinalEditStageGate); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	comparison, err := reporting.FinalEditSemanticComparison(ctx, server.Service, binding, copyDraft.Content)
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{draftID})
	}
	packetBytes, err := json.MarshalIndent(comparison, "", "  ")
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{draftID})
	}
	packet := string(packetBytes)
	server.Mu.Lock()
	current := server.State.StageDrafts[draftID]
	expectedOffset := 0
	if current != nil {
		expectedOffset = current.StyleReviewNextOffset
	}
	server.Mu.Unlock()
	if input.Offset != expectedOffset {
		return server.ErrorResult(call.Name, missionID, "validation", "style review reads must use contiguous next_offset values starting at 0", false, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := patchhandler.BoundedContent(packet, input.Offset, input.MaxBytes)
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	server.Mu.Lock()
	if current := server.State.StageDrafts[draftID]; current != nil {
		current.StyleReviewNextOffset = nextOffset
		current.StyleReviewComplete = !truncated
		current.UpdatedAt = nowUTC()
	}
	server.Mu.Unlock()
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"draft_id": draftID, "stage": copyDraft.Stage, "content": content, "offset": offset, "next_offset": nextOffset,
		"content_length": len([]byte(packet)), "changed_paragraph_count": len(comparison), "truncated": truncated,
	}}
}
