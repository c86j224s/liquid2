package web

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type reportRequirementStageRequest struct {
	missionID       string
	title           string
	directionHint   string
	executorName    string
	agentModel      string
	reasoningEffort string
	mcpMode         string
	pendingEventID  string
	planEventID     string
	planSessionID   string
	plan            agentSectionalReportPlan
}

func (server *Server) ensureReportRequirementMap(ctx context.Context, req reportRequirementStageRequest, progress sectionalReportProgress, executor AgentExecutor) (reporting.ReportRequirementMap, app.LedgerEvent, error) {
	if progress.hasRequirementMap {
		return progress.requirementMap, progress.requirementMapEvent, nil
	}
	if !progress.hasRequirementStage && (progress.hasPostPlanSectionStarted || len(progress.sections) > 0 || len(progress.parts) > 0) {
		// Reports interrupted before requirement mapping existed keep their
		// already-durable stage provenance and resume without retroactive policy.
		return reporting.ReportRequirementMap{}, app.LedgerEvent{}, nil
	}
	events, err := server.service.ListEvents(ctx, req.missionID)
	if err != nil {
		return reporting.ReportRequirementMap{}, app.LedgerEvent{}, longFormStageFailure("requirements", req.planEventID, 0, 0, err)
	}
	reviewEventIDs, err := app.ReportRequirementReviewEventIDs(events, req.pendingEventID)
	if err != nil {
		return reporting.ReportRequirementMap{}, app.LedgerEvent{}, longFormStageFailure("requirements", req.planEventID, 0, 0, err)
	}
	planSessionID := strings.TrimSpace(req.planSessionID)
	if planSessionID == "" {
		return reporting.ReportRequirementMap{}, app.LedgerEvent{}, longFormStageFailure("requirements", req.planEventID, 0, 0, fmt.Errorf("report requirement mapping requires a plan session"))
	}
	var durationMS int64
	lifecycle, err := server.reportRunner().RunReportRequirementMapLifecycle(ctx, reporting.ReportRequirementMapLifecycleRequest{
		MissionID: req.missionID, PendingEventID: req.pendingEventID, PlanEventID: req.planEventID,
		AgentExecutor: req.executorName, AgentModel: req.agentModel, AgentReasoningEffort: req.reasoningEffort,
		PreviousProviderSessionID: planSessionID, Plan: req.plan,
		Invoke: func(ctx context.Context, binding reporting.ReportRequirementMapBinding) (reporting.ReportRequirementMapAgentResult, error) {
			started := time.Now()
			result, runErr := executor.Run(ctx, AgentRequest{
				UserText:           "map explicit report requirements to the fixed long-form outline",
				Prompt:             agentReportRequirementMapPrompt(req.title, req.directionHint, req.plan, reviewEventIDs, binding),
				Model:              req.agentModel,
				ReasoningEffort:    req.reasoningEffort,
				MissionID:          req.missionID,
				ToolSessionID:      binding.ToolSessionID,
				PreviousSessionID:  planSessionID,
				AgentExecutor:      req.executorName,
				MCPMode:            req.mcpMode,
				ExtraMCPTools:      reportRequirementMCPTools(),
				ReplaceMCPTools:    true,
				ReportRequirements: &binding,
			})
			durationMS = time.Since(started).Milliseconds()
			if runErr != nil {
				return reporting.ReportRequirementMapAgentResult{}, reportAgentFailure(runErr, result, "report_requirements", durationMS, planSessionID)
			}
			validated, validateErr := validatedSameSessionResult(result, planSessionID)
			if validateErr != nil {
				return reporting.ReportRequirementMapAgentResult{}, reportAgentFailure(validateErr, result, "report_requirements", durationMS, planSessionID)
			}
			return reporting.ReportRequirementMapAgentResult{Text: validated.Text, SessionID: validated.SessionID}, nil
		},
	})
	if err != nil {
		return reporting.ReportRequirementMap{}, app.LedgerEvent{}, longFormStageFailure("requirements", req.planEventID, 0, 0, err)
	}
	_ = durationMS
	return lifecycle.RequirementMap, lifecycle.Event, nil
}
