package missionrepo

import (
	"context"
	"database/sql"

	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// SaveMissionProjection stores a calculated mission projection.
func (r *Repository) SaveMissionProjection(ctx context.Context, projection mission.Projection) error {
	scopeJSON, err := sqlitevalue.MarshalJSON(projection.Scope)
	if err != nil {
		return err
	}
	activeSessionsJSON, err := sqlitevalue.MarshalJSON(projection.ActiveSessionIDs)
	if err != nil {
		return err
	}
	acceptedClaimsJSON, err := sqlitevalue.MarshalJSON(projection.AcceptedClaimIDs)
	if err != nil {
		return err
	}
	openQuestionsJSON, err := sqlitevalue.MarshalJSON(projection.OpenQuestionIDs)
	if err != nil {
		return err
	}
	reasonsJSON, err := sqlitevalue.MarshalJSON(projection.NeedsReviewReasons)
	if err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, `
UPDATE plasma_missions
SET title = ?,
    objective = ?,
    scope_json = ?,
    lifecycle_state = ?,
    last_event_id = ?,
    last_sequence = ?,
    active_session_ids_json = ?,
    accepted_claim_ids_json = ?,
    open_question_ids_json = ?,
    active_report_version_id = ?,
    needs_review = ?,
    needs_review_reasons_json = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE mission_id = ?
  AND last_sequence <= ?`,
		projection.Title,
		projection.Objective,
		scopeJSON,
		projection.LifecycleState,
		projection.LastEventID,
		projection.LastSequence,
		activeSessionsJSON,
		acceptedClaimsJSON,
		openQuestionsJSON,
		projection.ActiveReportVersionID,
		sqlitevalue.BoolInt(projection.NeedsReview),
		reasonsJSON,
		projection.MissionID,
		projection.LastSequence)
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

// GetMissionProjection reads a mission projection without changing product state.
func (r *Repository) GetMissionProjection(ctx context.Context, missionID string) (mission.Projection, error) {
	var projection mission.Projection
	var scopeJSON string
	var activeSessionsJSON string
	var acceptedClaimsJSON string
	var openQuestionsJSON string
	var needsReview int
	var reasonsJSON string

	err := r.db.QueryRowContext(ctx, `
SELECT mission_id, title, objective, scope_json, lifecycle_state,
       last_event_id, last_sequence, active_session_ids_json,
       accepted_claim_ids_json, open_question_ids_json,
       active_report_version_id, needs_review, needs_review_reasons_json
FROM plasma_missions
WHERE mission_id = ?`, missionID).Scan(
		&projection.MissionID,
		&projection.Title,
		&projection.Objective,
		&scopeJSON,
		&projection.LifecycleState,
		&projection.LastEventID,
		&projection.LastSequence,
		&activeSessionsJSON,
		&acceptedClaimsJSON,
		&openQuestionsJSON,
		&projection.ActiveReportVersionID,
		&needsReview,
		&reasonsJSON)
	if err != nil {
		return mission.Projection{}, err
	}
	projection.NeedsReview = needsReview != 0
	if err := sqlitevalue.UnmarshalJSON(scopeJSON, &projection.Scope); err != nil {
		return mission.Projection{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(activeSessionsJSON, &projection.ActiveSessionIDs); err != nil {
		return mission.Projection{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(acceptedClaimsJSON, &projection.AcceptedClaimIDs); err != nil {
		return mission.Projection{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(openQuestionsJSON, &projection.OpenQuestionIDs); err != nil {
		return mission.Projection{}, err
	}
	if err := sqlitevalue.UnmarshalJSON(reasonsJSON, &projection.NeedsReviewReasons); err != nil {
		return mission.Projection{}, err
	}
	return projection, nil
}
