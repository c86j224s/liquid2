package source

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
)

// ArtifactReader is the narrow source-side read port used while building snapshots.
type ArtifactReader interface {
	GetRawArtifact(context.Context, string) (artifact.Raw, error)
}

// Store is the consumer-side persistence port for source snapshots.
type Store interface {
	CreateSourceSnapshot(context.Context, Snapshot) error
	GetSourceSnapshot(context.Context, string) (Snapshot, error)
}

// ListStore reads approved source snapshots for one mission.
type ListStore interface {
	ListSourceSnapshots(context.Context, string) ([]Snapshot, error)
}
