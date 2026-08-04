package reportrepo

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// ListReportBlocks reads report blocks ordered by block order and ID.
func (r *Repository) ListReportBlocks(ctx context.Context, versionID string) ([]app.ReportBlock, error) {
	rows, err := r.db.QueryContext(ctx, `
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
		if err := sqlitevalue.UnmarshalJSON(refsJSON, &block.SourceRefs); err != nil {
			return nil, err
		}
		if err := sqlitevalue.UnmarshalJSON(authorshipJSON, &block.Authorship); err != nil {
			return nil, err
		}
		if err := sqlitevalue.UnmarshalJSON(approvalJSON, &block.Approval); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, rows.Err()
}

// InsertReportBlockTx inserts a report block inside a caller-owned transaction or queryer.
func InsertReportBlockTx(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, block app.ReportBlock) error {
	contentJSON := string(block.Content)
	if contentJSON == "" {
		contentJSON = "{}"
	}
	if !json.Valid([]byte(contentJSON)) {
		return app.ErrInvalidInput
	}
	refsJSON, err := sqlitevalue.MarshalJSON(block.SourceRefs)
	if err != nil {
		return err
	}
	authorshipJSON, err := sqlitevalue.MarshalJSON(block.Authorship)
	if err != nil {
		return err
	}
	approvalJSON, err := sqlitevalue.MarshalJSON(block.Approval)
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
