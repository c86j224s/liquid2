package sqlite

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// SaveMissionProjection는 계산된 미션 projection을 SQLite에 저장한다.
func (s *Store) SaveMissionProjection(ctx context.Context, projection mission.Projection) error {
	return s.missions.SaveMissionProjection(ctx, projection)
}

// GetMissionProjection는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetMissionProjection(ctx context.Context, missionID string) (mission.Projection, error) {
	return s.missions.GetMissionProjection(ctx, missionID)
}

func marshalJSON(value any) (string, error) {
	return sqlitevalue.MarshalJSON(value)
}

func unmarshalJSON(text string, target any) error {
	return sqlitevalue.UnmarshalJSON(text, target)
}

func boolInt(value bool) int {
	return sqlitevalue.BoolInt(value)
}
