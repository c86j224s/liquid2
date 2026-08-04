package sqlite

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// GetModelDefaults는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetModelDefaults(ctx context.Context) (app.ModelDefaults, error) {
	return s.modelDefaults.GetModelDefaults(ctx)
}

// SaveModelDefaults는 모델 기본값 설정을 SQLite에 저장한다.
func (s *Store) SaveModelDefaults(ctx context.Context, defaults app.ModelDefaults) error {
	return s.modelDefaults.SaveModelDefaults(ctx, defaults)
}
