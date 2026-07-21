package reporting

import (
	"context"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	PartAssemblySubmittedEventType = "report.part_assembly.submitted"
	PartAssemblySubmittedKind      = "sectional_markdown_report_part_assembly_submission"
	PartAssemblySubmittedSentinel  = "PART_ASSEMBLY_SUBMITTED"
)

type PartTransition struct {
	AfterSectionIndex int    `json:"after_section_index"`
	Markdown          string `json:"markdown"`
}

type PartAssembly struct {
	Intro       string           `json:"intro"`
	Transitions []PartTransition `json:"transitions"`
	Closing     string           `json:"closing"`
}

type PartAssemblyBinding struct {
	MissionID                    string       `json:"mission_id"`
	PendingEventID               string       `json:"pending_event_id"`
	PlanEventID                  string       `json:"plan_event_id"`
	ToolSessionID                string       `json:"tool_session_id"`
	ProviderSessionID            string       `json:"provider_session_id"`
	PreviousProviderSessionID    string       `json:"previous_provider_session_id"`
	PartIndex                    int          `json:"part_index"`
	SectionCount                 int          `json:"section_count"`
	AgentExecutor                string       `json:"agent_executor"`
	AgentModel                   string       `json:"agent_model"`
	AgentReasoningEffort         string       `json:"agent_reasoning_effort"`
	AgentSelectionSource         string       `json:"agent_selection_source"`
	MCPMode                      string       `json:"mcp_mode"`
	ReportSessionPolicy          string       `json:"report_session_policy"`
	ReportSessionPolicySelection string       `json:"report_session_policy_selection"`
	PostReportHumanize           string       `json:"post_report_humanize"`
	GenerationGuidanceProfile    string       `json:"generation_guidance_profile"`
	GenerationGuidanceSHA256     string       `json:"generation_guidance_sha256"`
	SessionChainKind             string       `json:"session_chain_kind"`
	PreReportResearchSessionID   string       `json:"pre_report_research_session_id"`
	ReportPlanSessionID          string       `json:"report_plan_session_id"`
	ForkSourceAgentSessionID     string       `json:"fork_source_agent_session_id"`
	Producer                     app.Producer `json:"producer"`
}

type PartAssemblySubmittedEventRequest struct {
	EventID  string
	Binding  PartAssemblyBinding
	Assembly PartAssembly
}

type PartAssemblySubmission struct {
	Event    app.LedgerEvent
	Binding  PartAssemblyBinding
	Assembly PartAssembly
}

type partAssemblySubmittedPayload struct {
	Kind                         string       `json:"kind"`
	PendingEventID               string       `json:"pending_event_id"`
	PlanEventID                  string       `json:"plan_event_id"`
	ToolSessionID                string       `json:"tool_session_id"`
	ProviderSessionID            string       `json:"provider_session_id,omitempty"`
	PreviousProviderSessionID    string       `json:"previous_provider_session_id,omitempty"`
	PartIndex                    int          `json:"part_index"`
	SectionCount                 int          `json:"section_count"`
	AgentExecutor                string       `json:"agent_executor"`
	AgentModel                   string       `json:"agent_model,omitempty"`
	AgentReasoningEffort         string       `json:"agent_reasoning_effort,omitempty"`
	AgentSelectionSource         string       `json:"agent_selection_source,omitempty"`
	MCPMode                      string       `json:"mcp_mode,omitempty"`
	ReportSessionPolicy          string       `json:"report_session_policy,omitempty"`
	ReportSessionPolicySelection string       `json:"report_session_policy_selection,omitempty"`
	PostReportHumanize           string       `json:"post_report_humanize,omitempty"`
	GenerationGuidanceProfile    string       `json:"generation_guidance_profile,omitempty"`
	GenerationGuidanceSHA256     string       `json:"generation_guidance_sha256,omitempty"`
	SessionChainKind             string       `json:"session_chain_kind,omitempty"`
	PreReportResearchSessionID   string       `json:"pre_report_research_session_id,omitempty"`
	ReportPlanSessionID          string       `json:"report_plan_session_id,omitempty"`
	ForkSourceAgentSessionID     string       `json:"fork_source_agent_session_id,omitempty"`
	Assembly                     PartAssembly `json:"assembly"`
	Text                         string       `json:"text"`
}

type PartAssemblySubmissionStore interface {
	ListEvents(context.Context, string) ([]app.LedgerEvent, error)
}

func BuildPartAssemblySubmittedAppendRequest(req PartAssemblySubmittedEventRequest) app.AppendEventRequest {
	binding := normalizePartAssemblyBinding(req.Binding)
	payload := partAssemblySubmittedPayload{
		Kind:                         PartAssemblySubmittedKind,
		PendingEventID:               binding.PendingEventID,
		PlanEventID:                  binding.PlanEventID,
		ToolSessionID:                binding.ToolSessionID,
		ProviderSessionID:            binding.ProviderSessionID,
		PreviousProviderSessionID:    binding.PreviousProviderSessionID,
		PartIndex:                    binding.PartIndex,
		SectionCount:                 binding.SectionCount,
		AgentExecutor:                binding.AgentExecutor,
		AgentModel:                   binding.AgentModel,
		AgentReasoningEffort:         binding.AgentReasoningEffort,
		AgentSelectionSource:         binding.AgentSelectionSource,
		MCPMode:                      binding.MCPMode,
		ReportSessionPolicy:          binding.ReportSessionPolicy,
		ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
		PostReportHumanize:           binding.PostReportHumanize,
		GenerationGuidanceProfile:    binding.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     binding.GenerationGuidanceSHA256,
		SessionChainKind:             binding.SessionChainKind,
		PreReportResearchSessionID:   binding.PreReportResearchSessionID,
		ReportPlanSessionID:          binding.ReportPlanSessionID,
		ForkSourceAgentSessionID:     binding.ForkSourceAgentSessionID,
		Assembly:                     normalizePartAssembly(req.Assembly, binding.SectionCount),
		Text:                         "장문 리포트 파트 연결부를 MCP 편집 도구로 제출했습니다.",
	}
	return app.AppendEventRequest{
		EventID:          strings.TrimSpace(req.EventID),
		MissionID:        binding.MissionID,
		EventType:        PartAssemblySubmittedEventType,
		Producer:         app.Producer{Type: "mcp_server", ID: "plasma.report.part_assembly.submit"},
		CausationEventID: binding.PlanEventID,
		CorrelationID:    binding.PendingEventID,
		Payload:          mustJSON(payload),
	}
}
