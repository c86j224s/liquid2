package workflow

import (
	"context"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// SupervisorService is the durable state boundary used to reconcile workflow
// execution after requests, stops, and process restarts.
type SupervisorService interface {
	ListEvents(context.Context, string) ([]ledger.Event, error)
	AppendEvents(context.Context, string, []ledger.AppendRequest) ([]ledger.Event, error)
	ListWorkflowRuns(context.Context, string) ([]workflowstate.WorkflowRunView, error)
}

// RunExecutor advances one durable workflow run until it reaches a terminal
// state or the supplied context is canceled.
type RunExecutor interface {
	Run(context.Context, string, string) (workflowstate.WorkflowRunView, error)
}

// RunnerFactory wires the provider and model settings needed by one run. The
// supervisor owns process lifetime; the composition adapter owns provider
// selection.
type RunnerFactory func(context.Context, string, string) (RunExecutor, error)

// SupervisorOptions contains the replaceable boundaries required by the
// process-local workflow supervisor.
type SupervisorOptions struct {
	Service        SupervisorService
	RunnerFactory  RunnerFactory
	AgentAvailable func(string) bool
	NewID          func(string) string
}

// Supervisor owns process-local workflow execution, cancellation, and durable
// reconciliation. Durable run identity and status remain in the mission ledger.
type Supervisor struct {
	service        SupervisorService
	runnerFactory  RunnerFactory
	agentAvailable func(string) bool
	newID          func(string) string
	runs           runRegistry
	missions       missionLocks
}

// NewSupervisor constructs a workflow execution supervisor without starting
// background work.
func NewSupervisor(options SupervisorOptions) *Supervisor {
	return &Supervisor{
		service:        options.Service,
		runnerFactory:  options.RunnerFactory,
		agentAvailable: options.AgentAvailable,
		newID:          options.NewID,
	}
}

// Start begins one in-process runner unless the same durable run is already
// owned by this process.
func (supervisor *Supervisor) Start(missionID string, workflowRunID string, executorName string) bool {
	if supervisor == nil || supervisor.runnerFactory == nil {
		return false
	}
	workerCtx, cancel := context.WithCancel(context.Background())
	ownerID, ok := supervisor.runs.start(workflowRunID, cancel, supervisor.newID)
	if !ok {
		cancel()
		return false
	}
	go func() {
		defer cancel()
		defer supervisor.runs.finish(workflowRunID, ownerID)
		runner, err := supervisor.runnerFactory(context.Background(), missionID, executorName)
		if err != nil || runner == nil {
			return
		}
		_, _ = runner.Run(workerCtx, missionID, workflowRunID)
	}()
	return true
}

// Reconcile applies the existing queued, stopping, and interrupted recovery
// rules to the durable projection for one mission.
func (supervisor *Supervisor) Reconcile(ctx context.Context, missionID string) error {
	if supervisor == nil || supervisor.service == nil {
		return nil
	}
	unlock := supervisor.missions.lock(missionID)
	defer unlock()
	runs, err := supervisor.service.ListWorkflowRuns(ctx, missionID)
	if err != nil {
		return err
	}
	events, err := supervisor.service.ListEvents(ctx, missionID)
	if err != nil {
		return err
	}
	for _, run := range runs {
		switch run.Status {
		case workflowstate.WorkflowStatusInterrupted, workflowstate.WorkflowStatusFailed, workflowstate.WorkflowStatusStopped:
			_ = supervisor.closeOpenPending(ctx, run, events)
		case workflowstate.WorkflowStatusStopping:
			supervisor.Cancel(run.WorkflowRunID)
			if _, err := supervisor.StopRunNow(ctx, missionID, run.WorkflowRunID, firstNonEmpty(run.StopReason, "workflow stop requested")); err == nil {
				return nil
			}
		case workflowstate.WorkflowStatusQueued:
			if strings.TrimSpace(run.StartAfterEventID) == "" || !conversation.HasAgentTerminalEventForUser(events, run.StartAfterEventID) {
				continue
			}
			if supervisor.agentAvailable == nil || !supervisor.agentAvailable(run.AgentExecutor) {
				continue
			}
			supervisor.Start(missionID, run.WorkflowRunID, run.AgentExecutor)
			return nil
		}
	}
	return nil
}

// Cancel requests cancellation of a run currently owned by this process.
func (supervisor *Supervisor) Cancel(workflowRunID string) bool {
	return supervisor != nil && supervisor.runs.cancel(workflowRunID)
}

// Has reports whether this process currently owns the run.
func (supervisor *Supervisor) Has(workflowRunID string) bool {
	return supervisor != nil && supervisor.runs.has(workflowRunID)
}
