package reporting

import (
	"context"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

// PartEditStore는 part edit 제출과 조회에 필요한 report stage 저장소 포트다.
type PartEditStore interface {
	PartEditOutcomeStore
	AppendEventConditionally(context.Context, string, func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error)) (ledger.Event, bool, error)
	CreateRawArtifactWithEventConditionally(context.Context, artifactcontract.CreateRequest, func([]ledger.Event, artifactcontract.Raw) (ledger.AppendRequest, ledger.Event, bool, error)) (artifactcontract.Raw, ledger.Event, bool, error)
}

// PartEditStartStore는 part edit 시작 이벤트를 복원하기 위한 조회 포트다.
type PartEditStartStore interface {
	AppendEventConditionally(context.Context, string, func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error)) (ledger.Event, bool, error)
}

// PartEditOutcomeStore는 part edit outcome 복원에 필요한 조회 포트다.
type PartEditOutcomeStore interface {
	ListEvents(context.Context, string) ([]ledger.Event, error)
	GetRawArtifact(context.Context, string) (artifactcontract.Raw, error)
}
