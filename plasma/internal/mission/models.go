package mission

import (
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

const (
	ActiveWorkTurn     = "agent_turn"
	ActiveWorkReport   = "report_generation"
	ActiveWorkWorkflow = "workflow_run"

	BlockingReasonAgentTurn = "agent_turn_running"
	BlockingReasonReport    = "report_generation_running"
	BlockingReasonWorkflow  = "workflow_running"

	LifecycleActive   = "active"
	LifecycleArchived = "archived"
	ArchivedEvent     = "mission.archived"
	RestoredEvent     = "mission.restored"
)

// ActiveWorkView describes one durable in-progress activity for UI/API views.
type ActiveWorkView struct {
	Kind           string `json:"kind"`
	Status         string `json:"status"`
	ReasonCode     string `json:"reason_code"`
	Action         string `json:"action"`
	Target         string `json:"target"`
	PendingEventID string `json:"pending_event_id,omitempty"`
	WorkflowRunID  string `json:"workflow_run_id,omitempty"`
}

// ActiveWorkControl describes a blocked user control and stable reason codes.
type ActiveWorkControl struct {
	Control     string   `json:"control"`
	ReasonCodes []string `json:"reason_codes"`
}

// ActiveWorkState is the mission-level active-work summary. Items is the
// canonical field; Blocks remains populated for existing clients.
type ActiveWorkState struct {
	Items           []ActiveWorkView    `json:"items"`
	Blocks          []ActiveWorkView    `json:"blocks"`
	BlockedControls []ActiveWorkControl `json:"blocked_controls"`
}

// Mission is the durable mission identity and list projection.
type Mission struct {
	MissionID      string
	Title          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LifecycleState string          `json:"lifecycle_state"`
	Activity       ActivitySummary `json:"activity"`
}

// ListRequest controls whether archived missions are included.
type ListRequest struct {
	IncludeArchived bool
}

// ActivityInput is the minimal durable input for a mission list summary.
type ActivityInput struct {
	MissionID    string
	LastSequence int64
	Events       []ledger.Event
}

// ActivitySummary is a lightweight mission-list read model.
type ActivitySummary struct {
	LastSequence           int64                 `json:"last_sequence"`
	ActiveWork             ActiveWorkState       `json:"active_work"`
	LatestTerminalActivity *TerminalActivityView `json:"latest_terminal_activity,omitempty"`
}

type TerminalActivityKind string
type TerminalActivityOutcome string

// TerminalActivityView identifies the latest completed mission activity.
type TerminalActivityView struct {
	EventID  string                  `json:"event_id"`
	Sequence int64                   `json:"sequence"`
	Kind     TerminalActivityKind    `json:"kind"`
	Outcome  TerminalActivityOutcome `json:"outcome"`
}

const (
	TerminalActivityTurn     TerminalActivityKind = ActiveWorkTurn
	TerminalActivityReport   TerminalActivityKind = ActiveWorkReport
	TerminalActivityWorkflow TerminalActivityKind = ActiveWorkWorkflow

	TerminalActivityCompleted TerminalActivityOutcome = "completed"
	TerminalActivityFailed    TerminalActivityOutcome = "failed"
	TerminalActivityCanceled  TerminalActivityOutcome = "canceled"
	TerminalActivityPaused    TerminalActivityOutcome = "paused"
	TerminalActivityStopped   TerminalActivityOutcome = "stopped"
)

// CreateRequest contains the stable inputs for a new mission.
type CreateRequest struct {
	MissionID string
	Title     string
}

// Scope records explicitly included and excluded mission boundaries.
type Scope struct {
	Included []string `json:"included"`
	Excluded []string `json:"excluded"`
}

// CreatedEventRequest contains the fields for a mission.created event.
type CreatedEventRequest struct {
	EventID   string
	MissionID string
	Title     string
	Objective string
	Scope     Scope
	Producer  ledger.Producer
}

// Projection is a durable-ledger-derived mission read model.
type Projection struct {
	MissionID             string   `json:"mission_id"`
	LastEventID           string   `json:"last_event_id"`
	LastSequence          int64    `json:"last_sequence"`
	Title                 string   `json:"title"`
	Objective             string   `json:"objective"`
	Scope                 Scope    `json:"scope"`
	ActiveSessionIDs      []string `json:"active_session_ids"`
	AcceptedClaimIDs      []string `json:"accepted_claim_ids"`
	OpenQuestionIDs       []string `json:"open_question_ids"`
	ActiveReportVersionID string   `json:"active_report_version_id"`
	LifecycleState        string   `json:"lifecycle_state"`
	NeedsReview           bool     `json:"needs_review"`
	NeedsReviewReasons    []string `json:"needs_review_reasons"`
}
