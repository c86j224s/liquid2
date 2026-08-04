package researchrepo

import (
	"context"
	"database/sql"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// CreateProposalBundle stores one proposal bundle.
func (r *Repository) CreateProposalBundle(ctx context.Context, bundle app.ProposalBundle) error {
	return InsertProposalBundleTx(ctx, r.db, bundle)
}

// GetProposalBundle reads one proposal bundle by stable ID.
func (r *Repository) GetProposalBundle(ctx context.Context, proposalID string) (app.ProposalBundle, error) {
	var bundle app.ProposalBundle
	var refsJSON string
	var createdAt string
	var decidedAt string
	var updatedAt string
	err := r.db.QueryRowContext(ctx, `
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
	if err := sqlitevalue.UnmarshalJSON(refsJSON, &bundle.ObjectRefs); err != nil {
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

// ListProposalBundles reads mission proposal bundles ordered by creation time.
func (r *Repository) ListProposalBundles(ctx context.Context, missionID string) ([]app.ProposalBundle, error) {
	rows, err := r.db.QueryContext(ctx, `
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
		bundle, err := r.GetProposalBundle(ctx, proposalID)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, bundle)
	}
	return bundles, rows.Err()
}

// UpdateProposalBundleState updates proposal state with legacy RowsAffected semantics.
func (r *Repository) UpdateProposalBundleState(ctx context.Context, update app.ProposalBundleStateUpdate) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE plasma_proposal_bundles
SET state = ?,
    decision_event_id = ?,
    decided_at = ?,
    updated_at = ?
WHERE proposal_id = ?
  AND state = ?`,
		update.ToState,
		update.DecisionEventID,
		sqlitevalue.FormatTime(update.DecidedAt),
		sqlitevalue.FormatTime(update.UpdatedAt),
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

// InsertProposalBundleTx inserts a proposal bundle inside a caller-owned transaction or queryer.
func InsertProposalBundleTx(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, bundle app.ProposalBundle) error {
	refsJSON, err := sqlitevalue.MarshalJSON(bundle.ObjectRefs)
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
		sqlitevalue.FormatTime(bundle.CreatedAt),
		sqlitevalue.FormatOptionalTime(bundle.DecidedAt),
		sqlitevalue.FormatTime(bundle.UpdatedAt))
	return err
}
