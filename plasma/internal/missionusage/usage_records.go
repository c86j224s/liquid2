package missionusage

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportusage"
)

// usageRecord is the content-free projection input carried by agent events.
// CorrelationEventID links delayed usage to its canonical successful result.
type usageRecord struct {
	AgentUsage               json.RawMessage `json:"agent_usage"`
	WorkflowRunID            string          `json:"workflow_run_id"`
	CorrelationEventID       string          `json:"correlation_event_id"`
	ForkSourceAgentSessionID string          `json:"fork_source_agent_session_id"`
}

func usagePayload(raw []byte) (usageRecord, bool) {
	var payload usageRecord
	if json.Unmarshal(raw, &payload) != nil || len(payload.AgentUsage) == 0 || string(payload.AgentUsage) == "null" {
		return usageRecord{}, false
	}
	payload.WorkflowRunID = strings.TrimSpace(payload.WorkflowRunID)
	payload.CorrelationEventID = strings.TrimSpace(payload.CorrelationEventID)
	payload.ForkSourceAgentSessionID = strings.TrimSpace(payload.ForkSourceAgentSessionID)
	return payload, true
}

// requiredUsageSession recognizes successful report events whose agent usage
// may arrive later in a correlated report.agent_usage.recorded event.
func requiredUsageSession(event ledger.Event) (string, bool) {
	target, ok, err := reportusage.TargetForEvent(event)
	if err != nil {
		return "", false
	}
	if !ok {
		return "", false
	}
	return target.AgentSessionID, true
}

func rememberCumulativeBaseline(baselines map[string]agentusage.AgentUsage, sessionID string, usage agentusage.AgentUsage) {
	if _, _, ok := agentusage.IncrementalProviderUsage(usage, nil); ok {
		baselines[sessionID] = usage
	}
}
