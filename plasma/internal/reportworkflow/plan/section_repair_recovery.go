package plan

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// RecoverLongFormSectionRepair reconstructs the one durable plan amendment in
// the current report retry lineage. The canonical plan event remains immutable.
func (runner Runner) RecoverLongFormSectionRepair(ctx context.Context, input LongFormInput, planOut LongFormOutput) (LongFormSectionRepairOutput, bool, error) {
	events, err := runner.Service.ListEvents(ctx, input.MissionID)
	if err != nil {
		return LongFormSectionRepairOutput{}, false, err
	}
	result, ok, err := reporting.RecoverLongFormSectionPlanRepair(
		events, input.MissionID, input.PendingEventID, planOut.Event.EventID, planOut.Plan,
	)
	if err != nil || !ok {
		return LongFormSectionRepairOutput{}, ok, err
	}
	if result.Unrepairable {
		return LongFormSectionRepairOutput{}, true, fmt.Errorf("%w: planner previously found no supportable replacement Section", producterror.ErrConflict)
	}
	return LongFormSectionRepairOutput{
		Plan: result.Plan, Event: result.Event, Replacements: result.Replacements, Recovered: true,
	}, true, nil
}
