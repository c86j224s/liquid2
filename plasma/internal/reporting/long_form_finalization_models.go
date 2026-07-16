package reporting

import (
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type LongFormFinalizeBinding struct {
	MissionID                    string       `json:"mission_id"`
	PendingEventID               string       `json:"pending_event_id"`
	PlanEventID                  string       `json:"plan_event_id"`
	ArtifactID                   string       `json:"artifact_id"`
	Filename                     string       `json:"filename"`
	Title                        string       `json:"title"`
	ToolSessionID                string       `json:"tool_session_id"`
	IdempotencyKey               string       `json:"idempotency_key"`
	ProviderSessionID            string       `json:"provider_session_id"`
	PreviousProviderSessionID    string       `json:"previous_provider_session_id"`
	PartArtifactIDs              []string     `json:"part_artifact_ids"`
	SectionArtifactIDs           []string     `json:"section_artifact_ids"`
	SectionWordCount             int          `json:"section_word_count"`
	AgentExecutor                string       `json:"agent_executor"`
	AgentModel                   string       `json:"agent_model"`
	AgentReasoningEffort         string       `json:"agent_reasoning_effort"`
	AgentSelectionSource         string       `json:"agent_selection_source"`
	MCPMode                      string       `json:"mcp_mode"`
	RigorLevel                   string       `json:"rigor_level"`
	RigorLabel                   string       `json:"rigor_label"`
	ReportSessionPolicy          string       `json:"report_session_policy"`
	ReportSessionPolicySelection string       `json:"report_session_policy_selection"`
	PostReportHumanize           string       `json:"post_report_humanize"`
	GenerationGuidanceProfile    string       `json:"generation_guidance_profile"`
	GenerationGuidanceSHA256     string       `json:"generation_guidance_sha256"`
	SessionChainKind             string       `json:"session_chain_kind"`
	PreReportResearchSessionID   string       `json:"pre_report_research_session_id"`
	ReportPlanSessionID          string       `json:"report_plan_session_id"`
	ForkSourceAgentSessionID     string       `json:"fork_source_agent_session_id"`
	PlanToolSessionID            string       `json:"plan_tool_session_id"`
	StartedAt                    time.Time    `json:"started_at"`
	Producer                     app.Producer `json:"producer"`
}

type LongFormFinalizeRequest struct {
	Binding         LongFormFinalizeBinding
	EventID         string
	OpeningMarkdown string
	ClosingMarkdown string
}

type LongFormFinalizeResult struct {
	Artifact app.RawArtifact
	Event    app.LedgerEvent
	Replay   bool
}
