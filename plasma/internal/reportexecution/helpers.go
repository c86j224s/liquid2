package reportexecution

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentpolicy"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func pendingEventID(event ledger.Event) string {
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.PendingEventID)
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func validAgentExecutorOrEmpty(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	normalized, err := agentpolicy.NormalizeExecutorName(value)
	if err != nil {
		return ""
	}
	return normalized
}
