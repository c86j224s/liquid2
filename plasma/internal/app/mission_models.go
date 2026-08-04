package app

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
)

// Deprecated: capability code should import internal/mission directly.
type Mission = mission.Mission
type ListMissionsRequest = mission.ListRequest
type MissionActivityInput = mission.ActivityInput
type MissionActivitySummary = mission.ActivitySummary
type TerminalActivityKind = mission.TerminalActivityKind
type TerminalActivityOutcome = mission.TerminalActivityOutcome
type TerminalActivityView = mission.TerminalActivityView

const (
	TerminalActivityTurn      = mission.TerminalActivityTurn
	TerminalActivityReport    = mission.TerminalActivityReport
	TerminalActivityWorkflow  = mission.TerminalActivityWorkflow
	TerminalActivityCompleted = mission.TerminalActivityCompleted
	TerminalActivityFailed    = mission.TerminalActivityFailed
	TerminalActivityCanceled  = mission.TerminalActivityCanceled
	TerminalActivityPaused    = mission.TerminalActivityPaused
	TerminalActivityStopped   = mission.TerminalActivityStopped
)

// Deprecated: capability code should import internal/ledger directly.
type Producer = ledger.Producer
type LedgerEvent = ledger.Event
type AppendEventRequest = ledger.AppendRequest

// Deprecated: capability code should import internal/mission directly.
type CreateMissionRequest = mission.CreateRequest
type MissionCreatedEventRequest = mission.CreatedEventRequest
