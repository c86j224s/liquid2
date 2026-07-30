package reporting

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func partEditRequirementMapMatches(events []app.LedgerEvent, acceptedPending map[string]bool, binding PartEditBinding) bool {
	if binding.RequirementMapEventID == "" && binding.RequirementMapHash == "" {
		return true
	}
	if binding.RequirementMapEventID == "" || binding.RequirementMapHash == "" {
		return false
	}
	for _, event := range events {
		if event.EventID != binding.RequirementMapEventID || event.EventType != ReportRequirementsMappedEventType {
			continue
		}
		var payload struct {
			PendingEventID     string          `json:"pending_event_id"`
			PlanEventID        string          `json:"plan_event_id"`
			RequirementMapHash string          `json:"requirement_map_hash"`
			RequirementMap     json.RawMessage `json:"requirement_map"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil ||
			!acceptedPending[strings.TrimSpace(payload.PendingEventID)] ||
			strings.TrimSpace(payload.PlanEventID) != binding.PlanEventID ||
			event.CausationEventID != binding.PlanEventID ||
			event.CorrelationID != strings.TrimSpace(payload.PendingEventID) ||
			strings.TrimSpace(payload.RequirementMapHash) != binding.RequirementMapHash {
			return false
		}
		var requirementMap ReportRequirementMap
		if json.Unmarshal(payload.RequirementMap, &requirementMap) != nil {
			return false
		}
		hash, _, err := ReportRequirementMapHash(requirementMap)
		return err == nil && hash == binding.RequirementMapHash
	}
	return false
}
