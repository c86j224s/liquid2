package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// PartEditStartContract는 재실행과 검증에 쓰는 binding 계약이다.
type PartEditStartContract struct {
	MissionID                    string
	CurrentPendingEventID        string
	PlanEventID                  string
	SourcePartEventID            string
	SourceArtifactID             string
	PartIndex                    int
	IdempotencyKey               string
	RequirementMapEventID        string
	RequirementMapHash           string
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
	ForkSourceAgentSessionID     string
	ExpectedProviderSessionID    string
	ExcludedProviderSessionIDs   []string
}

// LoadCurrentPartEditStart는 현재 part edit 시작 이벤트와 binding을 장부에서 복원한다.
func LoadCurrentPartEditStart(ctx context.Context, store PartEditOutcomeStore, contract PartEditStartContract) (PartEditBinding, bool, error) {
	contract = normalizePartEditStartContract(contract)
	if err := validatePartEditStartContract(contract); err != nil {
		return PartEditBinding{}, false, err
	}
	events, err := store.ListEvents(ctx, contract.MissionID)
	if err != nil {
		return PartEditBinding{}, false, err
	}
	acceptedPending, err := longFormPendingLineage(events, contract.CurrentPendingEventID)
	if err != nil {
		return PartEditBinding{}, false, err
	}
	if !acceptedPending[contract.CurrentPendingEventID] {
		return PartEditBinding{}, false, fmt.Errorf("%w: current Part edit pending is invalid", app.ErrConflict)
	}
	var found PartEditBinding
	count := 0
	for _, event := range events {
		if event.EventType != PartEditStartedEventType || !partEditStartTargetsContract(event, contract) {
			continue
		}
		binding, ok := partEditBindingFromStartEvent(event)
		if !ok {
			return PartEditBinding{}, false, fmt.Errorf("%w: stored Part edit start is invalid", app.ErrConflict)
		}
		if err := validatePartEditStartBinding(ctx, store, events, acceptedPending, binding, contract); err != nil {
			return PartEditBinding{}, false, err
		}
		if existing, ok, err := canonicalPartEditEvent(events, binding); err != nil {
			return PartEditBinding{}, false, err
		} else if ok {
			if !partEditEventMatches(existing, binding) {
				return PartEditBinding{}, false, fmt.Errorf("%w: completed Part edit differs from start", app.ErrConflict)
			}
			if _, err := partEditResultFromEvent(ctx, store, binding, existing, true); err != nil {
				return PartEditBinding{}, false, fmt.Errorf("%w: completed Part edit artifact is invalid", app.ErrConflict)
			}
			continue
		}
		found, count = binding, count+1
	}
	if count > 1 {
		return PartEditBinding{}, false, fmt.Errorf("%w: multiple open Part edit starts match current pending", app.ErrConflict)
	}
	return found, count == 1, nil
}

func partEditStartTargetsContract(event app.LedgerEvent, contract PartEditStartContract) bool {
	payload := eventPayload(event)
	return strings.TrimSpace(event.MissionID) == contract.MissionID &&
		strings.TrimSpace(event.CorrelationID) == contract.IdempotencyKey &&
		payloadString(payload, "pending_event_id") == contract.CurrentPendingEventID &&
		payloadString(payload, "plan_event_id") == contract.PlanEventID &&
		payloadString(payload, "source_part_event_id") == contract.SourcePartEventID &&
		payloadString(payload, "source_artifact_id") == contract.SourceArtifactID &&
		jsonInt(payload["part_index"]) == contract.PartIndex
}

func validatePartEditStartBinding(ctx context.Context, store PartEditOutcomeStore, events []app.LedgerEvent, acceptedPending map[string]bool, binding PartEditBinding, contract PartEditStartContract) error {
	if !partEditStartBindingMatchesContract(binding, contract) {
		return fmt.Errorf("%w: Part edit start binding differs from current contract", app.ErrConflict)
	}
	if err := validatePartEditLineage(events, binding); err != nil {
		return err
	}
	if !partEditRequirementMapMatches(events, acceptedPending, binding) {
		return fmt.Errorf("%w: Part edit requirement map differs from binding", app.ErrConflict)
	}
	source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return err
	}
	if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" {
		return fmt.Errorf("%w: source Part artifact is foreign or not Markdown", app.ErrConflict)
	}
	return nil
}

