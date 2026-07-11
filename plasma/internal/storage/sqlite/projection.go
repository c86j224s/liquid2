package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (s *Store) SaveMissionProjection(ctx context.Context, projection app.MissionProjection) error {
	scopeJSON, err := marshalJSON(projection.Scope)
	if err != nil {
		return err
	}
	activeSessionsJSON, err := marshalJSON(projection.ActiveSessionIDs)
	if err != nil {
		return err
	}
	acceptedClaimsJSON, err := marshalJSON(projection.AcceptedClaimIDs)
	if err != nil {
		return err
	}
	openQuestionsJSON, err := marshalJSON(projection.OpenQuestionIDs)
	if err != nil {
		return err
	}
	reasonsJSON, err := marshalJSON(projection.NeedsReviewReasons)
	if err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx, `
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
		boolInt(projection.NeedsReview),
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

func (s *Store) GetMissionProjection(ctx context.Context, missionID string) (app.MissionProjection, error) {
	var projection app.MissionProjection
	var scopeJSON string
	var activeSessionsJSON string
	var acceptedClaimsJSON string
	var openQuestionsJSON string
	var needsReview int
	var reasonsJSON string

	err := s.db.QueryRowContext(ctx, `
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
		return app.MissionProjection{}, err
	}
	projection.NeedsReview = needsReview != 0
	if err := unmarshalJSON(scopeJSON, &projection.Scope); err != nil {
		return app.MissionProjection{}, err
	}
	if err := unmarshalJSON(activeSessionsJSON, &projection.ActiveSessionIDs); err != nil {
		return app.MissionProjection{}, err
	}
	if err := unmarshalJSON(acceptedClaimsJSON, &projection.AcceptedClaimIDs); err != nil {
		return app.MissionProjection{}, err
	}
	if err := unmarshalJSON(openQuestionsJSON, &projection.OpenQuestionIDs); err != nil {
		return app.MissionProjection{}, err
	}
	if err := unmarshalJSON(reasonsJSON, &projection.NeedsReviewReasons); err != nil {
		return app.MissionProjection{}, err
	}
	return projection, nil
}

func marshalJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal projection field: %w", err)
	}
	return string(encoded), nil
}

func unmarshalJSON(text string, target any) error {
	if err := json.Unmarshal([]byte(text), target); err != nil {
		return fmt.Errorf("unmarshal projection field: %w", err)
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
