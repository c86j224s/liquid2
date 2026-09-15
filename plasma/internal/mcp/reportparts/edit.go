package reportparts

import (
	"context"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Handler) CallReportPartEditStart(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportPartEditStartInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "Part edit start arguments are invalid", false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.EditBinding(common)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID || input.PartIndex != binding.PartIndex || input.SourceArtifactID != binding.SourceArtifactID {
		return server.ErrorResult(call.Name, common.MissionID, "binding", "Part edit start does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if draftID == "" {
		draftID = server.NewID("rpe")
	}
	if err := server.ValidateID("rpe_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	content, submitted, artifactID, eventID := "", false, "", ""
	if existing, ok, loadErr := reportingLoadPartEdit(ctx, server, binding); loadErr != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, loadErr, nil)
	} else if ok {
		content, submitted = string(existing.Artifact.Content), true
		artifactID, eventID = existing.Artifact.ArtifactID, existing.Event.EventID
	} else {
		artifact, getErr := server.Service.GetRawArtifact(ctx, binding.SourceArtifactID)
		if getErr != nil {
			return server.ErrorFromErr(call.Name, common.MissionID, getErr, []string{binding.SourceArtifactID})
		}
		if artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" {
			return server.ErrorResult(call.Name, common.MissionID, "conflict", "bound source Part is foreign or not Markdown", false, []string{binding.SourceArtifactID})
		}
		content = string(artifact.Content)
	}
	if strings.TrimSpace(content) == "" || len([]byte(content)) > patchhandler.ReportPatchMaxBytes || !utf8.ValidString(content) {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "bound Part is not readable UTF-8 Markdown", false, nil)
	}
	if _, _, err := reporting.StartPartEdit(ctx, server.Service, server.NewID("evt"), binding); err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, nil)
	}
	now := nowUTC()
	draft := &PartEditDraft{DraftID: draftID, MissionID: common.MissionID, SessionID: common.SessionID, Content: content, Submitted: submitted, ArtifactID: artifactID, EventID: eventID, CreatedAt: now, UpdatedAt: now}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if len(server.State.EditDrafts) >= reportPartEditMaxDrafts {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "too many in-process Part edit drafts", false, nil)
	}
	if _, exists := server.State.EditDrafts[draftID]; exists {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "Part edit draft already exists", false, []string{draftID})
	}
	server.State.EditDrafts[draftID] = draft
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: partEditFromState(*draft)}
}

func (server *Handler) CallReportPartEditRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	var input ReportPartEditReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "Part edit read arguments are invalid", false, nil)
	}
	missionID, sessionID, draftID := strings.TrimSpace(input.MissionID), strings.TrimSpace(input.SessionID), strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.ValidateID("ses_", sessionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.ValidateID("rpe_", draftID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	if _, err := server.EditBinding(wire.CommonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return server.ErrorResult(call.Name, missionID, "binding", err.Error(), false, nil)
	}
	server.Mu.Lock()
	draft, ok := server.State.EditDrafts[draftID]
	if !ok {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, missionID, "validation", "Part edit draft was not found", false, []string{draftID})
	}
	copyDraft := *draft
	server.Mu.Unlock()
	if err := validatePartEditAccess(&copyDraft, missionID, sessionID); err != nil {
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
