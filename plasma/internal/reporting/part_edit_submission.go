package reporting

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	PartEditStartedEventType  = "report.part_edit.started"
	PartEditedEventType       = "report.part.edited"
	PartEditedKind            = "sectional_markdown_report_part_edit"
	PartEditSubmittedSentinel = "PART_EDIT_SUBMITTED"
)

// BuildPartEditStartedAppendRequest는 보고서 생성 파이프라인에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildPartEditStartedAppendRequest(eventID string, binding PartEditBinding) app.AppendEventRequest {
	binding = normalizePartEditBinding(binding)
	return app.AppendEventRequest{
		EventID:          strings.TrimSpace(eventID),
		MissionID:        binding.MissionID,
		EventType:        PartEditStartedEventType,
		Producer:         app.Producer{Type: "agent_session", ID: binding.ProviderSessionID},
		CausationEventID: binding.SourcePartEventID,
		CorrelationID:    binding.IdempotencyKey,
		Payload: mustJSON(map[string]any{
			"kind":                            "sectional_markdown_report_part_edit_started",
			"pending_event_id":                binding.PendingEventID,
			"plan_event_id":                   binding.PlanEventID,
			"source_part_event_id":            binding.SourcePartEventID,
			"source_artifact_id":              binding.SourceArtifactID,
			"artifact_id":                     binding.EditedArtifactID,
			"filename":                        binding.Filename,
			"tool_session_id":                 binding.ToolSessionID,
			"provider_session_id":             binding.ProviderSessionID,
			"previous_provider_session_id":    binding.PreviousProviderSessionID,
			"idempotency_key":                 binding.IdempotencyKey,
			"part_index":                      binding.PartIndex,
			"requirement_map_event_id":        binding.RequirementMapEventID,
			"requirement_map_hash":            binding.RequirementMapHash,
			"agent_executor":                  binding.AgentExecutor,
			"agent_model":                     binding.AgentModel,
			"agent_reasoning_effort":          binding.AgentReasoningEffort,
			"agent_selection_source":          binding.AgentSelectionSource,
			"mcp_mode":                        binding.MCPMode,
			"report_session_policy":           binding.ReportSessionPolicy,
			"report_session_policy_selection": binding.ReportSessionPolicySelection,
			"generation_guidance_profile":     binding.GenerationGuidanceProfile,
			"generation_guidance_sha256":      binding.GenerationGuidanceSHA256,
			"session_chain_kind":              binding.SessionChainKind,
			"report_plan_session_id":          binding.ReportPlanSessionID,
			"fork_source_agent_session_id":    binding.ForkSourceAgentSessionID,
			"stage_kind":                      "part_edit",
			"stage_id":                        fmt.Sprintf("part-edit-%d", binding.PartIndex),
			"text":                            "조립된 Part의 별도 편집을 시작했습니다.",
		}),
	}
}

// PartEditBinding는 재실행과 검증에 쓰는 binding 계약이다.
type PartEditBinding struct {
	MissionID                    string `json:"mission_id"`
	PendingEventID               string `json:"pending_event_id"`
	PlanEventID                  string `json:"plan_event_id"`
	SourcePartEventID            string `json:"source_part_event_id"`
	SourceArtifactID             string `json:"source_artifact_id"`
	EditedArtifactID             string `json:"edited_artifact_id"`
	Filename                     string `json:"filename"`
	ToolSessionID                string `json:"tool_session_id"`
	ProviderSessionID            string `json:"provider_session_id"`
	PreviousProviderSessionID    string `json:"previous_provider_session_id"`
	IdempotencyKey               string `json:"idempotency_key"`
	PartIndex                    int    `json:"part_index"`
	RequirementMapEventID        string `json:"requirement_map_event_id,omitempty"`
	RequirementMapHash           string `json:"requirement_map_hash,omitempty"`
	AgentExecutor                string `json:"agent_executor"`
	AgentModel                   string `json:"agent_model"`
	AgentReasoningEffort         string `json:"agent_reasoning_effort"`
	AgentSelectionSource         string `json:"agent_selection_source"`
	MCPMode                      string `json:"mcp_mode"`
	ReportSessionPolicy          string `json:"report_session_policy"`
	ReportSessionPolicySelection string `json:"report_session_policy_selection"`
	GenerationGuidanceProfile    string `json:"generation_guidance_profile"`
	GenerationGuidanceSHA256     string `json:"generation_guidance_sha256"`
	SessionChainKind             string `json:"session_chain_kind"`
	ReportPlanSessionID          string `json:"report_plan_session_id"`
	ForkSourceAgentSessionID     string `json:"fork_source_agent_session_id"`
}

