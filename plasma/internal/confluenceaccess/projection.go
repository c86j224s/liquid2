package confluenceaccess

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

type accessPayload struct {
	ConnectorID  string `json:"connector_id"`
	ConnectionID string `json:"connection_id,omitempty"`
	CloudID      string `json:"cloud_id,omitempty"`
	SpaceKey     string `json:"space_key,omitempty"`
}

// Project derives mission access from durable connector-access events. Remote
// connection availability is applied separately by the application service.
func Project(missionID, connectorID string, events []ledger.Event) AccessProjection {
	projection := AccessProjection{MissionID: missionID, ConnectorID: connectorID, Status: AccessStatusDisabled}
	for _, event := range events {
		switch event.EventType {
		case AccessEventEnabled, AccessEventUpdated, AccessEventDisabled:
		default:
			continue
		}
		var payload accessPayload
		if json.Unmarshal(event.Payload, &payload) != nil || strings.TrimSpace(payload.ConnectorID) != connectorID {
			continue
		}
		projection.LastEventID = event.EventID
		projection.LastSequence = event.Sequence
		if event.EventType == AccessEventDisabled {
			projection.Enabled = false
			projection.ConnectionID = ""
			projection.CloudID = ""
			projection.SpaceKey = ""
			projection.Status = AccessStatusDisabled
			projection.InvalidReason = ""
			continue
		}
		projection.Enabled = true
		projection.ConnectionID = strings.TrimSpace(payload.ConnectionID)
		projection.CloudID = strings.TrimSpace(payload.CloudID)
		projection.SpaceKey = strings.TrimSpace(payload.SpaceKey)
		projection.Status = AccessStatusEnabled
		projection.InvalidReason = ""
	}
	return projection
}
