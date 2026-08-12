package workflow

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// Run advances a workflow until it reaches a terminal state. A terminal run is
// returned unchanged and an open agent turn prevents duplicate execution.
func (runner Runner) Run(ctx context.Context, missionID string, workflowRunID string) (workflowstate.WorkflowRunView, error) {
	if runner.Service == nil {
		return workflowstate.WorkflowRunView{}, fmt.Errorf("%w: workflow service is required", producterror.ErrInvalidInput)
	}
	if runner.Agent == nil {
		return workflowstate.WorkflowRunView{}, fmt.Errorf("%w: workflow agent executor is required", producterror.ErrInvalidInput)
	}
	view, err := runner.Service.GetWorkflowRun(ctx, missionID, workflowRunID)
	if err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	if workflowTerminalStatus(view.Status) {
		return view, nil
	}
	events, err := runner.Service.ListEvents(ctx, missionID)
	if err != nil {
		return workflowstate.WorkflowRunView{}, err
	}
	if view.StartAfterEventID != "" && !hasAgentTerminalEventForUser(events, view.StartAfterEventID) {
		return view, nil
	}
	if hasOpenAgentPending(events) {
		return view, fmt.Errorf("%w: agent turn is already running for this mission", producterror.ErrInvalidInput)
	}
	if view.StartedEventID == "" {
		claimedView, claimed, err := runner.Service.ClaimWorkflowRunStart(ctx, missionID, workflowRunID, runner.now())
		if err != nil {
			return workflowstate.WorkflowRunView{}, err
		}
		if !claimed {
			return claimedView, nil
		}
		view = claimedView
	}

	startedAt := runner.now()
	for {
		if err := ctx.Err(); err != nil {
			return runner.terminal(ctx, missionID, workflowRunID, workflowstate.WorkflowRunInterruptedEvent, "context canceled", err.Error())
		}
		view, err = runner.Service.GetWorkflowRun(ctx, missionID, workflowRunID)
		if err != nil {
			return workflowstate.WorkflowRunView{}, err
		}
		if workflowTerminalStatus(view.Status) {
			return view, nil
		}
		if view.StopRequestedEventID != "" {
			return runner.terminal(ctx, missionID, workflowRunID, workflowstate.WorkflowRunStoppedEvent, firstNonEmpty(view.StopReason, "stop requested"), "")
		}
		if view.MaxSteps > 0 && view.CompletedStepCount >= view.MaxSteps {
			return runner.limitReached(ctx, missionID, workflowRunID, view, "max_steps reached")
		}
		if view.MaxDurationMS > 0 && runner.now().Sub(startedAt).Milliseconds() >= view.MaxDurationMS {
			return runner.limitReached(ctx, missionID, workflowRunID, view, "max_duration reached")
		}
		compacted, err := runner.compactBeforeNextStep(ctx, view)
		if err != nil {
			return runner.terminal(ctx, missionID, workflowRunID, workflowstate.WorkflowRunFailedEvent, "proactive context compaction failed", err.Error())
		}
		if compacted {
			continue
		}
		if _, err := runner.runStep(ctx, view); err != nil {
			return runner.terminal(ctx, missionID, workflowRunID, workflowstate.WorkflowRunFailedEvent, "workflow step failed", err.Error())
		}
	}
}
