package reportworkflow

import (
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partassembly"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/requirements"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
)

func (runner Runner) prefixOutput(input DraftInput, planOut plan.LongFormOutput, reqOut requirements.Output, sections [][]sectiondraft.Draft, parts []partassembly.PartDraft, sectionArtifactIDs []string, partArtifactIDs []string, sectionWordTotal int, effort string, reportSessionID string) (PrefixOutput, error) {
	outSections := make([][]PrefixSection, len(sections))
	for partIndex, partSections := range sections {
		outSections[partIndex] = make([]PrefixSection, len(partSections))
		for sectionIndex, draft := range partSections {
			outSections[partIndex][sectionIndex] = PrefixSection{
				Title: draft.Title, Markdown: draft.Markdown,
				ArtifactID: draft.ArtifactID, WordCount: draft.WordCount,
			}
		}
	}
	outParts := make([]PrefixPart, len(parts))
	for index, part := range parts {
		outParts[index] = PrefixPart{Title: part.Title, Markdown: part.Markdown, ArtifactID: part.ArtifactID, WordCount: part.WordCount}
	}
	tail, err := finalTailForPipeline(planOut.FinalEditPipeline)
	if err != nil {
		return PrefixOutput{}, err
	}
	return PrefixOutput{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID,
		Title: input.Title, DirectionHint: input.DirectionHint,
		ExecutionStrategy: input.ExecutionStrategy,
		AgentExecutor:     planOut.AgentExecutor, AgentModel: planOut.AgentModel,
		AgentReasoningEffort: effort, AgentSelectionSource: planOut.AgentSelectionSource,
		MCPMode: planOut.MCPMode, Rigor: input.Rigor,
		ReportSessionPolicy:          planOut.ReportSessionPolicy,
		ReportSessionPolicySelection: planOut.ReportSessionPolicySelection,
		PostReportHumanize:           input.PostReportHumanize,
		GenerationGuidanceProfile:    planOut.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     planOut.GenerationGuidanceSHA256,
		ArtifactID:                   planOut.ArtifactID, PlanEvent: planOut.Event, Plan: planOut.Plan,
		RequirementMap: reqOut.RequirementMap, RequirementMapEvent: reqOut.Event,
		Parts: outParts, Sections: outSections, PartArtifactIDs: partArtifactIDs,
		SectionArtifactIDs: sectionArtifactIDs, SectionWordTotal: sectionWordTotal,
		SessionChainKind:           planOut.SessionChainKind,
		PreReportResearchSessionID: planOut.PreReportResearchSessionID,
		ReportPlanSessionID:        planOut.ReportPlanSessionID,
		ForkSourceAgentSessionID:   planOut.ForkSourceSessionID,
		ReportSessionID:            reportSessionID,
		PartEditEnabled:            planOut.PartEditEnabled, PartPlanningEnabled: planOut.PartPlanningEnabled,
		FinalEditPipeline: planOut.FinalEditPipeline, FinalTail: tail, StartedAt: planOut.StartedAt,
	}, nil
}
