package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) callReportLongFormStyleReviewRead(ctx context.Context, call ToolCall) ToolResult {
	var input reportLongFormEditReadInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "long-form style review read arguments are invalid", false, nil)
	}
	missionID, sessionID, draftID := strings.TrimSpace(input.MissionID), strings.TrimSpace(input.SessionID), strings.TrimSpace(input.DraftID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("ses_", sessionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	binding, err := server.requireFinalEditStageBinding(commonMutatingInput{MissionID: missionID, SessionID: sessionID}, reporting.FinalEditStageGate)
	if err != nil {
		return errorResult(call.Name, missionID, "binding", err.Error(), false, nil)
	}
	server.mu.Lock()
	draft, ok := server.longFormStageEditDrafts[draftID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, missionID, "validation", "long-form stage edit draft was not found in this MCP process", false, []string{draftID})
	}
	copyDraft := *draft
	server.mu.Unlock()
	if err := validateLongFormStageEditAccess(&copyDraft, missionID, sessionID, reporting.FinalEditStageGate); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	comparison, err := reporting.FinalEditSemanticComparison(ctx, server.service, binding, copyDraft.Content)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{draftID})
	}
	packetBytes, err := json.MarshalIndent(comparison, "", "  ")
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{draftID})
	}
	packet := string(packetBytes)
	server.mu.Lock()
	current := server.longFormStageEditDrafts[draftID]
	expectedOffset := 0
	if current != nil {
		expectedOffset = current.StyleReviewNextOffset
	}
	server.mu.Unlock()
	if input.Offset != expectedOffset {
		return errorResult(call.Name, missionID, "validation", "style review reads must use contiguous next_offset values starting at 0", false, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := boundedReportPatchContent(packet, input.Offset, input.MaxBytes)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	server.mu.Lock()
	if current := server.longFormStageEditDrafts[draftID]; current != nil {
		current.StyleReviewNextOffset = nextOffset
		current.StyleReviewComplete = !truncated
		current.UpdatedAt = nowUTC()
	}
	server.mu.Unlock()
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"draft_id": draftID, "stage": copyDraft.Stage, "content": content, "offset": offset, "next_offset": nextOffset,
		"content_length": len([]byte(packet)), "changed_paragraph_count": len(comparison), "truncated": truncated,
	}}
}
