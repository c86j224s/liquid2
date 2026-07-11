package web

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	reportHumanizeTerminalWriteTimeout = 10 * time.Second
)

type ReportHumanizeInput struct {
	Title                  string
	Markdown               string
	SourceArtifact         app.RawArtifact
	ExecutorName           string
	AgentModel             string
	ReasoningEffort        string
	MCPMode                string
	PreviousSessionID      string
	ReportMode             string
	PendingEventID         string
	HumanizePendingEventID string
	ToolSessionID          string
}

type reportHumanizeInput = ReportHumanizeInput

type ReportHumanizeResult struct {
	Artifact app.RawArtifact
	Event    app.LedgerEvent
	Markdown string
	Applied  bool
}

type ReportHumanizeIDFunc func(prefix string) string

type ReportHumanizeService interface {
	GetRawArtifact(context.Context, string) (app.RawArtifact, error)
	AppendEvent(context.Context, app.AppendEventRequest) (app.LedgerEvent, error)
}

type reportHumanizeEventLister interface {
	ListEvents(context.Context, string) ([]app.LedgerEvent, error)
}

func (server *Server) humanizeMarkdownReport(ctx context.Context, missionID string, input reportHumanizeInput, executor AgentExecutor) (reportHumanizeResult, error) {
	return HumanizeMarkdownReport(ctx, server.service, newID, missionID, input, executor)
}

type reportHumanizeResult = ReportHumanizeResult

type reportHumanizePendingPayload struct {
	Target               string `json:"target"`
	Profile              string `json:"profile"`
	PendingEventID       string `json:"pending_event_id"`
	ReportPendingEventID string `json:"report_pending_event_id"`
	Title                string `json:"title"`
	SourceArtifactID     string `json:"source_artifact_id"`
	SourceArtifactSHA256 string `json:"source_artifact_sha256"`
	AgentExecutor        string `json:"agent_executor"`
	AgentModel           string `json:"agent_model"`
	AgentReasoningEffort string `json:"agent_reasoning_effort"`
	PreviousSessionID    string `json:"previous_agent_session_id"`
	ToolSessionID        string `json:"tool_session_id"`
	MCPMode              string `json:"mcp_mode"`
	ReportMode           string `json:"report_mode"`
	ReportModeLabel      string `json:"report_mode_label"`
	HumanizeTransport    string `json:"humanize_transport"`
}

