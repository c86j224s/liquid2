package web

import (
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"strings"
)

// reportHumanizeInFlightPendingEventID interprets historical H5 cancellation lineage only.
func reportHumanizeInFlightPendingEventID(event ledger.Event) string {
	var payload struct {
		ReportPendingEventID string `json:"report_pending_event_id"`
	}
	_ = json.Unmarshal(event.Payload, &payload)
	return strings.TrimSpace(payload.ReportPendingEventID)
}
