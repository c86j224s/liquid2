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
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// RunLongForm은 section-first long-form 계획을 복구하거나 새 canonical plan event로 승격한다.
func (runner Runner) RunLongForm(ctx context.Context, input LongFormInput) (LongFormOutput, error) {
	if recovered, ok, err := runner.RecoverLongForm(ctx, input); err != nil || ok {
		recovered.StartedAt = time.Now()
		return recovered, err
	}
	startedAt := time.Now()
	artifactID := runner.id("art")
	reportSessionPolicy := firstNonEmpty(input.ReportSessionPolicy, reportexecution.SessionPolicySameSession)
	reportSessionPolicySelection := strings.TrimSpace(input.ReportSessionPolicySelection)
	previousSessionID := ""
	if runner.LatestSessionID != nil {
		previousSessionID = strings.TrimSpace(runner.LatestSessionID(ctx, input.MissionID, input.AgentExecutor))
	}
	reportStartSessionID, sessionChainKind, forkSourceSessionID, err := runner.longFormStartSession(ctx, reportSessionPolicy, previousSessionID, input.SectionFanout)
	if err != nil {
		return LongFormOutput{}, err
	}
	var planResult agentexec.AgentResult
	var returnedPlanSessionID string
	var planDurationMS int64
	userText := "plan sectional long-form markdown report"
	eventText := "섹션별 장문 Markdown 리포트 생성 계획을 만들었습니다."
	if input.SectionFanout {
		userText = "plan section-fanout long-form markdown report"
		eventText = "섹션 병렬 장문 Markdown 리포트 생성 계획을 만들었습니다."
		sessionChainKind = "section_fanout_report"
	}
	lifecycle, err := runner.Lifecycle.RunReportPlanLifecycle(ctx, reporting.ReportPlanLifecycleRequest{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID, ReportMode: reportexecution.ModeLongForm,
		AgentExecutor: input.AgentExecutor, AgentModel: input.AgentModel, AgentReasoningEffort: input.AgentReasoningEffort,
		PreviousProviderSessionID: reportStartSessionID,
		Invoke: func(ctx context.Context, binding reporting.ReportPlanLifecycleBinding) (reporting.ReportPlanLifecycleAgentResult, error) {
			planStarted := time.Now()
			result, runErr := runner.Executor.Run(ctx, agentexec.AgentRequest{
				UserText: userText,
				Prompt:   reportprompt.WithLongFormPlanningDirection(LongFormPrompt(input.Title, input.MissionID, binding.ToolSessionID, input.PendingEventID, binding.IdempotencyKey, input.Rigor, input.GenerationGuidanceProfile), input.DirectionHint),
				Model:    input.AgentModel, ReasoningEffort: input.AgentReasoningEffort, MissionID: input.MissionID,
				ToolSessionID: binding.ToolSessionID, PreviousSessionID: reportStartSessionID,
				AgentExecutor: input.AgentExecutor, MCPMode: input.MCPMode,
				ExtraMCPTools: MCPTools(), ReplaceMCPTools: true,
				ReportPlan: &agentexec.AgentReportPlanContext{
					PendingEventID: input.PendingEventID, ReportMode: reportexecution.ModeLongForm,
					IdempotencyKey: binding.IdempotencyKey, PreviousProviderSessionID: reportStartSessionID,
					AgentModel: input.AgentModel, AgentReasoningEffort: input.AgentReasoningEffort,
					RequireWritingContract: reportprompt.RequireReportWritingContract(input.GenerationGuidanceProfile),
				},
			})
			planDurationMS = time.Since(planStarted).Milliseconds()
			planResult = result
			if runErr != nil {
				return reporting.ReportPlanLifecycleAgentResult{}, longformutil.StageFailure("plan", "", 0, 0,
					reportAgentFailure(runErr, result, "report_plan", planDurationMS, reportStartSessionID))
			}
			returnedPlanSessionID = strings.TrimSpace(result.SessionID)
			validated, validateErr := validateSameSessionResult(result, reportStartSessionID)
			if validateErr != nil {
				return reporting.ReportPlanLifecycleAgentResult{}, longformutil.StageFailure("plan", "", 0, 0,
					reportAgentFailure(validateErr, result, "report_plan", planDurationMS, reportStartSessionID))
			}
			planResult = validated
			return reporting.ReportPlanLifecycleAgentResult{Text: validated.Text, SessionID: validated.SessionID}, nil
		},
		BuildCanonical: func(value any, _ reporting.ReportPlanSubmissionSelection, binding reporting.ReportPlanLifecycleBinding) (ledger.AppendRequest, error) {
			valuePlan, ok := value.(reporting.SectionalReportPlan)
			if !ok {
				return ledger.AppendRequest{}, fmt.Errorf("%w: invalid long-form report plan", producterror.ErrInvalidInput)
			}
			return reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
				MarkdownReportEventBase: reporting.MarkdownReportEventBase{
					EventID: runner.id("evt"), MissionID: input.MissionID, PendingEventID: input.PendingEventID,
					Title: input.Title, AgentExecutor: input.AgentExecutor, AgentModel: input.AgentModel,
					AgentReasoningEffort: input.AgentReasoningEffort, AgentSelectionSource: input.AgentSelectionSource,
					AgentSessionID: planResult.SessionID, PreviousAgentSessionID: reportStartSessionID,
					ReturnedAgentSessionID: returnedPlanSessionID, ToolSessionID: binding.ToolSessionID,
					MCPMode: input.MCPMode, RigorLevel: input.Rigor.Level, RigorLabel: input.Rigor.Label,
					ReportMode: reportexecution.ModeLongForm, ReportModeLabel: reportexecution.ModeLabel(reportexecution.ModeLongForm),
					ReportSessionPolicy: reportSessionPolicy, ReportSessionPolicySelection: reportSessionPolicySelection,
					PostReportHumanize: input.PostReportHumanize, HumanizeEnabled: input.PostReportHumanize != "disabled",
					GenerationGuidanceProfile: input.GenerationGuidanceProfile, GenerationGuidanceSHA256: input.GenerationGuidanceSHA256,
					SessionChainKind: sessionChainKind, PreReportResearchSessionID: previousSessionID,
					ReportPlanSessionID: planResult.SessionID, ForkSourceAgentSessionID: forkSourceSessionID,
					CompositionStrategy: "sectional_preserve_markdown", DurationMS: planDurationMS, Text: eventText,
					AgentUsage: planResult.Usage, AgentUsageSurface: "report_plan",
					AgentUsageDurationMS: planDurationMS, AgentResumed: planResult.Resumed,
					Producer: ledger.Producer{Type: "agent_session", ID: fallbackSessionID(planResult.SessionID, binding.ToolSessionID)},
				},
				ArtifactID: artifactID, Plan: valuePlan, AssemblyStrategy: "c4_normalized_section_headings",
				PartEditEnabled:     LongFormPartEditEnabled(input.GenerationGuidanceProfile),
				PartPlanningEnabled: input.SectionFanout && LongFormPartPlanningEnabled(input.GenerationGuidanceProfile),
				FinalEditPipeline:   LongFormFinalEditPipelineForPlan(input.GenerationGuidanceProfile),
				PlanReviewRequired:  false, PlanReviewState: "auto_accepted",
			}), nil
		},
	})
	if err != nil {
		return LongFormOutput{}, err
	}
	partEditEnabled, partPlanningEnabled, err := longFormActivationFlags(lifecycle.Event)
	if err != nil {
		return LongFormOutput{}, err
	}
	parent := reporting.PartPlanParentState{
		AgentExecutor: input.AgentExecutor, AgentModel: input.AgentModel,
		AgentReasoningEffort: input.AgentReasoningEffort, AgentSelectionSource: input.AgentSelectionSource,
		ReportSessionPolicy: reportSessionPolicy, ReportSessionPolicySelection: reportSessionPolicySelection,
		SessionChainKind: sessionChainKind, ReportPlanSessionID: planResult.SessionID,
		GenerationGuidanceProfile: input.GenerationGuidanceProfile, GenerationGuidanceSHA256: input.GenerationGuidanceSHA256,
	}
	if partPlanningEnabled {
		parent, err = partPlanningParent(lifecycle.Event, input.PendingEventID)
		if err != nil {
			return LongFormOutput{}, err
		}
	}
	valuePlan, ok := lifecycle.Plan.(reporting.SectionalReportPlan)
	if !ok {
		return LongFormOutput{}, fmt.Errorf("%w: invalid long-form report plan", producterror.ErrInvalidInput)
	}
	return LongFormOutput{
		Plan: valuePlan, Event: lifecycle.Event, ArtifactID: artifactID,
		ReportPlanSessionID: parent.ReportPlanSessionID,
		ReportSessionPolicy: parent.ReportSessionPolicy, ReportSessionPolicySelection: parent.ReportSessionPolicySelection,
		AgentExecutor: parent.AgentExecutor, AgentModel: parent.AgentModel,
		AgentReasoningEffort: parent.AgentReasoningEffort, AgentSelectionSource: parent.AgentSelectionSource,
		MCPMode: input.MCPMode, SessionChainKind: parent.SessionChainKind,
		PreReportResearchSessionID: previousSessionID, ForkSourceSessionID: forkSourceSessionID,
		GenerationGuidanceProfile: parent.GenerationGuidanceProfile, GenerationGuidanceSHA256: parent.GenerationGuidanceSHA256,
		PartEditEnabled: partEditEnabled, PartPlanningEnabled: partPlanningEnabled,
		FinalEditPipeline: LongFormFinalEditPipelineForPlan(input.GenerationGuidanceProfile),
		StartedAt:         startedAt,
	}, nil
}

