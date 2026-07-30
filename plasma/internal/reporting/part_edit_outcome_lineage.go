package reporting

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type PartEditOutcomeContract struct {
	MissionID                    string
	CurrentPendingEventID        string
	PlanEventID                  string
	SourcePartEventID            string
	SourceArtifactID             string
	PartIndex                    int
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSelectionSource         string
	MCPMode                      string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	SessionChainKind             string
	ReportPlanSessionID          string
	ExcludedProviderSessionIDs   []string
}

func partEditStartedBindingForOutcome(events []app.LedgerEvent, acceptedPending map[string]bool, edited app.LedgerEvent) (PartEditBinding, bool, error) {
	var found PartEditBinding
	count := 0
	for _, event := range events {
		if event.EventType != PartEditStartedEventType || event.CorrelationID != edited.CorrelationID {
			continue
		}
		binding, ok := partEditBindingFromStartEvent(event)
		if !ok {
			return PartEditBinding{}, false, fmt.Errorf("%w: stored Part edit start is invalid", app.ErrConflict)
		}
		if !acceptedPending[binding.PendingEventID] || !partEditEventMatches(edited, binding) {
			continue
		}
		found, count = binding, count+1
	}
	if count > 1 {
		return PartEditBinding{}, false, fmt.Errorf("%w: multiple Part edit starts match outcome", app.ErrConflict)
	}
	return found, count == 1, nil
}

func partEditBindingFromEditedEvent(event app.LedgerEvent) (PartEditBinding, bool) {
	payload := eventPayload(event)
	binding := PartEditBinding{
		MissionID: event.MissionID, PendingEventID: payloadString(payload, "pending_event_id"),
		PlanEventID: payloadString(payload, "plan_event_id"), SourcePartEventID: payloadString(payload, "source_part_event_id"),
		SourceArtifactID: payloadString(payload, "source_artifact_id"), EditedArtifactID: payloadString(payload, "artifact_id"),
		Filename: "part-edit-outcome.md", ToolSessionID: payloadString(payload, "tool_session_id"),
		ProviderSessionID: payloadString(payload, "provider_session_id"), PreviousProviderSessionID: payloadString(payload, "previous_provider_session_id"),
		IdempotencyKey: payloadString(payload, "idempotency_key"), PartIndex: jsonInt(payload["part_index"]),
		RequirementMapEventID: payloadString(payload, "requirement_map_event_id"), RequirementMapHash: payloadString(payload, "requirement_map_hash"),
		AgentExecutor: payloadString(payload, "agent_executor"), AgentModel: payloadString(payload, "agent_model"),
		AgentReasoningEffort: payloadString(payload, "agent_reasoning_effort"), AgentSelectionSource: payloadString(payload, "agent_selection_source"),
		MCPMode: payloadString(payload, "mcp_mode"), ReportSessionPolicy: payloadString(payload, "report_session_policy"),
		ReportSessionPolicySelection: payloadString(payload, "report_session_policy_selection"),
		GenerationGuidanceProfile:    payloadString(payload, "generation_guidance_profile"),
		GenerationGuidanceSHA256:     payloadString(payload, "generation_guidance_sha256"),
		SessionChainKind:             payloadString(payload, "session_chain_kind"),
		ReportPlanSessionID:          payloadString(payload, "report_plan_session_id"),
		ForkSourceAgentSessionID:     payloadString(payload, "fork_source_agent_session_id"),
	}
	binding = normalizePartEditBinding(binding)
	return binding, payloadString(payload, "kind") == PartEditedKind &&
		binding.MissionID != "" &&
		binding.PendingEventID != "" &&
		binding.IdempotencyKey != "" &&
		binding.PartIndex >= 1
}

