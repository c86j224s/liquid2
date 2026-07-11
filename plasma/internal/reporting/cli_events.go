package reporting

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type CLIMarkdownReportPlanCreatedEventRequest struct {
	EventID                      string
	MissionID                    string
	PendingEventID               string
	Title                        string
	AgentExecutor                string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ToolSessionID                string
	MCPMode                      string
	ReportMode                   string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	PostReportHumanize           string
	HumanizeEnabled              bool
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	SessionChainKind             string
	PreReportResearchSessionID   string
	ReportPlanSessionID          string
	ForkSourceAgentSessionID     string
	CompositionStrategy          string
	PlanText                     string
	Producer                     app.Producer
}

type CLIMarkdownReportArtifactCreatedEventRequest struct {
	EventID                      string
	MissionID                    string
	PendingEventID               string
	Title                        string
	Artifact                     app.RawArtifact
	AgentExecutor                string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ToolSessionID                string
	MCPMode                      string
	ReportMode                   string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	PostReportHumanize           string
	HumanizeEnabled              bool
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	SessionChainKind             string
	PreReportResearchSessionID   string
	ReportPlanSessionID          string
	ReportSessionID              string
	ForkSourceAgentSessionID     string
	CompositionStrategy          string
	PlanEventID                  string
	PlanToolSessionID            string
	DurationMS                   int64
	Producer                     app.Producer
}

func BuildCLIMarkdownReportPlanCreatedAppendRequest(req CLIMarkdownReportPlanCreatedEventRequest) app.AppendEventRequest {
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: "report.plan.created",
		Producer:  req.Producer,
		Payload: mustJSON(map[string]any{
			"kind":                            "markdown_report_plan",
			"pending_event_id":                strings.TrimSpace(req.PendingEventID),
			"title":                           strings.TrimSpace(req.Title),
			"agent_executor":                  strings.TrimSpace(req.AgentExecutor),
			"agent_session_id":                strings.TrimSpace(req.AgentSessionID),
			"previous_agent_session_id":       strings.TrimSpace(req.PreviousAgentSessionID),
			"tool_session_id":                 strings.TrimSpace(req.ToolSessionID),
			"mcp_mode":                        strings.TrimSpace(req.MCPMode),
			"report_mode":                     strings.TrimSpace(req.ReportMode),
			"report_mode_label":               ModeLabel(req.ReportMode),
			"report_session_policy":           strings.TrimSpace(req.ReportSessionPolicy),
			"report_session_policy_selection": strings.TrimSpace(req.ReportSessionPolicySelection),
			"post_report_humanize":            strings.TrimSpace(req.PostReportHumanize),
			"humanize_enabled":                req.HumanizeEnabled,
			"generation_guidance_profile":     strings.TrimSpace(req.GenerationGuidanceProfile),
			"generation_guidance_sha256":      strings.TrimSpace(req.GenerationGuidanceSHA256),
			"session_chain_kind":              strings.TrimSpace(req.SessionChainKind),
			"pre_report_research_session_id":  strings.TrimSpace(req.PreReportResearchSessionID),
			"report_plan_session_id":          strings.TrimSpace(req.ReportPlanSessionID),
			"report_session_id":               "",
			"fork_source_agent_session_id":    strings.TrimSpace(req.ForkSourceAgentSessionID),
			"post_report_research_session_id": "",
			"composition_strategy":            strings.TrimSpace(req.CompositionStrategy),
			"plan_review_required":            false,
			"plan_review_state":               "auto_accepted",
			"plan_text":                       strings.TrimSpace(req.PlanText),
			"text":                            "CLI Markdown 리포트 생성 계획을 만들었습니다.",
		}),
	}
}

func BuildCLIMarkdownReportArtifactCreatedAppendRequest(req CLIMarkdownReportArtifactCreatedEventRequest) app.AppendEventRequest {
	artifact := req.Artifact
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: "report.artifact.created",
		Producer:  req.Producer,
		Payload: mustJSON(map[string]any{
			"kind":                            "markdown_report_artifact",
			"pending_event_id":                strings.TrimSpace(req.PendingEventID),
			"title":                           strings.TrimSpace(req.Title),
			"artifact_id":                     artifact.ArtifactID,
			"media_type":                      artifact.MediaType,
			"agent_executor":                  strings.TrimSpace(req.AgentExecutor),
			"agent_session_id":                strings.TrimSpace(req.AgentSessionID),
			"previous_agent_session_id":       strings.TrimSpace(req.PreviousAgentSessionID),
			"tool_session_id":                 strings.TrimSpace(req.ToolSessionID),
			"mcp_mode":                        strings.TrimSpace(req.MCPMode),
			"report_mode":                     strings.TrimSpace(req.ReportMode),
			"report_mode_label":               ModeLabel(req.ReportMode),
			"report_session_policy":           strings.TrimSpace(req.ReportSessionPolicy),
			"report_session_policy_selection": strings.TrimSpace(req.ReportSessionPolicySelection),
			"post_report_humanize":            strings.TrimSpace(req.PostReportHumanize),
			"humanize_enabled":                req.HumanizeEnabled,
			"generation_guidance_profile":     strings.TrimSpace(req.GenerationGuidanceProfile),
			"generation_guidance_sha256":      strings.TrimSpace(req.GenerationGuidanceSHA256),
			"session_chain_kind":              strings.TrimSpace(req.SessionChainKind),
			"pre_report_research_session_id":  strings.TrimSpace(req.PreReportResearchSessionID),
			"report_plan_session_id":          strings.TrimSpace(req.ReportPlanSessionID),
			"report_session_id":               strings.TrimSpace(req.ReportSessionID),
			"fork_source_agent_session_id":    strings.TrimSpace(req.ForkSourceAgentSessionID),
			"post_report_research_session_id": "",
			"composition_strategy":            strings.TrimSpace(req.CompositionStrategy),
			"plan_event_id":                   strings.TrimSpace(req.PlanEventID),
			"plan_tool_session_id":            strings.TrimSpace(req.PlanToolSessionID),
			"duration_ms":                     req.DurationMS,
			"text":                            "Markdown 리포트 artifact를 생성했습니다.",
		}),
	}
}