func HumanizeMarkdownReport(ctx context.Context, service ReportHumanizeService, idFunc ReportHumanizeIDFunc, missionID string, input ReportHumanizeInput, executor AgentExecutor) (ReportHumanizeResult, error) {
	if executor == nil {
		return ReportHumanizeResult{}, nil
	}
	if idFunc == nil {
		idFunc = newID
	}
	original := strings.TrimSpace(input.Markdown)
	if original == "" {
		return ReportHumanizeResult{}, nil
	}
	toolSessionID := firstNonEmpty(strings.TrimSpace(input.ToolSessionID), idFunc("ses"))
	humanizePendingEventID := strings.TrimSpace(input.HumanizePendingEventID)
	if humanizePendingEventID == "" {
		pendingEvent, err := appendReportHumanizePending(ctx, service, idFunc, missionID, input, toolSessionID)
		if err != nil {
			return ReportHumanizeResult{}, nil
		}
		humanizePendingEventID = pendingEvent.EventID
	}
	patchReq := reportHumanizePatchRequest(input)
	started := time.Now()
	result, err := executor.Run(ctx, AgentRequest{
		UserText:          "humanize finalized markdown report tone",
		Prompt:            agentReportHumanizePatchPrompt(input.Title, missionID, toolSessionID, humanizePendingEventID, input.SourceArtifact.ArtifactID, patchReq),
		Model:             input.AgentModel,
		ReasoningEffort:   input.ReasoningEffort,
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: input.PreviousSessionID,
		AgentExecutor:     input.ExecutorName,
		MCPMode:           input.MCPMode,
		ExtraMCPTools:     reportPatchMCPTools(),
		ReplaceMCPTools:   true,
		ReportPatch: &AgentReportPatchContext{
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
		_, _ = appendReportHumanizeFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, fmt.Errorf("humanize agent failed: %w", err))
		return ReportHumanizeResult{}, nil
	}
	humanizedSessionID := strings.TrimSpace(result.SessionID)
	validated, err := validatedSameSessionResult(result, input.PreviousSessionID)
	if err != nil {
		_, _ = appendReportHumanizeFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, err)
		return ReportHumanizeResult{}, nil
	}
	finalizedEvent, ok, err := reportHumanizeFinalizedPatchEvent(ctx, service, missionID, humanizePendingEventID)
	if err != nil {
		_, _ = appendReportHumanizeFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, err)
		return ReportHumanizeResult{}, nil
	}
	if !ok {
		if reportHumanizeNoChangesResult(result.Text) {
			_, _ = appendReportHumanizeSkipped(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS)
			return ReportHumanizeResult{}, nil
		}
		activity, activityErr := reportHumanizePatchToolActivity(ctx, service, missionID, toolSessionID)
		if activityErr != nil {
			_, _ = appendReportHumanizeFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, activityErr)
			return ReportHumanizeResult{}, nil
		}
		if activity.Started && activity.ApplyCount == 0 && activity.FinalizeCount == 0 {
			_, _ = appendReportHumanizeSkipped(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS)
			return ReportHumanizeResult{}, nil
		}
		cause := fmt.Errorf("%w: H5 agent did not finalize through report patch MCP", app.ErrInvalidInput)
		if activity.LastError != "" {
			cause = fmt.Errorf("%w: H5 report patch MCP failed: %s", app.ErrInvalidInput, activity.LastError)
		}
		_, _ = appendReportHumanizeFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, reportAgentFailure(cause, result, "report_humanize_h5", durationMS, input.PreviousSessionID))
		return ReportHumanizeResult{}, nil
	}
	artifact, err := reportHumanizeFinalizedArtifact(ctx, service, missionID, finalizedEvent)
	if err != nil {
		_, _ = appendReportHumanizeFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, err)
		return ReportHumanizeResult{}, nil
	}
	humanized := strings.TrimSpace(string(artifact.Content))
	if humanized == "" {
		_, _ = appendReportHumanizeRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, artifact, finalizedEvent, "empty_humanized_markdown")
		_, _ = appendReportHumanizeFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, fmt.Errorf("%w: humanize agent returned empty Markdown", app.ErrInvalidInput))
		return ReportHumanizeResult{}, nil
	}
	if err := validateHumanizedMarkdown(original, humanized); err != nil {
		_, _ = appendReportHumanizeRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, artifact, finalizedEvent, "validation_failed")
		_, _ = appendReportHumanizeFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, err)
		return ReportHumanizeResult{}, nil
	}
	if humanized == original {
		_, _ = appendReportHumanizeRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, artifact, finalizedEvent, "unchanged_humanized_markdown")
		_, _ = appendReportHumanizeSkipped(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS)
		return ReportHumanizeResult{}, nil
	}
	if reportHumanizeTerminalExists(ctx, service, missionID, humanizePendingEventID) {
		_, _ = appendReportHumanizeRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, artifact, finalizedEvent, "terminal_already_closed")
		return ReportHumanizeResult{}, nil
	}
	event, err := service.AppendEvent(ctx, reporting.BuildHumanizedMarkdownExportAppendRequest(reporting.HumanizedMarkdownExportEventRequest{
		HumanizeEventBase:      reportHumanizeEventBase(idFunc("evt"), missionID, input, toolSessionID, humanizePendingEventID, app.Producer{Type: "agent_session", ID: fallbackSessionID(validated.SessionID, toolSessionID)}),
		PatchEventID:           finalizedEvent.EventID,
		Artifact:               artifact,
		AgentSessionID:         validated.SessionID,
		ReturnedAgentSessionID: humanizedSessionID,
		SourceWordCount:        reportWordCount(original),
		HumanizedWordCount:     reportWordCount(humanized),
		DurationMS:             durationMS,
		AgentUsage:             validated.Usage,
		AgentResumed:           validated.Resumed,
	}))
	if err != nil {
		_, _ = appendReportHumanizeFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, durationMS, err)
		return ReportHumanizeResult{}, nil
	}
	return ReportHumanizeResult{Artifact: artifact, Event: event, Markdown: humanized, Applied: true}, nil
}

