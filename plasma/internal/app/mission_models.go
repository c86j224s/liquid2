package app

import (
	"encoding/json"
	"time"
)

type Mission struct {
	MissionID string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Producer struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type LedgerEvent struct {
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

type CreateMissionRequest struct {
	MissionID string
	Title     string
}

type MissionCreatedEventRequest struct {
	EventID   string
	MissionID string
	Title     string
	Objective string
	Scope     MissionScope
	Producer  Producer
}

type AppendEventRequest struct {
	EventID          string
	MissionID        string
	EventType        string
	Producer         Producer
	CausationEventID string
	CorrelationID    string
	Payload          json.RawMessage
}
