package reporting

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type MarkdownReportEventBase struct {
	EventID                      string
	MissionID                    string
	PendingEventID               string
	Title                        string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ReturnedAgentSessionID       string
	ToolSessionID                string
	MCPMode                      string
	RigorLevel                   string
	RigorLabel                   string
	ReportMode                   string
	ReportModeLabel              string
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
	PostReportResearchSessionID  string
	CompositionStrategy          string
	DurationMS                   int64
	Text                         string
	AgentUsage                   agentusage.AgentUsage
	AgentUsageSurface            string
	AgentUsageDurationMS         int64
	AgentResumed                 bool
	Producer                     app.Producer
}

type MarkdownReportPlanCreatedEventRequest struct {
	MarkdownReportEventBase
	ArtifactID         string
	Plan               any
	AssemblyStrategy   string
	PlanReviewRequired bool
	PlanReviewState    string
}

type MarkdownReportArtifactCreatedEventRequest struct {
	MarkdownReportEventBase
	Artifact              app.RawArtifact
	PlanEventID           string
	PlanToolSessionID     string
	IncludePlanReview     bool
	PlanReviewRequired    bool
	PlanReviewState       string
	AssemblyStrategy      string
	SectionCount          int
	PartCount             int
	SectionArtifactIDs    []string
	PartArtifactIDs       []string
	SectionWordCount      int
	FinalWordCount        int
	PreservationRatio     float64
	IncludeLongFormFields bool
}

type MarkdownReportStageEventBase struct {
	EventID                      string
	MissionID                    string
	PendingEventID               string
	PlanEventID                  string
	Title                        string
	Artifact                     app.RawArtifact
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ReturnedAgentSessionID       string
	ToolSessionID                string
	ReportMode                   string
	ReportModeLabel              string
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
	PostReportResearchSessionID  string
	CompositionStrategy          string
	AssemblyStrategy             string
	DurationMS                   int64
	Text                         string
	AgentUsage                   agentusage.AgentUsage
	AgentUsageSurface            string
	AgentUsageDurationMS         int64
	AgentResumed                 bool
	Producer                     app.Producer
}

type MarkdownReportSectionCreatedEventRequest struct {
	MarkdownReportStageEventBase
	PartIndex    int
	SectionIndex int
	WordCount    int
}

type MarkdownReportPartCreatedEventRequest struct {
	MarkdownReportStageEventBase
	PartIndex    int
	SectionCount int
	WordCount    int
}

type PromotedMarkdownReportArtifactEventRequest struct {
	EventID             string
	MissionID           string
	PromotedFromEventID string
	Payload             map[string]any
	Producer            app.Producer
}

func BuildMarkdownReportPlanCreatedAppendRequest(req MarkdownReportPlanCreatedEventRequest) app.AppendEventRequest {
	base := req.MarkdownReportEventBase
	payload := markdownReportBasePayload(base)
	payload["kind"] = reportPlanKind(base.ReportMode)
	payload["artifact_id"] = req.ArtifactID
	putReportNonEmpty(payload, "assembly_strategy", req.AssemblyStrategy)
	payload["plan_review_required"] = req.PlanReviewRequired
	payload["plan_review_state"] = req.PlanReviewState
	payload["duration_ms"] = base.DurationMS
	payload["plan"] = req.Plan
	payload["text"] = base.Text
	addReportAgentUsage(payload, base)
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(base.EventID),
		MissionID: strings.TrimSpace(base.MissionID),
		EventType: "report.plan.created",
		Producer:  base.Producer,
		Payload:   mustJSON(payload),
	}
}

func BuildMarkdownReportArtifactCreatedAppendRequest(req MarkdownReportArtifactCreatedEventRequest) app.AppendEventRequest {
	base := req.MarkdownReportEventBase
	artifact := req.Artifact
	payload := markdownReportBasePayload(base)
	payload["kind"] = "markdown_report_artifact"
	payload["artifact_id"] = artifact.ArtifactID
	payload["media_type"] = artifact.MediaType
	putReportNonEmpty(payload, "plan_event_id", req.PlanEventID)
	putReportNonEmpty(payload, "plan_tool_session_id", req.PlanToolSessionID)
	if req.IncludePlanReview {
		payload["plan_review_required"] = req.PlanReviewRequired
		payload["plan_review_state"] = req.PlanReviewState
	} else {
		putReportNonEmpty(payload, "plan_review_state", req.PlanReviewState)
	}
	putReportNonEmpty(payload, "assembly_strategy", req.AssemblyStrategy)
	if req.IncludeLongFormFields {
		payload["section_count"] = req.SectionCount
		payload["part_count"] = req.PartCount
		payload["section_artifact_ids"] = req.SectionArtifactIDs
		payload["part_artifact_ids"] = req.PartArtifactIDs
		payload["section_word_count"] = req.SectionWordCount
		payload["final_word_count"] = req.FinalWordCount
		payload["preservation_ratio"] = req.PreservationRatio
	}
	payload["duration_ms"] = base.DurationMS
	payload["text"] = base.Text
	addReportAgentUsage(payload, base)
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(base.EventID),
		MissionID: strings.TrimSpace(base.MissionID),
		EventType: "report.artifact.created",
		Producer:  base.Producer,
		Payload:   mustJSON(payload),
	}
}