func appendReportHumanizeRejectedPatch(ctx context.Context, service ReportHumanizeService, idFunc ReportHumanizeIDFunc, missionID string, input reportHumanizeInput, toolSessionID string, humanizePendingEventID string, artifact app.RawArtifact, finalized app.LedgerEvent, reason string) (app.LedgerEvent, error) {
	ledgerCtx, cancel := reportHumanizeTerminalWriteContext(ctx)
	defer cancel()
	return service.AppendEvent(ledgerCtx, reporting.BuildHumanizePatchRejectedAppendRequest(reporting.HumanizePatchRejectedEventRequest{
		HumanizeEventBase: reportHumanizeEventBase(idFunc("evt"), missionID, input, toolSessionID, humanizePendingEventID, app.Producer{Type: "agent", ID: firstNonEmpty(input.ExecutorName, "plasma")}),
		PatchEventID:      finalized.EventID,
		Artifact:          artifact,
		Reason:            reason,
	}))
}

func appendReportHumanizePending(ctx context.Context, service ReportHumanizeService, idFunc ReportHumanizeIDFunc, missionID string, input reportHumanizeInput, toolSessionID string) (app.LedgerEvent, error) {
	eventID := idFunc("evt")
	return service.AppendEvent(ctx, reporting.BuildHumanizePendingAppendRequest(reporting.HumanizePendingEventRequest{
		HumanizeEventBase: reportHumanizeEventBase(eventID, missionID, input, toolSessionID, eventID, app.Producer{Type: "agent", ID: firstNonEmpty(input.ExecutorName, "plasma")}),
	}))
}

func appendReportHumanizeSkipped(ctx context.Context, service ReportHumanizeService, idFunc ReportHumanizeIDFunc, missionID string, input reportHumanizeInput, toolSessionID string, humanizePendingEventID string, durationMS int64) (app.LedgerEvent, error) {
	ledgerCtx, cancel := reportHumanizeTerminalWriteContext(ctx)
	defer cancel()
	if reportHumanizeTerminalExists(ledgerCtx, service, missionID, humanizePendingEventID) {
		return app.LedgerEvent{}, nil
	}
	return service.AppendEvent(ledgerCtx, reporting.BuildHumanizeSkippedAppendRequest(reporting.HumanizeSkippedEventRequest{
		HumanizeEventBase: reportHumanizeEventBase(idFunc("evt"), missionID, input, toolSessionID, humanizePendingEventID, app.Producer{Type: "agent", ID: firstNonEmpty(input.ExecutorName, "plasma")}),
		DurationMS:        durationMS,
	}))
}

func appendReportHumanizeFailed(ctx context.Context, service ReportHumanizeService, idFunc ReportHumanizeIDFunc, missionID string, input reportHumanizeInput, toolSessionID string, humanizePendingEventID string, durationMS int64, cause error) (app.LedgerEvent, error) {
	ledgerCtx, cancel := reportHumanizeTerminalWriteContext(ctx)
	defer cancel()
	if reportHumanizeTerminalExists(ledgerCtx, service, missionID, humanizePendingEventID) {
		return app.LedgerEvent{}, nil
	}
	return service.AppendEvent(ledgerCtx, reporting.BuildHumanizeFailedAppendRequest(reporting.HumanizeFailedEventRequest{
		HumanizeEventBase: reportHumanizeEventBase(idFunc("evt"), missionID, input, toolSessionID, humanizePendingEventID, app.Producer{Type: "agent", ID: firstNonEmpty(input.ExecutorName, "plasma")}),
		DurationMS:        durationMS,
		Error:             cause.Error(),
	}))
}