func partEditOutcomeMatchesContract(binding PartEditBinding, contract PartEditOutcomeContract) bool {
	if binding.MissionID != contract.MissionID || binding.PlanEventID != contract.PlanEventID ||
		binding.SourcePartEventID != contract.SourcePartEventID || binding.SourceArtifactID != contract.SourceArtifactID ||
		binding.PartIndex != contract.PartIndex || binding.AgentExecutor != contract.AgentExecutor ||
		binding.AgentModel != contract.AgentModel || binding.AgentReasoningEffort != contract.AgentReasoningEffort ||
		binding.AgentSelectionSource != contract.AgentSelectionSource || binding.MCPMode != contract.MCPMode ||
		binding.ReportSessionPolicy != contract.ReportSessionPolicy || binding.ReportSessionPolicySelection != contract.ReportSessionPolicySelection ||
		binding.GenerationGuidanceProfile != contract.GenerationGuidanceProfile || binding.GenerationGuidanceSHA256 != contract.GenerationGuidanceSHA256 ||
		binding.SessionChainKind != contract.SessionChainKind || binding.ReportPlanSessionID != contract.ReportPlanSessionID ||
		binding.ForkSourceAgentSessionID != contract.ReportPlanSessionID {
		return false
	}
	if binding.ProviderSessionID == "" || binding.ProviderSessionID == contract.ReportPlanSessionID || binding.PreviousProviderSessionID != binding.ProviderSessionID {
		return false
	}
	if binding.IdempotencyKey != fmt.Sprintf("report-part-edit:%s:%s:%d", binding.PendingEventID, binding.PlanEventID, binding.PartIndex) {
		return false
	}
	for _, sessionID := range contract.ExcludedProviderSessionIDs {
		if binding.ProviderSessionID == sessionID {
			return false
		}
	}
	return true
}

func normalizePartEditOutcomeContract(value PartEditOutcomeContract) PartEditOutcomeContract {
	value.MissionID = strings.TrimSpace(value.MissionID)
	value.CurrentPendingEventID = strings.TrimSpace(value.CurrentPendingEventID)
	value.PlanEventID = strings.TrimSpace(value.PlanEventID)
	value.SourcePartEventID = strings.TrimSpace(value.SourcePartEventID)
	value.SourceArtifactID = strings.TrimSpace(value.SourceArtifactID)
	value.AgentExecutor = strings.TrimSpace(strings.ToLower(value.AgentExecutor))
	value.AgentModel = strings.TrimSpace(value.AgentModel)
	value.AgentReasoningEffort = strings.TrimSpace(value.AgentReasoningEffort)
	value.AgentSelectionSource = strings.TrimSpace(value.AgentSelectionSource)
	value.MCPMode = strings.TrimSpace(value.MCPMode)
	value.ReportSessionPolicy = strings.TrimSpace(value.ReportSessionPolicy)
	value.ReportSessionPolicySelection = strings.TrimSpace(value.ReportSessionPolicySelection)
	value.GenerationGuidanceProfile = strings.TrimSpace(value.GenerationGuidanceProfile)
	value.GenerationGuidanceSHA256 = strings.TrimSpace(value.GenerationGuidanceSHA256)
	value.SessionChainKind = strings.TrimSpace(value.SessionChainKind)
	value.ReportPlanSessionID = strings.TrimSpace(value.ReportPlanSessionID)
	normalized := value.ExcludedProviderSessionIDs[:0]
	for _, sessionID := range value.ExcludedProviderSessionIDs {
		if sessionID = strings.TrimSpace(sessionID); sessionID != "" {
			normalized = append(normalized, sessionID)
		}
	}
	value.ExcludedProviderSessionIDs = normalized
	return value
}

func validatePartEditOutcomeContract(value PartEditOutcomeContract) error {
	if value.MissionID == "" || value.CurrentPendingEventID == "" || value.PlanEventID == "" || value.SourcePartEventID == "" || value.SourceArtifactID == "" || value.PartIndex < 1 || value.AgentExecutor == "" || value.ReportPlanSessionID == "" {
		return fmt.Errorf("%w: part edit outcome contract is incomplete", app.ErrInvalidInput)
	}
	return nil
}
