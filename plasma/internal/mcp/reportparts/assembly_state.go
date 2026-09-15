package reportparts

import (
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	reportPartAssemblyMaxDrafts     = 8
	reportPartAssemblyMaxPatchBytes = 64 * 1024
	reportPartAssemblyMaxOperations = 32
)

type PartAssemblyDraft struct {
	DraftID      string
	MissionID    string
	SessionID    string
	PendingID    string
	PlanEventID  string
	PartIndex    int
	SectionCount int
	Assembly     reporting.PartAssembly
	Operations   []PartAssemblyOperation
	Submitted    bool
	EventID      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PartAssemblyOperation struct {
	Field             string `json:"field"`
	AfterSectionIndex int    `json:"after_section_index,omitempty"`
	Summary           string `json:"summary,omitempty"`
	Bytes             int    `json:"bytes"`
}

type ReportPartAssemblyStartInput struct {
	wire.CommonMutatingInput
	DraftID        string `json:"draft_id"`
	PendingEventID string `json:"pending_event_id"`
	PlanEventID    string `json:"plan_event_id"`
	PartIndex      int    `json:"part_index"`
	SectionCount   int    `json:"section_count"`
}

type ReportPartAssemblyReadInput struct {
	MissionID string `json:"mission_id"`
	SessionID string `json:"session_id"`
	DraftID   string `json:"draft_id"`
}

type ReportPartSectionReadInput struct {
	MissionID    string `json:"mission_id"`
	SessionID    string `json:"session_id"`
	SectionIndex int    `json:"section_index"`
	Offset       int    `json:"offset"`
	MaxBytes     int    `json:"max_bytes"`
}

type ReportPartAssemblyPatchInput struct {
	wire.CommonMutatingInput
	DraftID           string `json:"draft_id"`
	Field             string `json:"field"`
	AfterSectionIndex int    `json:"after_section_index"`
	Markdown          string `json:"markdown"`
	Summary           string `json:"summary"`
}

type ReportPartAssemblySubmitInput struct {
	wire.CommonMutatingInput
	DraftID        string `json:"draft_id"`
	PendingEventID string `json:"pending_event_id"`
	PlanEventID    string `json:"plan_event_id"`
}

func validatePartAssemblyAccess(draft PartAssemblyDraft, missionID string, sessionID string) error {
	if draft.MissionID != strings.TrimSpace(missionID) || draft.SessionID != strings.TrimSpace(sessionID) {
		return fmt.Errorf("%w: part assembly draft is outside this MCP session", producterror.ErrInvalidInput)
	}
	return nil
}

func applyPartAssemblyPatch(draft *PartAssemblyDraft, field string, afterSectionIndex int, markdown string) error {
	switch field {
	case "intro":
		draft.Assembly.Intro = markdown
	case "closing":
		draft.Assembly.Closing = markdown
	case "transition":
		if afterSectionIndex < 1 || afterSectionIndex >= draft.SectionCount {
			return fmt.Errorf("%w: transition after_section_index must refer to a section before the next section", producterror.ErrInvalidInput)
		}
		draft.Assembly.Transitions = upsertPartTransition(draft.Assembly.Transitions, afterSectionIndex, markdown)
	default:
		return fmt.Errorf("%w: unsupported part assembly field", producterror.ErrInvalidInput)
	}
	return nil
}

func upsertPartTransition(transitions []reporting.PartTransition, afterSectionIndex int, markdown string) []reporting.PartTransition {
	out := make([]reporting.PartTransition, 0, len(transitions)+1)
	updated := false
	for _, transition := range transitions {
		if transition.AfterSectionIndex == afterSectionIndex {
			updated = true
			if markdown != "" {
				out = append(out, reporting.PartTransition{AfterSectionIndex: afterSectionIndex, Markdown: markdown})
			}
			continue
		}
		out = append(out, transition)
	}
	if !updated && markdown != "" {
		out = append(out, reporting.PartTransition{AfterSectionIndex: afterSectionIndex, Markdown: markdown})
	}
	return out
}

func partAssemblyFromState(draft PartAssemblyDraft) map[string]any {
	return map[string]any{
		"draft_id":         draft.DraftID,
		"mission_id":       draft.MissionID,
		"session_id":       draft.SessionID,
		"pending_event_id": draft.PendingID,
		"plan_event_id":    draft.PlanEventID,
		"part_index":       draft.PartIndex,
		"section_count":    draft.SectionCount,
		"assembly":         draft.Assembly,
		"operation_count":  len(draft.Operations),
		"operations":       draft.Operations,
		"submitted":        draft.Submitted,
		"event_id":         draft.EventID,
	}
}