func reportHumanizePendingPayloadFromEvent(event app.LedgerEvent) reportHumanizePendingPayload {
	var payload reportHumanizePendingPayload
	_ = json.Unmarshal(event.Payload, &payload)
	return payload
}

func reportHumanizeInFlightPendingEventID(event app.LedgerEvent) string {
	payload := reportHumanizePendingPayloadFromEvent(event)
	return strings.TrimSpace(payload.ReportPendingEventID)
}

func (server *Server) appendReportHumanizeStaleFailed(ctx context.Context, missionID string, pending app.LedgerEvent) (app.LedgerEvent, error) {
	if server.hasReportDraftTerminalEvent(ctx, missionID, pending.EventID) {
		return app.LedgerEvent{}, nil
	}
	payload := reportHumanizePendingPayloadFromEvent(pending)
	executor := firstNonEmpty(strings.TrimSpace(payload.AgentExecutor), "plasma")
	return server.service.AppendEvent(ctx, reporting.BuildHumanizeFailedAppendRequest(reporting.HumanizeFailedEventRequest{
		HumanizeEventBase: reporting.HumanizeEventBase{
			EventID:                newID("evt"),
			MissionID:              missionID,
			PendingEventID:         pending.EventID,
			ReportPendingEventID:   strings.TrimSpace(payload.ReportPendingEventID),
			Title:                  strings.TrimSpace(payload.Title),
			SourceArtifactID:       strings.TrimSpace(payload.SourceArtifactID),
			SourceArtifactSHA256:   strings.TrimSpace(payload.SourceArtifactSHA256),
			AgentExecutor:          executor,
			AgentModel:             strings.TrimSpace(payload.AgentModel),
			AgentReasoningEffort:   strings.TrimSpace(payload.AgentReasoningEffort),
			PreviousAgentSessionID: strings.TrimSpace(payload.PreviousSessionID),
			ToolSessionID:          strings.TrimSpace(payload.ToolSessionID),
			MCPMode:                strings.TrimSpace(payload.MCPMode),
			ReportMode:             strings.TrimSpace(payload.ReportMode),
			ReportModeLabel:        strings.TrimSpace(payload.ReportModeLabel),
			Target:                 firstNonEmpty(strings.TrimSpace(payload.Target), reporting.ExportTargetHumanizedMarkdown),
			Profile:                firstNonEmpty(strings.TrimSpace(payload.Profile), reporting.HumanizeProfileH5),
			HumanizeTransport:      firstNonEmpty(strings.TrimSpace(payload.HumanizeTransport), reporting.HumanizeTransportPatch),
			Producer:               app.Producer{Type: "agent", ID: executor},
		},
		Kind:         "humanized_markdown_report_stale_failed",
		Text:         "H5 말투 보정 작업이 중단된 상태로 남아 원본 Markdown artifact를 유지했습니다.",
		Error:        "stale humanized Markdown report generation was not running after restart",
		Relationship: "stale_post_report_tone_pass_of_source_artifact",
		OmitDuration: true,
		FailedAt:     time.Now().UTC().Format(time.RFC3339Nano),
	}))
}

