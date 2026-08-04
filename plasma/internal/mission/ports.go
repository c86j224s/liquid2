package mission

import "context"

// Store persists mission identity independently from ledger storage.
type Store interface {
	CreateMission(context.Context, Mission) error
}

// ListStore reads mission list records.
type ListStore interface {
	ListMissions(context.Context) ([]Mission, error)
}

// ActivityListStore bulk-loads the reduced ledger input for list summaries.
type ActivityListStore interface {
	ListMissionActivityInputs(context.Context, []string) ([]ActivityInput, error)
}

// ProjectionStore persists mission read models derived from the ledger.
type ProjectionStore interface {
	SaveMissionProjection(context.Context, Projection) error
	GetMissionProjection(context.Context, string) (Projection, error)
}
