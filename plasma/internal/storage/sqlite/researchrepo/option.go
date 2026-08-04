package researchrepo

import (
	"context"
	"database/sql"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// CreateOptionRecord stores one option record.
func (r *Repository) CreateOptionRecord(ctx context.Context, record app.OptionRecord) error {
	return InsertOptionRecordTx(ctx, r.db, record)
}

// GetOptionRecord reads one option record by stable ID.
func (r *Repository) GetOptionRecord(ctx context.Context, optionID string) (app.OptionRecord, error) {
	var record app.OptionRecord
	var prosJSON string
	var consJSON string
	var claimJSON string
	var createdAt string
	err := r.db.QueryRowContext(ctx, `
SELECT option_id, schema_version, object_kind, mission_id, state, title,
       description, pros_json, cons_json, supporting_claim_ids_json, risk_level,
       created_event_id, created_at
FROM plasma_option_records
WHERE option_id = ?`, optionID).Scan(
		&record.OptionID,
		&record.SchemaVersion,
		&record.ObjectKind,
		&record.MissionID,
		&record.State,
		&record.Title,
		&record.Description,
		&prosJSON,
		&consJSON,
		&claimJSON,
		&record.RiskLevel,
		&record.CreatedEventID,
		&createdAt)
	if err != nil {
		return app.OptionRecord{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(prosJSON, &record.Pros); err != nil {
		return app.OptionRecord{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(consJSON, &record.Cons); err != nil {
		return app.OptionRecord{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(claimJSON, &record.SupportingClaimIDs); err != nil {
		return app.OptionRecord{}, err
	}
	record.CreatedAt, err = parseRequiredTime(createdAt)
	if err != nil {
		return app.OptionRecord{}, err
	}
	return record, nil
}

// ListOptionRecords reads mission option records ordered by creation time.
func (r *Repository) ListOptionRecords(ctx context.Context, missionID string) ([]app.OptionRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT option_id
FROM plasma_option_records
WHERE mission_id = ?
ORDER BY created_at DESC, option_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []app.OptionRecord
	for rows.Next() {
		var optionID string
		if err := rows.Scan(&optionID); err != nil {
			return nil, err
		}
		record, err := r.GetOptionRecord(ctx, optionID)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

// InsertOptionRecordTx inserts an option inside a caller-owned transaction or queryer.
func InsertOptionRecordTx(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, record app.OptionRecord) error {
	prosJSON, err := sqlitevalue.MarshalJSON(record.Pros)
	if err != nil {
		return err
	}
	consJSON, err := sqlitevalue.MarshalJSON(record.Cons)
	if err != nil {
		return err
	}
	claimJSON, err := sqlitevalue.MarshalJSON(record.SupportingClaimIDs)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO plasma_option_records (
  option_id, schema_version, object_kind, mission_id, state, title,
  description, pros_json, cons_json, supporting_claim_ids_json, risk_level,
  created_event_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.OptionID,
		record.SchemaVersion,
		record.ObjectKind,
		record.MissionID,
		record.State,
		record.Title,
		record.Description,
		prosJSON,
		consJSON,
		claimJSON,
		record.RiskLevel,
		record.CreatedEventID,
		sqlitevalue.FormatTime(record.CreatedAt))
	return err
}
