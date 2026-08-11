package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
)

func (server *Server) createLongFormPrefixWorkflowDraft(ctx context.Context, missionID string, title string, directionHint string, executorName string, agentModel string, agentReasoningEffort string, agentSelectionSource string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, executionStrategy string, executor AgentExecutor) (map[string]any, error) {
	runner := reportworkflow.NewRunner(reportworkflow.RunnerConfig{
		Service:         server.service,
		Lifecycle:       reporting.Runner(server.reportRunner()),
		Executor:        executor,
		NewID:           newID,
		LatestSessionID: server.latestAgentSessionID,
	})
	output, err := runner.RunLongForm(ctx, reportworkflow.DraftInput{
		MissionID: missionID, PendingEventID: pendingEventID,
		Title: title, DirectionHint: directionHint, ExecutionStrategy: executionStrategy,
		AgentExecutor: executorName, AgentModel: agentModel, AgentReasoningEffort: agentReasoningEffort,
		AgentSelectionSource: agentSelectionSource, MCPMode: mcpMode, Rigor: reportWorkflowRigor(rigor),
		ReportMode: reportexecution.ModeLongForm, ReportSessionPolicy: reportSessionPolicy,
		ReportSessionPolicySelection: reportSessionPolicySelection, PostReportHumanize: postReportHumanize,
		GenerationGuidanceProfile: generationGuidanceProfile, GenerationGuidanceSHA256: generationGuidanceSHA256,
	})
	if err != nil {
		return nil, err
	}
	result := map[string]any{"artifact": output.Artifact, "event": output.Event, "markdown": output.Markdown}
	if output.Humanized != nil {
		result["humanized"] = *output.Humanized
	}
	return result, nil
}
