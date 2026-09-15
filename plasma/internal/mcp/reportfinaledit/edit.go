package reportfinaledit

import (
	"context"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Handler) CallReportLongFormEditStart(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportLongFormEditStartInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "long-form edit start arguments are invalid", false, nil)
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
		return server.ErrorResult(call.Name, common.MissionID, "binding", "long-form edit start does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if draftID == "" {
		draftID = server.NewID("rfe")
	}
	if err := server.ValidateID("rfe_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	content := ""
	submitted, artifactID, eventID := false, "", ""
	if existing, ok, loadErr := reporting.LoadLongFormFinalization(ctx, server.Service, binding); loadErr != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, loadErr, nil)
	} else if ok {
		content, submitted = string(existing.Artifact.Content), true
		artifactID, eventID = existing.Artifact.ArtifactID, existing.Event.EventID
	} else {
		content, err = reporting.PrepareLongFormEditingDraft(ctx, server.Service, binding)
		if err != nil {
			return server.ErrorFromErr(call.Name, common.MissionID, err, nil)
		}
	}
	if strings.TrimSpace(content) == "" || len([]byte(content)) > patchhandler.ReportPatchMaxBytes || !utf8.ValidString(content) {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "bound long-form manuscript is not readable UTF-8 Markdown", false, nil)
	}
	now := time.Now().UTC()
	draft := &LongFormEditDraft{
		DraftID: draftID, MissionID: common.MissionID, SessionID: common.SessionID,
		PendingID: binding.PendingEventID, PlanEventID: binding.PlanEventID, Content: content,
		Submitted: submitted, ArtifactID: artifactID, EventID: eventID, CreatedAt: now, UpdatedAt: now,
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if len(server.State.LegacyDrafts) >= reportLongFormEditMaxDrafts {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "too many in-process long-form edit drafts", false, nil)
	}
	if _, exists := server.State.LegacyDrafts[draftID]; exists {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "long-form edit draft already exists", false, []string{draftID})
	}
	server.State.LegacyDrafts[draftID] = draft
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: longFormEditFromState(*draft)}
}

func (server *Handler) CallReportLongFormEditRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	var input ReportLongFormEditReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "long-form edit read arguments are invalid", false, nil)
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
	if _, err := server.RequireLongFormBinding(wire.CommonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return server.ErrorResult(call.Name, missionID, "binding", err.Error(), false, nil)
	}
	server.Mu.Lock()
	draft, ok := server.State.LegacyDrafts[draftID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, missionID, "validation", "long-form edit draft was not found in this MCP process", false, []string{draftID})
	}
	copyDraft := *draft
	server.Mu.Unlock()
	if err := validateLongFormEditAccess(&copyDraft, missionID, sessionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := patchhandler.BoundedContent(copyDraft.Content, input.Offset, input.MaxBytes)
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"draft_id": draftID, "content": content, "offset": offset, "next_offset": nextOffset,
		"content_length": len([]byte(copyDraft.Content)), "truncated": truncated, "submitted": copyDraft.Submitted,
	}}
}
