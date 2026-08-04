package artifact

import "context"

// Store is the consumer-side raw artifact persistence port.
type Store interface {
	CreateRawArtifact(context.Context, Raw) error
	GetRawArtifact(context.Context, string) (Raw, error)
}

// ListStore reads artifact metadata for one mission.
type ListStore interface {
	ListRawArtifacts(context.Context, string) ([]Raw, error)
}
