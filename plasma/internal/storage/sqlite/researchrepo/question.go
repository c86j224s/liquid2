package researchrepo

import (
	"context"
	"database/sql"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// CreateQuestionRecord stores one question record.
func (r *Repository) CreateQuestionRecord(ctx context.Context, record app.QuestionRecord) error {
	return InsertQuestionRecordTx(ctx, r.db, record)
}

// GetQuestionRecord reads one question record by stable ID.
func (r *Repository) GetQuestionRecord(ctx context.Context, questionID string) (app.QuestionRecord, error) {
	var record app.QuestionRecord
	var blocking int
	var evidenceJSON string
	var claimJSON string
	var createdAt string
	err := r.db.QueryRowContext(ctx, `
SELECT question_id, schema_version, object_kind, mission_id, state, text,
       priority, blocking, related_evidence_ids_json, related_claim_ids_json,
       resolution, created_event_id, created_at
FROM plasma_question_records
WHERE question_id = ?`, questionID).Scan(
		&record.QuestionID,
		&record.SchemaVersion,
		&record.ObjectKind,
		&record.MissionID,
		&record.State,
		&record.Text,
		&record.Priority,
		&blocking,
		&evidenceJSON,
		&claimJSON,
		&record.Resolution,
		&record.CreatedEventID,
		&createdAt)
	if err != nil {
		return app.QuestionRecord{}, err
	}
	record.Blocking = blocking != 0
	if err := sqlitevalue.UnmarshalJSON(evidenceJSON, &record.RelatedEvidenceIDs); err != nil {
		return app.QuestionRecord{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(claimJSON, &record.RelatedClaimIDs); err != nil {
		return app.QuestionRecord{}, err
	}
	record.CreatedAt, err = parseRequiredTime(createdAt)
	if err != nil {
		return app.QuestionRecord{}, err
	}
	return record, nil
}

// ListQuestionRecords reads mission question records ordered by creation time.
func (r *Repository) ListQuestionRecords(ctx context.Context, missionID string) ([]app.QuestionRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT question_id
FROM plasma_question_records
WHERE mission_id = ?
ORDER BY created_at DESC, question_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []app.QuestionRecord
	for rows.Next() {
		var questionID string
		if err := rows.Scan(&questionID); err != nil {
			return nil, err
		}
		record, err := r.GetQuestionRecord(ctx, questionID)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

// InsertQuestionRecordTx inserts a question inside a caller-owned transaction or queryer.
func InsertQuestionRecordTx(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, record app.QuestionRecord) error {
	evidenceJSON, err := sqlitevalue.MarshalJSON(record.RelatedEvidenceIDs)
	if err != nil {
		return err
	}
	claimJSON, err := sqlitevalue.MarshalJSON(record.RelatedClaimIDs)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO plasma_question_records (
  question_id, schema_version, object_kind, mission_id, state, text, priority,
  blocking, related_evidence_ids_json, related_claim_ids_json, resolution,
  created_event_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.QuestionID,
		record.SchemaVersion,
		record.ObjectKind,
		record.MissionID,
		record.State,
		record.Text,
		record.Priority,
		sqlitevalue.BoolInt(record.Blocking),
		evidenceJSON,
		claimJSON,
		record.Resolution,
		record.CreatedEventID,
		sqlitevalue.FormatTime(record.CreatedAt))
	return err
}
