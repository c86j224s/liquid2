package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	reportPartAssemblyMaxDrafts     = 8
	reportPartAssemblyMaxPatchBytes = 64 * 1024
	reportPartAssemblyMaxOperations = 32
)

type partAssemblyDraft struct {
	DraftID      string
	MissionID    string
	SessionID    string
	PendingID    string
	PlanEventID  string
	PartIndex    int
	SectionCount int
	Assembly     reporting.PartAssembly
	Operations   []partAssemblyOperation
	Submitted    bool
	EventID      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type partAssemblyOperation struct {
	Field             string `json:"field"`
	AfterSectionIndex int    `json:"after_section_index,omitempty"`
	Summary           string `json:"summary,omitempty"`
	Bytes             int    `json:"bytes"`
}

type reportPartAssemblyStartInput struct {
	CommonMutatingInput
	DraftID        string `json:"draft_id"`
	PendingEventID string `json:"pending_event_id"`
	PlanEventID    string `json:"plan_event_id"`
	PartIndex      int    `json:"part_index"`
	SectionCount   int    `json:"section_count"`
}

type reportPartAssemblyReadInput struct {
	MissionID string `json:"mission_id"`
	SessionID string `json:"session_id"`
	DraftID   string `json:"draft_id"`
}

type reportPartSectionReadInput struct {
	MissionID    string `json:"mission_id"`
	SessionID    string `json:"session_id"`
	SectionIndex int    `json:"section_index"`
	Offset       int    `json:"offset"`
	MaxBytes     int    `json:"max_bytes"`
}

type reportPartAssemblyPatchInput struct {
	CommonMutatingInput
	DraftID           string `json:"draft_id"`
	Field             string `json:"field"`
	AfterSectionIndex int    `json:"after_section_index"`
	Markdown          string `json:"markdown"`
	Summary           string `json:"summary"`
}

type reportPartAssemblySubmitInput struct {
	CommonMutatingInput
	DraftID        string `json:"draft_id"`
	PendingEventID string `json:"pending_event_id"`
	PlanEventID    string `json:"plan_event_id"`
}

func (server *Server) requirePartAssemblyBinding(common commonMutatingInput) (reporting.PartAssemblyBinding, error) {
	if err := server.requireBoundWriteSession(common); err != nil {
		return reporting.PartAssemblyBinding{}, err
	}
	binding := server.partAssemblyBinding
	if err := ValidatePartAssemblyBinding(server.binding, binding); err != nil {
		return reporting.PartAssemblyBinding{}, err
	}
	return binding, nil
}

func (server *Server) anyPartAssemblyToolEnabled() bool {
	return server.toolEnabled(ToolReportPartAssemblyStart) ||
		server.toolEnabled(ToolReportPartAssemblyRead) ||
		server.toolEnabled(ToolReportPartSectionRead) ||
		server.toolEnabled(ToolReportPartAssemblyPatch) ||
		server.toolEnabled(ToolReportPartAssemblySubmit)
}

func (server *Server) partAssemblySectionReadToolEnabled() bool {
	return server.toolEnabled(ToolReportPartSectionRead) && reporting.ValidatePartAssemblySectionReadBinding(server.partAssemblyBinding) == nil && ValidatePartAssemblyBinding(server.binding, server.partAssemblyBinding) == nil
}

func (server *Server) partAssemblyToolEnabled(name string) bool {
	return server.toolEnabled(name) && ValidatePartAssemblyBinding(server.binding, server.partAssemblyBinding) == nil
}

func partAssemblyDisabledResult(call ToolCall) ToolResult {
	return errorResult(
		call.Name,
		missionIDFromArguments(call.Arguments),
		"binding",
		"part assembly tools are only enabled for bound long-form part assembly sessions",
		false,
		nil,
	)
}

func validatePartAssemblyAccess(draft partAssemblyDraft, missionID string, sessionID string) error {
	if draft.MissionID != strings.TrimSpace(missionID) || draft.SessionID != strings.TrimSpace(sessionID) {
		return fmt.Errorf("%w: part assembly draft is outside this MCP session", app.ErrInvalidInput)
	}
	return nil
}

func applyPartAssemblyPatch(draft *partAssemblyDraft, field string, afterSectionIndex int, markdown string) error {
	switch field {
	case "intro":
		draft.Assembly.Intro = markdown
	case "closing":
		draft.Assembly.Closing = markdown
	case "transition":
		if afterSectionIndex < 1 || afterSectionIndex >= draft.SectionCount {
			return fmt.Errorf("%w: transition after_section_index must refer to a section before the next section", app.ErrInvalidInput)
		}
		draft.Assembly.Transitions = upsertPartTransition(draft.Assembly.Transitions, afterSectionIndex, markdown)
	default:
		return fmt.Errorf("%w: unsupported part assembly field", app.ErrInvalidInput)
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

func partAssemblyFromState(draft partAssemblyDraft) map[string]any {
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

func nowUTC() time.Time {
	return time.Now().UTC()
}