func partEditStartBindingMatchesContract(binding PartEditBinding, contract PartEditStartContract) bool {
	if binding.MissionID != contract.MissionID ||
		binding.PendingEventID != contract.CurrentPendingEventID ||
		binding.PlanEventID != contract.PlanEventID ||
		binding.SourcePartEventID != contract.SourcePartEventID ||
		binding.SourceArtifactID != contract.SourceArtifactID ||
		binding.PartIndex != contract.PartIndex ||
		binding.IdempotencyKey != contract.IdempotencyKey ||
		binding.RequirementMapEventID != contract.RequirementMapEventID ||
		binding.RequirementMapHash != contract.RequirementMapHash ||
		binding.AgentExecutor != contract.AgentExecutor ||
		binding.AgentModel != contract.AgentModel ||
		binding.AgentReasoningEffort != contract.AgentReasoningEffort ||
		binding.AgentSelectionSource != contract.AgentSelectionSource ||
		binding.MCPMode != contract.MCPMode ||
		binding.ReportSessionPolicy != contract.ReportSessionPolicy ||
		binding.ReportSessionPolicySelection != contract.ReportSessionPolicySelection ||
		binding.GenerationGuidanceProfile != contract.GenerationGuidanceProfile ||
		binding.GenerationGuidanceSHA256 != contract.GenerationGuidanceSHA256 ||
		binding.SessionChainKind != contract.SessionChainKind ||
		binding.ReportPlanSessionID != contract.ReportPlanSessionID ||
		binding.ForkSourceAgentSessionID != contract.ForkSourceAgentSessionID {
		return false
	}
	if binding.ProviderSessionID == "" || binding.ProviderSessionID == binding.ReportPlanSessionID || binding.PreviousProviderSessionID != binding.ProviderSessionID {
		return false
	}
	if contract.ExpectedProviderSessionID != "" && binding.ProviderSessionID != contract.ExpectedProviderSessionID {
		return false
	}
	for _, excluded := range contract.ExcludedProviderSessionIDs {
		if binding.ProviderSessionID == excluded {
			return false
		}
	}
	return true
}

func normalizePartEditStartContract(value PartEditStartContract) PartEditStartContract {
	value.MissionID = strings.TrimSpace(value.MissionID)
	value.CurrentPendingEventID = strings.TrimSpace(value.CurrentPendingEventID)
	value.PlanEventID = strings.TrimSpace(value.PlanEventID)
	value.SourcePartEventID = strings.TrimSpace(value.SourcePartEventID)
	value.SourceArtifactID = strings.TrimSpace(value.SourceArtifactID)
	value.IdempotencyKey = strings.TrimSpace(value.IdempotencyKey)
	value.RequirementMapEventID = strings.TrimSpace(value.RequirementMapEventID)
	value.RequirementMapHash = strings.TrimSpace(value.RequirementMapHash)
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
	value.ForkSourceAgentSessionID = strings.TrimSpace(value.ForkSourceAgentSessionID)
	value.ExpectedProviderSessionID = strings.TrimSpace(value.ExpectedProviderSessionID)
	normalized := value.ExcludedProviderSessionIDs[:0]
	for _, sessionID := range value.ExcludedProviderSessionIDs {
		if sessionID = strings.TrimSpace(sessionID); sessionID != "" {
			normalized = append(normalized, sessionID)
		}
	}
	value.ExcludedProviderSessionIDs = normalized
	return value
}

func validatePartEditStartContract(value PartEditStartContract) error {
	if value.MissionID == "" ||
		value.CurrentPendingEventID == "" ||
		value.PlanEventID == "" ||
		value.SourcePartEventID == "" ||
		value.SourceArtifactID == "" ||
		value.PartIndex < 1 ||
		value.IdempotencyKey == "" ||
		value.AgentExecutor == "" ||
		value.ReportPlanSessionID == "" ||
		value.ForkSourceAgentSessionID == "" {
		return fmt.Errorf("%w: Part edit start contract is incomplete", app.ErrInvalidInput)
	}
	return nil
}
