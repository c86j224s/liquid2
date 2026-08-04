package reportrepo

import (
	"context"
	"database/sql"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// PromoteReportVersion promotes a report version with legacy RowsAffected semantics.
func (r *Repository) PromoteReportVersion(ctx context.Context, update app.ReportVersionPromotion) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
UPDATE plasma_report_versions
SET state = ?
WHERE report_version_id = ?
  AND state = ?`,
		update.ToState,
		update.ReportVersionID,
		update.FromState)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	result, err = tx.ExecContext(ctx, `
UPDATE plasma_reports
SET state = ?,
    active_version_id = ?,
    updated_at = ?
WHERE report_id = ?`,
		update.ReportState,
		update.ReportVersionID,
		sqlitevalue.FormatTime(update.UpdatedAt),
		update.ReportID)
	if err != nil {
		return err
	}
	affected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}
