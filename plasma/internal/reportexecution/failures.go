package reportexecution

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func (runner Runner) AppendDraftFailed(ctx context.Context, missionID string, pendingEventID string, executor string, reportMode string, cause error) (ledger.Event, error) {
	executor = firstNonEmpty(strings.TrimSpace(executor), "plasma")
	mode, err := NormalizeMode(reportMode)
	if err != nil {
		mode = DefaultMode
	}
	payload := map[string]any{
		"kind":              "report_draft_failed",
		"pending_event_id":  pendingEventID,
		"agent_executor":    executor,
		"report_mode":       mode,
		"report_mode_label": ModeLabel(mode),
		"text":              "리포트 초안 생성에 실패했습니다.",
		"error":             cause.Error(),
		"failed_at":         time.Now().UTC().Format(time.RFC3339Nano),
	}
	mergeFailurePayload(payload, cause)
	terminalID := runner.id("evt")
	terminal := ledger.AppendRequest{EventID: terminalID,
		MissionID: missionID,
		EventType: "report.draft.failed",
		Producer:  ledger.Producer{Type: "agent", ID: executor},
		Payload:   mustJSON(payload)}
	var stage *StageFailureError
	if !errors.As(cause, &stage) {
		appended, ok, err := runner.Service.AppendReportTerminalIfOpen(ctx, missionID, pendingEventID, []ledger.AppendRequest{terminal})
		if err != nil || !ok {
			return ledger.Event{}, err
		}
		return appended[0], nil
	}
	if strings.TrimSpace(stage.EventID) == "" {
		stage.EventID = runner.id("evt")
	}
	payload["failed_stage_kind"] = stage.Kind
	payload["failed_stage_id"] = stage.ID()
	payload["part_index"] = stage.PartIndex
	payload["section_index"] = stage.SectionIndex
	payload["stage_failure_event_id"] = stage.EventID
	payload["safe_error_class"] = stage.ErrorClass
	payload["safe_error_message"] = stage.Message
	terminal.Payload = mustJSON(payload)
	stageReq := stage.AppendRequest(missionID, pendingEventID, terminalID, ledger.Producer{Type: "agent", ID: executor})
	appended, ok, err := runner.Service.AppendReportTerminalIfOpen(ctx, missionID, pendingEventID, []ledger.AppendRequest{stageReq, terminal})
	if err != nil || !ok {
		return ledger.Event{}, err
	}
	return appended[1], nil
}

// AppendPatchFailed는 patch 실패 terminal 이벤트를 기록한다.
func (runner Runner) AppendPatchFailed(ctx context.Context, missionID string, pendingEventID string, executor string, baseArtifactID string, cause error) (ledger.Event, error) {
	executor = validAgentExecutorOrEmpty(executor)
	producerID := firstNonEmpty(executor, "plasma")
	payload := map[string]any{
		"kind":             "report_patch_failed",
		"pending_event_id": pendingEventID,
		"base_artifact_id": baseArtifactID,
		"agent_executor":   executor,
		"text":             "MCP 리포트 패치에 실패했습니다.",
		"error":            cause.Error(),
		"failed_at":        time.Now().UTC().Format(time.RFC3339Nano),
	}
	mergeFailurePayload(payload, cause)
	appended, ok, err := runner.Service.AppendReportTerminalIfOpen(ctx, missionID, pendingEventID, []ledger.AppendRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.patch.failed",
		Producer:  ledger.Producer{Type: "agent", ID: producerID},
		Payload:   mustJSON(payload),
	}})
	if err != nil || !ok {
		return ledger.Event{}, err
	}
	return appended[0], nil
}

// AppendHumanizeFailed는 humanize 실패 terminal 이벤트를 기록한다.
func (runner Runner) AppendHumanizeFailed(ctx context.Context, missionID string, pendingEventID string, executor string, sourceArtifactID string, reportMode string, cause error) (ledger.Event, error) {
	executor = validAgentExecutorOrEmpty(executor)
	producerID := firstNonEmpty(executor, "plasma")
	mode, err := NormalizeMode(reportMode)
	if err != nil {
		mode = DefaultMode
	}
	payload := map[string]any{
		"kind":                        "humanized_markdown_report_failed",
		"pending_event_id":            pendingEventID,
		"source_artifact_id":          sourceArtifactID,
		"agent_executor":              executor,
		"target":                      "humanized_markdown",
		"profile":                     "h5-full-report-tone-pass",
		"humanize_transport":          "mcp_patch",
		"report_mode":                 mode,
		"report_mode_label":           ModeLabel(mode),
		"relationship":                "failed_post_report_tone_pass_of_source_artifact",
		"preserved_original_markdown": true,
		"text":                        "H5 말투 보정이 실패해 원본 Markdown artifact를 유지했습니다.",
		"error":                       cause.Error(),
		"failed_at":                   time.Now().UTC().Format(time.RFC3339Nano),
	}
	mergeFailurePayload(payload, cause)
	appended, ok, err := runner.Service.AppendReportTerminalIfOpen(ctx, missionID, pendingEventID, []ledger.AppendRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.humanize.failed",
		Producer:  ledger.Producer{Type: "agent", ID: producerID},
		Payload:   mustJSON(payload),
	}})
	if err != nil || !ok {
		return ledger.Event{}, err
	}
	return appended[0], nil
}

// AppendDesignFailed는 design 생성 실패 terminal 이벤트를 기록한다.
func (runner Runner) AppendDesignFailed(ctx context.Context, missionID string, pendingEventID string, executor string, sourceArtifactID string, rendererVersion string, cause error) (ledger.Event, error) {
	executor = firstNonEmpty(strings.TrimSpace(executor), "plasma")
	payload := map[string]any{
		"kind":               "designed_html_report_failed",
		"pending_event_id":   pendingEventID,
		"source_artifact_id": sourceArtifactID,
		"agent_executor":     executor,
		"target":             DesignTargetDesigned,
		"renderer_version":   rendererVersion,
		"text":               "Designed HTML 리포트 artifact 생성에 실패했습니다.",
		"error":              cause.Error(),
		"failed_at":          time.Now().UTC().Format(time.RFC3339Nano),
	}
	mergeFailurePayload(payload, cause)
	appended, ok, err := runner.Service.AppendReportTerminalIfOpen(ctx, missionID, pendingEventID, []ledger.AppendRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.design.failed",
		Producer:  ledger.Producer{Type: "agent", ID: executor},
		Payload:   mustJSON(payload),
	}})
	if err != nil || !ok {
		return ledger.Event{}, err
	}
	return appended[0], nil
}

func mergeFailurePayload(payload map[string]any, cause error) {
	var provider FailurePayloadProvider
	if !errors.As(cause, &provider) {
		return
	}
	for key, value := range provider.FailurePayload() {
		if !allowedFailurePayloadKey(key) || value == nil {
			continue
		}
		payload[key] = value
	}
}

func allowedFailurePayloadKey(key string) bool {
	switch key {
	case "agent_usage",
		"failed_surface",
		"agent_session_id",
		"previous_agent_session_id",
		"returned_agent_session_id",
		"tool_session_id",
		"resumed":
		return true
	default:
		return false
	}
}
