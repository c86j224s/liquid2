package reporthumanize

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// AppendStaleFailed closes an abandoned H5 pending event after restart while
// preserving the original report artifact.
func AppendStaleFailed(ctx context.Context, service Service, idFunc IDFunc, missionID string, pending ledger.Event) (ledger.Event, error) {
	payload := PendingPayloadFromEvent(pending)
	executor := firstNonEmpty(strings.TrimSpace(payload.AgentExecutor), "plasma")
	event, _, err := appendTerminal(ctx, service, missionID, pending.EventID, reporting.BuildHumanizeFailedAppendRequest(reporting.HumanizeFailedEventRequest{
		HumanizeEventBase: reporting.HumanizeEventBase{
			EventID:                idFunc("evt"),
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
			Target:                 firstNonEmpty(strings.TrimSpace(payload.Target), reportexecution.ExportTargetHumanizedMarkdown),
			Profile:                firstNonEmpty(strings.TrimSpace(payload.Profile), reportexecution.HumanizeProfileH5),
			HumanizeTransport:      firstNonEmpty(strings.TrimSpace(payload.HumanizeTransport), reportexecution.HumanizeTransportPatch),
			Producer:               ledger.Producer{Type: "agent", ID: executor},
		},
		Kind:         "humanized_markdown_report_stale_failed",
		Text:         "H5 말투 보정 작업이 중단된 상태로 남아 원본 Markdown artifact를 유지했습니다.",
		Error:        "stale humanized Markdown report generation was not running after restart",
		Relationship: "stale_post_report_tone_pass_of_source_artifact",
		OmitDuration: true,
		FailedAt:     time.Now().UTC().Format(time.RFC3339Nano),
	}))
	return event, err
}

// RecoverFinalizedPatch validates and promotes a finalized H5 patch that
// completed before restart but whose H5 pending event was not terminally closed.
func RecoverFinalizedPatch(ctx context.Context, service Service, idFunc IDFunc, missionID string, pending ledger.Event) (bool, error) {
	if terminalExists(ctx, service, missionID, pending.EventID) {
		return true, nil
	}
	finalized, ok, err := finalizedPatchEvent(ctx, service, missionID, pending.EventID)
	if err != nil || !ok {
		return ok, err
	}
	payload := PendingPayloadFromEvent(pending)
	toolSessionID := strings.TrimSpace(payload.ToolSessionID)
	patchArtifact, artifactErr := finalizedArtifact(ctx, service, missionID, finalized)
	sourceArtifact, sourceErr := service.GetRawArtifact(ctx, strings.TrimSpace(payload.SourceArtifactID))
	input := InputFromPendingPayload(payload, sourceArtifact)
	if artifactErr != nil {
		_, err := appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, pending.EventID, 0, fmt.Errorf("recover finalized H5 patch artifact: %w", artifactErr))
		return true, err
	}
	if sourceErr != nil {
		if _, err := appendRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, pending.EventID, patchArtifact, finalized, "missing_source_artifact_for_recovery"); err != nil {
			return true, err
		}
		_, err := appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, pending.EventID, 0, fmt.Errorf("recover finalized H5 patch source artifact: %w", sourceErr))
		return true, err
	}
	return recoverPatchArtifact(ctx, service, idFunc, missionID, input, toolSessionID, pending.EventID, patchArtifact, finalized, sourceArtifact)
}

// InputFromPendingPayload reconstructs the H5 input used by recovery from the
// durable pending event payload and, when available, the original source artifact.
func InputFromPendingPayload(payload PendingPayload, sourceArtifact artifact.Raw) Input {
	if strings.TrimSpace(sourceArtifact.ArtifactID) == "" {
		sourceArtifact.ArtifactID = strings.TrimSpace(payload.SourceArtifactID)
	}
	if strings.TrimSpace(sourceArtifact.SHA256) == "" {
		sourceArtifact.SHA256 = strings.TrimSpace(payload.SourceArtifactSHA256)
	}
	return Input{
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

func recoverPatchArtifact(ctx context.Context, service Service, idFunc IDFunc, missionID string, input Input, toolSessionID string, humanizePendingEventID string, patchArtifact artifact.Raw, finalized ledger.Event, sourceArtifact artifact.Raw) (bool, error) {
	original := strings.TrimSpace(string(sourceArtifact.Content))
	humanized := strings.TrimSpace(string(patchArtifact.Content))
	if humanized == "" {
		if _, err := appendRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, patchArtifact, finalized, "empty_humanized_markdown_recovered_after_restart"); err != nil {
			return true, err
		}
		_, err := appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, 0, fmt.Errorf("%w: recovered H5 patch artifact is empty", producterror.ErrInvalidInput))
		return true, err
	}
	if err := ValidateMarkdown(original, humanized); err != nil {
		if _, rejectErr := appendRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, patchArtifact, finalized, "validation_failed_recovered_after_restart"); rejectErr != nil {
			return true, rejectErr
		}
		_, failErr := appendFailed(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, 0, err)
		return true, failErr
	}
	if humanized == original {
		if _, err := appendRejectedPatch(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, patchArtifact, finalized, "unchanged_humanized_markdown_recovered_after_restart"); err != nil {
			return true, err
		}
		_, err := appendSkipped(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, 0)
		return true, err
	}
	if _, err := appendRecoveredExport(ctx, service, idFunc, missionID, input, toolSessionID, humanizePendingEventID, patchArtifact, finalized, original, humanized); err != nil {
		return true, err
	}
	return true, nil
}

func appendRecoveredExport(ctx context.Context, service Service, idFunc IDFunc, missionID string, input Input, toolSessionID string, humanizePendingEventID string, artifact artifact.Raw, finalized ledger.Event, original string, humanized string) (ledger.Event, error) {
	ledgerCtx, cancel := terminalWriteContext(ctx)
	defer cancel()
	producerID := firstNonEmpty(strings.TrimSpace(input.PreviousSessionID), strings.TrimSpace(toolSessionID), strings.TrimSpace(finalized.CorrelationID))
	event, _, err := appendTerminal(ledgerCtx, service, missionID, humanizePendingEventID, reporting.BuildHumanizedMarkdownExportAppendRequest(reporting.HumanizedMarkdownExportEventRequest{
		HumanizeEventBase:      eventBase(idFunc("evt"), missionID, input, toolSessionID, humanizePendingEventID, ledger.Producer{Type: "agent_session", ID: producerID}),
		PatchEventID:           finalized.EventID,
		Artifact:               artifact,
		AgentSessionID:         strings.TrimSpace(input.PreviousSessionID),
		ReturnedAgentSessionID: strings.TrimSpace(input.PreviousSessionID),
		SourceWordCount:        reportWordCount(original),
		HumanizedWordCount:     reportWordCount(humanized),
		RecoveredAfterRestart:  true,
		Text:                   "서버 재시작 전에 완료된 H5 말투 보정 Markdown artifact를 검증해 복구했습니다.",
	}))
	return event, err
}
