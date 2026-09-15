package reportrun

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportusage"
)

func validateUsageEvents(state completionState, events []ledger.Event) (map[string]agentusage.AgentUsage, error) {
	targets := make(map[string]completionTarget, len(state.Targets))
	for _, target := range state.Targets {
		targets[target.Event.EventID] = target
	}
	usageByTarget := make(map[string]agentusage.AgentUsage, len(targets))
	usageEventByTarget := make(map[string]string, len(targets))
	for _, event := range events {
		if event.EventType != reportusage.ReportAgentUsageRecordedEventType {
			continue
		}
		var payload struct {
			Kind                     string                `json:"kind"`
			PendingEventID           string                `json:"pending_event_id"`
			CorrelationEventID       string                `json:"correlation_event_id"`
			ForkSourceAgentSessionID string                `json:"fork_source_agent_session_id"`
			AgentUsage               agentusage.AgentUsage `json:"agent_usage"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode report usage %s: %w", event.EventID, err)
		}
		target, isTarget := targets[strings.TrimSpace(payload.CorrelationEventID)]
		if !isTarget {
			continue
		}
		if err := validateUsageEvent(event, payload.Kind, payload.PendingEventID, payload.CorrelationEventID, target); err != nil {
			return nil, err
		}
		if priorID, exists := usageEventByTarget[target.Event.EventID]; exists {
			if priorID != event.EventID {
				return nil, fmt.Errorf("conflicting report usage events for %s", target.Event.EventID)
			}
			continue
		}
		usageEventByTarget[target.Event.EventID] = event.EventID
		usageByTarget[target.Event.EventID] = payload.AgentUsage
	}
	return usageByTarget, nil
}

func validateUsageEvent(event ledger.Event, kind, pendingID, correlationID string, target completionTarget) error {
	if event.EventID != reportusage.ReportAgentUsageEventID(correlationID) || event.MissionID != target.Event.MissionID || event.EventType != reportusage.ReportAgentUsageRecordedEventType || event.CausationEventID != target.Event.EventID || event.CorrelationID != pendingID || strings.TrimSpace(kind) != "report_agent_usage" || strings.TrimSpace(pendingID) != target.Pending.EventID || strings.TrimSpace(correlationID) != target.Event.EventID {
		return fmt.Errorf("invalid report usage envelope for %s", target.Event.EventID)
	}
	if target.Meta.AgentSessionID == "" || event.Producer != (ledger.Producer{Type: "agent_session", ID: target.Meta.AgentSessionID}) {
		return fmt.Errorf("invalid report usage producer for %s", target.Event.EventID)
	}
	var payload struct {
		ForkSourceAgentSessionID string                `json:"fork_source_agent_session_id"`
		AgentUsage               agentusage.AgentUsage `json:"agent_usage"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil || payload.AgentUsage.SchemaVersion != agentusage.SchemaVersion || payload.AgentUsage.Empty() || (payload.AgentUsage.ProviderUsage == nil && !payload.AgentUsage.UsageUnavailable) || (payload.AgentUsage.ProviderUsage != nil && payload.AgentUsage.UsageUnavailable) || (payload.AgentUsage.UsageUnavailable && strings.TrimSpace(payload.AgentUsage.UsageUnavailableReason) == "") {
		return fmt.Errorf("invalid report usage schema for %s", target.Event.EventID)
	}
	if payload.AgentUsage.Session.AgentSessionID != target.Meta.AgentSessionID || payload.AgentUsage.Session.PreviousAgentSessionID != target.Meta.PreviousAgentSessionID || payload.AgentUsage.Surface != target.Meta.Surface || payload.AgentUsage.Executor != target.Meta.AgentExecutor || payload.AgentUsage.Model != target.Meta.AgentModel || payload.AgentUsage.ReasoningEffort != target.Meta.AgentReasoningEffort || strings.TrimSpace(payload.ForkSourceAgentSessionID) != target.Meta.ForkSourceAgentSessionID {
		return fmt.Errorf("report usage metadata differs for %s", target.Event.EventID)
	}
	return nil
}

func actualUsageAppendRequest(state completionState, target completionTarget, actual reportusage.ReportAgentUsageRequest) (ledger.AppendRequest, error) {
	if strings.TrimSpace(actual.MissionID) != target.Event.MissionID || strings.TrimSpace(actual.CanonicalEventID) != target.Event.EventID {
		return ledger.AppendRequest{}, fmt.Errorf("actual report usage does not match target")
	}
	if strings.TrimSpace(actual.AgentSessionID) != target.Meta.AgentSessionID ||
		strings.TrimSpace(actual.PreviousAgentSessionID) != target.Meta.PreviousAgentSessionID ||
		strings.TrimSpace(actual.Surface) != target.Meta.Surface ||
		strings.TrimSpace(actual.ForkSourceAgentSessionID) != target.Meta.ForkSourceAgentSessionID ||
		strings.TrimSpace(actual.Usage.Executor) != target.Meta.AgentExecutor ||
		strings.TrimSpace(actual.Usage.Model) != target.Meta.AgentModel ||
		strings.TrimSpace(actual.Usage.ReasoningEffort) != target.Meta.AgentReasoningEffort {
		return ledger.AppendRequest{}, fmt.Errorf("actual report usage metadata differs from target")
	}
	if actual.Usage.SchemaVersion != agentusage.SchemaVersion || actual.Usage.Empty() ||
		(actual.Usage.ProviderUsage == nil && !actual.Usage.UsageUnavailable) ||
		(actual.Usage.ProviderUsage != nil && actual.Usage.UsageUnavailable) ||
		(actual.Usage.UsageUnavailable && strings.TrimSpace(actual.Usage.UsageUnavailableReason) == "") {
		return ledger.AppendRequest{}, fmt.Errorf("actual report usage has invalid outcome")
	}
	actual.PendingEventID = target.Pending.EventID
	actual.Surface = target.Meta.Surface
	actual.AgentSessionID = target.Meta.AgentSessionID
	actual.PreviousAgentSessionID = target.Meta.PreviousAgentSessionID
	actual.ForkSourceAgentSessionID = target.Meta.ForkSourceAgentSessionID
	actual.Usage.Executor = target.Meta.AgentExecutor
	actual.Usage.Model = target.Meta.AgentModel
	actual.Usage.ReasoningEffort = target.Meta.AgentReasoningEffort
	if actual.AgentSessionID == "" {
		return ledger.AppendRequest{}, fmt.Errorf("report usage target has no agent session")
	}
	request, ok, err := reportusage.BuildReportAgentUsageAppendRequest(actual)
	if err != nil {
		return ledger.AppendRequest{}, err
	}
	if !ok {
		return ledger.AppendRequest{}, fmt.Errorf("actual report usage is empty")
	}
	return request, nil
}

func unavailableUsageRequest(state completionState, target completionTarget) (ledger.AppendRequest, error) {
	if target.Meta.AgentSessionID == "" || target.Meta.Surface == "report_" {
		return ledger.AppendRequest{}, fmt.Errorf("report usage target metadata is incomplete")
	}
	usage := agentusage.New("", target.Meta.AgentExecutor, target.Meta.AgentModel, target.Meta.AgentReasoningEffort, "")
	usage = usage.WithSurface(target.Meta.Surface).WithSession(target.Meta.PreviousAgentSessionID, target.Meta.AgentSessionID, false, false).WithUnavailable(ReportRunCompletionReason)
	encoded, _ := json.Marshal(map[string]any{
		"kind": "report_agent_usage", "pending_event_id": target.Pending.EventID,
		"correlation_event_id": target.Event.EventID, "agent_usage": usage,
		"fork_source_agent_session_id": target.Meta.ForkSourceAgentSessionID,
	})
	return ledger.AppendRequest{
		EventID: reportusage.ReportAgentUsageEventID(target.Event.EventID), MissionID: target.Event.MissionID,
		EventType: reportusage.ReportAgentUsageRecordedEventType, Producer: ledger.Producer{Type: "agent_session", ID: target.Meta.AgentSessionID},
		CausationEventID: target.Event.EventID, CorrelationID: target.Pending.EventID, Payload: encoded,
	}, nil
}
