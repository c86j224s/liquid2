package web

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
	"log"
	"strings"
)

func (server *Server) createReportWorkflowDraft(ctx context.Context, missionID string, title string, directionHint string, executorName string, agentModel string, agentReasoningEffort string, agentSelectionSource string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, reportMode string, executionStrategy string, executor AgentExecutor, outputKind string, articleIntent reportexecution.ArticleIntent) (map[string]any, error) {
	runner := reportworkflow.NewRunner(reportworkflow.RunnerConfig{
		Service:         server.service,
		Lifecycle:       reporting.Runner(server.reportRunner()),
		Executor:        executor,
		NewID:           newID,
		LatestSessionID: server.latestAgentSessionID,
	})
	output, err := runner.RunDraft(ctx, reportworkflow.DraftInput{
		MissionID:                    missionID,
		PendingEventID:               pendingEventID,
		Title:                        title,
		DirectionHint:                directionHint,
		ExecutionStrategy:            executionStrategy,
		AgentExecutor:                executorName,
		AgentModel:                   agentModel,
		AgentReasoningEffort:         agentReasoningEffort,
		AgentSelectionSource:         agentSelectionSource,
		MCPMode:                      mcpMode,
		Rigor:                        reportWorkflowRigor(rigor),
		ReportMode:                   reportMode,
		ReportSessionPolicy:          reportSessionPolicy,
		ReportSessionPolicySelection: strings.TrimSpace(reportSessionPolicySelection),
		PostReportHumanize:           postReportHumanize,
		GenerationGuidanceProfile:    generationGuidanceProfile,
		GenerationGuidanceSHA256:     generationGuidanceSHA256,
		OutputKind:                   outputKind,
		ArticleIntent:                articleIntent,
	})
	if err != nil {
		return nil, err
	}
	result := map[string]any{"artifact": output.Artifact, "event": output.Event, "markdown": output.Markdown}
	return result, nil
}

func reportWorkflowRigor(rigor reportRigorProfile) reportprompt.RigorProfile {
	return reportprompt.RigorProfile{
		Level:        rigor.level,
		Label:        rigor.label,
		Description:  rigor.description,
		Instructions: rigor.instructions,
	}
}

func (server *Server) createSectionalLongFormReportDraft(ctx context.Context, missionID string, title string, directionHint string, executorName string, agentModel string, agentReasoningEffort string, agentSelectionSource string, mcpMode string, rigor reportRigorProfile, reportSessionPolicy string, reportSessionPolicySelection string, postReportHumanize string, generationGuidanceProfile string, generationGuidanceSHA256 string, pendingEventID string, executor AgentExecutor) (map[string]any, error) {
	return server.createLongFormPrefixWorkflowDraft(ctx, missionID, title, directionHint, executorName, agentModel, agentReasoningEffort, agentSelectionSource, mcpMode, rigor, reportSessionPolicy, reportSessionPolicySelection, postReportHumanize, generationGuidanceProfile, generationGuidanceSHA256, pendingEventID, reportExecutionStrategySerial, executor)
}

func logLongFormFinalObservation(missionID, pendingEventID, planEventID string, attempt int, boundSessionID string, result AgentResult, durationMS int64) {
	returnedSessionID := strings.TrimSpace(result.SessionID)
	inputTokens, outputTokens, totalTokens := 0, 0, 0
	usageAvailable := result.Usage.ProviderUsage != nil
	if usageAvailable {
		inputTokens = result.Usage.ProviderUsage.InputTokens
		outputTokens = result.Usage.ProviderUsage.OutputTokens
		totalTokens = result.Usage.ProviderUsage.TotalTokens
		if totalTokens == 0 {
			totalTokens = inputTokens + outputTokens
		}
	}
	log.Printf("report_long_form_final_observed mission_id=%q pending_event_id=%q plan_event_id=%q attempt_count=%d returned_session_present=%t returned_session_matches_bound=%t usage_available=%t input_tokens=%d output_tokens=%d total_tokens=%d resumed=%t duration_ms=%d", missionID, pendingEventID, planEventID, attempt, returnedSessionID != "", returnedSessionID != "" && returnedSessionID == strings.TrimSpace(boundSessionID), usageAvailable, inputTokens, outputTokens, totalTokens, result.Resumed, durationMS)
}
