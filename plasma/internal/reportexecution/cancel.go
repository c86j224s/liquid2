package reportexecution

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func (runner Runner) AppendCanceled(ctx context.Context, missionID string, pending ledger.Event, canceledInFlight bool, producer ledger.Producer) (ledger.Event, error) {
	switch pending.EventType {
	case "report.design.pending":
		return runner.AppendDesignCanceled(ctx, missionID, pending, canceledInFlight, producer)
	case "report.humanize.pending":
		return runner.AppendHumanizeCanceled(ctx, missionID, pending, canceledInFlight, producer)
	case "report.patch.pending":
		return runner.AppendPatchCanceled(ctx, missionID, pending, canceledInFlight, producer)
	default:
		return runner.AppendDraftCanceled(ctx, missionID, pending, canceledInFlight, producer)
	}
}

// AppendDraftCanceled는 draft 생성 취소 terminal 이벤트를 기록한다.
func (runner Runner) AppendDraftCanceled(ctx context.Context, missionID string, pending ledger.Event, canceledInFlight bool, producer ledger.Producer) (ledger.Event, error) {
	var payload struct {
		AgentExecutor string `json:"agent_executor"`
		ReportMode    string `json:"report_mode"`
	}
	_ = json.Unmarshal(pending.Payload, &payload)
	executor := firstNonEmpty(payload.AgentExecutor, "plasma")
	mode, err := NormalizeMode(payload.ReportMode)
	if err != nil {
		mode = DefaultMode
	}
	req := ledger.AppendRequest{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.draft.failed",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":              "report_draft_canceled",
			"pending_event_id":  pending.EventID,
			"agent_executor":    executor,
			"report_mode":       mode,
			"report_mode_label": ModeLabel(mode),
			"text":              "리포트 초안 생성이 취소되었습니다.",
			"error":             "report draft canceled by user",
			"canceled":          true,
			"in_flight":         canceledInFlight,
			"canceled_at":       time.Now().UTC().Format(time.RFC3339Nano),
		}),
	}
	appended, ok, err := runner.Service.AppendReportTerminalIfOpen(ctx, missionID, pending.EventID, []ledger.AppendRequest{req})
	if err != nil || !ok {
		return ledger.Event{}, err
	}
	return appended[0], nil
}

// AppendPatchCanceled는 patch 취소 terminal 이벤트를 기록한다.
func (runner Runner) AppendPatchCanceled(ctx context.Context, missionID string, pending ledger.Event, canceledInFlight bool, producer ledger.Producer) (ledger.Event, error) {
	req, err := PatchRequestFromPendingEvent(pending)
	if err != nil {
		req.AgentExecutor = "plasma"
	}
	executor := validAgentExecutorOrEmpty(req.AgentExecutor)
	appended, ok, err := runner.Service.AppendReportTerminalIfOpen(ctx, missionID, pending.EventID, []ledger.AppendRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.patch.failed",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":             "report_patch_canceled",
			"pending_event_id": pending.EventID,
			"base_artifact_id": req.BaseArtifactID,
			"agent_executor":   executor,
			"text":             "MCP 리포트 패치가 취소되었습니다.",
			"error":            "report patch canceled by user",
			"canceled":         true,
			"in_flight":        canceledInFlight,
			"canceled_at":      time.Now().UTC().Format(time.RFC3339Nano),
		}),
	}})
	if err != nil || !ok {
		return ledger.Event{}, err
	}
	return appended[0], nil
}

// AppendDesignCanceled는 design 생성 취소 terminal 이벤트를 기록한다.
func (runner Runner) AppendDesignCanceled(ctx context.Context, missionID string, pending ledger.Event, canceledInFlight bool, producer ledger.Producer) (ledger.Event, error) {
	var payload struct {
		SourceArtifactID string `json:"source_artifact_id"`
		AgentExecutor    string `json:"agent_executor"`
		RendererVersion  string `json:"renderer_version"`
	}
	_ = json.Unmarshal(pending.Payload, &payload)
	executor := firstNonEmpty(payload.AgentExecutor, "plasma")
	appended, ok, err := runner.Service.AppendReportTerminalIfOpen(ctx, missionID, pending.EventID, []ledger.AppendRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.design.failed",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":               "designed_html_report_canceled",
			"pending_event_id":   pending.EventID,
			"source_artifact_id": payload.SourceArtifactID,
			"agent_executor":     executor,
			"target":             DesignTargetDesigned,
			"renderer_version":   payload.RendererVersion,
			"text":               "Designed HTML 리포트 artifact 생성이 취소되었습니다.",
			"error":              "designed HTML report generation canceled by user",
			"canceled":           true,
			"in_flight":          canceledInFlight,
			"canceled_at":        time.Now().UTC().Format(time.RFC3339Nano),
		}),
	}})
	if err != nil || !ok {
		return ledger.Event{}, err
	}
	return appended[0], nil
}

// AppendHumanizeCanceled는 humanize 취소 terminal 이벤트를 기록한다.
func (runner Runner) AppendHumanizeCanceled(ctx context.Context, missionID string, pending ledger.Event, canceledInFlight bool, producer ledger.Producer) (ledger.Event, error) {
	payload := humanizePendingPayloadFromEvent(pending)
	executor := firstNonEmpty(payload.AgentExecutor, "plasma")
	appended, ok, err := runner.Service.AppendReportTerminalIfOpen(ctx, missionID, pending.EventID, []ledger.AppendRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.humanize.failed",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":                        "humanized_markdown_report_canceled",
			"target":                      firstNonEmpty(payload.Target, ExportTargetHumanizedMarkdown),
			"profile":                     firstNonEmpty(payload.Profile, HumanizeProfileH5),
			"pending_event_id":            pending.EventID,
			"report_pending_event_id":     strings.TrimSpace(payload.ReportPendingEventID),
			"title":                       strings.TrimSpace(payload.Title),
			"source_artifact_id":          strings.TrimSpace(payload.SourceArtifactID),
			"source_artifact_sha256":      strings.TrimSpace(payload.SourceArtifactSHA256),
			"agent_executor":              executor,
			"agent_model":                 strings.TrimSpace(payload.AgentModel),
			"agent_reasoning_effort":      strings.TrimSpace(payload.AgentReasoningEffort),
			"previous_agent_session_id":   strings.TrimSpace(payload.PreviousSessionID),
			"tool_session_id":             strings.TrimSpace(payload.ToolSessionID),
			"mcp_mode":                    strings.TrimSpace(payload.MCPMode),
			"report_mode":                 strings.TrimSpace(payload.ReportMode),
			"report_mode_label":           strings.TrimSpace(payload.ReportModeLabel),
			"humanize_transport":          firstNonEmpty(payload.HumanizeTransport, HumanizeTransportPatch),
			"text":                        "H5 말투 보정이 취소되어 원본 Markdown artifact를 유지했습니다.",
			"error":                       "humanized Markdown report generation canceled by user",
			"canceled":                    true,
			"in_flight":                   canceledInFlight,
			"canceled_at":                 time.Now().UTC().Format(time.RFC3339Nano),
			"relationship":                "canceled_post_report_tone_pass_of_source_artifact",
			"preserved_original_markdown": true,
		}),
	}})
	if err != nil || !ok {
		return ledger.Event{}, err
	}
	return appended[0], nil
}
