package reporting

import (
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	// LongFormCompositionPreserveMarkdown은 섹션/파트 markdown을 구조 보존 방식으로 조립한다.
	LongFormCompositionPreserveMarkdown = "sectional_preserve_markdown"
	// LongFormCompositionNarrativeEdit은 조립 후 서사 흐름을 다듬는 최종 편집 전략이다.
	LongFormCompositionNarrativeEdit = "sectional_narrative_edit"
)

// LongFormFinalizeBinding은 장문 보고서 최종화가 replay와 검증에 사용하는
// 지속 계약이다. artifact, stage, session, 모델 선택, humanize 설정이 모두
// 여기 묶여야 하며, 최종화 단계는 저장된 binding과 다른 결과를 canonical로
// 승격하지 않는다.
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
	CompositionStrategy          string       `json:"composition_strategy,omitempty"`
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

// LongFormFinalizeRequest는 장문 보고서 최종 artifact를 만들거나 replay하기 위한
// 입력이다. Binding이 동일하면 같은 canonical event를 재사용해야 하고, final edit
// pipeline 필드는 최종 편집 단계의 검증 결과를 함께 전달한다.
type LongFormFinalizeRequest struct {
	Binding                   LongFormFinalizeBinding
	EventID                   string
	OpeningMarkdown           string
	ClosingMarkdown           string
	ManuscriptMarkdown        string
	FinalEditPipeline         string
	GateFindings              []StoredFinalEditGateFinding
	SemanticReview            FinalEditSemanticAttestation
	FinalEditActualArtifactID string
	FinalEditGateEventID      string
	FinalEditGateChanged      bool
}

// LongFormFinalizeResult는 장문 보고서 최종화 결과다. Replay가 true이면 새
// artifact/event를 만들지 않고 기존 canonical 결과를 반환했다는 뜻이다.
type LongFormFinalizeResult struct {
	Artifact app.RawArtifact
	Event    app.LedgerEvent
	Replay   bool
}
