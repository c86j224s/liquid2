package sqlite

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
)

// CreateMission는 새 미션과 최초 장부 이벤트를 저장한다.
func (s *Store) CreateMission(ctx context.Context, missionRecord mission.Mission) error {
	return s.missions.CreateMission(ctx, missionRecord)
}

// ListMissions는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListMissions(ctx context.Context) ([]mission.Mission, error) {
	return s.missions.ListMissions(ctx)
}

// ListMissionActivityInputs는 모든 미션의 목록 관련 장부 이벤트를 한 번의 쿼리로 읽는다.
// 전체 원장 읽기는 detail projection 경계에서만 사용한다.
func (s *Store) ListMissionActivityInputs(ctx context.Context, missionIDs []string) ([]mission.ActivityInput, error) {
	return s.missions.ListMissionActivityInputs(ctx, missionIDs)
}

// AppendLedgerEvent는 단일 장부 이벤트를 SQLite에 추가한다.
func (s *Store) AppendLedgerEvent(ctx context.Context, event ledger.Event) (ledger.Event, error) {
	return s.missions.AppendLedgerEvent(ctx, event)
}

// AppendLedgerEventsConditionally는 조건 검사를 통과한 이벤트 묶음만 SQLite에 추가한다.
func (s *Store) AppendLedgerEventsConditionally(ctx context.Context, missionID string, build func([]ledger.Event) ([]ledger.Event, error)) ([]ledger.Event, error) {
	return s.missions.AppendLedgerEventsConditionally(ctx, missionID, build)
}

// ListLedgerEvents는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListLedgerEvents(ctx context.Context, missionID string) ([]ledger.Event, error) {
	return s.missions.ListLedgerEvents(ctx, missionID)
}
