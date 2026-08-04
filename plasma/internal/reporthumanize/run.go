package reporthumanize

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportpatch"
)

// HumanizeMarkdownReport attempts the H5 tone pass through MCP patch tools. It
// never replaces the source artifact; failures and no-op outcomes close the H5
// pending event while preserving the original Markdown artifact.
func HumanizeMarkdownReport(ctx context.Context, service Service, idFunc IDFunc, missionID string, input Input, executor agentexec.AgentExecutor) (Result, error) {
	if executor == nil {
		return Result{}, nil
	}
	if idFunc == nil {
		idFunc = defaultID
	}
	original := strings.TrimSpace(input.Markdown)
	if original == "" {
		return Result{}, nil
	}
	toolSessionID := firstNonEmpty(strings.TrimSpace(input.ToolSessionID), idFunc("ses"))
	humanizePendingEventID := strings.TrimSpace(input.HumanizePendingEventID)
	if humanizePendingEventID == "" {
		pendingEvent, err := appendPending(ctx, service, idFunc, missionID, input, toolSessionID)
		if err != nil {
			return Result{}, nil
		}
		humanizePendingEventID = pendingEvent.EventID
	}
	patchReq := patchRequest(input)
	started := time.Now()
	result, err := executor.Run(ctx, agentexec.AgentRequest{
		UserText:          "humanize finalized markdown report tone",
		Prompt:            Prompt(input.Title, missionID, toolSessionID, humanizePendingEventID, input.SourceArtifact.ArtifactID, patchReq),
		Model:             input.AgentModel,
		ReasoningEffort:   input.ReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: input.PreviousSessionID,
		AgentExecutor:     input.ExecutorName,
		MCPMode:           input.MCPMode,
		ExtraMCPTools:     reportpatch.MCPTools(),
		ReplaceMCPTools:   true,
		ReportPatch: &agentexec.AgentReportPatchContext{
			BaseArtifactID:               input.SourceArtifact.ArtifactID,
			PendingEventID:               humanizePendingEventID,
			AgentExecutor:                input.ExecutorName,
			AgentModel:                   input.AgentModel,
			AgentReasoningEffort:         input.ReasoningEffort,
			MCPMode:                      input.MCPMode,
			AgentSessionID:               input.PreviousSessionID,
			PreviousAgentSessionID:       patchReq.PreviousAgentSessionID,
			ReturnedAgentSessionID:       input.PreviousSessionID,
			ReportSessionID:              input.PreviousSessionID,
			ReportSessionPolicy:          patchReq.ReportSessionPolicy,
			ReportSessionPolicySelection: patchReq.ReportSessionPolicySelection,
			SessionChainKind:             patchReq.SessionChainKind,
		},
	})
	durationMS := time.Since(started).Milliseconds()
	if err != nil {
		_, _ = appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, fmt.Errorf("humanize agent failed: %w", err))
		return Result{}, nil
	}
	return handleAgentResult(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, original, result, durationMS)
}

func handleAgentResult(ctx context.Context, service Service, idFunc IDFunc, missionID string, input Input, toolSessionID string, humanizePendingEventID string, original string, result agentexec.AgentResult, durationMS int64) (Result, error) {
	humanizedSessionID := strings.TrimSpace(result.SessionID)
	validated, err := validatedSameSessionResult(result, input.PreviousSessionID)
	if err != nil {
		_, _ = appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, err)
		return Result{}, nil
	}
	finalizedEvent, ok, err := finalizedPatchEvent(ctx, service, missionID, humanizePendingEventID)
	if err != nil {
		_, _ = appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, err)
		return Result{}, nil
	}
	if !ok {
		return handleMissingFinalize(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, result, durationMS)
	}
	artifact, err := finalizedArtifact(ctx, service, missionID, finalizedEvent)
	if err != nil {
		_, _ = appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, err)
		return Result{}, nil
	}
	return acceptFinalizedArtifact(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, original, artifactAcceptInput{
		artifact:           artifact,
		finalizedEvent:     finalizedEvent,
		validated:          validated,
		humanizedSessionID: humanizedSessionID,
		durationMS:         durationMS,
	})
}

