package reportrepo

import (
	"context"
	"database/sql"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// CreateReportVersion stores a version and its blocks in one transaction.
func (r *Repository) CreateReportVersion(ctx context.Context, version app.ReportVersion, blocks []app.ReportBlock) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := InsertReportVersionTx(ctx, tx, version); err != nil {
		return err
	}
	for _, block := range blocks {
		if err := InsertReportBlockTx(ctx, tx, block); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetReportVersion reads one report version by stable ID.
func (r *Repository) GetReportVersion(ctx context.Context, versionID string) (app.ReportVersion, error) {
	var version app.ReportVersion
	var blockIDsJSON string
	var scopeJSON string
	var createdAt string
	err := r.db.QueryRowContext(ctx, `
SELECT report_version_id, schema_version, object_kind, report_id, mission_id,
       base_version_id, state, root_block_id, block_ids_json,
       included_evidence_scope_json, created_event_id, created_at
FROM plasma_report_versions
WHERE report_version_id = ?`, versionID).Scan(
		&version.ReportVersionID,
		&version.SchemaVersion,
		&version.ObjectKind,
		&version.ReportID,
		&version.MissionID,
		&version.BaseVersionID,
		&version.State,
		&version.RootBlockID,
		&blockIDsJSON,
		&scopeJSON,
		&version.CreatedEventID,
		&createdAt)
	if err != nil {
		return app.ReportVersion{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(blockIDsJSON, &version.BlockIDs); err != nil {
		return app.ReportVersion{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(scopeJSON, &version.IncludedEvidenceScope); err != nil {
		return app.ReportVersion{}, err
	}
	var parseErr error
	version.CreatedAt, parseErr = parseRequiredTime(createdAt)
	if parseErr != nil {
		return app.ReportVersion{}, parseErr
	}
	return version, nil
}

// ListReportVersions reads mission report versions ordered by creation time.
func (r *Repository) ListReportVersions(ctx context.Context, missionID string) ([]app.ReportVersion, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT report_version_id
FROM plasma_report_versions
WHERE mission_id = ?
ORDER BY created_at DESC, report_version_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []app.ReportVersion
	for rows.Next() {
		var versionID string
		if err := rows.Scan(&versionID); err != nil {
			return nil, err
		}
		version, err := r.GetReportVersion(ctx, versionID)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

// InsertReportVersionTx inserts a report version inside a caller-owned transaction or queryer.
func InsertReportVersionTx(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, version app.ReportVersion) error {
	blockIDsJSON, err := sqlitevalue.MarshalJSON(version.BlockIDs)
	if err != nil {
		return err
	}
	scopeJSON, err := sqlitevalue.MarshalJSON(version.IncludedEvidenceScope)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO plasma_report_versions (
  report_version_id, schema_version, object_kind, report_id, mission_id,
  base_version_id, state, root_block_id, block_ids_json,
  included_evidence_scope_json, created_event_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		version.ReportVersionID,
		version.SchemaVersion,
		version.ObjectKind,
		version.ReportID,
		version.MissionID,
		version.BaseVersionID,
		version.State,
		version.RootBlockID,
		blockIDsJSON,
		scopeJSON,
		version.CreatedEventID,
		sqlitevalue.FormatTime(version.CreatedAt))
	return err
}
