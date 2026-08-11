package plan

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// RunMarkdown는 planned 보고서의 canonical plan을 복구하거나 새로 생성한다.
func (runner Runner) RunMarkdown(ctx context.Context, input Input) (Output, error) {
	if recovered, ok, err := runner.RecoverMarkdown(ctx, input.MissionID, input.PendingEventID); err != nil || ok {
		recovered.StartedAt = time.Now()
		return recovered, err
	}
	startedAt := time.Now()
	artifactID := runner.id("art")
	previousSessionID := ""
	if runner.LatestSessionID != nil {
		previousSessionID = strings.TrimSpace(runner.LatestSessionID(ctx, input.MissionID, input.AgentExecutor))
	}
	reportStartSessionID, sessionChainKind, forkSourceSessionID, err := runner.startSession(ctx, input.ReportSessionPolicy, previousSessionID)
	if err != nil {
		return Output{}, err
	}
	var planResult agentexec.AgentResult
	var returnedPlanSessionID string
	var planDurationMS int64
	lifecycle, err := runner.Lifecycle.RunReportPlanLifecycle(ctx, reporting.ReportPlanLifecycleRequest{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID, ReportMode: reportexecution.ModePlanned, AgentExecutor: input.AgentExecutor, AgentModel: input.AgentModel, AgentReasoningEffort: input.AgentReasoningEffort, PreviousProviderSessionID: reportStartSessionID,
		Invoke: func(ctx context.Context, binding reporting.ReportPlanLifecycleBinding) (reporting.ReportPlanLifecycleAgentResult, error) {
			planStarted := time.Now()
			result, runErr := runner.Executor.Run(ctx, agentexec.AgentRequest{
				UserText: "plan markdown report artifact", Prompt: reportprompt.WithReportDirection(reportprompt.MarkdownReportPlanPrompt(input.Title, input.MissionID, binding.ToolSessionID, input.PendingEventID, binding.IdempotencyKey, input.Rigor, input.GenerationGuidanceProfile), input.DirectionHint),
				Model: input.AgentModel, ReasoningEffort: input.AgentReasoningEffort, MissionID: input.MissionID, ToolSessionID: binding.ToolSessionID, PreviousSessionID: reportStartSessionID, AgentExecutor: input.AgentExecutor, MCPMode: input.MCPMode,
				ExtraMCPTools: MCPTools(), ReplaceMCPTools: true, ReportPlan: &agentexec.AgentReportPlanContext{PendingEventID: input.PendingEventID, ReportMode: reportexecution.ModePlanned, IdempotencyKey: binding.IdempotencyKey, PreviousProviderSessionID: reportStartSessionID, AgentModel: input.AgentModel, AgentReasoningEffort: input.AgentReasoningEffort, RequireWritingContract: reportprompt.RequireReportWritingContract(input.GenerationGuidanceProfile)},
			})
			planDurationMS = time.Since(planStarted).Milliseconds()
			planResult = result
			if runErr != nil {
				return reporting.ReportPlanLifecycleAgentResult{}, fmt.Errorf("report planning agent failed: %w", reportAgentFailure(runErr, result, "report_plan", planDurationMS, reportStartSessionID))
			}
			returnedPlanSessionID = strings.TrimSpace(result.SessionID)
			validated, validateErr := validateSameSessionResult(result, reportStartSessionID)
			if validateErr != nil {
				return reporting.ReportPlanLifecycleAgentResult{}, reportAgentFailure(validateErr, result, "report_plan", planDurationMS, reportStartSessionID)
			}
			planResult = validated
			return reporting.ReportPlanLifecycleAgentResult{Text: validated.Text, SessionID: validated.SessionID}, nil
		},
		BuildCanonical: func(value any, _ reporting.ReportPlanSubmissionSelection, binding reporting.ReportPlanLifecycleBinding) (ledger.AppendRequest, error) {
			valuePlan, ok := value.(reporting.ReportPlan)
			if !ok {
				return ledger.AppendRequest{}, fmt.Errorf("%w: invalid planned report plan", producterror.ErrInvalidInput)
			}
			return reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
				MarkdownReportEventBase: reporting.MarkdownReportEventBase{
					EventID:                      runner.id("evt"),
					MissionID:                    input.MissionID,
					PendingEventID:               input.PendingEventID,
					Title:                        input.Title,
					AgentExecutor:                input.AgentExecutor,
					AgentModel:                   input.AgentModel,
					AgentReasoningEffort:         input.AgentReasoningEffort,
					AgentSelectionSource:         input.AgentSelectionSource,
					AgentSessionID:               planResult.SessionID,
					PreviousAgentSessionID:       reportStartSessionID,
					ReturnedAgentSessionID:       returnedPlanSessionID,
					ToolSessionID:                binding.ToolSessionID,
					MCPMode:                      input.MCPMode,
					RigorLevel:                   input.Rigor.Level,
					RigorLabel:                   input.Rigor.Label,
					ReportMode:                   reportexecution.ModePlanned,
					ReportModeLabel:              reportexecution.ModeLabel(reportexecution.ModePlanned),
					ReportSessionPolicy:          input.ReportSessionPolicy,
					ReportSessionPolicySelection: input.ReportSessionPolicySelection,
					PostReportHumanize:           input.PostReportHumanize,
					HumanizeEnabled:              input.PostReportHumanize != "disabled",
					GenerationGuidanceProfile:    input.GenerationGuidanceProfile,
					GenerationGuidanceSHA256:     input.GenerationGuidanceSHA256,
					SessionChainKind:             sessionChainKind,
					PreReportResearchSessionID:   previousSessionID,
					ReportPlanSessionID:          planResult.SessionID,
					ForkSourceAgentSessionID:     forkSourceSessionID,
					CompositionStrategy:          "planned_markdown",
					DurationMS:                   planDurationMS,
					Text:                         "Markdown 리포트 생성 계획을 만들었습니다.",
					AgentUsage:                   planResult.Usage,
					AgentUsageSurface:            "report_plan",
					AgentUsageDurationMS:         planDurationMS,
					AgentResumed:                 planResult.Resumed,
					Producer:                     ledger.Producer{Type: "agent_session", ID: fallbackSessionID(planResult.SessionID, binding.ToolSessionID)},
				},
				ArtifactID:         artifactID,
				Plan:               valuePlan,
				PlanReviewRequired: false,
				PlanReviewState:    "auto_accepted",
			}), nil
		},
	})
	if err != nil {
		return Output{}, err
	}
	valuePlan, ok := lifecycle.Plan.(reporting.ReportPlan)
	if !ok {
		return Output{}, fmt.Errorf("%w: invalid planned report plan", producterror.ErrInvalidInput)
	}
	return Output{
		Plan:                         valuePlan,
		Event:                        lifecycle.Event,
		ArtifactID:                   artifactID,
		PlanToolSessionID:            lifecycle.Binding.ToolSessionID,
		ReportPlanSessionID:          planResult.SessionID,
		ReportSessionPolicy:          input.ReportSessionPolicy,
		ReportSessionPolicySelection: input.ReportSessionPolicySelection,
		SessionChainKind:             sessionChainKind,
		PreReportResearchSessionID:   previousSessionID,
		ForkSourceSessionID:          forkSourceSessionID,
		StartedAt:                    startedAt,
	}, nil
}

func (runner Runner) startSession(ctx context.Context, policy string, previousSessionID string) (string, string, string, error) {
	policy = firstNonEmpty(policy, reportexecution.SessionPolicySameSession)
	if policy == reportexecution.SessionPolicyFreshSession {
		return "", "fresh_session_report", "", nil
	}
	if policy != reportexecution.SessionPolicyIsolatedFork {
		return previousSessionID, "same_session_report", "", nil
	}
	if previousSessionID == "" {
		return "", "", "", fmt.Errorf("%w: isolated report session requires a pre-report research session", producterror.ErrInvalidInput)
	}
	forker, ok := runner.Executor.(agentexec.AgentSessionForker)
	if !ok {
		return "", "", "", reportexecution.ValidateSessionPolicy(policy, reportexecution.ModePlanned, false, previousSessionID != "", false)
	}
	fork, err := forker.ForkSession(ctx, previousSessionID)
	if err != nil {
		return "", "", "", fmt.Errorf("report session fork failed: %w", err)
	}
	forkSourceSessionID := firstNonEmpty(fork.SourceSessionID, previousSessionID)
	return fork.SessionID, "isolated_fork_report", forkSourceSessionID, nil
}
