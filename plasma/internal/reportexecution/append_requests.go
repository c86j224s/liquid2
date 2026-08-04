package reportexecution

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func BuildSelfContainedHTMLExportAppendRequest(req SelfContainedHTMLExportEventRequest) ledger.AppendRequest {
	artifact := req.Artifact
	return ledger.AppendRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "report.artifact.exported",
		Producer:  req.Producer,
		Payload: mustJSON(map[string]any{
			"kind":               ExportKindSelfContainedHTML,
			"source_artifact_id": req.SourceArtifactID,
			"artifact_id":        artifact.ArtifactID,
			"media_type":         artifact.MediaType,
			"target":             ExportTargetSelfContainedHTML,
			"renderer_version":   req.RendererVersion,
			"text":               "Self-contained HTML 리포트 artifact를 생성했습니다.",
		}),
	}
}

// BuildDesignedHTMLExportAppendRequest는 보고서 생성 파이프라인에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildDesignedHTMLExportAppendRequest(req DesignedHTMLExportEventRequest) ledger.AppendRequest {
	artifact := req.Artifact
	payload := map[string]any{
		"kind":                      ExportKindDesignedHTML,
		"pending_event_id":          req.PendingEventID,
		"source_artifact_id":        req.SourceArtifactID,
		"content_model_artifact_id": req.ContentModelArtifactID,
		"artifact_id":               artifact.ArtifactID,
		"media_type":                artifact.MediaType,
		"target":                    ExportTargetDesignedHTML,
		"renderer_version":          req.RendererVersion,
		"content_model_contract":    DesignedContentModelContract,
		"image_set_fingerprint":     req.ImageSetFingerprint,
		"agent_executor":            req.AgentExecutor,
		"agent_model":               req.AgentModel,
		"agent_reasoning_effort":    req.AgentReasoningEffort,
		"agent_session_id":          req.AgentSessionID,
		"tool_session_id":           req.ToolSessionID,
		"duration_ms":               req.DurationMS,
		"text":                      "Designed HTML 리포트 artifact를 생성했습니다.",
	}
	if eventUsage, ok := req.AgentUsage.ForEvent("report_design", req.AgentDurationMS, "", req.AgentSessionID, req.AgentResumed, false); ok {
		payload["agent_usage"] = eventUsage
	}
	return ledger.AppendRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "report.artifact.exported",
		Producer:  req.Producer,
		Payload:   mustJSON(payload),
	}
}

// BuildPatchFinalizedAppendRequest는 보고서 생성 파이프라인에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildPatchFinalizedAppendRequest(req PatchFinalizedEventRequest) ledger.AppendRequest {
	artifact := req.Artifact
	return ledger.AppendRequest{
		EventID:       strings.TrimSpace(req.EventID),
		MissionID:     strings.TrimSpace(req.MissionID),
		EventType:     "report.patch.finalized",
		Producer:      req.Producer,
		CorrelationID: strings.TrimSpace(req.CorrelationID),
		Payload: mustJSON(map[string]any{
			"kind":                            "markdown_report_patch_finalized",
			"pending_event_id":                req.PendingEventID,
			"title":                           req.Title,
			"artifact_id":                     artifact.ArtifactID,
			"media_type":                      artifact.MediaType,
			"byte_size":                       artifact.ByteSize,
			"sha256":                          artifact.SHA256,
			"filename":                        artifact.Filename,
			"base_artifact_id":                req.BaseArtifactID,
			"base_report_artifact_id":         req.BaseArtifactID,
			"patch_id":                        req.PatchID,
			"patch_instruction":               req.PatchInstruction,
			"patch_summary":                   req.PatchSummary,
			"operation_count":                 req.OperationCount,
			"operations":                      req.Operations,
			"agent_executor":                  req.AgentExecutor,
			"agent_model":                     req.AgentModel,
			"agent_reasoning_effort":          req.AgentReasoningEffort,
			"agent_session_id":                req.AgentSessionID,
			"previous_agent_session_id":       req.PreviousAgentSessionID,
			"returned_agent_session_id":       req.ReturnedAgentSessionID,
			"report_session_id":               req.ReportSessionID,
			"fork_source_agent_session_id":    req.ForkSourceAgentSessionID,
			"report_session_policy":           req.ReportSessionPolicy,
			"report_session_policy_selection": req.ReportSessionPolicySelection,
			"tool_session_id":                 req.ToolSessionID,
			"mcp_mode":                        req.MCPMode,
			"producer_tool_name":              req.ProducerToolName,
			"composition_strategy":            "mcp_patch_markdown",
			"session_chain_kind":              req.SessionChainKind,
			"post_report_research_session_id": "",
			"text":                            "MCP 패치 방식으로 Markdown 리포트 artifact 새 버전을 생성했습니다.",
		}),
	}
}