func BuildMarkdownReportSectionCreatedAppendRequest(req MarkdownReportSectionCreatedEventRequest) app.AppendEventRequest {
	base := req.MarkdownReportStageEventBase
	payload := markdownReportStagePayload(base)
	payload["kind"] = "sectional_markdown_report_section"
	payload["part_index"] = req.PartIndex
	payload["section_index"] = req.SectionIndex
	payload["word_count"] = req.WordCount
	payload["duration_ms"] = base.DurationMS
	payload["text"] = base.Text
	addReportStageAgentUsage(payload, base)
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(base.EventID),
		MissionID: strings.TrimSpace(base.MissionID),
		EventType: "report.section.created",
		Producer:  base.Producer,
		Payload:   mustJSON(payload),
	}
}

func BuildMarkdownReportPartCreatedAppendRequest(req MarkdownReportPartCreatedEventRequest) app.AppendEventRequest {
	base := req.MarkdownReportStageEventBase
	payload := markdownReportStagePayload(base)
	payload["kind"] = "sectional_markdown_report_part"
	payload["part_index"] = req.PartIndex
	payload["section_count"] = req.SectionCount
	payload["word_count"] = req.WordCount
	payload["duration_ms"] = base.DurationMS
	payload["text"] = base.Text
	addReportStageAgentUsage(payload, base)
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(base.EventID),
		MissionID: strings.TrimSpace(base.MissionID),
		EventType: "report.part.created",
		Producer:  base.Producer,
		Payload:   mustJSON(payload),
	}
}

func BuildPromotedMarkdownReportArtifactAppendRequest(req PromotedMarkdownReportArtifactEventRequest) app.AppendEventRequest {
	payload := copyReportPayload(req.Payload)
	payload["kind"] = "markdown_report_artifact"
	payload["promoted_from_event_id"] = strings.TrimSpace(req.PromotedFromEventID)
	return app.AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: "report.artifact.created",
		Producer:  req.Producer,
		Payload:   mustJSON(payload),
	}
}

func markdownReportBasePayload(req MarkdownReportEventBase) map[string]any {
	return map[string]any{
		"pending_event_id":                req.PendingEventID,
		"title":                           req.Title,
		"agent_executor":                  req.AgentExecutor,
		"agent_model":                     req.AgentModel,
		"agent_reasoning_effort":          req.AgentReasoningEffort,
		"agent_session_id":                req.AgentSessionID,
		"previous_agent_session_id":       req.PreviousAgentSessionID,
		"returned_agent_session_id":       req.ReturnedAgentSessionID,
		"tool_session_id":                 req.ToolSessionID,
		"mcp_mode":                        req.MCPMode,
		"rigor_level":                     req.RigorLevel,
		"rigor_label":                     req.RigorLabel,
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
		"post_report_research_session_id": req.PostReportResearchSessionID,
		"composition_strategy":            req.CompositionStrategy,
	}
}

func markdownReportStagePayload(req MarkdownReportStageEventBase) map[string]any {
	artifact := req.Artifact
	return map[string]any{
		"pending_event_id":                req.PendingEventID,
		"plan_event_id":                   req.PlanEventID,
		"title":                           req.Title,
		"artifact_id":                     artifact.ArtifactID,
		"media_type":                      artifact.MediaType,
		"agent_executor":                  req.AgentExecutor,
		"agent_model":                     req.AgentModel,
		"agent_reasoning_effort":          req.AgentReasoningEffort,
		"agent_session_id":                req.AgentSessionID,
		"previous_agent_session_id":       req.PreviousAgentSessionID,
		"returned_agent_session_id":       req.ReturnedAgentSessionID,
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
		"post_report_research_session_id": req.PostReportResearchSessionID,
		"composition_strategy":            req.CompositionStrategy,
		"assembly_strategy":               req.AssemblyStrategy,
	}
}

func addReportAgentUsage(payload map[string]any, req MarkdownReportEventBase) {
	if eventUsage, ok := req.AgentUsage.ForEvent(req.AgentUsageSurface, req.AgentUsageDurationMS, req.PreviousAgentSessionID, req.AgentSessionID, req.AgentResumed, false); ok {
		payload["agent_usage"] = eventUsage
	}
}

func addReportStageAgentUsage(payload map[string]any, req MarkdownReportStageEventBase) {
	if eventUsage, ok := req.AgentUsage.ForEvent(req.AgentUsageSurface, req.AgentUsageDurationMS, req.PreviousAgentSessionID, req.AgentSessionID, req.AgentResumed, false); ok {
		payload["agent_usage"] = eventUsage
	}
}

func reportPlanKind(mode string) string {
	if strings.TrimSpace(mode) == ModeLongForm {
		return "sectional_markdown_report_plan"
	}
	return "markdown_report_plan"
}

func putReportNonEmpty(payload map[string]any, key string, value string) {
	if strings.TrimSpace(value) != "" {
		payload[key] = value
	}
}

func copyReportPayload(payload map[string]any) map[string]any {
	copied := make(map[string]any, len(payload)+2)
	for key, value := range payload {
		copied[key] = value
	}
	return copied
}
