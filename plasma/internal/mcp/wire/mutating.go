package wire

import "github.com/c86j224s/liquid2/plasma/internal/ledger"

// CommonMutatingInput is the shared protocol envelope used by root guards and feature handlers.
type CommonMutatingInput struct {
	MissionID      string          `json:"mission_id"`
	SessionID      string          `json:"session_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Producer       ledger.Producer `json:"producer"`
}