func (server *Server) recoverStaleReportHumanizeFinalizedPatch(ctx context.Context, missionID string, pending app.LedgerEvent) (bool, error) {
	if server.hasReportDraftTerminalEvent(ctx, missionID, pending.EventID) {
		return true, nil
	}
	finalized, ok, err := reportHumanizeFinalizedPatchEvent(ctx, server.service, missionID, pending.EventID)
	if err != nil || !ok {
		return ok, err
	}
	payload := reportHumanizePendingPayloadFromEvent(pending)
	toolSessionID := strings.TrimSpace(payload.ToolSessionID)
	patchArtifact, artifactErr := reportHumanizeFinalizedArtifact(ctx, server.service, missionID, finalized)
	sourceArtifact, sourceErr := server.service.GetRawArtifact(ctx, strings.TrimSpace(payload.SourceArtifactID))
	input := reportHumanizeInputFromPendingPayload(payload, sourceArtifact)
	if artifactErr != nil {
		_, err := appendReportHumanizeFailed(ctx, server.service, newID, missionID, input, toolSessionID, pending.EventID, 0, fmt.Errorf("recover finalized H5 patch artifact: %w", artifactErr))
		return true, err
	}
	if sourceErr != nil {
		if _, err := appendReportHumanizeRejectedPatch(ctx, server.service, newID, missionID, input, toolSessionID, pending.EventID, patchArtifact, finalized, "missing_source_artifact_for_recovery"); err != nil {
			return true, err
		}
		_, err := appendReportHumanizeFailed(ctx, server.service, newID, missionID, input, toolSessionID, pending.EventID, 0, fmt.Errorf("recover finalized H5 patch source artifact: %w", sourceErr))
		return true, err
	}
	original := strings.TrimSpace(string(sourceArtifact.Content))
	humanized := strings.TrimSpace(string(patchArtifact.Content))
	if humanized == "" {
		if _, err := appendReportHumanizeRejectedPatch(ctx, server.service, newID, missionID, input, toolSessionID, pending.EventID, patchArtifact, finalized, "empty_humanized_markdown_recovered_after_restart"); err != nil {
			return true, err
		}
		_, err := appendReportHumanizeFailed(ctx, server.service, newID, missionID, input, toolSessionID, pending.EventID, 0, fmt.Errorf("%w: recovered H5 patch artifact is empty", app.ErrInvalidInput))
		return true, err
	}
	if err := validateHumanizedMarkdown(original, humanized); err != nil {
		if _, rejectErr := appendReportHumanizeRejectedPatch(ctx, server.service, newID, missionID, input, toolSessionID, pending.EventID, patchArtifact, finalized, "validation_failed_recovered_after_restart"); rejectErr != nil {
			return true, rejectErr
		}
		_, failErr := appendReportHumanizeFailed(ctx, server.service, newID, missionID, input, toolSessionID, pending.EventID, 0, err)
		return true, failErr
	}
	if humanized == original {
		if _, err := appendReportHumanizeRejectedPatch(ctx, server.service, newID, missionID, input, toolSessionID, pending.EventID, patchArtifact, finalized, "unchanged_humanized_markdown_recovered_after_restart"); err != nil {
			return true, err
		}
		_, err := appendReportHumanizeSkipped(ctx, server.service, newID, missionID, input, toolSessionID, pending.EventID, 0)
		return true, err
	}
	if _, err := appendReportHumanizeRecoveredExport(ctx, server.service, newID, missionID, input, toolSessionID, pending.EventID, patchArtifact, finalized, original, humanized); err != nil {
		return true, err
	}
	return true, nil
}

func reportHumanizeInputFromPendingPayload(payload reportHumanizePendingPayload, sourceArtifact app.RawArtifact) reportHumanizeInput {
	if strings.TrimSpace(sourceArtifact.ArtifactID) == "" {
		sourceArtifact.ArtifactID = strings.TrimSpace(payload.SourceArtifactID)
	}
	if strings.TrimSpace(sourceArtifact.SHA256) == "" {
		sourceArtifact.SHA256 = strings.TrimSpace(payload.SourceArtifactSHA256)
	}
	return reportHumanizeInput{
		Title:             strings.TrimSpace(payload.Title),
		Markdown:          strings.TrimSpace(string(sourceArtifact.Content)),
		SourceArtifact:    sourceArtifact,
		ExecutorName:      firstNonEmpty(strings.TrimSpace(payload.AgentExecutor), "plasma"),
		AgentModel:        strings.TrimSpace(payload.AgentModel),
		ReasoningEffort:   strings.TrimSpace(payload.AgentReasoningEffort),
		MCPMode:           strings.TrimSpace(payload.MCPMode),
		PreviousSessionID: strings.TrimSpace(payload.PreviousSessionID),
		ReportMode:        strings.TrimSpace(payload.ReportMode),
		PendingEventID:    strings.TrimSpace(payload.ReportPendingEventID),
	}
}

