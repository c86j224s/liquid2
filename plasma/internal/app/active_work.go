package app

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// ActiveWorkFromMissionState preserves the existing app facade while the
// mission capability owns the projection rule.
func ActiveWorkFromMissionState(events []ledger.Event, runs []workflowstate.WorkflowRunView) mission.ActiveWorkState {
	return mission.ActiveWorkFromState(events, runs)
}
