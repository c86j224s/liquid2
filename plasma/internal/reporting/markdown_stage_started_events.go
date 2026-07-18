package reporting

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type MarkdownReportSectionStartedEventRequest struct {
	MarkdownReportStageEventBase
	PartIndex    int
	SectionIndex int
}

func BuildMarkdownReportSectionStartedAppendRequest(req MarkdownReportSectionStartedEventRequest) app.AppendEventRequest {
	base := req.MarkdownReportStageEventBase
	payload := markdownReportStageStartedPayload(base)
	payload["kind"] = "sectional_markdown_report_section_started"
	payload["part_index"] = req.PartIndex
	payload["section_index"] = req.SectionIndex
	payload["text"] = firstNonEmpty(base.Text, "장문 리포트 섹션 Markdown 생성을 시작했습니다.")
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(base.EventID),
		MissionID: strings.TrimSpace(base.MissionID),
		EventType: "report.section.started",
		Producer:  base.Producer,
		Payload:   mustJSON(payload),
	}
}

func markdownReportStageStartedPayload(req MarkdownReportStageEventBase) map[string]any {
	return map[string]any{
		"pending_event_id":                req.PendingEventID,
		"plan_event_id":                   req.PlanEventID,
		"title":                           req.Title,
		"agent_executor":                  req.AgentExecutor,
		"agent_model":                     req.AgentModel,
		"agent_reasoning_effort":          req.AgentReasoningEffort,
		"agent_session_id":                req.AgentSessionID,
		"previous_agent_session_id":       req.PreviousAgentSessionID,
		"tool_session_id":                 req.ToolSessionID,
		"report_mode":                     req.ReportMode,
		"report_mode_label":               firstNonEmpty(req.ReportModeLabel, ModeLabel(req.ReportMode)),
		"report_session_policy":           req.ReportSessionPolicy,
		"report_session_policy_selection": req.ReportSessionPolicySelection,
		"post_report_humanize":            req.PostReportHumanize,
		"humanize_enabled":                req.HumanizeEnabled,
		"generation_guidance_profile":     req.GenerationGuidanceProfile,
		"generation_guidance_sha256":      req.GenerationGuidanceSHA256,
		"session_chain_kind":              req.SessionChainKind,
		"pre_report_research_session_id":  req.PreReportResearchSessionID,
		"report_plan_session_id":          req.ReportPlanSessionID,
		"report_session_id":               req.ReportSessionID,
		"fork_source_agent_session_id":    req.ForkSourceAgentSessionID,
		"composition_strategy":            req.CompositionStrategy,
		"assembly_strategy":               req.AssemblyStrategy,
	}
}
