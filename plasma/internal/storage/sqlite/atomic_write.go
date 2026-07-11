package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (s *Store) CommitAtomicWrite(ctx context.Context, write app.AtomicWrite) (app.AtomicWriteResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.AtomicWriteResult{}, err
	}
	defer tx.Rollback()

	events := make([]app.LedgerEvent, 0, len(write.Events))
	for _, event := range write.Events {
		committed, err := appendLedgerEventTx(ctx, tx, event)
		if err != nil {
			return app.AtomicWriteResult{}, err
		}
		events = append(events, committed)
	}
	for _, artifact := range write.RawArtifacts {
		if err := insertRawArtifactTx(ctx, tx, artifact); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, snapshot := range write.SourceSnapshots {
		if err := insertSourceSnapshotTx(ctx, tx, snapshot); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, record := range write.EvidenceRecords {
		if err := insertEvidenceRecordTx(ctx, tx, record); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, record := range write.ClaimRecords {
		if err := insertClaimRecordTx(ctx, tx, record); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, record := range write.QuestionRecords {
		if err := insertQuestionRecordTx(ctx, tx, record); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, bundle := range write.ProposalBundles {
		if err := insertProposalBundleTx(ctx, tx, bundle); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, report := range write.Reports {
		if err := insertReportTx(ctx, tx, report); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, version := range write.ReportVersions {
		if err := insertReportVersionTx(ctx, tx, version); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, block := range write.ReportBlocks {
		if err := insertReportBlockTx(ctx, tx, block); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return app.AtomicWriteResult{}, err
	}
	return app.AtomicWriteResult{Events: events}, nil
}

func appendLedgerEventTx(ctx context.Context, tx *sql.Tx, event app.LedgerEvent) (app.LedgerEvent, error) {
	var sequence int64
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(MAX(sequence), 0) + 1
FROM plasma_ledger_events
WHERE mission_id = ?`, event.MissionID).Scan(&sequence); err != nil {
		return app.LedgerEvent{}, err
	}
	event.Sequence = sequence
	if _, err := tx.ExecContext(ctx, `
INSERT INTO plasma_ledger_events (
  event_id, mission_id, sequence, event_type, producer_type, producer_id,
  causation_event_id, correlation_id, payload_json, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.EventID,
		event.MissionID,
		event.Sequence,
		event.EventType,
		event.Producer.Type,
		event.Producer.ID,
		event.CausationEventID,
		event.CorrelationID,
		string(event.Payload),
		formatTime(event.CreatedAt)); err != nil {
		return app.LedgerEvent{}, fmt.Errorf("append ledger event: %w", err)
	}
	return event, nil
}

func insertRawArtifactTx(ctx context.Context, tx *sql.Tx, artifact app.RawArtifact) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO plasma_raw_artifacts (
  artifact_id, mission_id, media_type, byte_size, sha256, storage_uri, filename,
  producer_type, producer_id, created_at, content_blob
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		artifact.ArtifactID,
		artifact.MissionID,
		artifact.MediaType,
		artifact.ByteSize,
		artifact.SHA256,
		artifact.StorageURI,
		artifact.Filename,
		artifact.Producer.Type,
		artifact.Producer.ID,
		formatTime(artifact.CreatedAt),
		artifact.Content)
	return err
}

func insertSourceSnapshotTx(ctx context.Context, tx *sql.Tx, snapshot app.SourceSnapshot) error {
	locatorsJSON := string(snapshot.Locators)
	if locatorsJSON == "" {
		locatorsJSON = "[]"
	}
	if !json.Valid([]byte(locatorsJSON)) {
		return app.ErrInvalidInput
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO plasma_source_snapshots (
  snapshot_id, mission_id, connector_id, connector_type, external_source_id,
  external_uri, external_version, connector_version, title, captured_at,
  external_updated_at, content_hash_algorithm, content_hash_value,
  locators_json, access_visibility, access_license, retrieval_policy
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.SnapshotID,
		snapshot.MissionID,
		snapshot.Connector.ConnectorID,
		snapshot.Connector.ConnectorType,
		snapshot.Connector.ExternalSourceID,
		snapshot.Connector.ExternalURI,
		snapshot.Connector.ExternalVersion,
		snapshot.Connector.ConnectorVersion,
		snapshot.Title,
		formatTime(snapshot.CapturedAt),
		formatOptionalTime(snapshot.ExternalUpdatedAt),
		snapshot.ContentHash.Algorithm,
		snapshot.ContentHash.Value,
		locatorsJSON,
		snapshot.Access.Visibility,
		snapshot.Access.License,
		snapshot.Access.RetrievalPolicy); err != nil {
		return err
	}
	for i, artifactID := range snapshot.ArtifactIDs {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO plasma_source_snapshot_artifacts (snapshot_id, artifact_id, ordinal)
VALUES (?, ?, ?)`,
			snapshot.SnapshotID,
			artifactID,
			i); err != nil {
			return err
		}
	}
	return nil
}

func insertEvidenceRecordTx(ctx context.Context, tx *sql.Tx, record app.EvidenceRecord) error {
	snapshotRefsJSON, err := marshalJSON(record.SnapshotRefs)
	if err != nil {
		return err
	}
	confidenceJSON, err := marshalJSON(record.Confidence)
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
		formatTime(record.CreatedAt))
	return err
}

func insertClaimRecordTx(ctx context.Context, tx *sql.Tx, record app.ClaimRecord) error {
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
		formatTime(record.CreatedAt))
	return err
}

func insertQuestionRecordTx(ctx context.Context, tx *sql.Tx, record app.QuestionRecord) error {
	evidenceJSON, err := marshalJSON(record.RelatedEvidenceIDs)
	if err != nil {
		return err
	}
	claimJSON, err := marshalJSON(record.RelatedClaimIDs)
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
		boolInt(record.Blocking),
		evidenceJSON,
		claimJSON,
		record.Resolution,
		record.CreatedEventID,
		formatTime(record.CreatedAt))
	return err
}

func insertProposalBundleTx(ctx context.Context, tx *sql.Tx, bundle app.ProposalBundle) error {
	refsJSON, err := marshalJSON(bundle.ObjectRefs)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
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
