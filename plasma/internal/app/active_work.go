package app

import "github.com/c86j224s/liquid2/plasma/internal/mission"

const (
	ActiveWorkTurn          = mission.ActiveWorkTurn
	ActiveWorkReport        = mission.ActiveWorkReport
	ActiveWorkWorkflow      = mission.ActiveWorkWorkflow
	BlockingReasonAgentTurn = mission.BlockingReasonAgentTurn
	BlockingReasonReport    = mission.BlockingReasonReport
	BlockingReasonWorkflow  = mission.BlockingReasonWorkflow
)

type ActiveWorkView = mission.ActiveWorkView
type ActiveWorkControl = mission.ActiveWorkControl
type ActiveWorkState = mission.ActiveWorkState

// ActiveWorkFromMissionState preserves the existing app facade while the
// mission capability owns the projection rule.
func ActiveWorkFromMissionState(events []LedgerEvent, runs []WorkflowRunView) ActiveWorkState {
	return mission.ActiveWorkFromState(events, runs)
}
