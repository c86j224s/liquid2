package reporting

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// HumanizeEventBase는 H5 보정 이벤트들이 공유하는 artifact와 agent 실행 metadata다.
type HumanizeEventBase struct {
	EventID                string
	MissionID              string
	PendingEventID         string
	ReportPendingEventID   string
	Title                  string
	SourceArtifactID       string
	SourceArtifactSHA256   string
	AgentExecutor          string
	AgentModel             string
	AgentReasoningEffort   string
	PreviousAgentSessionID string
	ToolSessionID          string
	MCPMode                string
	ReportMode             string
	ReportModeLabel        string
	Target                 string
	Profile                string
	HumanizeTransport      string
	Producer               app.Producer
}

// HumanizePendingEventRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type HumanizePendingEventRequest struct {
	HumanizeEventBase
}

// HumanizeSkippedEventRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type HumanizeSkippedEventRequest struct {
	HumanizeEventBase
	DurationMS int64
}

// HumanizeFailedEventRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type HumanizeFailedEventRequest struct {
	HumanizeEventBase
	Kind         string
	Error        string
	Text         string
	Relationship string
	DurationMS   int64
	OmitDuration bool
	FailedAt     string
}

// HumanizePatchRejectedEventRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type HumanizePatchRejectedEventRequest struct {
	HumanizeEventBase
	PatchEventID string
	Artifact     app.RawArtifact
	Reason       string
}

// HumanizedMarkdownExportEventRequest는 보고서 생성 파이프라인에 전달되는 요청 값이다.
type HumanizedMarkdownExportEventRequest struct {
	HumanizeEventBase
	PatchEventID           string
	Artifact               app.RawArtifact
	AgentSessionID         string
	ReturnedAgentSessionID string
	SourceWordCount        int
	HumanizedWordCount     int
	DurationMS             int64
	AgentUsage             agentusage.AgentUsage
	AgentResumed           bool
	RecoveredAfterRestart  bool
	Text                   string
}

// BuildHumanizePendingAppendRequest는 보고서 생성 파이프라인에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildHumanizePendingAppendRequest(req HumanizePendingEventRequest) app.AppendEventRequest {
	base := req.HumanizeEventBase
	pendingEventID := strings.TrimSpace(base.EventID)
	payload := humanizeBasePayload(base, pendingEventID)
	payload["kind"] = "humanized_markdown_report_pending"
	payload["text"] = "H5 말투 보정 Markdown artifact를 생성하는 중입니다."
	payload["relationship"] = "pending_post_report_tone_pass_of_source_artifact"
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(base.EventID),
		MissionID: strings.TrimSpace(base.MissionID),
		EventType: "report.humanize.pending",
		Producer:  base.Producer,
		Payload:   mustJSON(payload),
	}
}

// BuildHumanizeSkippedAppendRequest는 보고서 생성 파이프라인에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildHumanizeSkippedAppendRequest(req HumanizeSkippedEventRequest) app.AppendEventRequest {
	payload := humanizeBasePayload(req.HumanizeEventBase, req.PendingEventID)
	payload["kind"] = "humanized_markdown_report_skipped"
	payload["duration_ms"] = req.DurationMS
	payload["text"] = "H5 말투 보정 결과가 원본과 같아 별도 artifact를 만들지 않았습니다."
	payload["relationship"] = "no_change_post_report_tone_pass_of_source_artifact"
	payload["preserved_original_markdown"] = true
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: "report.humanize.skipped",
		Producer:  req.Producer,
		Payload:   mustJSON(payload),
	}
}

// BuildHumanizeFailedAppendRequest는 보고서 생성 파이프라인에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildHumanizeFailedAppendRequest(req HumanizeFailedEventRequest) app.AppendEventRequest {
	payload := humanizeBasePayload(req.HumanizeEventBase, req.PendingEventID)
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		kind = "humanized_markdown_report_failed"
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		text = "H5 말투 보정이 실패해 원본 Markdown artifact를 유지했습니다."
	}
	payload["kind"] = kind
	if !req.OmitDuration {
		payload["duration_ms"] = req.DurationMS
	}
	payload["error"] = req.Error
	payload["text"] = text
	payload["relationship"] = firstNonEmpty(req.Relationship, "failed_post_report_tone_pass_of_source_artifact")
	payload["preserved_original_markdown"] = true
	if failedAt := strings.TrimSpace(req.FailedAt); failedAt != "" {
		payload["failed_at"] = failedAt
	}
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: "report.humanize.failed",
		Producer:  req.Producer,
		Payload:   mustJSON(payload),
	}
}

