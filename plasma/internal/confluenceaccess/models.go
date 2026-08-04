package confluenceaccess

import (
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

const (
	ConnectorID  = "confluence"
	AuthOAuth    = "oauth"
	AuthAPIToken = "api_token"

	AccessEventEnabled  = "mission.connector_access.enabled"
	AccessEventUpdated  = "mission.connector_access.updated"
	AccessEventDisabled = "mission.connector_access.disabled"

	AccessStatusDisabled = "disabled"
	AccessStatusEnabled  = "enabled"
	AccessStatusInvalid  = "invalid"
)

// Site is one Confluence site available to a stored connection.
type Site struct {
	CloudID string   `json:"cloud_id"`
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Scopes  []string `json:"scopes,omitempty"`
}

type SiteListResult struct {
	Sites []Site `json:"sites"`
}

// Connection is the durable Confluence credential record. Tokens must never
// be serialized into API responses or logs.
type Connection struct {
	ConnectionID   string    `json:"connection_id"`
	DisplayName    string    `json:"display_name"`
	AuthType       string    `json:"auth_type"`
	AccountID      string    `json:"account_id,omitempty"`
	AccountName    string    `json:"account_name,omitempty"`
	AccessToken    string    `json:"-"`
	RefreshToken   string    `json:"-"`
	TokenExpiresAt time.Time `json:"token_expires_at,omitempty"`
	Scopes         []string  `json:"scopes,omitempty"`
	Sites          []Site    `json:"sites,omitempty"`
	Revoked        bool      `json:"revoked"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UpsertRequest struct {
	ConnectionID   string
	DisplayName    string
	AuthType       string
	AccountID      string
	AccountName    string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt time.Time
	Scopes         []string
	Sites          []Site
	Revoked        bool
}

// AccessProjection is one mission's current permission to use the connector.
type AccessProjection struct {
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

type SetAccessRequest struct {
	EventID      string
	MissionID    string
	ConnectorID  string
	Enabled      bool
	ConnectionID string
	CloudID      string
	SpaceKey     string
	Producer     ledger.Producer
}

type AccessChangeResult struct {
	Access AccessProjection `json:"access"`
	Event  ledger.Event     `json:"event"`
}
