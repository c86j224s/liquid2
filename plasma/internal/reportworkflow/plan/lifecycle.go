package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const ReportPlanSubmittedSentinel = "PLAN_SUBMITTED"

// ReportPlanLifecycleService is the durable plan submission and promotion port.
type ReportPlanLifecycleService interface {
	SelectReportPlanSubmission(context.Context, reporting.ReportPlanSubmissionQuery) (reporting.ReportPlanSubmissionSelection, error)
	PromoteReportPlan(context.Context, reporting.PromoteReportPlanRequest) (ledger.Event, error)
}

// ReportPlanLifecycleBinding carries the provider tool and idempotency identities.
type ReportPlanLifecycleBinding struct {
	ToolSessionID, IdempotencyKey string
}

// ReportPlanLifecycleAgentResult carries the provider sentinel and session identity.
type ReportPlanLifecycleAgentResult struct {
	Text, SessionID string
}

// ReportPlanLifecycleRequest is the stage-owned plan lifecycle input.
type ReportPlanLifecycleRequest struct {
	MissionID, PendingEventID, ReportMode, AgentExecutor, AgentModel, AgentReasoningEffort, PreviousProviderSessionID string
	Invoke                                                                                                            func(context.Context, ReportPlanLifecycleBinding) (ReportPlanLifecycleAgentResult, error)
	BuildCanonical                                                                                                    func(any, reporting.ReportPlanSubmissionSelection, ReportPlanLifecycleBinding) (ledger.AppendRequest, error)
}

// ReportPlanLifecycleResult returns the normalized submitted plan and promoted event.
type ReportPlanLifecycleResult struct {
	Plan       any
	Event      ledger.Event
	Binding    ReportPlanLifecycleBinding
	Submission reporting.ReportPlanSubmissionSelection
	Agent      ReportPlanLifecycleAgentResult
}

// RunReportPlanLifecycle links provider submission to durable selection and canonical promotion.
func (runner Runner) RunReportPlanLifecycle(ctx context.Context, req ReportPlanLifecycleRequest) (ReportPlanLifecycleResult, error) {
	service, ok := runner.Lifecycle.Service.(ReportPlanLifecycleService)
	if !ok {
		return ReportPlanLifecycleResult{}, fmt.Errorf("%w: durable report plan lifecycle service is required", producterror.ErrInvalidInput)
	}
	if req.Invoke == nil || req.BuildCanonical == nil {
		return ReportPlanLifecycleResult{}, fmt.Errorf("%w: report plan lifecycle callbacks are required", producterror.ErrInvalidInput)
	}
	binding := ReportPlanLifecycleBinding{ToolSessionID: lifecycleID(runner, "ses"), IdempotencyKey: lifecycleID(runner, "rpk")}
	agent, err := req.Invoke(ctx, binding)
	if err != nil {
		return ReportPlanLifecycleResult{}, err
	}
	if agent.Text != ReportPlanSubmittedSentinel {
		return ReportPlanLifecycleResult{}, fmt.Errorf("%w: report planning agent did not confirm plan submission", producterror.ErrInvalidInput)
	}
	previousProviderSessionID := strings.TrimSpace(req.PreviousProviderSessionID)
	selection, err := service.SelectReportPlanSubmission(ctx, reporting.ReportPlanSubmissionQuery{MissionID: req.MissionID, PendingEventID: req.PendingEventID, ReportMode: req.ReportMode, ToolSessionID: binding.ToolSessionID, PreviousProviderSessionID: previousProviderSessionID, AgentExecutor: req.AgentExecutor, AgentModel: req.AgentModel, AgentReasoningEffort: req.AgentReasoningEffort, IdempotencyKey: binding.IdempotencyKey})
	if err != nil {
		return ReportPlanLifecycleResult{}, err
	}
	plan, err := decodeReportPlan(req.ReportMode, selection.Plan)
	if err != nil {
		return ReportPlanLifecycleResult{}, err
	}
	canonical, err := req.BuildCanonical(plan, selection, binding)
	if err != nil {
		return ReportPlanLifecycleResult{}, err
	}
	event, err := service.PromoteReportPlan(ctx, reporting.PromoteReportPlanRequest{MissionID: req.MissionID, PendingEventID: req.PendingEventID, ReportMode: req.ReportMode, ToolSessionID: binding.ToolSessionID, PreviousProviderSessionID: previousProviderSessionID, AgentExecutor: req.AgentExecutor, AgentModel: req.AgentModel, AgentReasoningEffort: req.AgentReasoningEffort, IdempotencyKey: binding.IdempotencyKey, ArgumentsHash: selection.ArgumentsHash, PlanHash: selection.PlanHash, SubmissionEventID: selection.EventID, Canonical: canonical})
	if err != nil {
		return ReportPlanLifecycleResult{}, err
	}
	return ReportPlanLifecycleResult{Plan: plan, Event: event, Binding: binding, Submission: selection, Agent: agent}, nil
}

func lifecycleID(runner Runner, prefix string) string {
	if runner.Lifecycle.NewID == nil {
		return prefix + "_report"
	}
	return runner.Lifecycle.NewID(prefix)
}

func decodeReportPlan(mode string, payload json.RawMessage) (any, error) {
	switch mode {
	case reportexecution.ModePlanned:
		var plan reporting.ReportPlan
		if json.Unmarshal(payload, &plan) != nil {
			return nil, fmt.Errorf("%w: invalid submitted planned report plan", producterror.ErrInvalidInput)
		}
		return reporting.NormalizeReportPlan(plan)
	case reportexecution.ModeLongForm:
		var plan reporting.SectionalReportPlan
		if json.Unmarshal(payload, &plan) != nil {
			return nil, fmt.Errorf("%w: invalid submitted long-form report plan", producterror.ErrInvalidInput)
		}
		return reporting.NormalizeSectionalReportPlan(plan)
	default:
		return nil, fmt.Errorf("%w: unsupported report mode", producterror.ErrInvalidInput)
	}
}