func (runner Runner) longFormStartSession(ctx context.Context, policy string, previousSessionID string, fanout bool) (string, string, string, error) {
	policy = firstNonEmpty(policy, reportexecution.SessionPolicySameSession)
	if policy == reportexecution.SessionPolicyFreshSession {
		if fanout {
			return "", "section_fanout_report", "", nil
		}
		return "", "fresh_session_report", "", nil
	}
	if policy != reportexecution.SessionPolicyIsolatedFork {
		if fanout {
			return previousSessionID, "section_fanout_report", "", nil
		}
		return previousSessionID, "same_session_report", "", nil
	}
	if previousSessionID == "" {
		return "", "", "", fmt.Errorf("%w: isolated report session requires a pre-report research session", producterror.ErrInvalidInput)
	}
	forker, ok := runner.Executor.(agentexec.AgentSessionForker)
	if !ok {
		return "", "", "", reportexecution.ValidateSessionPolicy(policy, reportexecution.ModeLongForm, false, previousSessionID != "", false)
	}
	fork, err := forker.ForkSession(ctx, previousSessionID)
	if err != nil {
		return "", "", "", fmt.Errorf("report session fork failed: %w", err)
	}
	if fanout {
		return fork.SessionID, "section_fanout_report", firstNonEmpty(fork.SourceSessionID, previousSessionID), nil
	}
	return fork.SessionID, "isolated_fork_report", firstNonEmpty(fork.SourceSessionID, previousSessionID), nil
}
