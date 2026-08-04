package source

import "context"

// Store is the consumer-side persistence port for source snapshots.
type Store interface {
	CreateSourceSnapshot(context.Context, Snapshot) error
	GetSourceSnapshot(context.Context, string) (Snapshot, error)
}

// ListStore reads approved source snapshots for one mission.
type ListStore interface {
	ListSourceSnapshots(context.Context, string) ([]Snapshot, error)
}
