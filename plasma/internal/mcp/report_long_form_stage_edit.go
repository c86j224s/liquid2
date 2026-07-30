package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type longFormStageEditDraft struct {
	DraftID             string
	Stage               string
	MissionID           string
	SessionID           string
	PendingID           string
	PlanEventID         string
	Content             string
	Operations          []reportPatchOperation
	Finalizing          bool
	Submitted           bool
	StageSubmitted      bool
	StageOperationCount int
	ArtifactID          string
	EventID             string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type reportLongFormStageEditSubmitInput struct {
	reportLongFormEditSubmitInput
	GateFindings *[]reportLongFormGateFindingInput `json:"gate_findings"`
}

type reportLongFormGateFindingInput struct {
	Statement      string   `json:"statement"`
	Classification string   `json:"classification"`
	RepairAction   string   `json:"repair_action"`
	EvidenceIDs    []string `json:"evidence_ids"`
}

func (server *Server) callReportLongFormStageEditStart(ctx context.Context, call ToolCall, expectedStage string) ToolResult {
	var input reportLongFormEditStartInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "long-form stage edit start arguments are invalid", false, nil)
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
		return errorResult(call.Name, common.MissionID, "binding", "long-form stage edit start does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if draftID == "" {
		draftID = newMCPID("rfe")
	}
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	content, submitted, stageSubmitted, stageOperationCount, artifactID, eventID, err := server.loadFinalEditStageDraftContent(ctx, binding)
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, nil)
	}
	if strings.TrimSpace(content) == "" || len([]byte(content)) > reportPatchMaxBytes || !utf8.ValidString(content) {
		return errorResult(call.Name, common.MissionID, "validation", "bound long-form stage manuscript is not readable UTF-8 Markdown", false, nil)
	}
	now := time.Now().UTC()
	draft := &longFormStageEditDraft{
		DraftID: draftID, Stage: binding.Stage, MissionID: common.MissionID, SessionID: common.SessionID,
		PendingID: binding.PendingEventID, PlanEventID: binding.PlanEventID, Content: content,
		Submitted: submitted, StageSubmitted: stageSubmitted, StageOperationCount: stageOperationCount,
		ArtifactID: artifactID, EventID: eventID, CreatedAt: now, UpdatedAt: now,
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.longFormStageEditDrafts) >= reportLongFormEditMaxDrafts {
		return errorResult(call.Name, common.MissionID, "validation", "too many in-process long-form stage edit drafts", false, nil)
	}
	if _, exists := server.longFormStageEditDrafts[draftID]; exists {
		return errorResult(call.Name, common.MissionID, "conflict", "long-form stage edit draft already exists", false, []string{draftID})
	}
	server.longFormStageEditDrafts[draftID] = draft
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: longFormStageEditFromState(*draft)}
}

func (server *Server) loadFinalEditStageDraftContent(ctx context.Context, binding reporting.FinalEditStageBinding) (string, bool, bool, int, string, string, error) {
	if existing, ok, err := reporting.LoadFinalEditStageSubmission(ctx, server.service, binding); err != nil {
		return "", false, false, 0, "", "", err
	} else if ok {
		if binding.Stage == reporting.FinalEditStageGate {
			return string(existing.Artifact.Content), false, true, existing.OperationCount, existing.Artifact.ArtifactID, existing.Event.EventID, nil
		}
		return string(existing.Artifact.Content), true, false, 0, existing.Artifact.ArtifactID, existing.Event.EventID, nil
	}
	if _, _, err := reporting.StartFinalEditStage(ctx, server.service, newMCPID("evt"), binding); err != nil {
		return "", false, false, 0, "", "", err
	}
	source, err := server.service.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return "", false, false, 0, "", "", err
	}
	if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.Filename != binding.Filename {
		return "", false, false, 0, "", "", fmt.Errorf("%w: final edit stage source artifact is outside the binding", app.ErrConflict)
	}
	return string(source.Content), false, false, 0, "", "", nil
}

func (server *Server) callReportLongFormStageEditRead(ctx context.Context, call ToolCall, expectedStage string) ToolResult {
	_ = ctx
	var input reportLongFormEditReadInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "long-form stage edit read arguments are invalid", false, nil)
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
	if _, err := server.requireFinalEditStageBinding(commonMutatingInput{MissionID: missionID, SessionID: sessionID}, expectedStage); err != nil {
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
	if err := validateLongFormStageEditAccess(&copyDraft, missionID, sessionID, expectedStage); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := boundedReportPatchContent(copyDraft.Content, input.Offset, input.MaxBytes)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"draft_id": draftID, "stage": copyDraft.Stage, "content": content, "offset": offset, "next_offset": nextOffset,
		"content_length": len([]byte(copyDraft.Content)), "truncated": truncated, "submitted": copyDraft.Submitted || copyDraft.StageSubmitted,
	}}
}

func (server *Server) callReportLongFormStageEditPatch(ctx context.Context, call ToolCall, expectedStage string) ToolResult {
	_ = ctx
	var input reportLongFormEditPatchInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "long-form stage edit patch arguments are invalid", false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if _, err := server.requireFinalEditStageBinding(common, expectedStage); err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	return server.patchLongFormStageEditDraft(call, common, input, expectedStage)
}
