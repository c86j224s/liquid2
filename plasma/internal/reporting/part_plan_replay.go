package reporting

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// StoredPartPlanExpectation는 저장된 part plan이 어느 parent plan과 stage에 속하는지 검증하는 값이다.
type StoredPartPlanExpectation struct {
	MissionID                    string
	PendingEventID               string
	PlanEventID                  string
	PartIndex                    int
	PartCount                    int
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSelectionSource         string
	ReportMode                   string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	SessionChainKind             string
	ReportPlanSessionID          string
}

func matchingPartPlanEvents(events []app.LedgerEvent, pendingEventID string, planEventID string, partIndex int) []app.LedgerEvent {
	matches := []app.LedgerEvent{}
	for _, event := range events {
		if event.EventType != PartPlanCreatedEventType {
			continue
		}
		var payload struct {
			PendingEventID string `json:"pending_event_id"`
			PlanEventID    string `json:"plan_event_id"`
			PartIndex      int    `json:"part_index"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && strings.TrimSpace(payload.PendingEventID) == strings.TrimSpace(pendingEventID) && strings.TrimSpace(payload.PlanEventID) == strings.TrimSpace(planEventID) && payload.PartIndex == partIndex {
			matches = append(matches, event)
		}
	}
	return matches
}

// DecodeStoredPartPlan는 저장된 part plan artifact를 재실행 가능한 plan 값으로 복원한다.
func DecodeStoredPartPlan(event app.LedgerEvent, expected StoredPartPlanExpectation) (PartPlanResult, bool, error) {
	expected = normalizeStoredPartPlanExpectation(expected)
	if event.EventType != PartPlanCreatedEventType {
		return PartPlanResult{}, false, nil
	}
	payload := map[string]any{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return PartPlanResult{}, false, fmt.Errorf("%w: stored Part plan payload is invalid", app.ErrConflict)
	}
	if payloadString(payload, "pending_event_id") != expected.PendingEventID {
		return PartPlanResult{}, false, nil
	}
	if err := validateStoredPartPlanExpectation(expected); err != nil {
		return PartPlanResult{}, false, err
	}
	brief := strings.TrimSpace(payloadString(payload, "brief"))
	partIndex := jsonInt(payload["part_index"])
	if brief == "" || len([]byte(brief)) > maxPartPlanBriefBytes {
		return PartPlanResult{}, false, fmt.Errorf("%w: stored Part plan brief is invalid", app.ErrConflict)
	}
	if partIndex < 1 || partIndex > expected.PartCount || (expected.PartIndex > 0 && partIndex != expected.PartIndex) {
		return PartPlanResult{}, false, fmt.Errorf("%w: stored Part plan index is outside the report plan", app.ErrConflict)
	}
	ownerSessionID := payloadString(payload, "agent_session_id")
	reportPlanSessionID := expected.ReportPlanSessionID
	correlationID := fmt.Sprintf("report-part-plan:%s:%s:%d", expected.PendingEventID, expected.PlanEventID, partIndex)
	if strings.TrimSpace(event.MissionID) != expected.MissionID ||
		strings.TrimSpace(event.CausationEventID) != expected.PlanEventID ||
		strings.TrimSpace(event.CorrelationID) != correlationID ||
		payloadString(payload, "kind") != PartPlanCreatedKind ||
		payloadString(payload, "plan_event_id") != expected.PlanEventID ||
		payloadString(payload, "stage_kind") != "part_plan" ||
		payloadString(payload, "stage_id") != fmt.Sprintf("part-plan-%d", partIndex) ||
		payloadString(payload, "agent_executor") != expected.AgentExecutor ||
		payloadString(payload, "agent_model") != expected.AgentModel ||
		payloadString(payload, "agent_reasoning_effort") != expected.AgentReasoningEffort ||
		payloadString(payload, "agent_selection_source") != expected.AgentSelectionSource ||
		payloadString(payload, "report_mode") != expected.ReportMode ||
		payloadString(payload, "report_session_policy") != expected.ReportSessionPolicy ||
		payloadString(payload, "report_session_policy_selection") != expected.ReportSessionPolicySelection ||
		payloadString(payload, "generation_guidance_profile") != expected.GenerationGuidanceProfile ||
		payloadString(payload, "generation_guidance_sha256") != expected.GenerationGuidanceSHA256 ||
		payloadString(payload, "session_chain_kind") != expected.SessionChainKind ||
		payloadString(payload, "report_plan_session_id") != reportPlanSessionID ||
		payloadString(payload, "fork_source_agent_session_id") != reportPlanSessionID {
		return PartPlanResult{}, false, fmt.Errorf("%w: stored Part plan provenance differs", app.ErrConflict)
	}
	if ownerSessionID == "" ||
		payloadString(payload, "tool_session_id") == "" ||
		ownerSessionID == reportPlanSessionID ||
		event.Producer != (app.Producer{Type: "agent_session", ID: ownerSessionID}) ||
		payloadString(payload, "previous_agent_session_id") != ownerSessionID ||
		payloadString(payload, "returned_agent_session_id") != ownerSessionID ||
		payloadString(payload, "report_session_id") != ownerSessionID {
		return PartPlanResult{}, false, fmt.Errorf("%w: stored Part plan provider session is invalid", app.ErrConflict)
	}
	return PartPlanResult{Event: event, Brief: brief, ProviderSessionID: ownerSessionID, PartIndex: partIndex}, true, nil
}

func validatePartPlanCreatedEvent(event app.LedgerEvent, expected StoredPartPlanExpectation) error {
	_, ok, err := DecodeStoredPartPlan(event, expected)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: stored Part plan does not match replay expectation", app.ErrConflict)
	}
	return nil
}

func validatePartPlanCreatedRequest(request app.AppendEventRequest, expected StoredPartPlanExpectation) error {
	return validatePartPlanCreatedEvent(app.LedgerEvent{
		EventID:          request.EventID,
		MissionID:        request.MissionID,
		EventType:        request.EventType,
		Producer:         request.Producer,
		CausationEventID: request.CausationEventID,
		CorrelationID:    request.CorrelationID,
		Payload:          request.Payload,
	}, expected)
}

func partPlanExpectationForRequest(req PartPlanCreatedEventRequest, partCount int) StoredPartPlanExpectation {
	base := req.MarkdownReportStageEventBase
	return StoredPartPlanExpectation{
		MissionID: strings.TrimSpace(base.MissionID), PendingEventID: strings.TrimSpace(base.PendingEventID),
		PlanEventID: strings.TrimSpace(base.PlanEventID), PartIndex: req.PartIndex, PartCount: partCount,
		AgentExecutor: strings.TrimSpace(strings.ToLower(base.AgentExecutor)), AgentModel: strings.TrimSpace(base.AgentModel),
		AgentReasoningEffort: strings.TrimSpace(base.AgentReasoningEffort), AgentSelectionSource: strings.TrimSpace(base.AgentSelectionSource),
		ReportMode: strings.TrimSpace(base.ReportMode), ReportSessionPolicy: strings.TrimSpace(base.ReportSessionPolicy),
		ReportSessionPolicySelection: strings.TrimSpace(base.ReportSessionPolicySelection),
		GenerationGuidanceProfile:    strings.TrimSpace(base.GenerationGuidanceProfile),
		GenerationGuidanceSHA256:     strings.TrimSpace(base.GenerationGuidanceSHA256),
		SessionChainKind:             strings.TrimSpace(base.SessionChainKind),
		ReportPlanSessionID:          strings.TrimSpace(base.ReportPlanSessionID),
	}
}

func normalizeStoredPartPlanExpectation(value StoredPartPlanExpectation) StoredPartPlanExpectation {
	value.MissionID = strings.TrimSpace(value.MissionID)
	value.PendingEventID = strings.TrimSpace(value.PendingEventID)
	value.PlanEventID = strings.TrimSpace(value.PlanEventID)
	value.AgentExecutor = strings.TrimSpace(strings.ToLower(value.AgentExecutor))
	value.AgentModel = strings.TrimSpace(value.AgentModel)
	value.AgentReasoningEffort = strings.TrimSpace(value.AgentReasoningEffort)
	value.AgentSelectionSource = strings.TrimSpace(value.AgentSelectionSource)
	value.ReportMode = strings.TrimSpace(value.ReportMode)
	value.ReportSessionPolicy = strings.TrimSpace(value.ReportSessionPolicy)
	value.ReportSessionPolicySelection = strings.TrimSpace(value.ReportSessionPolicySelection)
	value.GenerationGuidanceProfile = strings.TrimSpace(value.GenerationGuidanceProfile)
	value.GenerationGuidanceSHA256 = strings.TrimSpace(value.GenerationGuidanceSHA256)
	value.SessionChainKind = strings.TrimSpace(value.SessionChainKind)
	value.ReportPlanSessionID = strings.TrimSpace(value.ReportPlanSessionID)
	return value
}

func validateStoredPartPlanExpectation(value StoredPartPlanExpectation) error {
	if value.MissionID == "" ||
		value.PendingEventID == "" ||
		value.PlanEventID == "" ||
		value.PartCount < 1 ||
		value.AgentExecutor == "" ||
		value.ReportMode == "" ||
		value.ReportSessionPolicy == "" ||
		value.SessionChainKind == "" ||
		value.ReportPlanSessionID == "" {
		return fmt.Errorf("%w: stored Part plan expectation is incomplete", app.ErrInvalidInput)
	}
	return nil
}