func appendReportHumanizeRecoveredExport(ctx context.Context, service ReportHumanizeService, idFunc ReportHumanizeIDFunc, missionID string, input reportHumanizeInput, toolSessionID string, humanizePendingEventID string, artifact app.RawArtifact, finalized app.LedgerEvent, original string, humanized string) (app.LedgerEvent, error) {
	ledgerCtx, cancel := reportHumanizeTerminalWriteContext(ctx)
	defer cancel()
	if reportHumanizeTerminalExists(ledgerCtx, service, missionID, humanizePendingEventID) {
		return app.LedgerEvent{}, nil
	}
	producerID := firstNonEmpty(strings.TrimSpace(input.PreviousSessionID), strings.TrimSpace(toolSessionID), strings.TrimSpace(finalized.CorrelationID))
	return service.AppendEvent(ledgerCtx, reporting.BuildHumanizedMarkdownExportAppendRequest(reporting.HumanizedMarkdownExportEventRequest{
		HumanizeEventBase:      reportHumanizeEventBase(idFunc("evt"), missionID, input, toolSessionID, humanizePendingEventID, app.Producer{Type: "agent_session", ID: producerID}),
		PatchEventID:           finalized.EventID,
		Artifact:               artifact,
		AgentSessionID:         strings.TrimSpace(input.PreviousSessionID),
		ReturnedAgentSessionID: strings.TrimSpace(input.PreviousSessionID),
		SourceWordCount:        reportWordCount(original),
		HumanizedWordCount:     reportWordCount(humanized),
		RecoveredAfterRestart:  true,
		Text:                   "서버 재시작 전에 완료된 H5 말투 보정 Markdown artifact를 검증해 복구했습니다.",
	}))
}

func reportHumanizeEventBase(eventID string, missionID string, input reportHumanizeInput, toolSessionID string, pendingEventID string, producer app.Producer) reporting.HumanizeEventBase {
	return reporting.HumanizeEventBase{
		EventID:                eventID,
		MissionID:              missionID,
		PendingEventID:         pendingEventID,
		ReportPendingEventID:   input.PendingEventID,
		Title:                  input.Title,
		SourceArtifactID:       input.SourceArtifact.ArtifactID,
		SourceArtifactSHA256:   input.SourceArtifact.SHA256,
		AgentExecutor:          input.ExecutorName,
		AgentModel:             input.AgentModel,
		AgentReasoningEffort:   input.ReasoningEffort,
		PreviousAgentSessionID: input.PreviousSessionID,
		ToolSessionID:          toolSessionID,
		MCPMode:                input.MCPMode,
		ReportMode:             input.ReportMode,
		Producer:               producer,
	}
}

func reportHumanizeTerminalExists(ctx context.Context, service ReportHumanizeService, missionID string, pendingEventID string) bool {
	pendingEventID = strings.TrimSpace(pendingEventID)
	if pendingEventID == "" {
		return false
	}
	lister, ok := service.(reportHumanizeEventLister)
	if !ok {
		return false
	}
	events, err := lister.ListEvents(ctx, missionID)
	if err != nil {
		return false
	}
	_, ok = reporting.CompletedPendingEventIDs(events)[pendingEventID]
	return ok
}

func reportHumanizeTerminalWriteContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx != nil && ctx.Err() == nil {
		return ctx, func() {}
	}
	return context.WithTimeout(context.Background(), reportHumanizeTerminalWriteTimeout)
}

func reportHumanizePatchRequest(input ReportHumanizeInput) reporting.PatchRequest {
	return reporting.PatchRequest{
		BaseArtifactID:               strings.TrimSpace(input.SourceArtifact.ArtifactID),
		Instruction:                  reportHumanizePatchInstruction(),
		Title:                        firstNonEmpty(strings.TrimSpace(input.Title)+" humanized", "Humanized report"),
		AgentExecutor:                strings.TrimSpace(input.ExecutorName),
		AgentModel:                   strings.TrimSpace(input.AgentModel),
		AgentReasoningEffort:         strings.TrimSpace(input.ReasoningEffort),
		MCPMode:                      strings.TrimSpace(input.MCPMode),
		ReportSessionID:              strings.TrimSpace(input.PreviousSessionID),
		PreviousAgentSessionID:       strings.TrimSpace(input.PreviousSessionID),
		ReportSessionPolicy:          reportSessionPolicySameSession,
		ReportSessionPolicySelection: "auto_same_report_session_h5",
		SessionChainKind:             "same_report_session_h5_humanize_patch",
	}
}

