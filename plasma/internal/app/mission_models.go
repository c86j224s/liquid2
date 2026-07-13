package app

import (
	"encoding/json"
	"time"
)

type Mission struct {
	MissionID string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
	Activity  MissionActivitySummary `json:"activity"`
}

// MissionActivityInput is the minimum durable ledger input needed to derive a
// mission-list activity summary. LastSequence includes every ledger event;
// Events contains only activity-relevant event types.
type MissionActivityInput struct {
	MissionID    string
	LastSequence int64
	Events       []LedgerEvent
}

// MissionActivitySummary is the lightweight, mission-scoped activity view for
// lists. It is derived from the durable ledger and never stores read state.
type MissionActivitySummary struct {
	LastSequence           int64                 `json:"last_sequence"`
	ActiveWork             ActiveWorkState       `json:"active_work"`
	LatestTerminalActivity *TerminalActivityView `json:"latest_terminal_activity,omitempty"`
}

type TerminalActivityKind string

type TerminalActivityOutcome string

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

type Producer struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type LedgerEvent struct {
	EventID          string
	MissionID        string
	Sequence         int64
	EventType        string
	Producer         Producer
	CausationEventID string
	CorrelationID    string
	Payload          json.RawMessage
	CreatedAt        time.Time
}

type CreateMissionRequest struct {
	MissionID string
	Title     string
}

type MissionCreatedEventRequest struct {
	EventID   string
	MissionID string
	Title     string
	Objective string
	Scope     MissionScope
	Producer  Producer
}

type AppendEventRequest struct {
	EventID          string
	MissionID        string
	EventType        string
	Producer         Producer
	CausationEventID string
	CorrelationID    string
	Payload          json.RawMessage
}
