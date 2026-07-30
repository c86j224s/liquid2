package reporting

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type PartEditStore interface {
	PartEditOutcomeStore
	AppendEventConditionally(context.Context, string, func([]app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.LedgerEvent, bool, error)
	CreateRawArtifactWithEventConditionally(context.Context, app.CreateRawArtifactRequest, func([]app.LedgerEvent, app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.RawArtifact, app.LedgerEvent, bool, error)
}

type PartEditStartStore interface {
	AppendEventConditionally(context.Context, string, func([]app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.LedgerEvent, bool, error)
}

type PartEditOutcomeStore interface {
	ListEvents(context.Context, string) ([]app.LedgerEvent, error)
	GetRawArtifact(context.Context, string) (app.RawArtifact, error)
}
