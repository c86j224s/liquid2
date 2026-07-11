package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (s *Store) CreateReport(ctx context.Context, report app.Report) error {
	_, err := s.db.ExecContext(ctx, `
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
		formatTime(report.CreatedAt),
		formatTime(report.UpdatedAt))
	return err
}

func (s *Store) GetReport(ctx context.Context, reportID string) (app.Report, error) {
	var report app.Report
	var createdAt string
	var updatedAt string
	err := s.db.QueryRowContext(ctx, `
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

func (s *Store) ListReports(ctx context.Context, missionID string) ([]app.Report, error) {
	rows, err := s.db.QueryContext(ctx, `
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
		report, err := s.GetReport(ctx, reportID)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

func (s *Store) CreateReportVersion(ctx context.Context, version app.ReportVersion, blocks []app.ReportBlock) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := insertReportVersionTx(ctx, tx, version); err != nil {
		return err
	}
	for _, block := range blocks {
		if err := insertReportBlockTx(ctx, tx, block); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetReportVersion(ctx context.Context, versionID string) (app.ReportVersion, error) {
	var version app.ReportVersion
	var blockIDsJSON string
	var scopeJSON string
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
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
	if err := unmarshalJSON(blockIDsJSON, &version.BlockIDs); err != nil {
		return app.ReportVersion{}, err
	}
	if err := unmarshalJSON(scopeJSON, &version.IncludedEvidenceScope); err != nil {
		return app.ReportVersion{}, err
	}
	var parseErr error
	version.CreatedAt, parseErr = parseRequiredTime(createdAt)
	if parseErr != nil {
		return app.ReportVersion{}, parseErr
	}
	return version, nil
}

func (s *Store) ListReportVersions(ctx context.Context, missionID string) ([]app.ReportVersion, error) {
	rows, err := s.db.QueryContext(ctx, `
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
		version, err := s.GetReportVersion(ctx, versionID)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *Store) ListReportBlocks(ctx context.Context, versionID string) ([]app.ReportBlock, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT block_id, schema_version, object_kind, report_version_id, mission_id,
       block_type, parent_block_id, block_order, content_json, source_refs_json,
       authorship_json, approval_json
FROM plasma_report_blocks
WHERE report_version_id = ?
ORDER BY block_order, block_id`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []app.ReportBlock
	for rows.Next() {
		var block app.ReportBlock
		var contentJSON string
		var refsJSON string
		var authorshipJSON string
		var approvalJSON string
		if err := rows.Scan(
			&block.BlockID,
			&block.SchemaVersion,
			&block.ObjectKind,
			&block.ReportVersionID,
			&block.MissionID,
			&block.BlockType,
			&block.ParentBlockID,
			&block.Order,
			&contentJSON,
			&refsJSON,
			&authorshipJSON,
			&approvalJSON); err != nil {
			return nil, err
		}
		block.Content = append([]byte(nil), contentJSON...)
		if err := unmarshalJSON(refsJSON, &block.SourceRefs); err != nil {
			return nil, err
		}
		if err := unmarshalJSON(authorshipJSON, &block.Authorship); err != nil {
			return nil, err
		}
		if err := unmarshalJSON(approvalJSON, &block.Approval); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, rows.Err()
}

func (s *Store) PromoteReportVersion(ctx context.Context, update app.ReportVersionPromotion) error {
	tx, err := s.db.BeginTx(ctx, nil)
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
		formatTime(update.UpdatedAt),
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

func insertReportTx(ctx context.Context, tx *sql.Tx, report app.Report) error {
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
		formatTime(report.CreatedAt),
		formatTime(report.UpdatedAt))
	return err
}

func insertReportVersionTx(ctx context.Context, tx *sql.Tx, version app.ReportVersion) error {
	blockIDsJSON, err := marshalJSON(version.BlockIDs)
	if err != nil {
		return err
	}
	scopeJSON, err := marshalJSON(version.IncludedEvidenceScope)
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
		formatTime(version.CreatedAt))
	return err
}

func insertReportBlockTx(ctx context.Context, tx *sql.Tx, block app.ReportBlock) error {
	contentJSON := string(block.Content)
	if contentJSON == "" {
		contentJSON = "{}"
	}
	if !json.Valid([]byte(contentJSON)) {
		return app.ErrInvalidInput
	}
	refsJSON, err := marshalJSON(block.SourceRefs)
	if err != nil {
		return err
	}
	authorshipJSON, err := marshalJSON(block.Authorship)
	if err != nil {
		return err
	}
	approvalJSON, err := marshalJSON(block.Approval)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO plasma_report_blocks (
  block_id, schema_version, object_kind, report_version_id, mission_id,
  block_type, parent_block_id, block_order, content_json, source_refs_json,
  authorship_json, approval_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		block.BlockID,
		block.SchemaVersion,
		block.ObjectKind,
		block.ReportVersionID,
		block.MissionID,
		block.BlockType,
		block.ParentBlockID,
		block.Order,
		contentJSON,
		refsJSON,
		authorshipJSON,
		approvalJSON)
	return err
}
