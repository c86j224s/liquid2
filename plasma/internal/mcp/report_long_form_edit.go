package mcp

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) callReportLongFormEditStart(ctx context.Context, call ToolCall) ToolResult {
	var input reportLongFormEditStartInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "long-form edit start arguments are invalid", false, nil)
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
		return errorResult(call.Name, common.MissionID, "binding", "long-form edit start does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if draftID == "" {
		draftID = newMCPID("rfe")
	}
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	content := ""
	submitted, artifactID, eventID := false, "", ""
	if existing, ok, loadErr := reporting.LoadLongFormFinalization(ctx, server.service, binding); loadErr != nil {
		return errorFromErr(call.Name, common.MissionID, loadErr, nil)
	} else if ok {
		content, submitted = string(existing.Artifact.Content), true
		artifactID, eventID = existing.Artifact.ArtifactID, existing.Event.EventID
	} else {
		content, err = reporting.PrepareLongFormEditingDraft(ctx, server.service, binding)
		if err != nil {
			return errorFromErr(call.Name, common.MissionID, err, nil)
		}
	}
	if strings.TrimSpace(content) == "" || len([]byte(content)) > reportPatchMaxBytes || !utf8.ValidString(content) {
		return errorResult(call.Name, common.MissionID, "validation", "bound long-form manuscript is not readable UTF-8 Markdown", false, nil)
	}
	now := time.Now().UTC()
	draft := &longFormEditDraft{
		DraftID: draftID, MissionID: common.MissionID, SessionID: common.SessionID,
		PendingID: binding.PendingEventID, PlanEventID: binding.PlanEventID, Content: content,
		Submitted: submitted, ArtifactID: artifactID, EventID: eventID, CreatedAt: now, UpdatedAt: now,
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.longFormEditDrafts) >= reportLongFormEditMaxDrafts {
		return errorResult(call.Name, common.MissionID, "validation", "too many in-process long-form edit drafts", false, nil)
	}
	if _, exists := server.longFormEditDrafts[draftID]; exists {
		return errorResult(call.Name, common.MissionID, "conflict", "long-form edit draft already exists", false, []string{draftID})
	}
	server.longFormEditDrafts[draftID] = draft
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: longFormEditFromState(*draft)}
}

func (server *Server) callReportLongFormEditRead(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input reportLongFormEditReadInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "long-form edit read arguments are invalid", false, nil)
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
	if _, err := server.requireLongFormEditBinding(commonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return errorResult(call.Name, missionID, "binding", err.Error(), false, nil)
	}
	server.mu.Lock()
	draft, ok := server.longFormEditDrafts[draftID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, missionID, "validation", "long-form edit draft was not found in this MCP process", false, []string{draftID})
	}
	copyDraft := *draft
	server.mu.Unlock()
	if err := validateLongFormEditAccess(&copyDraft, missionID, sessionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := boundedReportPatchContent(copyDraft.Content, input.Offset, input.MaxBytes)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"draft_id": draftID, "content": content, "offset": offset, "next_offset": nextOffset,
		"content_length": len([]byte(copyDraft.Content)), "truncated": truncated, "submitted": copyDraft.Submitted,
	}}
}