func reportHumanizePatchInstruction() string {
	return "Apply the H5 Korean tone pass to this Markdown report. Smooth stiff AI-like Korean phrasing, repetitive transitions, and unnatural sentence endings, but preserve the report structure, claims, citations, numbers, tables, code, links, headings, paragraph boundaries, and useful detail. Use bounded MCP reads and small targeted patch operations. Do not rewrite or summarize the whole report."
}

func reportHumanizeNoChangesResult(text string) bool {
	return strings.Contains(strings.TrimSpace(text), "NO_H5_CHANGES")
}

type reportHumanizePatchActivity struct {
	Started       bool
	ApplyCount    int
	FinalizeCount int
	LastError     string
}

func reportHumanizePatchToolActivity(ctx context.Context, service ReportHumanizeService, missionID string, toolSessionID string) (reportHumanizePatchActivity, error) {
	lister, ok := service.(reportHumanizeEventLister)
	if !ok {
		return reportHumanizePatchActivity{}, fmt.Errorf("%w: H5 MCP patch requires event listing", app.ErrInvalidInput)
	}
	events, err := lister.ListEvents(ctx, missionID)
	if err != nil {
		return reportHumanizePatchActivity{}, err
	}
	toolSessionID = strings.TrimSpace(toolSessionID)
	var activity reportHumanizePatchActivity
	for _, event := range events {
		if event.EventType != "mcp.tool.called" {
			continue
		}
		var payload struct {
			ToolName      string `json:"tool_name"`
			ToolSessionID string `json:"tool_session_id"`
			Success       bool   `json:"success"`
			Result        struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			} `json:"result"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return reportHumanizePatchActivity{}, fmt.Errorf("%w: invalid MCP tool call payload", app.ErrInvalidInput)
		}
		if strings.TrimSpace(payload.ToolSessionID) != toolSessionID {
			continue
		}
		switch payload.ToolName {
		case plasmamcp.ToolReportPatchStart:
			activity.Started = true
		case plasmamcp.ToolReportPatchApply:
			activity.ApplyCount++
		case plasmamcp.ToolReportPatchFinalize:
			activity.FinalizeCount++
		}
		if !payload.Success {
			if message := strings.TrimSpace(payload.Result.Error.Message); message != "" {
				activity.LastError = message
			}
		}
	}
	return activity, nil
}

func reportHumanizeFinalizedPatchEvent(ctx context.Context, service ReportHumanizeService, missionID string, pendingEventID string) (app.LedgerEvent, bool, error) {
	lister, ok := service.(reportHumanizeEventLister)
	if !ok {
		return app.LedgerEvent{}, false, fmt.Errorf("%w: H5 MCP patch requires event listing", app.ErrInvalidInput)
	}
	events, err := lister.ListEvents(ctx, missionID)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	pendingEventID = strings.TrimSpace(pendingEventID)
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		if event.EventType != "report.patch.finalized" {
			continue
		}
		var payload struct {
			PendingEventID string `json:"pending_event_id"`
			ArtifactID     string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.PendingEventID) == pendingEventID && strings.TrimSpace(payload.ArtifactID) != "" {
			return event, true, nil
		}
	}
	return app.LedgerEvent{}, false, nil
}

func reportHumanizeFinalizedArtifact(ctx context.Context, service ReportHumanizeService, missionID string, finalized app.LedgerEvent) (app.RawArtifact, error) {
	var payload struct {
		ArtifactID string `json:"artifact_id"`
	}
	if err := json.Unmarshal(finalized.Payload, &payload); err != nil {
		return app.RawArtifact{}, fmt.Errorf("%w: invalid H5 patch finalized payload", app.ErrInvalidInput)
	}
	artifactID := strings.TrimSpace(payload.ArtifactID)
	if artifactID == "" {
		return app.RawArtifact{}, fmt.Errorf("%w: H5 patch finalized payload is missing artifact_id", app.ErrInvalidInput)
	}
	artifact, err := service.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return app.RawArtifact{}, err
	}
	if artifact.MissionID != missionID {
		return app.RawArtifact{}, fmt.Errorf("%w: H5 patch artifact belongs to another mission", app.ErrInvalidInput)
	}
	if !isMarkdownMediaType(artifact.MediaType) {
		return app.RawArtifact{}, fmt.Errorf("%w: H5 patch artifact must be Markdown", app.ErrInvalidInput)
	}
	return artifact, nil
}

func agentReportHumanizePatchPrompt(title string, missionID string, toolSessionID string, pendingEventID string, baseArtifactID string, req reporting.PatchRequest) string {
	return fmt.Sprintf(`You are applying the approved Plasma H5 Korean report humanize pass through MCP report patch tools.

This is a post-report tone pass after Markdown report generation. It is not a planner, source selector, content model rewrite, AST redesign, or Designed HTML improvement.

Do not rewrite the full report in your response. Use the report patch MCP tools to inspect bounded chunks, apply small targeted edits, and finalize a new Markdown artifact.

Mission ID: %s
Base report artifact ID: %s
Report title: %s
Patch instruction: %s

Plasma tool binding:
- Use mission_id %s.
- Use session_id %s and producer {"type":"agent_session","id":"%s"} for all report patch tool calls.

Required MCP flow:
1. Call %s with base_artifact_id %s, title %s, and the patch instruction. Do not provide patch_id; use the patch_id returned by this call for later patch tool calls.
2. Use %s to inspect relevant ranges. Continue with next_offset when a chunk is truncated and more content is needed.
3. Use %s with small replace operations only. Do not use append, insert_after, or replace_all. Prefer exact targeted edits over broad rewrites.
4. Call %s exactly once after edits are complete.

Finalize metadata is server-bound Plasma lineage. Do not infer it from the report text, previous pending events, or tool responses. When the finalize schema asks for these fields, use these exact values:
- pending_event_id: %s
- agent_executor: %s
- agent_model: %s
- agent_reasoning_effort: %s
- mcp_mode: %s
- agent_session_id: %s
- previous_agent_session_id: %s
- returned_agent_session_id: %s
- report_session_id: %s
- fork_source_agent_session_id: %s
- report_session_policy: %s
- report_session_policy_selection: %s
- session_chain_kind: %s

Rules:
- Keep the same report register: clear, public-facing Korean report prose, not casual chat.
- Smooth stiff Korean phrasing and transitions where possible.
- Do not add, remove, merge, split, reorder, or summarize sections or paragraphs.
- Preserve heading levels and order exactly.
- Preserve tables, code fences, links, footnotes, source labels, citations, quotes, numbers, dates, model names, technical terms, and uncertainty/caveat wording.
- Do not introduce new claims, sources, evidence, recommendations, or caveats.
- If a sentence cannot be improved without risking fidelity, keep it unchanged.
- If the report is already natural enough or every possible change would risk fidelity, do not finalize. Return exactly: NO_H5_CHANGES
- After successful finalization, return only a short Korean summary and the artifact ID returned by the tool.
`, missionID, baseArtifactID, strconv.Quote(title), strconv.Quote(req.Instruction), missionID, toolSessionID, toolSessionID,
		plasmamcp.ToolReportPatchStart, baseArtifactID, strconv.Quote(req.Title),
		plasmamcp.ToolReportPatchRead,
		plasmamcp.ToolReportPatchApply,
		plasmamcp.ToolReportPatchFinalize,
		pendingEventID,
		req.AgentExecutor,
		req.AgentModel,
		req.AgentReasoningEffort,
		req.MCPMode,
		req.ReportSessionID,
		req.PreviousAgentSessionID,
		req.ReportSessionID,
		req.ReportSessionID,
		req.ForkSourceAgentSessionID,
		req.ReportSessionPolicy,
		req.ReportSessionPolicySelection,
		req.SessionChainKind)
}
