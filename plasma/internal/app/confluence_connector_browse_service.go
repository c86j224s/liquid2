package app

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"strings"
)

// SearchConfluenceSources는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) SearchConfluenceSources(
	ctx context.Context,
	connector confluencesource.ConfluenceSourceConnector,
	req confluencesource.ConfluenceSourceSearchRequest,
) (confluencesource.ConfluenceSourceSearchResult, error) {
	if connector == nil {
		return confluencesource.ConfluenceSourceSearchResult{}, fmt.Errorf("%w: confluence connector is required", ErrInvalidInput)
	}
	normalized, err := confluencesource.NormalizeSearchRequest(req)
	if err != nil {
		return confluencesource.ConfluenceSourceSearchResult{}, err
	}
	result, err := connector.SearchConfluenceSources(ctx, normalized)
	if err != nil {
		return confluencesource.ConfluenceSourceSearchResult{}, err
	}
	result.MissionID = normalized.MissionID
	result.CloudID = normalized.CloudID
	result.NextCursor = strings.TrimSpace(result.NextCursor)
	for i := range result.Candidates {
		candidate, err := confluencesource.NormalizeCandidate(result.Candidates[i], normalized.CloudID)
		if err != nil {
			return confluencesource.ConfluenceSourceSearchResult{}, err
		}
		result.Candidates[i] = candidate
	}
	return result, nil
}

// ListConfluenceSpaces는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListConfluenceSpaces(
	ctx context.Context,
	connector confluencesource.ConfluenceBrowserConnector,
	req confluencesource.ConfluenceSpaceListRequest,
) (confluencesource.ConfluenceSpaceListResult, error) {
	if connector == nil {
		return confluencesource.ConfluenceSpaceListResult{}, fmt.Errorf("%w: confluence browser connector is required", ErrInvalidInput)
	}
	normalized, err := confluencesource.NormalizeSpaceListRequest(req)
	if err != nil {
		return confluencesource.ConfluenceSpaceListResult{}, err
	}
	result, err := connector.ListConfluenceSpaces(ctx, normalized)
	if err != nil {
		return confluencesource.ConfluenceSpaceListResult{}, err
	}
	result.MissionID = normalized.MissionID
	result.CloudID = normalized.CloudID
	result.NextCursor = strings.TrimSpace(result.NextCursor)
	for i := range result.Spaces {
		result.Spaces[i] = confluencesource.NormalizeSpaceSummary(result.Spaces[i], normalized.CloudID)
	}
	return result, nil
}

// ListConfluenceSpacePages는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListConfluenceSpacePages(
	ctx context.Context,
	connector confluencesource.ConfluenceBrowserConnector,
	req confluencesource.ConfluenceSpacePagesRequest,
) (confluencesource.ConfluencePageListResult, error) {
	if connector == nil {
		return confluencesource.ConfluencePageListResult{}, fmt.Errorf("%w: confluence browser connector is required", ErrInvalidInput)
	}
	normalized, err := confluencesource.NormalizeSpacePagesRequest(req)
	if err != nil {
		return confluencesource.ConfluencePageListResult{}, err
	}
	result, err := connector.ListConfluenceSpacePages(ctx, normalized)
	if err != nil {
		return confluencesource.ConfluencePageListResult{}, err
	}
	result.MissionID = normalized.MissionID
	result.CloudID = normalized.CloudID
	result.NextCursor = strings.TrimSpace(result.NextCursor)
	for i := range result.Pages {
		result.Pages[i] = confluencesource.NormalizePageSummary(result.Pages[i], normalized.CloudID)
	}
	return result, nil
}

// ListConfluencePageChildren는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListConfluencePageChildren(
	ctx context.Context,
	connector confluencesource.ConfluenceBrowserConnector,
	req confluencesource.ConfluencePageChildrenRequest,
) (confluencesource.ConfluencePageListResult, error) {
	if connector == nil {
		return confluencesource.ConfluencePageListResult{}, fmt.Errorf("%w: confluence browser connector is required", ErrInvalidInput)
	}
	normalized, err := confluencesource.NormalizePageChildrenRequest(req)
	if err != nil {
		return confluencesource.ConfluencePageListResult{}, err
	}
	result, err := connector.ListConfluencePageChildren(ctx, normalized)
	if err != nil {
		return confluencesource.ConfluencePageListResult{}, err
	}
	result.MissionID = normalized.MissionID
	result.CloudID = normalized.CloudID
	result.NextCursor = strings.TrimSpace(result.NextCursor)
	for i := range result.Pages {
		result.Pages[i] = confluencesource.NormalizePageSummary(result.Pages[i], normalized.CloudID)
	}
	return result, nil
}
