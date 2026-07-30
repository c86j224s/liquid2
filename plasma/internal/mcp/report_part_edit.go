package mcp

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) callReportPartEditStart(ctx context.Context, call ToolCall) ToolResult {
	var input reportPartEditStartInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "Part edit start arguments are invalid", false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.requirePartEditBinding(common)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID || input.PartIndex != binding.PartIndex || input.SourceArtifactID != binding.SourceArtifactID {
		return errorResult(call.Name, common.MissionID, "binding", "Part edit start does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if draftID == "" {
		draftID = newMCPID("rpe")
	}
	if err := validateID("rpe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	content, submitted, artifactID, eventID := "", false, "", ""
	if existing, ok, loadErr := reportingLoadPartEdit(ctx, server, binding); loadErr != nil {
		return errorFromErr(call.Name, common.MissionID, loadErr, nil)
	} else if ok {
		content, submitted = string(existing.Artifact.Content), true
		artifactID, eventID = existing.Artifact.ArtifactID, existing.Event.EventID
	} else {
		artifact, getErr := server.service.GetRawArtifact(ctx, binding.SourceArtifactID)
		if getErr != nil {
			return errorFromErr(call.Name, common.MissionID, getErr, []string{binding.SourceArtifactID})
		}
		if artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" {
			return errorResult(call.Name, common.MissionID, "conflict", "bound source Part is foreign or not Markdown", false, []string{binding.SourceArtifactID})
		}
		content = string(artifact.Content)
	}
	if strings.TrimSpace(content) == "" || len([]byte(content)) > reportPatchMaxBytes || !utf8.ValidString(content) {
		return errorResult(call.Name, common.MissionID, "validation", "bound Part is not readable UTF-8 Markdown", false, nil)
	}
	if _, _, err := reporting.StartPartEdit(ctx, server.service, newMCPID("evt"), binding); err != nil {
		return errorFromErr(call.Name, common.MissionID, err, nil)
	}
	now := nowUTC()
	draft := &partEditDraft{DraftID: draftID, MissionID: common.MissionID, SessionID: common.SessionID, Content: content, Submitted: submitted, ArtifactID: artifactID, EventID: eventID, CreatedAt: now, UpdatedAt: now}
	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.partEditDrafts) >= reportPartEditMaxDrafts {
		return errorResult(call.Name, common.MissionID, "validation", "too many in-process Part edit drafts", false, nil)
	}
	if _, exists := server.partEditDrafts[draftID]; exists {
		return errorResult(call.Name, common.MissionID, "conflict", "Part edit draft already exists", false, []string{draftID})
	}
	server.partEditDrafts[draftID] = draft
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: partEditFromState(*draft)}
}

func (server *Server) callReportPartEditRead(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	var input reportPartEditReadInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "Part edit read arguments are invalid", false, nil)
	}
	missionID, sessionID, draftID := strings.TrimSpace(input.MissionID), strings.TrimSpace(input.SessionID), strings.TrimSpace(input.DraftID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("ses_", sessionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("rpe_", draftID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	if _, err := server.requirePartEditBinding(commonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return errorResult(call.Name, missionID, "binding", err.Error(), false, nil)
	}
	server.mu.Lock()
	draft, ok := server.partEditDrafts[draftID]
	if !ok {
		server.mu.Unlock()
		return errorResult(call.Name, missionID, "validation", "Part edit draft was not found", false, []string{draftID})
	}
	copyDraft := *draft
	server.mu.Unlock()
	if err := validatePartEditAccess(&copyDraft, missionID, sessionID); err != nil {
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
