package reportworkflow

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/legacyfinalize"
)

// runLegacyFinalTail은 legacy long-form finalizer만를 실행한다.
func (runner Runner) runLegacyFinalTail(ctx context.Context, prefix PrefixOutput) (DraftOutput, error) {
	finalUserText := "finalize sectional long-form markdown report"
	if prefix.ExecutionStrategy == "section_fanout" {
		finalUserText = "finalize section-fanout long-form markdown report"
	}
	finalSessionID := prefix.ReportSessionID
	finalForkSourceID := prefix.ForkSourceAgentSessionID
	if prefix.ExecutionStrategy == "section_fanout" {
		forker, ok := runner.executor.(agentexec.AgentSessionForker)
		if !ok {
			return DraftOutput{}, finaledit.StageFailure("final", prefix.PlanEvent.EventID, reportexecution.ValidateSessionPolicy(reportexecution.SessionPolicyIsolatedFork, reportexecution.ModeLongForm, false, true, false))
		}
		var err error
		finalSessionID, finalForkSourceID, err = legacyfinalize.ForkSession(ctx, forker, prefix.ReportPlanSessionID)
		if err != nil {
			return DraftOutput{}, finaledit.StageFailure("final", prefix.PlanEvent.EventID, err)
		}
		if finalForkSourceID == "" {
			finalForkSourceID = prefix.ReportPlanSessionID
		}
	} else if prefix.PartEditEnabled {
		forker, ok := runner.executor.(agentexec.AgentSessionForker)
		if !ok {
			return DraftOutput{}, finaledit.StageFailure("final", prefix.PlanEvent.EventID, fmt.Errorf("%w: final editor requires an agent session forker", producterror.ErrInvalidInput))
		}
		sourceSessionID := finaledit.FirstNonEmpty(prefix.ReportPlanSessionID, prefix.ReportSessionID)
		var err error
		finalSessionID, finalForkSourceID, err = legacyfinalize.ForkSession(ctx, forker, sourceSessionID)
		if err != nil {
			return DraftOutput{}, finaledit.StageFailure("final", prefix.PlanEvent.EventID, err)
		}
		if finalForkSourceID == "" {
			finalForkSourceID = sourceSessionID
		}
	}
	done := runner.observeStart(NodeLegacyFinal)
	finalized, err := runner.legacyFinalizeRunner.Run(ctx, legacyInput(prefix, finalUserText, finalSessionID, finalForkSourceID), runner.executor)
	done(err, false)
	if err != nil {
		return DraftOutput{}, err
	}
	out := DraftOutput{Artifact: finalized.Artifact, Event: finalized.Event, Markdown: finalized.Markdown, ReportSessionID: finalSessionID}

	if err := runner.complete(context.WithoutCancel(ctx), out, nil); err != nil {
		return DraftOutput{}, err
	}
	return out, nil
}

func legacyInput(prefix PrefixOutput, finalUserText string, finalSessionID string, finalForkSourceID string) legacyfinalize.Input {
	parts := make([]legacyfinalize.Part, len(prefix.Parts))
	for index, part := range prefix.Parts {
		parts[index] = legacyfinalize.Part{Title: part.Title, Markdown: part.Markdown, ArtifactID: part.ArtifactID, WordCount: part.WordCount}
	}
	return legacyfinalize.Input{
		MissionID: prefix.MissionID, Title: prefix.Title, DirectionHint: prefix.DirectionHint,
		FinalUserText: finalUserText, ExecutorName: prefix.AgentExecutor,
		AgentModel: prefix.AgentModel, AgentReasoningEffort: prefix.AgentReasoningEffort,
		AgentSelectionSource: prefix.AgentSelectionSource, MCPMode: prefix.MCPMode, Rigor: prefix.Rigor,
		ReportSessionPolicy: prefix.ReportSessionPolicy, ReportSessionPolicySelection: prefix.ReportSessionPolicySelection,
		PostReportHumanize: prefix.PostReportHumanize, GenerationGuidanceProfile: prefix.GenerationGuidanceProfile,
		GenerationGuidanceSHA256: prefix.GenerationGuidanceSHA256, PendingEventID: prefix.PendingEventID,
		ArtifactID: prefix.ArtifactID, PlanEvent: prefix.PlanEvent, Plan: prefix.Plan, RequirementMap: prefix.RequirementMap,
		Parts: parts, PartArtifactIDs: prefix.PartArtifactIDs, SectionArtifactIDs: prefix.SectionArtifactIDs,
		SectionWordTotal: prefix.SectionWordTotal, SessionChainKind: prefix.SessionChainKind,
		PreReportResearchSessionID: prefix.PreReportResearchSessionID, ReportPlanSessionID: prefix.ReportPlanSessionID,
		FinalSessionID: finalSessionID, FinalForkSourceID: finalForkSourceID, StartedAt: prefix.StartedAt,
	}
}
