package ledger

import (
	"encoding/json"
	"time"
)

// Producer identifies the actor that created a ledger event.
type Producer struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Event is one durable entry in a mission ledger. Product projections derive
// from this sequence; generated files and provider sessions are not identity.
type Event struct {
	EventID          string
	MissionID        string
	Sequence         int64
	EventType        string
	Producer         Producer
	CausationEventID string
	CorrelationID    string
	Payload          json.RawMessage
	CreatedAt        time.Time
}

// AppendRequest contains the caller-owned data needed to build an Event.
type AppendRequest struct {
	EventID          string
	MissionID        string
	EventType        string
	Producer         Producer
	CausationEventID string
	CorrelationID    string
	Payload          json.RawMessage
}