// PartEditResult는 part edit 제출 이벤트와 artifact를 함께 반환한다.
type PartEditResult struct {
	Artifact app.RawArtifact
	Event    app.LedgerEvent
	Replay   bool
}

type partEditedPayload struct {
	Kind                         string `json:"kind"`
	PendingEventID               string `json:"pending_event_id"`
	PlanEventID                  string `json:"plan_event_id"`
	SourcePartEventID            string `json:"source_part_event_id"`
	SourceArtifactID             string `json:"source_artifact_id"`
	ArtifactID                   string `json:"artifact_id"`
	ToolSessionID                string `json:"tool_session_id"`
	ProviderSessionID            string `json:"provider_session_id"`
	PreviousProviderSessionID    string `json:"previous_provider_session_id"`
	IdempotencyKey               string `json:"idempotency_key"`
	PartIndex                    int    `json:"part_index"`
	RequirementMapEventID        string `json:"requirement_map_event_id,omitempty"`
	RequirementMapHash           string `json:"requirement_map_hash,omitempty"`
	AgentExecutor                string `json:"agent_executor"`
	AgentModel                   string `json:"agent_model,omitempty"`
	AgentReasoningEffort         string `json:"agent_reasoning_effort,omitempty"`
	AgentSelectionSource         string `json:"agent_selection_source,omitempty"`
	MCPMode                      string `json:"mcp_mode,omitempty"`
	ReportSessionPolicy          string `json:"report_session_policy,omitempty"`
	ReportSessionPolicySelection string `json:"report_session_policy_selection,omitempty"`
	GenerationGuidanceProfile    string `json:"generation_guidance_profile,omitempty"`
	GenerationGuidanceSHA256     string `json:"generation_guidance_sha256,omitempty"`
	SessionChainKind             string `json:"session_chain_kind,omitempty"`
	ReportPlanSessionID          string `json:"report_plan_session_id,omitempty"`
	ForkSourceAgentSessionID     string `json:"fork_source_agent_session_id,omitempty"`
	OperationCount               int    `json:"operation_count"`
	SourceWordCount              int    `json:"source_word_count"`
	EditedWordCount              int    `json:"edited_word_count"`
	Changed                      bool   `json:"changed"`
	Text                         string `json:"text"`
}

func buildPartEditedAppendRequest(eventID string, binding PartEditBinding, source, artifact app.RawArtifact, operationCount int, changed bool) app.AppendEventRequest {
	payload := partEditedPayload{
		Kind:                         PartEditedKind,
		PendingEventID:               binding.PendingEventID,
		PlanEventID:                  binding.PlanEventID,
		SourcePartEventID:            binding.SourcePartEventID,
		SourceArtifactID:             binding.SourceArtifactID,
		ArtifactID:                   artifact.ArtifactID,
		ToolSessionID:                binding.ToolSessionID,
		ProviderSessionID:            binding.ProviderSessionID,
		PreviousProviderSessionID:    binding.PreviousProviderSessionID,
		IdempotencyKey:               binding.IdempotencyKey,
		PartIndex:                    binding.PartIndex,
		RequirementMapEventID:        binding.RequirementMapEventID,
		RequirementMapHash:           binding.RequirementMapHash,
		AgentExecutor:                binding.AgentExecutor,
		AgentModel:                   binding.AgentModel,
		AgentReasoningEffort:         binding.AgentReasoningEffort,
		AgentSelectionSource:         binding.AgentSelectionSource,
		MCPMode:                      binding.MCPMode,
		ReportSessionPolicy:          binding.ReportSessionPolicy,
		ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
		GenerationGuidanceProfile:    binding.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     binding.GenerationGuidanceSHA256,
		SessionChainKind:             binding.SessionChainKind,
		ReportPlanSessionID:          binding.ReportPlanSessionID,
		ForkSourceAgentSessionID:     binding.ForkSourceAgentSessionID,
		OperationCount:               operationCount,
		SourceWordCount:              len(strings.Fields(string(source.Content))),
		EditedWordCount:              len(strings.Fields(string(artifact.Content))),
		Changed:                      changed,
		Text:                         "조립된 Part를 별도 편집 단계에서 검토하고 편집본으로 확정했습니다.",
	}
	return app.AppendEventRequest{
		EventID:          strings.TrimSpace(eventID),
		MissionID:        binding.MissionID,
		EventType:        PartEditedEventType,
		Producer:         app.Producer{Type: "agent_session", ID: binding.ProviderSessionID},
		CausationEventID: binding.SourcePartEventID,
		CorrelationID:    binding.IdempotencyKey,
		Payload:          mustJSON(payload),
	}
}