func handleMissingFinalize(ctx context.Context, service Service, idFunc IDFunc, missionID string, input Input, toolSessionID string, humanizePendingEventID string, result agentexec.AgentResult, durationMS int64) (Result, error) {
	if noChangesResult(result.Text) {
		_, _ = appendSkipped(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS)
		return Result{}, nil
	}
	activity, activityErr := patchToolActivity(ctx, service, missionID, toolSessionID)
	if activityErr != nil {
		_, _ = appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, activityErr)
		return Result{}, nil
	}
	if activity.Started && activity.ApplyCount == 0 && activity.FinalizeCount == 0 {
		_, _ = appendSkipped(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS)
		return Result{}, nil
	}
	cause := fmt.Errorf("%w: H5 agent did not finalize through report patch MCP", producterror.ErrInvalidInput)
	if activity.LastError != "" {
		cause = fmt.Errorf("%w: H5 report patch MCP failed: %s", producterror.ErrInvalidInput, activity.LastError)
	}
	_, _ = appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, agentFailure(cause, result, "report_humanize_h5", durationMS, input.PreviousSessionID))
	return Result{}, nil
}

type artifactAcceptInput struct {
	artifact           artifact.Raw
	finalizedEvent     ledger.Event
	validated          agentexec.AgentResult
	humanizedSessionID string
	durationMS         int64
}

func acceptFinalizedArtifact(ctx context.Context, service Service, idFunc IDFunc, missionID string, input Input, toolSessionID string, humanizePendingEventID string, original string, accepted artifactAcceptInput) (Result, error) {
	humanized := strings.TrimSpace(string(accepted.artifact.Content))
	if humanized == "" {
		_, _ = appendRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, accepted.artifact, accepted.finalizedEvent, "empty_humanized_markdown")
		_, _ = appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, accepted.durationMS, fmt.Errorf("%w: humanize agent returned empty Markdown", producterror.ErrInvalidInput))
		return Result{}, nil
	}
	if err := ValidateMarkdown(original, humanized); err != nil {
		_, _ = appendRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, accepted.artifact, accepted.finalizedEvent, "validation_failed")
		_, _ = appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, accepted.durationMS, err)
		return Result{}, nil
	}
	if humanized == original {
		_, _ = appendRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, accepted.artifact, accepted.finalizedEvent, "unchanged_humanized_markdown")
		_, _ = appendSkipped(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, accepted.durationMS)
		return Result{}, nil
	}
	if terminalExists(ctx, service, missionID, humanizePendingEventID) {
		_, _ = appendRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, accepted.artifact, accepted.finalizedEvent, "terminal_already_closed")
		return Result{}, nil
	}
	event, closed, err := appendTerminal(ctx, service, missionID, humanizePendingEventID, reporting.BuildHumanizedMarkdownExportAppendRequest(reporting.HumanizedMarkdownExportEventRequest{
		HumanizeEventBase:      eventBase(idFunc("evt"), missionID, input, toolSessionID, humanizePendingEventID, ledger.Producer{Type: "agent_session", ID: fallbackSessionID(accepted.validated.SessionID, toolSessionID)}),
		PatchEventID:           accepted.finalizedEvent.EventID,
		Artifact:               accepted.artifact,
		AgentSessionID:         accepted.validated.SessionID,
		ReturnedAgentSessionID: accepted.humanizedSessionID,
		SourceWordCount:        reportWordCount(original),
		HumanizedWordCount:     reportWordCount(humanized),
		DurationMS:             accepted.durationMS,
		AgentUsage:             accepted.validated.Usage,
		AgentResumed:           accepted.validated.Resumed,
	}))
	if err != nil {
		_, _ = appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, accepted.durationMS, err)
		return Result{}, nil
	}
	if !closed {
		return Result{}, nil
	}
	return Result{Artifact: accepted.artifact, Event: event, Markdown: humanized, Applied: true}, nil
}
