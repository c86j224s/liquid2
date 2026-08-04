package researchrepo

import (
	"context"
	"database/sql"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// CreateClaimRecord stores one claim record.
func (r *Repository) CreateClaimRecord(ctx context.Context, record app.ClaimRecord) error {
	return InsertClaimRecordTx(ctx, r.db, record)
}

// GetClaimRecord reads one claim record by stable ID.
func (r *Repository) GetClaimRecord(ctx context.Context, claimID string) (app.ClaimRecord, error) {
	var record app.ClaimRecord
	var supportingJSON string
	var opposingJSON string
	var questionJSON string
	var confidenceJSON string
	var approvalJSON string
	var createdAt string
	err := r.db.QueryRowContext(ctx, `
SELECT claim_id, schema_version, object_kind, mission_id, state, text, claim_type,
       supporting_evidence_ids_json, opposing_evidence_ids_json,
       depends_on_question_ids_json, user_assertion_event_id, confidence_json,
       approval_json, created_event_id, created_at
FROM plasma_claim_records
WHERE claim_id = ?`, claimID).Scan(
		&record.ClaimID,
		&record.SchemaVersion,
		&record.ObjectKind,
		&record.MissionID,
		&record.State,
		&record.Text,
		&record.ClaimType,
		&supportingJSON,
		&opposingJSON,
		&questionJSON,
		&record.UserAssertionEventID,
		&confidenceJSON,
		&approvalJSON,
		&record.CreatedEventID,
		&createdAt)
	if err != nil {
		return app.ClaimRecord{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(supportingJSON, &record.SupportingEvidenceIDs); err != nil {
		return app.ClaimRecord{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(opposingJSON, &record.OpposingEvidenceIDs); err != nil {
		return app.ClaimRecord{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(questionJSON, &record.DependsOnQuestionIDs); err != nil {
		return app.ClaimRecord{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(confidenceJSON, &record.Confidence); err != nil {
		return app.ClaimRecord{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(approvalJSON, &record.Approval); err != nil {
		return app.ClaimRecord{}, err
	}
	record.CreatedAt, err = parseRequiredTime(createdAt)
	if err != nil {
		return app.ClaimRecord{}, err
	}
	return record, nil
}

// ListClaimRecords reads mission claim records ordered by creation time.
func (r *Repository) ListClaimRecords(ctx context.Context, missionID string) ([]app.ClaimRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT claim_id
FROM plasma_claim_records
WHERE mission_id = ?
ORDER BY created_at DESC, claim_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []app.ClaimRecord
	for rows.Next() {
		var claimID string
		if err := rows.Scan(&claimID); err != nil {
			return nil, err
		}
		record, err := r.GetClaimRecord(ctx, claimID)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

// InsertClaimRecordTx inserts a claim inside a caller-owned transaction or queryer.
func InsertClaimRecordTx(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, record app.ClaimRecord) error {
	supportingJSON, err := sqlitevalue.MarshalJSON(record.SupportingEvidenceIDs)
	if err != nil {
		return err
	}
	opposingJSON, err := sqlitevalue.MarshalJSON(record.OpposingEvidenceIDs)
	if err != nil {
		return err
	}
	questionJSON, err := sqlitevalue.MarshalJSON(record.DependsOnQuestionIDs)
	if err != nil {
		return err
	}
	confidenceJSON, err := sqlitevalue.MarshalJSON(record.Confidence)
	if err != nil {
		return err
	}
	approvalJSON, err := sqlitevalue.MarshalJSON(record.Approval)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO plasma_claim_records (
  claim_id, schema_version, object_kind, mission_id, state, text, claim_type,
  supporting_evidence_ids_json, opposing_evidence_ids_json,
  depends_on_question_ids_json, user_assertion_event_id, confidence_json,
  approval_json, created_event_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ClaimID,
		record.SchemaVersion,
		record.ObjectKind,
		record.MissionID,
		record.State,
		record.Text,
		record.ClaimType,
		supportingJSON,
		opposingJSON,
		questionJSON,
		record.UserAssertionEventID,
		confidenceJSON,
		approvalJSON,
		record.CreatedEventID,
		sqlitevalue.FormatTime(record.CreatedAt))
	return err
}
