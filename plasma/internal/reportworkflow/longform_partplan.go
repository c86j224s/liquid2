package reportworkflow

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partplan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/requirements"
)

func (runner Runner) runPartPlans(ctx context.Context, input DraftInput, planOut plan.LongFormOutput, reqOut requirements.Output, progress longFormProgress) (map[int]partplan.Output, error) {
	plans := map[int]partplan.Output{}
	for index, value := range progress.partPlans {
		plans[index] = value
	}
	type task struct {
		index int
		input partplan.Input
	}
	tasks := []task{}
	forker, ok := runner.partPlanRunner.Executor.(agentexec.AgentSessionForker)
	if !ok {
		return nil, longformutil.StageFailure("part_plan", planOut.Event.EventID, 0, 0, fmt.Errorf("%w: Part planning requires an agent session forker", producterror.ErrInvalidInput))
	}
	for index, part := range planOut.Plan.Parts {
		if _, ok := plans[index]; ok {
			continue
		}
		if _, hasPart := progress.parts[index]; hasPart || hasRecoveredSection(progress, index) {
			return nil, fmt.Errorf("%w: Part output exists without its planning state", producterror.ErrConflict)
		}
		sessionID, forkSource, err := forkLongFormSession(ctx, forker, planOut.ReportPlanSessionID)
		if err != nil {
			return nil, longformutil.StageFailure("part_plan", planOut.Event.EventID, index+1, 0, err)
		}
		tasks = append(tasks, task{index: index, input: partplan.Input{
			Base: partPlanBase(input, planOut, reqOut), Part: part, PartIndex: index,
			ProviderSessionID: sessionID, ForkSourceSession: firstNonEmpty(forkSource, planOut.ReportPlanSessionID),
		}})
	}
	if len(tasks) == 0 {
		return plans, nil
	}
	outputs := make([]partplan.Output, len(tasks))
	err := runLimited(len(tasks), longFormWorkerLimit, func(i int) error {
		out, err := runner.partPlanRunner.Run(ctx, tasks[i].input)
		if err != nil {
			return err
		}
		outputs[i] = out
		return nil
	})
	if err != nil {
		return nil, err
	}
	for index, task := range tasks {
		plans[task.index] = outputs[index]
	}
	if len(plans) != len(planOut.Plan.Parts) {
		return nil, fmt.Errorf("%w: Part planning left a Part incomplete", producterror.ErrConflict)
	}
	return plans, nil
}