// BuildHumanizePatchRejectedAppendRequest는 보고서 생성 파이프라인에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildHumanizePatchRejectedAppendRequest(req HumanizePatchRejectedEventRequest) app.AppendEventRequest {
	payload := humanizeBasePayload(req.HumanizeEventBase, req.PendingEventID)
	payload["kind"] = "markdown_report_patch_rejected"
	payload["patch_event_id"] = req.PatchEventID
	payload["artifact_id"] = req.Artifact.ArtifactID
	payload["media_type"] = req.Artifact.MediaType
	payload["reason"] = req.Reason
	payload["text"] = "H5 말투 보정 패치 artifact가 검증을 통과하지 못해 기본 연구 조회면에서 제외되었습니다."
	payload["relationship"] = "rejected_post_report_tone_pass_patch_artifact"
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: "report.patch.rejected",
		Producer:  req.Producer,
		Payload:   mustJSON(payload),
	}
}

// BuildHumanizedMarkdownExportAppendRequest는 보고서 생성 파이프라인에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildHumanizedMarkdownExportAppendRequest(req HumanizedMarkdownExportEventRequest) app.AppendEventRequest {
	payload := humanizeBasePayload(req.HumanizeEventBase, req.PendingEventID)
	agentSessionID := req.AgentSessionID
	payload["kind"] = ExportKindHumanizedMarkdown
	payload["patch_event_id"] = req.PatchEventID
	payload["artifact_id"] = req.Artifact.ArtifactID
	payload["media_type"] = req.Artifact.MediaType
	payload["agent_session_id"] = agentSessionID
	payload["returned_agent_session_id"] = req.ReturnedAgentSessionID
	payload["source_word_count"] = req.SourceWordCount
	payload["humanized_word_count"] = req.HumanizedWordCount
	payload["duration_ms"] = req.DurationMS
	text := strings.TrimSpace(req.Text)
	if text == "" {
		text = "H5 말투 보정 Markdown artifact를 생성했습니다."
	}
	payload["text"] = text
	payload["relationship"] = "post_report_tone_pass_of_source_artifact"
	payload["preserved_original_markdown"] = true
	if req.RecoveredAfterRestart {
		payload["recovered_after_restart"] = true
	}
	if eventUsage, ok := req.AgentUsage.ForEvent("report_humanize_h5", req.DurationMS, req.PreviousAgentSessionID, agentSessionID, req.AgentResumed, false); ok {
		payload["agent_usage"] = eventUsage
	}
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: "report.artifact.exported",
		Producer:  req.Producer,
		Payload:   mustJSON(payload),
	}
}

func humanizeBasePayload(req HumanizeEventBase, pendingEventID string) map[string]any {
	return map[string]any{
		"target":                    firstNonEmpty(req.Target, ExportTargetHumanizedMarkdown),
		"profile":                   firstNonEmpty(req.Profile, HumanizeProfileH5),
		"pending_event_id":          pendingEventID,
		"report_pending_event_id":   req.ReportPendingEventID,
		"title":                     req.Title,
		"source_artifact_id":        req.SourceArtifactID,
		"source_artifact_sha256":    req.SourceArtifactSHA256,
		"agent_executor":            req.AgentExecutor,
		"agent_model":               req.AgentModel,
		"agent_reasoning_effort":    req.AgentReasoningEffort,
		"previous_agent_session_id": req.PreviousAgentSessionID,
		"tool_session_id":           req.ToolSessionID,
		"mcp_mode":                  req.MCPMode,
		"report_mode":               req.ReportMode,
		"report_mode_label":         firstNonEmpty(req.ReportModeLabel, ModeLabel(req.ReportMode)),
		"humanize_transport":        firstNonEmpty(req.HumanizeTransport, HumanizeTransportPatch),
	}
}
