package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/confluenceaccess"
)

// ConfluenceConnectionStore는 Confluence 연결 저장과 조회를 Service 뒤에 숨기는 저장소 포트다.
type ConfluenceConnectionStore = confluenceaccess.ConnectionStore

// UpsertConfluenceConnection는 Confluence 연결 설정을 저장하거나 갱신한다.
func (s *Service) UpsertConfluenceConnection(ctx context.Context, req confluenceaccess.UpsertRequest) (confluenceaccess.Connection, error) {
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	connection, err := confluenceaccess.NormalizeConfluenceConnection(req)
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	now := time.Now().UTC()
	if connection.CreatedAt.IsZero() {
		connection.CreatedAt = now
	}
	connection.UpdatedAt = now
	if err := store.UpsertConfluenceConnection(ctx, connection); err != nil {
		return confluenceaccess.Connection{}, err
	}
	return connection, nil
}

// GetConfluenceConnection는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetConfluenceConnection(ctx context.Context, connectionID string) (confluenceaccess.Connection, error) {
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	trimmed := strings.TrimSpace(connectionID)
	if err := validateID("cnf_", trimmed); err != nil {
		return confluenceaccess.Connection{}, err
	}
	return store.GetConfluenceConnection(ctx, trimmed)
}

// ListConfluenceConnections는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListConfluenceConnections(ctx context.Context) ([]confluenceaccess.Connection, error) {
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return nil, err
	}
	return store.ListConfluenceConnections(ctx)
}

// DeleteConfluenceConnection는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) DeleteConfluenceConnection(ctx context.Context, connectionID string) error {
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(connectionID)
	if err := validateID("cnf_", trimmed); err != nil {
		return err
	}
	return store.DeleteConfluenceConnection(ctx, trimmed)
}

// RenameConfluenceConnection는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) RenameConfluenceConnection(ctx context.Context, connectionID string, displayName string) (confluenceaccess.Connection, error) {
	connection, err := s.GetConfluenceConnection(ctx, connectionID)
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return confluenceaccess.Connection{}, fmt.Errorf("%w: confluence display name is required", ErrInvalidInput)
	}
	connection.DisplayName = displayName
	connection.UpdatedAt = time.Now().UTC()
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	if err := store.UpsertConfluenceConnection(ctx, connection); err != nil {
		return confluenceaccess.Connection{}, err
	}
	return connection, nil
}

// RevokeConfluenceConnection는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) RevokeConfluenceConnection(ctx context.Context, connectionID string) (confluenceaccess.Connection, error) {
	connection, err := s.GetConfluenceConnection(ctx, connectionID)
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	connection.AccessToken = ""
	connection.RefreshToken = ""
	connection.TokenExpiresAt = time.Time{}
	connection.Revoked = true
	connection.UpdatedAt = time.Now().UTC()
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	if err := store.UpsertConfluenceConnection(ctx, connection); err != nil {
		return confluenceaccess.Connection{}, err
	}
	return connection, nil
}

// RefreshConfluenceConnectionSites는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) RefreshConfluenceConnectionSites(
	ctx context.Context,
	connectionID string,
	lister confluenceaccess.SiteLister,
) (confluenceaccess.Connection, error) {
	if lister == nil {
		return confluenceaccess.Connection{}, fmt.Errorf("%w: confluence site lister is required", ErrInvalidInput)
	}
	connection, err := s.GetConfluenceConnection(ctx, connectionID)
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	result, err := lister.ListConfluenceSites(ctx)
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	sites, err := confluenceaccess.NormalizeConfluenceSites(result.Sites, connection.AuthType)
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	connection.Sites = sites
	connection.UpdatedAt = time.Now().UTC()
	store, err := s.confluenceConnectionStore()
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	if err := store.UpsertConfluenceConnection(ctx, connection); err != nil {
		return confluenceaccess.Connection{}, err
	}
	return connection, nil
}

func (s *Service) confluenceConnectionStore() (ConfluenceConnectionStore, error) {
	store, ok := s.store.(ConfluenceConnectionStore)
	if !ok {
		return nil, fmt.Errorf("%w: confluence connection store is required", ErrInvalidInput)
	}
	return store, nil
}
