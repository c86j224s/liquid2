package ledger

import "context"

// Store is the consumer-side port for one mission's durable event sequence.
type Store interface {
	AppendLedgerEvent(context.Context, Event) (Event, error)
	ListLedgerEvents(context.Context, string) ([]Event, error)
}

// ConditionalStore atomically derives and appends events from current state.
// Returning no events is a successful idempotent no-op.
type ConditionalStore interface {
	AppendLedgerEventsConditionally(context.Context, string, func([]Event) ([]Event, error)) ([]Event, error)
}
