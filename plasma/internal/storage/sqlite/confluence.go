package sqlite

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/confluenceaccess"
)

// UpsertConfluenceConnection는 Confluence 연결 설정을 저장하거나 갱신한다.
func (s *Store) UpsertConfluenceConnection(ctx context.Context, connection confluenceaccess.Connection) error {
	return s.confluence.UpsertConfluenceConnection(ctx, connection)
}

// GetConfluenceConnection는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetConfluenceConnection(ctx context.Context, connectionID string) (confluenceaccess.Connection, error) {
	return s.confluence.GetConfluenceConnection(ctx, connectionID)
}

// ListConfluenceConnections는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListConfluenceConnections(ctx context.Context) ([]confluenceaccess.Connection, error) {
	return s.confluence.ListConfluenceConnections(ctx)
}

// DeleteConfluenceConnection는 SQLite 저장소 어댑터의 명시적 상태 전이를 수행한다. 결과는 SQLite 기록으로 확인한다.
func (s *Store) DeleteConfluenceConnection(ctx context.Context, connectionID string) error {
	return s.confluence.DeleteConfluenceConnection(ctx, connectionID)
}
