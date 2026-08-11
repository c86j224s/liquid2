package requirements

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// Run은 요구사항 map을 복구하거나 provider에게 MCP 제출을 요청해 canonical map을 채택한다.
func (runner Runner) Run(ctx context.Context, input Input) (Output, error) {
	events, err := runner.Service.ListEvents(ctx, input.MissionID)
	if err != nil {
		return Output{}, longformutil.StageFailure("requirements", input.PlanEventID, 0, 0, err)
	}
	recovered, err := recoverState(events, input.PendingEventID, input.PlanEventID, input.Plan)
	if err != nil {
		return Output{}, longformutil.StageFailure("requirements", input.PlanEventID, 0, 0, err)
	}
	if recovered.hasMap {
		return Output{RequirementMap: recovered.requirementMap, Event: recovered.event, Recovered: true}, nil
	}
	if !recovered.hasStage && (recovered.hasValidatedWorkStart || input.ValidatedDownstream) {
		return Output{RequirementMap: reporting.ReportRequirementMap{}, Recovered: true}, nil
	}
	reviewEventIDs, err := reviewEventIDs(events, input.PendingEventID)
	if err != nil {
		return Output{}, longformutil.StageFailure("requirements", input.PlanEventID, 0, 0, err)
	}
	if strings.TrimSpace(input.PlanSessionID) == "" {
		return Output{}, longformutil.StageFailure("requirements", input.PlanEventID, 0, 0,
			fmt.Errorf("report requirement mapping requires a plan session"))
	}
	lifecycle, err := runner.Lifecycle.RunReportRequirementMapLifecycle(ctx, reporting.ReportRequirementMapLifecycleRequest{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID, PlanEventID: input.PlanEventID,
		AgentExecutor: input.AgentExecutor, AgentModel: input.AgentModel, AgentReasoningEffort: input.ReasoningEffort,
		PreviousProviderSessionID: input.PlanSessionID, Plan: input.Plan,
		Invoke: func(ctx context.Context, binding reporting.ReportRequirementMapBinding) (reporting.ReportRequirementMapAgentResult, error) {
			started := time.Now()
			result, runErr := runner.Executor.Run(ctx, agentexec.AgentRequest{
				UserText: "map explicit report requirements to the fixed long-form outline",
				Prompt:   Prompt(input.Title, input.DirectionHint, input.Plan, reviewEventIDs, binding),
				Model:    input.AgentModel, ReasoningEffort: input.ReasoningEffort, MissionID: input.MissionID,
				ToolSessionID: binding.ToolSessionID, PreviousSessionID: input.PlanSessionID,
				AgentExecutor: input.AgentExecutor, MCPMode: input.MCPMode,
				ExtraMCPTools: MCPTools(), ReplaceMCPTools: true, ReportRequirements: &binding,
			})
			durationMS := time.Since(started).Milliseconds()
			if runErr != nil {
				return reporting.ReportRequirementMapAgentResult{}, longformutil.AgentFailure(runErr, result, "report_requirements", durationMS, input.PlanSessionID)
			}
			validated, validateErr := longformutil.ValidateSameSessionResult(result, input.PlanSessionID)
			if validateErr != nil {
				return reporting.ReportRequirementMapAgentResult{}, longformutil.AgentFailure(validateErr, result, "report_requirements", durationMS, input.PlanSessionID)
			}
			return reporting.ReportRequirementMapAgentResult{Text: strings.TrimSpace(validated.Text), SessionID: validated.SessionID}, nil
		},
	})
	if err != nil {
		return Output{}, longformutil.StageFailure("requirements", input.PlanEventID, 0, 0, err)
	}
	return Output{RequirementMap: lifecycle.RequirementMap, Event: lifecycle.Event}, nil
}
