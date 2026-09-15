package app

import (
	"context"
	"strings"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

// CreateRawArtifact는 원문 artifact를 저장하고 저장 record를 반환한다.
func (s *Service) CreateRawArtifact(ctx context.Context, req artifactcontract.CreateRequest) (artifactcontract.Raw, error) {
	built, err := artifactcontract.Build(req)
	if err != nil {
		return artifactcontract.Raw{}, err
	}
	if err := s.store.CreateRawArtifact(ctx, built); err != nil {
		return artifactcontract.Raw{}, err
	}
	return built, nil
}

// GetRawArtifact는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetRawArtifact(ctx context.Context, artifactID string) (artifactcontract.Raw, error) {
	trimmed := strings.TrimSpace(artifactID)
	if err := validateID("art_", trimmed); err != nil {
		return artifactcontract.Raw{}, err
	}
	return s.store.GetRawArtifact(ctx, trimmed)
}

// CreateSourceSnapshot는 source snapshot record를 저장하고 반환한다.
func (s *Service) CreateSourceSnapshot(ctx context.Context, req sourcecontract.CreateRequest) (sourcecontract.Snapshot, error) {
	snapshot, err := source.BuildSnapshot(ctx, s.store, req, nil)
	if err != nil {
		return sourcecontract.Snapshot{}, err
	}
	if err := s.store.CreateSourceSnapshot(ctx, snapshot); err != nil {
		return sourcecontract.Snapshot{}, err
	}
	return snapshot, nil
}

// GetSourceSnapshot는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetSourceSnapshot(ctx context.Context, snapshotID string) (sourcecontract.Snapshot, error) {
	trimmed := strings.TrimSpace(snapshotID)
	if err := validateID("src_", trimmed); err != nil {
		return sourcecontract.Snapshot{}, err
	}
	snapshot, err := s.store.GetSourceSnapshot(ctx, trimmed)
	if err != nil {
		return sourcecontract.Snapshot{}, err
	}
	state, err := s.sourceState(ctx, snapshot.MissionID, snapshot.SnapshotID)
	if err != nil {
		return sourcecontract.Snapshot{}, err
	}
	snapshot.State = state
	return snapshot, nil
}

// ListSourceSnapshots는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListSourceSnapshots(ctx context.Context, missionID string) ([]sourcecontract.Snapshot, error) {
	return s.ListSourceSnapshotsWithState(ctx, sourcecontract.ListRequest{MissionID: missionID})
}
