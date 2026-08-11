package reportworkflow

import (
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partassembly"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partedit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partplan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/requirements"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
)

func fillPlanRuntime(input DraftInput, out plan.LongFormOutput) plan.LongFormOutput {
	out.AgentExecutor = firstNonEmpty(out.AgentExecutor, input.AgentExecutor)
	out.AgentModel = firstNonEmpty(out.AgentModel, input.AgentModel)
	out.AgentReasoningEffort = firstNonEmpty(out.AgentReasoningEffort, input.AgentReasoningEffort)
	out.AgentSelectionSource = firstNonEmpty(out.AgentSelectionSource, input.AgentSelectionSource)
	out.MCPMode = firstNonEmpty(out.MCPMode, input.MCPMode)
	out.ReportSessionPolicy = firstNonEmpty(out.ReportSessionPolicy, input.ReportSessionPolicy)
	out.ReportSessionPolicySelection = firstNonEmpty(out.ReportSessionPolicySelection, input.ReportSessionPolicySelection)
	out.GenerationGuidanceProfile = firstNonEmpty(out.GenerationGuidanceProfile, input.GenerationGuidanceProfile)
	out.GenerationGuidanceSHA256 = firstNonEmpty(out.GenerationGuidanceSHA256, input.GenerationGuidanceSHA256)
	return out
}

func requirementsInput(input DraftInput, planOut plan.LongFormOutput, progress longFormProgress) requirements.Input {
	return requirements.Input{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID,
		PlanEventID: planOut.Event.EventID, PlanSessionID: planOut.ReportPlanSessionID,
		ValidatedDownstream: progress.hasValidatedDownstream(),
		Title:               input.Title, DirectionHint: input.DirectionHint,
		AgentExecutor:   firstNonEmpty(planOut.AgentExecutor, input.AgentExecutor),
		AgentModel:      firstNonEmpty(planOut.AgentModel, input.AgentModel),
		ReasoningEffort: firstNonEmpty(planOut.AgentReasoningEffort, input.AgentReasoningEffort),
		MCPMode:         firstNonEmpty(planOut.MCPMode, input.MCPMode),
		Plan:            planOut.Plan,
	}
}

func sectionBase(input DraftInput, planOut plan.LongFormOutput, reqOut requirements.Output) sectiondraft.BaseInput {
	return sectiondraft.BaseInput{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID,
		Title: input.Title, DirectionHint: input.DirectionHint,
		AgentExecutor: planOut.AgentExecutor, AgentModel: planOut.AgentModel,
		AgentReasoningEffort: planOut.AgentReasoningEffort, AgentSelectionSource: planOut.AgentSelectionSource,
		MCPMode: planOut.MCPMode, Rigor: input.Rigor,
		ReportSessionPolicy: planOut.ReportSessionPolicy, ReportSessionPolicySelection: planOut.ReportSessionPolicySelection,
		PostReportHumanize: input.PostReportHumanize, GenerationGuidanceProfile: planOut.GenerationGuidanceProfile,
		GenerationGuidanceSHA256: planOut.GenerationGuidanceSHA256,
		Plan:                     planOut.Plan, PlanEvent: planOut.Event, ReportPlanSessionID: planOut.ReportPlanSessionID,
		SessionChainKind: planOut.SessionChainKind, PreReportResearchSessionID: planOut.PreReportResearchSessionID,
		ForkSourceSessionID: planOut.ForkSourceSessionID, RequirementMap: reqOut.RequirementMap,
	}
}

func partPlanBase(input DraftInput, planOut plan.LongFormOutput, reqOut requirements.Output) partplan.BaseInput {
	return partplan.BaseInput{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID,
		Title: input.Title, DirectionHint: input.DirectionHint, AgentExecutor: planOut.AgentExecutor,
		AgentModel: planOut.AgentModel, AgentReasoningEffort: planOut.AgentReasoningEffort,
		AgentSelectionSource: planOut.AgentSelectionSource, MCPMode: planOut.MCPMode,
		Rigor: input.Rigor, PostReportHumanize: input.PostReportHumanize,
		ReportSessionPolicy: planOut.ReportSessionPolicy, ReportSessionPolicySelection: planOut.ReportSessionPolicySelection,
		GenerationGuidanceProfile: planOut.GenerationGuidanceProfile, GenerationGuidanceSHA256: planOut.GenerationGuidanceSHA256,
		Plan: planOut.Plan, PlanEvent: planOut.Event, ReportPlanSessionID: planOut.ReportPlanSessionID,
		SessionChainKind: planOut.SessionChainKind, PreReportResearchSessionID: planOut.PreReportResearchSessionID,
		RequirementMap: reqOut.RequirementMap,
	}
}

func partAssemblyBase(input DraftInput, planOut plan.LongFormOutput) partassembly.BaseInput {
	return partassembly.BaseInput{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID,
		Title: input.Title, DirectionHint: input.DirectionHint,
		AgentExecutor: planOut.AgentExecutor, AgentModel: planOut.AgentModel,
		AgentReasoningEffort: planOut.AgentReasoningEffort, AgentSelectionSource: planOut.AgentSelectionSource,
		MCPMode: planOut.MCPMode, Rigor: input.Rigor,
		ReportSessionPolicy: planOut.ReportSessionPolicy, ReportSessionPolicySelection: planOut.ReportSessionPolicySelection,
		PostReportHumanize: input.PostReportHumanize, GenerationGuidanceProfile: planOut.GenerationGuidanceProfile,
		GenerationGuidanceSHA256: planOut.GenerationGuidanceSHA256,
		Plan:                     planOut.Plan, PlanEvent: planOut.Event, ReportPlanSessionID: planOut.ReportPlanSessionID,
		SessionChainKind: planOut.SessionChainKind, PreReportResearchSessionID: planOut.PreReportResearchSessionID,
	}
}

func partEditBase(input DraftInput, planOut plan.LongFormOutput, reqOut requirements.Output, effort string) partedit.BaseInput {
	return partedit.BaseInput{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID,
		Title: input.Title, DirectionHint: input.DirectionHint,
		AgentExecutor: planOut.AgentExecutor, AgentModel: planOut.AgentModel,
		AgentReasoningEffort: effort, AgentSelectionSource: planOut.AgentSelectionSource,
		MCPMode: planOut.MCPMode, Rigor: input.Rigor,
		ReportSessionPolicy: planOut.ReportSessionPolicy, ReportSessionPolicySelection: planOut.ReportSessionPolicySelection,
		GenerationGuidanceProfile: planOut.GenerationGuidanceProfile, GenerationGuidanceSHA256: planOut.GenerationGuidanceSHA256,
		Plan: planOut.Plan, PlanEvent: planOut.Event, RequirementMap: reqOut.RequirementMap,
		RequirementMapEvent: reqOut.Event, ReportPlanSessionID: planOut.ReportPlanSessionID,
		SessionChainKind: planOut.SessionChainKind,
	}
}
