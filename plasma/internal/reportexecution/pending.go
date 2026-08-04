package reportexecution

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func DraftRequestFromPendingEvent(event ledger.Event) (DraftRequest, error) {
	var payload struct {
		Title                        string `json:"title"`
		DirectionHint                string `json:"direction_hint"`
		ExecutionStrategy            string `json:"execution_strategy"`
		AgentExecutor                string `json:"agent_executor"`
		AgentModel                   string `json:"agent_model"`
		AgentReasoningEffort         string `json:"agent_reasoning_effort"`
		AgentSelectionSource         string `json:"agent_selection_source"`
		MCPMode                      string `json:"mcp_mode"`
		RigorLevel                   string `json:"rigor_level"`
		RigorLabel                   string `json:"rigor_label"`
		ReportMode                   string `json:"report_mode"`
		ReportSessionPolicy          string `json:"report_session_policy"`
		ReportSessionPolicySelection string `json:"report_session_policy_selection"`
		PostReportHumanize           string `json:"post_report_humanize"`
		GenerationGuidanceProfile    string `json:"generation_guidance_profile"`
		GenerationGuidanceSHA256     string `json:"generation_guidance_sha256"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return DraftRequest{}, fmt.Errorf("%w: invalid report pending payload", producterror.ErrInvalidInput)
	}
	return normalizeDraftRequest(DraftRequest{
		Title:                        firstNonEmpty(payload.Title, "Mission report"),
		DirectionHint:                payload.DirectionHint,
		ExecutionStrategy:            payload.ExecutionStrategy,
		AgentExecutor:                firstNonEmpty(payload.AgentExecutor, "codex"),
		AgentModel:                   payload.AgentModel,
		AgentReasoningEffort:         payload.AgentReasoningEffort,
		AgentSelectionSource:         payload.AgentSelectionSource,
		MCPMode:                      firstNonEmpty(payload.MCPMode, "auto"),
		RigorLevel:                   payload.RigorLevel,
		RigorLabel:                   payload.RigorLabel,
		ReportMode:                   payload.ReportMode,
		ReportSessionPolicy:          payload.ReportSessionPolicy,
		ReportSessionPolicySelection: payload.ReportSessionPolicySelection,
		PostReportHumanize:           payload.PostReportHumanize,
		GenerationGuidanceProfile:    payload.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     payload.GenerationGuidanceSHA256,
	}), nil
}

// DesignRequestFromPendingEvent는 design export pending payload에서 재개용 DesignRequest를 복원한다.
func DesignRequestFromPendingEvent(event ledger.Event) (DesignRequest, error) {
	var payload struct {
		SourceArtifactID     string `json:"source_artifact_id"`
		SourceMediaType      string `json:"source_media_type"`
		Title                string `json:"title"`
		AgentExecutor        string `json:"agent_executor"`
		AgentModel           string `json:"agent_model"`
		AgentReasoningEffort string `json:"agent_reasoning_effort"`
		RendererVersion      string `json:"renderer_version"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return DesignRequest{}, fmt.Errorf("%w: invalid designed HTML pending payload", producterror.ErrInvalidInput)
	}
	return DesignRequest{
		SourceArtifactID:     strings.TrimSpace(payload.SourceArtifactID),
		SourceMediaType:      strings.TrimSpace(payload.SourceMediaType),
		Title:                firstNonEmpty(payload.Title, "Mission report"),
		AgentExecutor:        firstNonEmpty(payload.AgentExecutor, "codex"),
		AgentModel:           strings.TrimSpace(payload.AgentModel),
		AgentReasoningEffort: strings.TrimSpace(payload.AgentReasoningEffort),
		RendererVersion:      strings.TrimSpace(payload.RendererVersion),
	}, nil
}

// HumanizeRequestFromPendingEvent는 humanize pending payload에서 재개용 HumanizeRequest를 복원한다.
func HumanizeRequestFromPendingEvent(event ledger.Event) (HumanizeRequest, error) {
	var payload humanizePendingPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return HumanizeRequest{}, fmt.Errorf("%w: invalid H5 humanize pending payload", producterror.ErrInvalidInput)
	}
	return normalizeHumanizeRequest(HumanizeRequest{
		SourceArtifactID:       payload.SourceArtifactID,
		SourceArtifactSHA256:   payload.SourceArtifactSHA256,
		SourceMediaType:        payload.SourceMediaType,
		Title:                  payload.Title,
		AgentExecutor:          payload.AgentExecutor,
		AgentModel:             payload.AgentModel,
		AgentReasoningEffort:   payload.AgentReasoningEffort,
		MCPMode:                payload.MCPMode,
		PreviousAgentSessionID: payload.PreviousSessionID,
		ToolSessionID:          payload.ToolSessionID,
		ReportMode:             payload.ReportMode,
		ReportPendingEventID:   payload.ReportPendingEventID,
	}), nil
}

