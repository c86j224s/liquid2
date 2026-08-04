package reportrepo

import (
	"context"
	"database/sql"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// CreateReport stores one report row.
func (r *Repository) CreateReport(ctx context.Context, report app.Report) error {
	return InsertReportTx(ctx, r.db, report)
}

// GetReport reads one report by stable ID.
func (r *Repository) GetReport(ctx context.Context, reportID string) (app.Report, error) {
	var report app.Report
	var createdAt string
	var updatedAt string
	err := r.db.QueryRowContext(ctx, `
SELECT report_id, schema_version, object_kind, mission_id, title,
       active_version_id, state, created_at, updated_at
FROM plasma_reports
WHERE report_id = ?`, reportID).Scan(
		&report.ReportID,
		&report.SchemaVersion,
		&report.ObjectKind,
		&report.MissionID,
		&report.Title,
		&report.ActiveVersionID,
		&report.State,
		&createdAt,
		&updatedAt)
	if err != nil {
		return app.Report{}, err
	}
	var parseErr error
	report.CreatedAt, parseErr = parseRequiredTime(createdAt)
	if parseErr != nil {
		return app.Report{}, parseErr
	}
	report.UpdatedAt, parseErr = parseRequiredTime(updatedAt)
	if parseErr != nil {
		return app.Report{}, parseErr
	}
	return report, nil
}

// ListReports reads mission reports ordered by creation time.
func (r *Repository) ListReports(ctx context.Context, missionID string) ([]app.Report, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT report_id
FROM plasma_reports
WHERE mission_id = ?
ORDER BY created_at DESC, report_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []app.Report
	for rows.Next() {
		var reportID string
		if err := rows.Scan(&reportID); err != nil {
			return nil, err
		}
		report, err := r.GetReport(ctx, reportID)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

// InsertReportTx inserts a report inside a caller-owned transaction or queryer.
func InsertReportTx(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, report app.Report) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO plasma_reports (
  report_id, schema_version, object_kind, mission_id, title, active_version_id,
  state, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ReportID,
		report.SchemaVersion,
		report.ObjectKind,
		report.MissionID,
		report.Title,
		report.ActiveVersionID,
		report.State,
		sqlitevalue.FormatTime(report.CreatedAt),
		sqlitevalue.FormatTime(report.UpdatedAt))
	return err
}
