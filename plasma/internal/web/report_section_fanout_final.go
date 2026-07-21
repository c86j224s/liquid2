package web

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) finalizeSectionFanoutLongForm(ctx context.Context, req sectionFanoutLongFormRequest, state sectionFanoutPlanState, parts []sectionalReportPartDraft, sectionArtifactIDs []string, partArtifactIDs []string, sectionWordTotal int, finalSessionID string, finalForkSourceID string, started time.Time, executor AgentExecutor) (app.RawArtifact, app.LedgerEvent, AgentResult, error) {
	toolSessionID := newID("ses")
	binding := reporting.LongFormFinalizeBinding{
		MissionID: req.missionID, PendingEventID: req.pendingEventID, PlanEventID: state.planEvent.EventID, ArtifactID: state.artifactID,
		Filename: safeFilename(req.title, ".md"), Title: req.title, ToolSessionID: toolSessionID,
		IdempotencyKey:    "report-long-form-finalize:" + req.pendingEventID + ":" + state.planEvent.EventID,
		ProviderSessionID: finalSessionID, PreviousProviderSessionID: finalSessionID,
		PartArtifactIDs: partArtifactIDs, SectionArtifactIDs: sectionArtifactIDs, SectionWordCount: sectionWordTotal,
		AgentExecutor: req.executorName, AgentModel: req.agentModel, AgentReasoningEffort: req.agentReasoningEffort, AgentSelectionSource: req.agentSelectionSource,
		MCPMode: req.mcpMode, RigorLevel: req.rigor.level, RigorLabel: req.rigor.label,
		ReportSessionPolicy: state.reportSessionPolicy, ReportSessionPolicySelection: state.reportSessionPolicySelection,
		PostReportHumanize: req.postReportHumanize, GenerationGuidanceProfile: req.generationGuidanceProfile, GenerationGuidanceSHA256: req.generationGuidanceSHA256,
		SessionChainKind: state.sessionChainKind, PreReportResearchSessionID: state.preReportResearchSessionID, ReportPlanSessionID: state.reportPlanSessionID,
		ForkSourceAgentSessionID: finalForkSourceID, PlanToolSessionID: reportEventString(state.planEvent, "tool_session_id"), StartedAt: started,
		Producer: app.Producer{Type: "agent_session", ID: finalSessionID},
	}
	var finalResult AgentResult
	var finalization reporting.LongFormFinalizeResult
	var hint reporting.LongFormFinalizationHint
	canonical := false
	for attempt := 1; attempt <= 2; attempt++ {
		attemptStarted := time.Now()
		result, runErr := executor.Run(ctx, AgentRequest{
			UserText: "finalize section-fanout long-form markdown report",
			Prompt:   agentLongFormFinalizePrompt(req.title, req.missionID, req.rigor, state.plan, parts, req.generationGuidanceProfile, binding, attempt, canonical, hint),
			Model:    req.agentModel, ReasoningEffort: req.agentReasoningEffort, MissionID: req.missionID, ToolSessionID: toolSessionID,
			PreviousSessionID: finalSessionID, AgentExecutor: req.executorName, MCPMode: req.mcpMode,
			ExtraMCPTools: reportFinalizeMCPTools(), ReplaceMCPTools: true, LongFormFinalize: &binding,
		})
		durationMS := time.Since(attemptStarted).Milliseconds()
		logLongFormFinalObservation(req.missionID, req.pendingEventID, state.planEvent.EventID, attempt, finalSessionID, result, durationMS)
		if runErr == nil {
			result, runErr = validatedSameSessionResult(result, finalSessionID)
		}
		if runErr == nil {
			finalResult = result
		}
		loaded, exists, loadErr := reporting.LoadLongFormFinalization(context.WithoutCancel(ctx), server.service, binding)
		if loadErr != nil {
			return app.RawArtifact{}, app.LedgerEvent{}, finalResult, longFormStageFailure("final", state.planEvent.EventID, 0, 0, loadErr)
		}
		canonical = exists
		if exists {
			finalization = loaded
		}
		if runErr == nil && canonical && result.Text == "REPORT_FINALIZED" {
			return finalization.Artifact, finalization.Event, finalResult, nil
		}
		if attempt == 1 {
			hint = reporting.RecoverLongFormFinalizationHint(result.Text)
			continue
		}
		cause := runErr
		if cause == nil {
			cause = fmt.Errorf("%w: finalization acknowledgement was not exact", app.ErrConflict)
		}
		if canonical {
			return finalization.Artifact, finalization.Event, finalResult, nil
		}
		return app.RawArtifact{}, app.LedgerEvent{}, finalResult, longFormStageFailure("final", state.planEvent.EventID, 0, 0, reportAgentFailure(cause, result, "report_frame", durationMS, finalSessionID))
	}
	return finalization.Artifact, finalization.Event, finalResult, nil
}

func forkSectionFanoutSession(ctx context.Context, forker AgentSessionForker, sourceSessionID string) (string, string, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return "", "", fmt.Errorf("%w: section fanout requires a report plan provider session", app.ErrConflict)
	}
	fork, err := forker.ForkSession(ctx, sourceSessionID)
	if err != nil {
		return "", "", fmt.Errorf("section fanout session fork failed: %w", err)
	}
	if strings.TrimSpace(fork.SessionID) == "" {
		return "", "", fmt.Errorf("%w: section fanout session fork returned an empty session", app.ErrConflict)
	}
	return strings.TrimSpace(fork.SessionID), strings.TrimSpace(fork.SourceSessionID), nil
}
