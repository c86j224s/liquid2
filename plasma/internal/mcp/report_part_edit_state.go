package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	reportPartEditMaxDrafts     = 8
	reportPartEditMaxOperations = 64
)

type partEditDraft struct {
	DraftID    string
	MissionID  string
	SessionID  string
	Content    string
	Operations []reportPatchOperation
	Finalizing bool
	Submitted  bool
	ArtifactID string
	EventID    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type reportPartEditStartInput struct {
	CommonMutatingInput
	DraftID          string `json:"draft_id"`
	PendingEventID   string `json:"pending_event_id"`
	PlanEventID      string `json:"plan_event_id"`
	PartIndex        int    `json:"part_index"`
	SourceArtifactID string `json:"source_artifact_id"`
}

type reportPartEditReadInput struct {
	MissionID string `json:"mission_id"`
	SessionID string `json:"session_id"`
	DraftID   string `json:"draft_id"`
	Offset    int    `json:"offset"`
	MaxBytes  int    `json:"max_bytes"`
}

type reportPartEditPatchInput struct {
	CommonMutatingInput
	DraftID     string `json:"draft_id"`
	Operation   string `json:"operation"`
	MatchText   string `json:"match_text"`
	Replacement string `json:"replacement"`
	Occurrence  int    `json:"occurrence"`
	ReplaceAll  bool   `json:"replace_all"`
	Summary     string `json:"summary"`
}

type reportPartEditSubmitInput struct {
	CommonMutatingInput
	DraftID        string `json:"draft_id"`
	PendingEventID string `json:"pending_event_id"`
	PlanEventID    string `json:"plan_event_id"`
}

func (server *Server) requirePartEditBinding(common commonMutatingInput) (reporting.PartEditBinding, error) {
	if err := server.requireBoundWriteSession(common); err != nil {
		return reporting.PartEditBinding{}, err
	}
	if err := ValidatePartEditBinding(server.binding, server.partEditBinding); err != nil {
		return reporting.PartEditBinding{}, err
	}
	return server.partEditBinding, nil
}

func (server *Server) partEditToolEnabled(name string) bool {
	return server.toolEnabled(name) && ValidatePartEditBinding(server.binding, server.partEditBinding) == nil
}

func partEditDisabledResult(call ToolCall) ToolResult {
	return errorResult(call.Name, missionIDFromArguments(call.Arguments), "binding", "Part editor tools are only enabled for one bound assembled Part", false, nil)
}

func validatePartEditAccess(draft *partEditDraft, missionID, sessionID string) error {
	if draft == nil || draft.MissionID != strings.TrimSpace(missionID) || draft.SessionID != strings.TrimSpace(sessionID) {
		return fmt.Errorf("%w: Part edit draft is outside this MCP session", app.ErrInvalidInput)
	}
	return nil
}

func partEditFromState(draft partEditDraft) map[string]any {
	state := "open"
	if draft.Submitted {
		state = "submitted"
	} else if draft.Finalizing {
		state = "finalizing"
	}
	return map[string]any{
		"draft_id": draft.DraftID, "mission_id": draft.MissionID, "session_id": draft.SessionID,
		"state": state, "content_length": len([]byte(draft.Content)), "operation_count": len(draft.Operations),
		"submitted": draft.Submitted, "artifact_id": draft.ArtifactID, "event_id": draft.EventID,
	}
}
