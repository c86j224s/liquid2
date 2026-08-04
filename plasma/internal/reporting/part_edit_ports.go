package reporting

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// PartEditStore는 part edit 제출과 조회에 필요한 report stage 저장소 포트다.
type PartEditStore interface {
	PartEditOutcomeStore
	AppendEventConditionally(context.Context, string, func([]app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.LedgerEvent, bool, error)
	CreateRawArtifactWithEventConditionally(context.Context, app.CreateRawArtifactRequest, func([]app.LedgerEvent, app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.RawArtifact, app.LedgerEvent, bool, error)
}

// PartEditStartStore는 part edit 시작 이벤트를 복원하기 위한 조회 포트다.
type PartEditStartStore interface {
	AppendEventConditionally(context.Context, string, func([]app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.LedgerEvent, bool, error)
}

// PartEditOutcomeStore는 part edit outcome 복원에 필요한 조회 포트다.
type PartEditOutcomeStore interface {
	ListEvents(context.Context, string) ([]app.LedgerEvent, error)
	GetRawArtifact(context.Context, string) (app.RawArtifact, error)
}
