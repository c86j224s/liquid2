package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partplan"
	workflowplan "github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
)

type sectionFanoutPartPlanTask struct {
	partIndex         int
	part              agentReportPart
	providerSession   string
	forkSourceSession string
}

type sectionFanoutPartPlanOutcome struct {
	partIndex int
	plan      sectionFanoutPartPlan
	err       error
}

func longFormPartPlanningEnabled(profile string) bool {
	return workflowplan.LongFormPartPlanningEnabled(profile)
}

func (server *Server) runSectionFanoutPartPlan(ctx context.Context, req sectionFanoutLongFormRequest, state sectionFanoutPlanState, task sectionFanoutPartPlanTask, executor AgentExecutor) sectionFanoutPartPlanOutcome {
	out, err := (partplan.Runner{Service: server.service, Executor: executor, NewID: newID}).Run(ctx, partPlanInput(req, state, task))
	if err != nil {
		return sectionFanoutPartPlanOutcome{partIndex: task.partIndex, err: err}
	}
	return sectionFanoutPartPlanOutcome{partIndex: task.partIndex, plan: sectionFanoutPartPlan{
		brief: out.Brief, providerSessionID: out.ProviderSessionID, event: out.Event,
	}}
}

func agentPartPlanningPrompt(req sectionFanoutLongFormRequest, state sectionFanoutPlanState, task sectionFanoutPartPlanTask) string {
	return partplan.Prompt(partPlanInput(req, state, task))
}