type humanizePendingPayload struct {
	Target               string `json:"target"`
	Profile              string `json:"profile"`
	PendingEventID       string `json:"pending_event_id"`
	ReportPendingEventID string `json:"report_pending_event_id"`
	Title                string `json:"title"`
	SourceArtifactID     string `json:"source_artifact_id"`
	SourceArtifactSHA256 string `json:"source_artifact_sha256"`
	SourceMediaType      string `json:"source_media_type"`
	AgentExecutor        string `json:"agent_executor"`
	AgentModel           string `json:"agent_model"`
	AgentReasoningEffort string `json:"agent_reasoning_effort"`
	PreviousSessionID    string `json:"previous_agent_session_id"`
	ToolSessionID        string `json:"tool_session_id"`
	MCPMode              string `json:"mcp_mode"`
	ReportMode           string `json:"report_mode"`
	ReportModeLabel      string `json:"report_mode_label"`
	HumanizeTransport    string `json:"humanize_transport"`
}

func humanizePendingPayloadFromEvent(event ledger.Event) humanizePendingPayload {
	var payload humanizePendingPayload
	_ = json.Unmarshal(event.Payload, &payload)
	return payload
}

// PatchRequestFromPendingEvent는 patch pending payload에서 재개용 PatchRequest를 복원한다.
func PatchRequestFromPendingEvent(event ledger.Event) (PatchRequest, error) {
	var payload struct {
		BaseArtifactID               string `json:"base_artifact_id"`
		Instruction                  string `json:"instruction"`
		Title                        string `json:"title"`
		AgentExecutor                string `json:"agent_executor"`
		AgentModel                   string `json:"agent_model"`
		AgentReasoningEffort         string `json:"agent_reasoning_effort"`
		MCPMode                      string `json:"mcp_mode"`
		ReportSessionID              string `json:"report_session_id"`
		PreviousAgentSessionID       string `json:"previous_agent_session_id"`
		ForkSourceAgentSessionID     string `json:"fork_source_agent_session_id"`
		ReportSessionPolicy          string `json:"report_session_policy"`
		ReportSessionPolicySelection string `json:"report_session_policy_selection"`
		SessionChainKind             string `json:"session_chain_kind"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return PatchRequest{}, fmt.Errorf("%w: invalid report patch pending payload", producterror.ErrInvalidInput)
	}
	return normalizePatchRequest(PatchRequest{
		BaseArtifactID:               payload.BaseArtifactID,
		Instruction:                  payload.Instruction,
		Title:                        payload.Title,
		AgentExecutor:                payload.AgentExecutor,
		AgentModel:                   payload.AgentModel,
		AgentReasoningEffort:         payload.AgentReasoningEffort,
		MCPMode:                      payload.MCPMode,
		ReportSessionID:              payload.ReportSessionID,
		PreviousAgentSessionID:       payload.PreviousAgentSessionID,
		ForkSourceAgentSessionID:     payload.ForkSourceAgentSessionID,
		ReportSessionPolicy:          payload.ReportSessionPolicy,
		ReportSessionPolicySelection: payload.ReportSessionPolicySelection,
		SessionChainKind:             payload.SessionChainKind,
	}), nil
}

// CompletedPendingEventIDs는 장부에서 완료된 pending event ID 집합을 계산한다.
func CompletedPendingEventIDs(events []ledger.Event) map[string]struct{} {
	completed := map[string]struct{}{}
	for _, event := range events {
		switch event.EventType {
		case "report.drafted", "report.artifact.created", "report.artifact.exported":
			if pendingEventID := pendingEventID(event); pendingEventID != "" {
				completed[pendingEventID] = struct{}{}
			}
		case "report.draft.failed", "report.design.failed", "report.humanize.failed", "report.humanize.skipped", "report.patch.failed":
			var payload struct {
				PendingEventID string `json:"pending_event_id"`
			}
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				continue
			}
			if pendingEventID := strings.TrimSpace(payload.PendingEventID); pendingEventID != "" {
				completed[pendingEventID] = struct{}{}
			}
		}
	}
	return completed
}
