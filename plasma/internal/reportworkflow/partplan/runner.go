package partplan

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
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// Run은 단일 Part planning agent를 실행하고 canonical part_plan event를 채택한다.
func (runner Runner) Run(ctx context.Context, input Input) (Output, error) {
	toolSessionID := runner.id("ses")
	started := time.Now()
	result, err := runner.Executor.Run(ctx, agentexec.AgentRequest{
		UserText: fmt.Sprintf("plan the reading flow for Part %d of the long-form report", input.PartIndex+1),
		Prompt:   Prompt(input), Model: input.Base.AgentModel, ReasoningEffort: input.Base.AgentReasoningEffort,
		MissionID: input.Base.MissionID, ToolSessionID: toolSessionID, PreviousSessionID: input.ProviderSessionID,
		AgentExecutor: input.Base.AgentExecutor, MCPMode: input.Base.MCPMode, ReplaceMCPTools: true,
	})
	durationMS := time.Since(started).Milliseconds()
	if err == nil {
		result, err = longformutil.ValidateSameSessionResult(result, input.ProviderSessionID)
	}
	if err != nil {
		return Output{}, longformutil.StageFailure("part_plan", input.Base.PlanEvent.EventID, input.PartIndex+1, 0,
			longformutil.AgentFailure(err, result, "report_part_plan", durationMS, input.ProviderSessionID))
	}
	brief := strings.TrimSpace(result.Text)
	if brief == "" {
		return Output{}, longformutil.StageFailure("part_plan", input.Base.PlanEvent.EventID, input.PartIndex+1, 0,
			fmt.Errorf("%w: Part planner returned an empty brief", producterror.ErrInvalidInput))
	}
	finalized, err := reporting.FinalizePartPlan(context.WithoutCancel(ctx), runner.Service, reporting.PartPlanCreatedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: runner.id("evt"), MissionID: input.Base.MissionID, PendingEventID: input.Base.PendingEventID,
			PlanEventID: input.Base.PlanEvent.EventID, Title: input.Part.Title,
			AgentExecutor: input.Base.AgentExecutor, AgentModel: input.Base.AgentModel,
			AgentReasoningEffort: input.Base.AgentReasoningEffort, AgentSelectionSource: input.Base.AgentSelectionSource,
			AgentSessionID: result.SessionID, PreviousAgentSessionID: input.ProviderSessionID,
			ReturnedAgentSessionID: result.SessionID, ToolSessionID: toolSessionID,
			ReportMode: reportexecution.ModeLongForm, ReportModeLabel: reportexecution.ModeLabel(reportexecution.ModeLongForm),
			ReportSessionPolicy: input.Base.ReportSessionPolicy, ReportSessionPolicySelection: input.Base.ReportSessionPolicySelection,
			PostReportHumanize: input.Base.PostReportHumanize, HumanizeEnabled: input.Base.PostReportHumanize != "disabled",
			GenerationGuidanceProfile: input.Base.GenerationGuidanceProfile,
			GenerationGuidanceSHA256:  input.Base.GenerationGuidanceSHA256,
			SessionChainKind:          input.Base.SessionChainKind, PreReportResearchSessionID: input.Base.PreReportResearchSessionID,
			ReportPlanSessionID: input.Base.ReportPlanSessionID, ReportSessionID: result.SessionID,
			ForkSourceAgentSessionID: input.ForkSourceSession, CompositionStrategy: "sectional_preserve_markdown",
			AssemblyStrategy: "c4_normalized_section_headings", DurationMS: durationMS,
			AgentUsage: result.Usage, AgentUsageSurface: "report_part_plan", AgentUsageDurationMS: durationMS,
			AgentResumed: result.Resumed, Producer: ledger.Producer{Type: "agent_session", ID: result.SessionID},
		},
		PartIndex: input.PartIndex + 1,
		Brief:     brief,
	})
	if err != nil {
		return Output{}, longformutil.StageFailure("part_plan", input.Base.PlanEvent.EventID, input.PartIndex+1, 0, err)
	}
	return Output{
		PartIndex: input.PartIndex, Brief: finalized.Brief,
		ProviderSessionID: finalized.ProviderSessionID, Event: finalized.Event,
	}, nil
}

func (runner Runner) id(prefix string) string {
	if runner.NewID != nil {
		return runner.NewID(prefix)
	}
	return prefix + "_missing"
}
