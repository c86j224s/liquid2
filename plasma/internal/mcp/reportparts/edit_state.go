package reportparts

import (
	"fmt"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const (
	reportPartEditMaxDrafts     = 8
	reportPartEditMaxOperations = 64
)

type PartEditDraft struct {
	DraftID    string
	MissionID  string
	SessionID  string
	Content    string
	Operations []patchhandler.Operation
	Finalizing bool
	Submitted  bool
	ArtifactID string
	EventID    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ReportPartEditStartInput struct {
	wire.CommonMutatingInput
	DraftID          string `json:"draft_id"`
	PendingEventID   string `json:"pending_event_id"`
	PlanEventID      string `json:"plan_event_id"`
	PartIndex        int    `json:"part_index"`
	SourceArtifactID string `json:"source_artifact_id"`
}

type ReportPartEditReadInput struct {
	MissionID string `json:"mission_id"`
	SessionID string `json:"session_id"`
	DraftID   string `json:"draft_id"`
	Offset    int    `json:"offset"`
	MaxBytes  int    `json:"max_bytes"`
}

type ReportPartEditPatchInput struct {
	wire.CommonMutatingInput
	DraftID     string `json:"draft_id"`
	Operation   string `json:"operation"`
	MatchText   string `json:"match_text"`
	Replacement string `json:"replacement"`
	Occurrence  int    `json:"occurrence"`
	ReplaceAll  bool   `json:"replace_all"`
	Summary     string `json:"summary"`
}

type ReportPartEditSubmitInput struct {
	wire.CommonMutatingInput
	DraftID        string `json:"draft_id"`
	PendingEventID string `json:"pending_event_id"`
	PlanEventID    string `json:"plan_event_id"`
}

func validatePartEditAccess(draft *PartEditDraft, missionID, sessionID string) error {
	if draft == nil || draft.MissionID != strings.TrimSpace(missionID) || draft.SessionID != strings.TrimSpace(sessionID) {
		return fmt.Errorf("%w: Part edit draft is outside this MCP session", producterror.ErrInvalidInput)
	}
	return nil
}

func partEditFromState(draft PartEditDraft) map[string]any {
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
