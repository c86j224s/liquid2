package app

const (
	ConnectorAccessEventEnabled  = "mission.connector_access.enabled"
	ConnectorAccessEventUpdated  = "mission.connector_access.updated"
	ConnectorAccessEventDisabled = "mission.connector_access.disabled"

	ConnectorAccessStatusDisabled = "disabled"
	ConnectorAccessStatusEnabled  = "enabled"
	ConnectorAccessStatusInvalid  = "invalid"
)

type ConnectorAccessProjection struct {
	MissionID     string `json:"mission_id"`
	ConnectorID   string `json:"connector_id"`
	Enabled       bool   `json:"enabled"`
	ConnectionID  string `json:"connection_id,omitempty"`
	CloudID       string `json:"cloud_id,omitempty"`
	SpaceKey      string `json:"space_key,omitempty"`
	Status        string `json:"status"`
	InvalidReason string `json:"invalid_reason,omitempty"`
	LastEventID   string `json:"last_event_id,omitempty"`
	LastSequence  int64  `json:"last_sequence,omitempty"`
}

type SetConnectorAccessRequest struct {
	EventID      string
	MissionID    string
	ConnectorID  string
	Enabled      bool
	ConnectionID string
	CloudID      string
	SpaceKey     string
	Producer     Producer
}

type ConnectorAccessChangeResult struct {
	Access ConnectorAccessProjection `json:"access"`
	Event  LedgerEvent               `json:"event"`
}
