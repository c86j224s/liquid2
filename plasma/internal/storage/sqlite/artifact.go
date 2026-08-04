package sqlite

import (
	"context"
	"time"

	artifactmodel "github.com/c86j224s/liquid2/plasma/internal/artifact"
	sourcemodel "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// CreateRawArtifact는 원문 artifact를 저장하고 저장 record를 반환한다.
func (s *Store) CreateRawArtifact(ctx context.Context, artifact artifactmodel.Raw) error {
	return s.artifacts.CreateRawArtifact(ctx, artifact)
}

// GetRawArtifact는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetRawArtifact(ctx context.Context, artifactID string) (artifactmodel.Raw, error) {
	return s.artifacts.GetRawArtifact(ctx, artifactID)
}

// ListRawArtifacts는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListRawArtifacts(ctx context.Context, missionID string) ([]artifactmodel.Raw, error) {
	return s.artifacts.ListRawArtifacts(ctx, missionID)
}

// CreateSourceSnapshot는 source snapshot record를 저장하고 반환한다.
func (s *Store) CreateSourceSnapshot(ctx context.Context, snapshot sourcemodel.Snapshot) error {
	return s.artifacts.CreateSourceSnapshot(ctx, snapshot)
}

// GetSourceSnapshot는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetSourceSnapshot(ctx context.Context, snapshotID string) (sourcemodel.Snapshot, error) {
	return s.artifacts.GetSourceSnapshot(ctx, snapshotID)
}

// ListSourceSnapshots는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListSourceSnapshots(ctx context.Context, missionID string) ([]sourcemodel.Snapshot, error) {
	return s.artifacts.ListSourceSnapshots(ctx, missionID)
}

func formatOptionalTime(t time.Time) string {
	return sqlitevalue.FormatOptionalTime(t)
}

func formatTime(t time.Time) string {
	return sqlitevalue.FormatTime(t)
}
