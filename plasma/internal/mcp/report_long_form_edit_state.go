package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	reportLongFormEditMaxDrafts     = 2
	reportLongFormEditMaxOperations = 64
)

type longFormEditDraft struct {
	DraftID     string
	MissionID   string
	SessionID   string
	PendingID   string
	PlanEventID string
	Content     string
	Operations  []reportPatchOperation
	Finalizing  bool
	Submitted   bool
	ArtifactID  string
	EventID     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type reportLongFormEditStartInput struct {
	CommonMutatingInput
	DraftID        string `json:"draft_id"`
	PendingEventID string `json:"pending_event_id"`
	PlanEventID    string `json:"plan_event_id"`
}

type reportLongFormEditReadInput struct {
	MissionID string `json:"mission_id"`
	SessionID string `json:"session_id"`
	DraftID   string `json:"draft_id"`
	Offset    int    `json:"offset"`
	MaxBytes  int    `json:"max_bytes"`
}

type reportLongFormEditPatchInput struct {
	CommonMutatingInput
	DraftID     string `json:"draft_id"`
	Operation   string `json:"operation"`
	MatchText   string `json:"match_text"`
	Replacement string `json:"replacement"`
	Occurrence  int    `json:"occurrence"`
	ReplaceAll  bool   `json:"replace_all"`
	Summary     string `json:"summary"`
}

type reportLongFormEditSubmitInput struct {
	CommonMutatingInput
	DraftID        string `json:"draft_id"`
	PendingEventID string `json:"pending_event_id"`
	PlanEventID    string `json:"plan_event_id"`
}

func (server *Server) requireLongFormEditBinding(common commonMutatingInput) (reporting.LongFormFinalizeBinding, error) {
	if err := server.requireBoundWriteSession(common); err != nil {
		return reporting.LongFormFinalizeBinding{}, err
	}
	binding := server.longFormFinalizeBinding
	if err := ValidateLongFormFinalizeBinding(server.binding, binding); err != nil {
		return reporting.LongFormFinalizeBinding{}, err
	}
	if binding.CompositionStrategy != reporting.LongFormCompositionNarrativeEdit {
		return reporting.LongFormFinalizeBinding{}, fmt.Errorf("%w: long-form final editor is not enabled for this composition strategy", app.ErrInvalidInput)
	}
	return binding, nil
}

func (server *Server) longFormEditToolEnabled(name string) bool {
	binding := server.longFormFinalizeBinding
	return server.toolEnabled(name) && binding.CompositionStrategy == reporting.LongFormCompositionNarrativeEdit && ValidateLongFormFinalizeBinding(server.binding, binding) == nil
}

func longFormEditDisabledResult(call ToolCall) ToolResult {
	return errorResult(call.Name, missionIDFromArguments(call.Arguments), "binding", "long-form final editor tools are only enabled for a bound narrative-edit session", false, nil)
}

func validateLongFormEditAccess(draft *longFormEditDraft, missionID string, sessionID string) error {
	if draft == nil || draft.MissionID != strings.TrimSpace(missionID) || draft.SessionID != strings.TrimSpace(sessionID) {
		return fmt.Errorf("%w: long-form edit draft is outside this MCP session", app.ErrInvalidInput)
	}
	return nil
}

func longFormEditFromState(draft longFormEditDraft) map[string]any {
	state := "open"
	if draft.Submitted {
		state = "submitted"
	} else if draft.Finalizing {
		state = "finalizing"
	}
	return map[string]any{
		"draft_id": draft.DraftID, "mission_id": draft.MissionID, "session_id": draft.SessionID,
		"pending_event_id": draft.PendingID, "plan_event_id": draft.PlanEventID,
		"state": state, "content_length": len([]byte(draft.Content)), "operation_count": len(draft.Operations),
		"submitted": draft.Submitted, "artifact_id": draft.ArtifactID, "event_id": draft.EventID,
	}
}
