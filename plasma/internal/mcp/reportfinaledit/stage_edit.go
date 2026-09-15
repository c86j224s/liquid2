package reportfinaledit

import (
	"context"
	"fmt"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type LongFormStageEditDraft struct {
	DraftID               string
	Stage                 string
	MissionID             string
	SessionID             string
	PendingID             string
	PlanEventID           string
	Content               string
	Operations            []patchhandler.Operation
	Finalizing            bool
	Submitted             bool
	StageSubmitted        bool
	StageOperationCount   int
	StyleReviewNextOffset int
	StyleReviewComplete   bool
	ArtifactID            string
	EventID               string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type ReportLongFormStageEditSubmitInput struct {
	ReportLongFormEditSubmitInput
	GateFindings       *[]ReportLongFormGateFindingInput        `json:"gate_findings"`
	SemanticAcceptance *[]ReportLongFormSemanticAcceptanceInput `json:"semantic_acceptance"`
}

type ReportLongFormGateFindingInput struct {
	StatementSHA256 string   `json:"statement_sha256"`
	Statement       string   `json:"statement"`
	Classification  string   `json:"classification"`
	RepairAction    string   `json:"repair_action"`
	EvidenceIDs     []string `json:"evidence_ids"`
}

type ReportLongFormSemanticAcceptanceInput struct {
	ParagraphOrdinal      int    `json:"paragraph_ordinal"`
	FinalParagraphOrdinal int    `json:"final_paragraph_ordinal"`
	Verdict               string `json:"verdict"`
}

func (server *Handler) CallReportLongFormStageEditStart(ctx context.Context, call wire.ToolCall, expectedStage string) wire.ToolResult {
	var input ReportLongFormEditStartInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "long-form stage edit start arguments are invalid", false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.RequireStageBinding(common, expectedStage)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return server.ErrorResult(call.Name, common.MissionID, "binding", "long-form stage edit start does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if draftID == "" {
		draftID = server.NewID("rfe")
	}
	if err := server.ValidateID("rfe_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	content, submitted, stageSubmitted, stageOperationCount, artifactID, eventID, err := server.loadFinalEditStageDraftContent(ctx, binding)
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, nil)
	}
	if strings.TrimSpace(content) == "" || len([]byte(content)) > patchhandler.ReportPatchMaxBytes || !utf8.ValidString(content) {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "bound long-form stage manuscript is not readable UTF-8 Markdown", false, nil)
	}
	now := time.Now().UTC()
	draft := &LongFormStageEditDraft{
		DraftID: draftID, Stage: binding.Stage, MissionID: common.MissionID, SessionID: common.SessionID,
		PendingID: binding.PendingEventID, PlanEventID: binding.PlanEventID, Content: content,
		Submitted: submitted, StageSubmitted: stageSubmitted, StageOperationCount: stageOperationCount,
		ArtifactID: artifactID, EventID: eventID, CreatedAt: now, UpdatedAt: now,
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if len(server.State.StageDrafts) >= reportLongFormEditMaxDrafts {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "too many in-process long-form stage edit drafts", false, nil)
	}
	if _, exists := server.State.StageDrafts[draftID]; exists {
		return server.ErrorResult(call.Name, common.MissionID, "conflict", "long-form stage edit draft already exists", false, []string{draftID})
	}
	server.State.StageDrafts[draftID] = draft
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, Content: longFormStageEditFromState(*draft)}
}

func (server *Handler) loadFinalEditStageDraftContent(ctx context.Context, binding reporting.FinalEditStageBinding) (string, bool, bool, int, string, string, error) {
	if existing, ok, err := reporting.LoadFinalEditStageSubmission(ctx, server.Service, binding); err != nil {
		return "", false, false, 0, "", "", err
	} else if ok {
		if binding.Stage == reporting.FinalEditStageGate {
			return string(existing.Artifact.Content), false, true, existing.OperationCount, existing.Artifact.ArtifactID, existing.Event.EventID, nil
		}
		return string(existing.Artifact.Content), true, false, 0, existing.Artifact.ArtifactID, existing.Event.EventID, nil
	}
	if _, _, err := reporting.StartFinalEditStage(ctx, server.Service, server.NewID("evt"), binding); err != nil {
		return "", false, false, 0, "", "", err
	}
	source, err := server.Service.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return "", false, false, 0, "", "", err
	}
	if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.Filename != binding.Filename {
		return "", false, false, 0, "", "", fmt.Errorf("%w: final edit stage source artifact is outside the binding", producterror.ErrConflict)
	}
	return string(source.Content), false, false, 0, "", "", nil
}

func (server *Handler) CallReportLongFormStageEditRead(ctx context.Context, call wire.ToolCall, expectedStage string) wire.ToolResult {
	_ = ctx
	var input ReportLongFormEditReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "long-form stage edit read arguments are invalid", false, nil)
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
	if _, err := server.RequireStageBinding(wire.CommonMutatingInput{MissionID: missionID, SessionID: sessionID}, expectedStage); err != nil {
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
	if err := validateLongFormStageEditAccess(&copyDraft, missionID, sessionID, expectedStage); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := patchhandler.BoundedContent(copyDraft.Content, input.Offset, input.MaxBytes)
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"draft_id": draftID, "stage": copyDraft.Stage, "content": content, "offset": offset, "next_offset": nextOffset,
		"content_length": len([]byte(copyDraft.Content)), "truncated": truncated, "submitted": copyDraft.Submitted || copyDraft.StageSubmitted,
	}}
}

func (server *Handler) CallReportLongFormStageEditPatch(ctx context.Context, call wire.ToolCall, expectedStage string) wire.ToolResult {
	_ = ctx
	var input ReportLongFormEditPatchInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "long-form stage edit patch arguments are invalid", false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if _, err := server.RequireStageBinding(common, expectedStage); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	return server.patchLongFormStageEditDraft(call, common, input, expectedStage)
}
