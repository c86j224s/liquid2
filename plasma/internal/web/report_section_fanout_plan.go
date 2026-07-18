package web

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) ensureSectionFanoutPlan(ctx context.Context, req sectionFanoutLongFormRequest, progress sectionalReportProgress, executor AgentExecutor) (sectionFanoutPlanState, error) {
	artifactID := strings.TrimSpace(progress.artifactID)
	if artifactID == "" {
		artifactID = newID("art")
	}
	reportSessionPolicy := firstNonEmpty(req.reportSessionPolicy, reportSessionPolicySameSession)
	reportSessionPolicySelection := strings.TrimSpace(req.reportSessionPolicySelection)
	if progress.hasPlan {
		reportSessionPolicy = firstNonEmpty(progress.reportSessionPolicy, reportSessionPolicy)
		reportSessionPolicySelection = firstNonEmpty(progress.reportSessionPolicySelection, reportSessionPolicySelection)
		return sectionFanoutPlanState{
			artifactID:                   artifactID,
			plan:                         progress.plan,
			planEvent:                    progress.planEvent,
			reportPlanSessionID:          firstNonEmpty(progress.reportPlanSessionID, progress.currentSessionID),
			reportSessionPolicy:          reportSessionPolicy,
			reportSessionPolicySelection: reportSessionPolicySelection,
			sessionChainKind:             firstNonEmpty(progress.sessionChainKind, "section_fanout_report"),
			preReportResearchSessionID:   strings.TrimSpace(progress.preReportResearchSessionID),
			forkSourceSessionID:          strings.TrimSpace(progress.forkSourceSessionID),
		}, nil
	}

	previousSessionID := server.latestAgentSessionID(ctx, req.missionID, req.executorName)
	preReportResearchSessionID := previousSessionID
	reportStartSessionID := previousSessionID
	forkSourceSessionID := ""
	if reportSessionPolicy == reportSessionPolicyIsolatedFork {
		if strings.TrimSpace(previousSessionID) == "" {
			return sectionFanoutPlanState{}, fmt.Errorf("%w: isolated report session requires a pre-report research session", app.ErrInvalidInput)
		}
		forker, ok := executor.(AgentSessionForker)
		if !ok {
			return sectionFanoutPlanState{}, reporting.ValidateSessionPolicy(reportSessionPolicy, reportModeLongForm, false, strings.TrimSpace(previousSessionID) != "", false)
		}
		fork, err := forker.ForkSession(ctx, previousSessionID)
		if err != nil {
			return sectionFanoutPlanState{}, fmt.Errorf("report session fork failed: %w", err)
		}
		reportStartSessionID = fork.SessionID
		forkSourceSessionID = firstNonEmpty(fork.SourceSessionID, previousSessionID)
	}

	var planResult AgentResult
	var returnedPlanSessionID string
	var planDurationMS int64
	lifecycle, err := server.reportRunner().RunReportPlanLifecycle(ctx, reporting.ReportPlanLifecycleRequest{
		MissionID:                 req.missionID,
		PendingEventID:            req.pendingEventID,
		ReportMode:                reportModeLongForm,
		AgentExecutor:             req.executorName,
		AgentModel:                req.agentModel,
		AgentReasoningEffort:      req.agentReasoningEffort,
		PreviousProviderSessionID: reportStartSessionID,
		Invoke: func(ctx context.Context, binding reporting.ReportPlanLifecycleBinding) (reporting.ReportPlanLifecycleAgentResult, error) {
			planStarted := time.Now()
			result, runErr := executor.Run(ctx, AgentRequest{
				UserText:          "plan section-fanout long-form markdown report",
				Prompt:            withReportDirection(agentSectionalReportPlanPrompt(req.title, req.missionID, binding.ToolSessionID, req.pendingEventID, binding.IdempotencyKey, req.rigor, req.generationGuidanceProfile), req.directionHint),
				Model:             req.agentModel,
				ReasoningEffort:   req.agentReasoningEffort,
				MissionID:         req.missionID,
				ToolSessionID:     binding.ToolSessionID,
				PreviousSessionID: reportStartSessionID,
				AgentExecutor:     req.executorName,
				MCPMode:           req.mcpMode,
				ExtraMCPTools:     []string{plasmamcp.ToolReportPlanSubmit},
				ReportPlan: &AgentReportPlanContext{
					PendingEventID:            req.pendingEventID,
					ReportMode:                reportModeLongForm,
					IdempotencyKey:            binding.IdempotencyKey,
					PreviousProviderSessionID: reportStartSessionID,
					AgentModel:                req.agentModel,
					AgentReasoningEffort:      req.agentReasoningEffort,
				},
			})
			planDurationMS = time.Since(planStarted).Milliseconds()
			planResult = result
			if runErr != nil {
				return reporting.ReportPlanLifecycleAgentResult{}, longFormStageFailure("plan", "", 0, 0, reportAgentFailure(runErr, result, "report_plan", planDurationMS, reportStartSessionID))
			}
			returnedPlanSessionID = strings.TrimSpace(result.SessionID)
			validated, validateErr := validatedSameSessionResult(result, reportStartSessionID)
			if validateErr != nil {
				return reporting.ReportPlanLifecycleAgentResult{}, longFormStageFailure("plan", "", 0, 0, reportAgentFailure(validateErr, result, "report_plan", planDurationMS, reportStartSessionID))
			}
			planResult = validated
			return reporting.ReportPlanLifecycleAgentResult{Text: validated.Text, SessionID: validated.SessionID}, nil
		},
		BuildCanonical: func(value any, _ app.ReportPlanSubmissionSelection, binding reporting.ReportPlanLifecycleBinding) (app.AppendEventRequest, error) {
			valuePlan, ok := value.(reporting.SectionalReportPlan)
			if !ok {
				return app.AppendEventRequest{}, fmt.Errorf("%w: invalid long-form report plan", app.ErrInvalidInput)
			}
			return reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
				MarkdownReportEventBase: reporting.MarkdownReportEventBase{
					EventID:                      newID("evt"),
					MissionID:                    req.missionID,
					PendingEventID:               req.pendingEventID,
					Title:                        req.title,
					AgentExecutor:                req.executorName,
					AgentModel:                   req.agentModel,
					AgentReasoningEffort:         req.agentReasoningEffort,
					AgentSelectionSource:         req.agentSelectionSource,
					AgentSessionID:               planResult.SessionID,
					PreviousAgentSessionID:       reportStartSessionID,
					ReturnedAgentSessionID:       returnedPlanSessionID,
					ToolSessionID:                binding.ToolSessionID,
					MCPMode:                      req.mcpMode,
					RigorLevel:                   req.rigor.level,
					RigorLabel:                   req.rigor.label,
					ReportMode:                   reportModeLongForm,
					ReportModeLabel:              reportModeLabel(reportModeLongForm),
					ReportSessionPolicy:          reportSessionPolicy,
					ReportSessionPolicySelection: reportSessionPolicySelection,
					PostReportHumanize:           req.postReportHumanize,
					HumanizeEnabled:              req.postReportHumanize != "disabled",
					GenerationGuidanceProfile:    req.generationGuidanceProfile,
					GenerationGuidanceSHA256:     req.generationGuidanceSHA256,
					SessionChainKind:             "section_fanout_report",
					PreReportResearchSessionID:   preReportResearchSessionID,
					ReportPlanSessionID:          planResult.SessionID,
					ForkSourceAgentSessionID:     forkSourceSessionID,
					CompositionStrategy:          "sectional_preserve_markdown",
					DurationMS:                   planDurationMS,
					Text:                         "섹션 병렬 장문 Markdown 리포트 생성 계획을 만들었습니다.",
					AgentUsage:                   planResult.Usage,
					AgentUsageSurface:            "report_plan",
					AgentUsageDurationMS:         planDurationMS,
					AgentResumed:                 planResult.Resumed,
					Producer:                     app.Producer{Type: "agent_session", ID: fallbackSessionID(planResult.SessionID, binding.ToolSessionID)},
				},
				ArtifactID:         artifactID,
				Plan:               valuePlan,
				AssemblyStrategy:   "c4_normalized_section_headings",
				PlanReviewRequired: false,
				PlanReviewState:    "auto_accepted",
			}), nil
		},
	})
	if err != nil {
		return sectionFanoutPlanState{}, err
	}
	return sectionFanoutPlanState{
		artifactID:                   artifactID,
		plan:                         lifecycle.Plan.(reporting.SectionalReportPlan),
		planEvent:                    lifecycle.Event,
		reportPlanSessionID:          planResult.SessionID,
		reportSessionPolicy:          reportSessionPolicy,
		reportSessionPolicySelection: reportSessionPolicySelection,
		sessionChainKind:             "section_fanout_report",
		preReportResearchSessionID:   preReportResearchSessionID,
		forkSourceSessionID:          forkSourceSessionID,
	}, nil
}
