package missionrepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// CreateMission stores a new mission row.
func (r *Repository) CreateMission(ctx context.Context, missionRecord mission.Mission) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO plasma_missions (mission_id, title, created_at, updated_at)
VALUES (?, ?, ?, ?)`,
		missionRecord.MissionID,
		missionRecord.Title,
		sqlitevalue.FormatTime(missionRecord.CreatedAt),
		sqlitevalue.FormatTime(missionRecord.UpdatedAt))
	return err
}

// ListMissions reads mission rows without changing product state.
func (r *Repository) ListMissions(ctx context.Context) ([]mission.Mission, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT mission_id, title, created_at, updated_at, lifecycle_state
FROM plasma_missions
ORDER BY updated_at DESC, created_at DESC, mission_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var missions []mission.Mission
	for rows.Next() {
		var missionRecord mission.Mission
		var createdAt string
		var updatedAt string
		if err := rows.Scan(&missionRecord.MissionID, &missionRecord.Title, &createdAt, &updatedAt, &missionRecord.LifecycleState); err != nil {
			return nil, err
		}
		missionRecord.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		missionRecord.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, err
		}
		missions = append(missions, missionRecord)
	}
	return missions, rows.Err()
}

// ListMissionActivityInputs reads ledger inputs needed for mission activity projections.
func (r *Repository) ListMissionActivityInputs(ctx context.Context, missionIDs []string) ([]mission.ActivityInput, error) {
	eventTypes := mission.ActivityEventTypes()
	placeholders := strings.TrimRight(strings.Repeat("?,", len(eventTypes)), ",")
	missionFilter := ""
	if len(missionIDs) > 0 {
		missionFilter = "\nWHERE m.mission_id IN (" + strings.TrimRight(strings.Repeat("?,", len(missionIDs)), ",") + ")"
	}
	query := fmt.Sprintf(`
SELECT m.mission_id, COALESCE((
  SELECT latest.sequence
  FROM plasma_ledger_events latest
  WHERE latest.mission_id = m.mission_id
  ORDER BY latest.sequence DESC
  LIMIT 1
), 0),
       e.event_id, e.sequence, e.event_type, e.producer_type, e.producer_id,
       e.causation_event_id, e.correlation_id, e.payload_json, e.created_at
FROM plasma_missions m
LEFT JOIN plasma_ledger_events e
  ON e.mission_id = m.mission_id AND e.event_type IN (%s)
	%s
ORDER BY m.mission_id, e.sequence`, placeholders, missionFilter)
	args := make([]any, 0, len(eventTypes)+len(missionIDs))
	for _, eventType := range eventTypes {
		args = append(args, eventType)
	}
	for _, missionID := range missionIDs {
		args = append(args, missionID)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inputs := make([]mission.ActivityInput, 0)
	for rows.Next() {
		var missionID string
		var lastSequence int64
		var eventID, eventType, producerType, producerID, causationID, correlationID, payload, createdAt sql.NullString
		var sequence sql.NullInt64
		if err := rows.Scan(
			&missionID, &lastSequence,
			&eventID, &sequence, &eventType, &producerType, &producerID,
			&causationID, &correlationID, &payload, &createdAt,
		); err != nil {
			return nil, err
		}
		if len(inputs) == 0 || inputs[len(inputs)-1].MissionID != missionID {
			inputs = append(inputs, mission.ActivityInput{MissionID: missionID, LastSequence: lastSequence})
		}
		if !eventID.Valid {
			continue
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt.String)
		if err != nil {
			return nil, err
		}
		input := &inputs[len(inputs)-1]
		input.Events = append(input.Events, ledger.Event{
			EventID:          eventID.String,
			MissionID:        missionID,
			Sequence:         sequence.Int64,
			EventType:        eventType.String,
			Producer:         ledger.Producer{Type: producerType.String, ID: producerID.String},
			CausationEventID: causationID.String,
			CorrelationID:    correlationID.String,
			Payload:          []byte(payload.String),
			CreatedAt:        parsedCreatedAt,
		})
	}
	return inputs, rows.Err()
}

// AppendLedgerEvent appends one ledger event in its own transaction.
func (r *Repository) AppendLedgerEvent(ctx context.Context, event ledger.Event) (ledger.Event, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ledger.Event{}, err
	}
	defer tx.Rollback()

	event, err = AppendLedgerEventTx(ctx, tx, event)
	if err != nil {
		return ledger.Event{}, err
	}
	if err := tx.Commit(); err != nil {
		return ledger.Event{}, err
	}
	return event, nil
}

// AppendLedgerEventsConditionally appends a built event batch if the callback succeeds.
func (r *Repository) AppendLedgerEventsConditionally(ctx context.Context, missionID string, build func([]ledger.Event) ([]ledger.Event, error)) ([]ledger.Event, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	events, err := ListLedgerEventsTx(ctx, tx, missionID)
	if err != nil {
		return nil, err
	}
	toAppend, err := build(events)
	if err != nil {
		return nil, err
	}
	appended := make([]ledger.Event, 0, len(toAppend))
	for _, event := range toAppend {
		committed, err := AppendLedgerEventTx(ctx, tx, event)
		if err != nil {
			return nil, err
		}
		appended = append(appended, committed)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return appended, nil
}

// ListLedgerEvents reads mission ledger events ordered by sequence.
func (r *Repository) ListLedgerEvents(ctx context.Context, missionID string) ([]ledger.Event, error) {
	return ListLedgerEventsTx(ctx, r.db, missionID)
}
