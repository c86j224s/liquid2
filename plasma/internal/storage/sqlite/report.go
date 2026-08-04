package sqlite

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// CreateReport는 새 report record를 SQLite에 저장한다.
func (s *Store) CreateReport(ctx context.Context, report app.Report) error {
	return s.reports.CreateReport(ctx, report)
}

// GetReport는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetReport(ctx context.Context, reportID string) (app.Report, error) {
	return s.reports.GetReport(ctx, reportID)
}

// ListReports는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListReports(ctx context.Context, missionID string) ([]app.Report, error) {
	return s.reports.ListReports(ctx, missionID)
}

// CreateReportVersion는 report artifact의 새 version record를 저장한다.
func (s *Store) CreateReportVersion(ctx context.Context, version app.ReportVersion, blocks []app.ReportBlock) error {
	return s.reports.CreateReportVersion(ctx, version, blocks)
}

// GetReportVersion는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetReportVersion(ctx context.Context, versionID string) (app.ReportVersion, error) {
	return s.reports.GetReportVersion(ctx, versionID)
}

// ListReportVersions는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListReportVersions(ctx context.Context, missionID string) ([]app.ReportVersion, error) {
	return s.reports.ListReportVersions(ctx, missionID)
}

// ListReportBlocks는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListReportBlocks(ctx context.Context, versionID string) ([]app.ReportBlock, error) {
	return s.reports.ListReportBlocks(ctx, versionID)
}

// PromoteReportVersion는 선택한 report version을 현재 version으로 승격한다.
func (s *Store) PromoteReportVersion(ctx context.Context, update app.ReportVersionPromotion) error {
	return s.reports.PromoteReportVersion(ctx, update)
}
