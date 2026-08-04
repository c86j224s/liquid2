package researchrepo

import (
	"context"
	"database/sql"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// CreateEvidenceRecord stores one evidence record.
func (r *Repository) CreateEvidenceRecord(ctx context.Context, record app.EvidenceRecord) error {
	return InsertEvidenceRecordTx(ctx, r.db, record)
}

// GetEvidenceRecord reads one evidence record by stable ID.
func (r *Repository) GetEvidenceRecord(ctx context.Context, evidenceID string) (app.EvidenceRecord, error) {
	var record app.EvidenceRecord
	var snapshotRefsJSON string
	var confidenceJSON string
	var createdAt string
	err := r.db.QueryRowContext(ctx, `
SELECT evidence_id, schema_version, object_kind, mission_id, state, summary,
       evidence_type, snapshot_refs_json, confidence_json, producer_type,
       producer_id, created_event_id, created_at
FROM plasma_evidence_records
WHERE evidence_id = ?`, evidenceID).Scan(
		&record.EvidenceID,
		&record.SchemaVersion,
		&record.ObjectKind,
		&record.MissionID,
		&record.State,
		&record.Summary,
		&record.EvidenceType,
		&snapshotRefsJSON,
		&confidenceJSON,
		&record.Producer.Type,
		&record.Producer.ID,
		&record.CreatedEventID,
		&createdAt)
	if err != nil {
		return app.EvidenceRecord{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(snapshotRefsJSON, &record.SnapshotRefs); err != nil {
		return app.EvidenceRecord{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(confidenceJSON, &record.Confidence); err != nil {
		return app.EvidenceRecord{}, err
	}
	record.CreatedAt, err = parseRequiredTime(createdAt)
	if err != nil {
		return app.EvidenceRecord{}, err
	}
	return record, nil
}

// ListEvidenceRecords reads mission evidence records ordered by creation time.
func (r *Repository) ListEvidenceRecords(ctx context.Context, missionID string) ([]app.EvidenceRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT evidence_id
FROM plasma_evidence_records
WHERE mission_id = ?
ORDER BY created_at DESC, evidence_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []app.EvidenceRecord
	for rows.Next() {
		var evidenceID string
		if err := rows.Scan(&evidenceID); err != nil {
			return nil, err
		}
		record, err := r.GetEvidenceRecord(ctx, evidenceID)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

// InsertEvidenceRecordTx inserts evidence inside a caller-owned transaction or queryer.
func InsertEvidenceRecordTx(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, record app.EvidenceRecord) error {
	snapshotRefsJSON, err := sqlitevalue.MarshalJSON(record.SnapshotRefs)
	if err != nil {
		return err
	}
	confidenceJSON, err := sqlitevalue.MarshalJSON(record.Confidence)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO plasma_evidence_records (
  evidence_id, schema_version, object_kind, mission_id, state, summary,
  evidence_type, snapshot_refs_json, confidence_json, producer_type,
  producer_id, created_event_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.EvidenceID,
		record.SchemaVersion,
		record.ObjectKind,
		record.MissionID,
		record.State,
		record.Summary,
		record.EvidenceType,
		snapshotRefsJSON,
		confidenceJSON,
		record.Producer.Type,
		record.Producer.ID,
		record.CreatedEventID,
		sqlitevalue.FormatTime(record.CreatedAt))
	return err
}
