package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const ReportPlanSubmittedSentinel = "PLAN_SUBMITTED"

type ReportPlanLifecycleService interface {
	SelectReportPlanSubmission(context.Context, app.ReportPlanSubmissionQuery) (app.ReportPlanSubmissionSelection, error)
	PromoteReportPlan(context.Context, app.PromoteReportPlanRequest) (app.LedgerEvent, error)
}

type ReportPlanLifecycleBinding struct {
	ToolSessionID, IdempotencyKey string
}

type ReportPlanLifecycleAgentResult struct {
	Text, SessionID string
}

type ReportPlanLifecycleRequest struct {
	MissionID, PendingEventID, ReportMode, AgentExecutor, AgentModel, AgentReasoningEffort, PreviousProviderSessionID string
	Invoke                                                                                                            func(context.Context, ReportPlanLifecycleBinding) (ReportPlanLifecycleAgentResult, error)
	BuildCanonical                                                                                                    func(any, app.ReportPlanSubmissionSelection, ReportPlanLifecycleBinding) (app.AppendEventRequest, error)
}

type ReportPlanLifecycleResult struct {
	Plan       any
	Event      app.LedgerEvent
	Binding    ReportPlanLifecycleBinding
	Submission app.ReportPlanSubmissionSelection
	Agent      ReportPlanLifecycleAgentResult
}

func (runner Runner) RunReportPlanLifecycle(ctx context.Context, req ReportPlanLifecycleRequest) (ReportPlanLifecycleResult, error) {
	service, ok := runner.Service.(ReportPlanLifecycleService)
	if !ok {
		return ReportPlanLifecycleResult{}, fmt.Errorf("%w: durable report plan lifecycle service is required", app.ErrInvalidInput)
	}
	if req.Invoke == nil || req.BuildCanonical == nil {
		return ReportPlanLifecycleResult{}, fmt.Errorf("%w: report plan lifecycle callbacks are required", app.ErrInvalidInput)
	}
	binding := ReportPlanLifecycleBinding{ToolSessionID: runner.id("ses"), IdempotencyKey: runner.id("rpk")}
	agent, err := req.Invoke(ctx, binding)
	if err != nil {
		return ReportPlanLifecycleResult{}, err
	}
	if agent.Text != ReportPlanSubmittedSentinel {
		return ReportPlanLifecycleResult{}, fmt.Errorf("%w: report planning agent did not confirm plan submission", app.ErrInvalidInput)
	}
	selection, err := service.SelectReportPlanSubmission(ctx, app.ReportPlanSubmissionQuery{MissionID: req.MissionID, PendingEventID: req.PendingEventID, ReportMode: req.ReportMode, ToolSessionID: binding.ToolSessionID, PreviousProviderSessionID: strings.TrimSpace(req.PreviousProviderSessionID), AgentExecutor: req.AgentExecutor, AgentModel: req.AgentModel, AgentReasoningEffort: req.AgentReasoningEffort, IdempotencyKey: binding.IdempotencyKey})
	if err != nil {
		return ReportPlanLifecycleResult{}, err
	}
	plan, err := decodeLifecycleReportPlan(req.ReportMode, selection.Plan)
	if err != nil {
		return ReportPlanLifecycleResult{}, err
	}
	canonical, err := req.BuildCanonical(plan, selection, binding)
	if err != nil {
		return ReportPlanLifecycleResult{}, err
	}
	event, err := service.PromoteReportPlan(ctx, app.PromoteReportPlanRequest{MissionID: req.MissionID, PendingEventID: req.PendingEventID, ReportMode: req.ReportMode, ToolSessionID: binding.ToolSessionID, PreviousProviderSessionID: strings.TrimSpace(req.PreviousProviderSessionID), AgentExecutor: req.AgentExecutor, AgentModel: req.AgentModel, AgentReasoningEffort: req.AgentReasoningEffort, IdempotencyKey: binding.IdempotencyKey, ArgumentsHash: selection.ArgumentsHash, PlanHash: selection.PlanHash, SubmissionEventID: selection.EventID, Canonical: canonical})
	if err != nil {
		return ReportPlanLifecycleResult{}, err
	}
	return ReportPlanLifecycleResult{Plan: plan, Event: event, Binding: binding, Submission: selection, Agent: agent}, nil
}

func decodeLifecycleReportPlan(mode string, payload json.RawMessage) (any, error) {
	switch mode {
	case ModePlanned:
		var plan ReportPlan
		if json.Unmarshal(payload, &plan) != nil {
			return nil, fmt.Errorf("%w: invalid submitted planned report plan", app.ErrInvalidInput)
		}
		return NormalizeReportPlan(plan)
	case ModeLongForm:
		var plan SectionalReportPlan
		if json.Unmarshal(payload, &plan) != nil {
			return nil, fmt.Errorf("%w: invalid submitted long-form report plan", app.ErrInvalidInput)
		}
		return NormalizeSectionalReportPlan(plan)
	default:
		return nil, fmt.Errorf("%w: unsupported report mode", app.ErrInvalidInput)
	}
}
