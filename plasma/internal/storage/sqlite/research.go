package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (s *Store) CreateEvidenceRecord(ctx context.Context, record app.EvidenceRecord) error {
	snapshotRefsJSON, err := marshalJSON(record.SnapshotRefs)
	if err != nil {
		return err
	}
	confidenceJSON, err := marshalJSON(record.Confidence)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
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
		formatTime(record.CreatedAt))
	return err
}

func (s *Store) GetEvidenceRecord(ctx context.Context, evidenceID string) (app.EvidenceRecord, error) {
	var record app.EvidenceRecord
	var snapshotRefsJSON string
	var confidenceJSON string
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
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
	if err := unmarshalJSON(snapshotRefsJSON, &record.SnapshotRefs); err != nil {
		return app.EvidenceRecord{}, err
	}
	if err := unmarshalJSON(confidenceJSON, &record.Confidence); err != nil {
		return app.EvidenceRecord{}, err
	}
	record.CreatedAt, err = parseRequiredTime(createdAt)
	if err != nil {
		return app.EvidenceRecord{}, err
	}
	return record, nil
}

func (s *Store) ListEvidenceRecords(ctx context.Context, missionID string) ([]app.EvidenceRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
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
		record, err := s.GetEvidenceRecord(ctx, evidenceID)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) CreateClaimRecord(ctx context.Context, record app.ClaimRecord) error {
	supportingJSON, err := marshalJSON(record.SupportingEvidenceIDs)
	if err != nil {
		return err
	}
	opposingJSON, err := marshalJSON(record.OpposingEvidenceIDs)
	if err != nil {
		return err
	}
	questionJSON, err := marshalJSON(record.DependsOnQuestionIDs)
	if err != nil {
		return err
	}
	confidenceJSON, err := marshalJSON(record.Confidence)
	if err != nil {
		return err
	}
	approvalJSON, err := marshalJSON(record.Approval)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
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
		formatTime(record.CreatedAt))
	return err
}

func (s *Store) GetClaimRecord(ctx context.Context, claimID string) (app.ClaimRecord, error) {
	var record app.ClaimRecord
	var supportingJSON string
	var opposingJSON string
	var questionJSON string
	var confidenceJSON string
	var approvalJSON string
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
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
	if err := unmarshalJSON(supportingJSON, &record.SupportingEvidenceIDs); err != nil {
		return app.ClaimRecord{}, err
	}
	if err := unmarshalJSON(opposingJSON, &record.OpposingEvidenceIDs); err != nil {
		return app.ClaimRecord{}, err
	}
	if err := unmarshalJSON(questionJSON, &record.DependsOnQuestionIDs); err != nil {
		return app.ClaimRecord{}, err
	}
	if err := unmarshalJSON(confidenceJSON, &record.Confidence); err != nil {
		return app.ClaimRecord{}, err
	}
	if err := unmarshalJSON(approvalJSON, &record.Approval); err != nil {
		return app.ClaimRecord{}, err
	}
	record.CreatedAt, err = parseRequiredTime(createdAt)
	if err != nil {
		return app.ClaimRecord{}, err
	}
	return record, nil
}

func (s *Store) ListClaimRecords(ctx context.Context, missionID string) ([]app.ClaimRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
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
		record, err := s.GetClaimRecord(ctx, claimID)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) CreateQuestionRecord(ctx context.Context, record app.QuestionRecord) error {
	evidenceJSON, err := marshalJSON(record.RelatedEvidenceIDs)
	if err != nil {
		return err
	}
	claimJSON, err := marshalJSON(record.RelatedClaimIDs)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
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
		boolInt(record.Blocking),
		evidenceJSON,
		claimJSON,
		record.Resolution,
		record.CreatedEventID,
		formatTime(record.CreatedAt))
	return err
}

func (s *Store) GetQuestionRecord(ctx context.Context, questionID string) (app.QuestionRecord, error) {
	var record app.QuestionRecord
	var blocking int
	var evidenceJSON string
	var claimJSON string
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
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
	if err := unmarshalJSON(evidenceJSON, &record.RelatedEvidenceIDs); err != nil {
		return app.QuestionRecord{}, err
	}
	if err := unmarshalJSON(claimJSON, &record.RelatedClaimIDs); err != nil {
		return app.QuestionRecord{}, err
	}
	record.CreatedAt, err = parseRequiredTime(createdAt)
	if err != nil {
		return app.QuestionRecord{}, err
	}
	return record, nil
}

func (s *Store) ListQuestionRecords(ctx context.Context, missionID string) ([]app.QuestionRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
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
		record, err := s.GetQuestionRecord(ctx, questionID)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) CreateOptionRecord(ctx context.Context, record app.OptionRecord) error {
	prosJSON, err := marshalJSON(record.Pros)
	if err != nil {
		return err
	}
	consJSON, err := marshalJSON(record.Cons)
	if err != nil {
		return err
	}
	claimJSON, err := marshalJSON(record.SupportingClaimIDs)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
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
		formatTime(record.CreatedAt))
	return err
}

func (s *Store) GetOptionRecord(ctx context.Context, optionID string) (app.OptionRecord, error) {
	var record app.OptionRecord
	var prosJSON string
	var consJSON string
	var claimJSON string
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
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
	if err := unmarshalJSON(prosJSON, &record.Pros); err != nil {
		return app.OptionRecord{}, err
	}
	if err := unmarshalJSON(consJSON, &record.Cons); err != nil {
		return app.OptionRecord{}, err
	}
	if err := unmarshalJSON(claimJSON, &record.SupportingClaimIDs); err != nil {
		return app.OptionRecord{}, err
	}
	record.CreatedAt, err = parseRequiredTime(createdAt)
	if err != nil {
		return app.OptionRecord{}, err
	}
	return record, nil
}

func (s *Store) ListOptionRecords(ctx context.Context, missionID string) ([]app.OptionRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
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
		record, err := s.GetOptionRecord(ctx, optionID)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) CreateProposalBundle(ctx context.Context, bundle app.ProposalBundle) error {
	refsJSON, err := marshalJSON(bundle.ObjectRefs)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO plasma_proposal_bundles (
  proposal_id, schema_version, object_kind, mission_id, state, title,
  object_refs_json, requested_decision, created_event_id, decision_event_id,
  created_at, decided_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bundle.ProposalID,
		bundle.SchemaVersion,
		bundle.ObjectKind,
		bundle.MissionID,
		bundle.State,
		bundle.Title,
		refsJSON,
		bundle.RequestedDecision,
		bundle.CreatedEventID,
		bundle.DecisionEventID,
		formatTime(bundle.CreatedAt),
		formatOptionalTime(bundle.DecidedAt),
		formatTime(bundle.UpdatedAt))
	return err
}

func (s *Store) GetProposalBundle(ctx context.Context, proposalID string) (app.ProposalBundle, error) {
	var bundle app.ProposalBundle
	var refsJSON string
	var createdAt string
	var decidedAt string
	var updatedAt string
	err := s.db.QueryRowContext(ctx, `
SELECT proposal_id, schema_version, object_kind, mission_id, state, title,
       object_refs_json, requested_decision, created_event_id, decision_event_id,
       created_at, decided_at, updated_at
FROM plasma_proposal_bundles
WHERE proposal_id = ?`, proposalID).Scan(
		&bundle.ProposalID,
		&bundle.SchemaVersion,
		&bundle.ObjectKind,
		&bundle.MissionID,
		&bundle.State,
		&bundle.Title,
		&refsJSON,
		&bundle.RequestedDecision,
		&bundle.CreatedEventID,
		&bundle.DecisionEventID,
		&createdAt,
		&decidedAt,
		&updatedAt)
	if err != nil {
		return app.ProposalBundle{}, err
	}
	if err := unmarshalJSON(refsJSON, &bundle.ObjectRefs); err != nil {
		return app.ProposalBundle{}, err
	}
	bundle.CreatedAt, err = parseRequiredTime(createdAt)
	if err != nil {
		return app.ProposalBundle{}, err
	}
	bundle.DecidedAt, err = parseOptionalTime(decidedAt)
	if err != nil {
		return app.ProposalBundle{}, err
	}
	bundle.UpdatedAt, err = parseRequiredTime(updatedAt)
	if err != nil {
		return app.ProposalBundle{}, err
	}
	return bundle, nil
}

func (s *Store) ListProposalBundles(ctx context.Context, missionID string) ([]app.ProposalBundle, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT proposal_id
FROM plasma_proposal_bundles
WHERE mission_id = ?
ORDER BY created_at DESC, proposal_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bundles []app.ProposalBundle
	for rows.Next() {
		var proposalID string
		if err := rows.Scan(&proposalID); err != nil {
			return nil, err
		}
		bundle, err := s.GetProposalBundle(ctx, proposalID)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, bundle)
	}
	return bundles, rows.Err()
}

func (s *Store) UpdateProposalBundleState(ctx context.Context, update app.ProposalBundleStateUpdate) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE plasma_proposal_bundles
SET state = ?,
    decision_event_id = ?,
    decided_at = ?,
    updated_at = ?
WHERE proposal_id = ?
  AND state = ?`,
		update.ToState,
		update.DecisionEventID,
		formatTime(update.DecidedAt),
		formatTime(update.UpdatedAt),
		update.ProposalID,
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
	return nil
}

func parseRequiredTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}
