package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (s *Store) CreateMission(ctx context.Context, mission app.Mission) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO plasma_missions (mission_id, title, created_at, updated_at)
VALUES (?, ?, ?, ?)`,
		mission.MissionID,
		mission.Title,
		formatTime(mission.CreatedAt),
		formatTime(mission.UpdatedAt))
	return err
}

func (s *Store) ListMissions(ctx context.Context) ([]app.Mission, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT mission_id, title, created_at, updated_at, lifecycle_state
FROM plasma_missions
ORDER BY updated_at DESC, created_at DESC, mission_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var missions []app.Mission
	for rows.Next() {
		var mission app.Mission
		var createdAt string
		var updatedAt string
		if err := rows.Scan(&mission.MissionID, &mission.Title, &createdAt, &updatedAt, &mission.LifecycleState); err != nil {
			return nil, err
		}
		mission.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		mission.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, err
		}
		missions = append(missions, mission)
	}
	return missions, rows.Err()
}

// ListMissionActivityInputs reads every mission's list-relevant ledger events
// in one query. Full ledger reads remain available only for detail projections.
func (s *Store) ListMissionActivityInputs(ctx context.Context, missionIDs []string) ([]app.MissionActivityInput, error) {
	eventTypes := app.MissionActivityEventTypes()
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
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inputs := make([]app.MissionActivityInput, 0)
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
			inputs = append(inputs, app.MissionActivityInput{MissionID: missionID, LastSequence: lastSequence})
		}
		if !eventID.Valid {
			continue
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt.String)
		if err != nil {
			return nil, err
		}
		input := &inputs[len(inputs)-1]
		input.Events = append(input.Events, app.LedgerEvent{
			EventID:          eventID.String,
			MissionID:        missionID,
			Sequence:         sequence.Int64,
			EventType:        eventType.String,
			Producer:         app.Producer{Type: producerType.String, ID: producerID.String},
			CausationEventID: causationID.String,
			CorrelationID:    correlationID.String,
			Payload:          []byte(payload.String),
			CreatedAt:        parsedCreatedAt,
		})
	}
	return inputs, rows.Err()
}

func (s *Store) AppendLedgerEvent(ctx context.Context, event app.LedgerEvent) (app.LedgerEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.LedgerEvent{}, err
	}
	defer tx.Rollback()

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
	if err := tx.Commit(); err != nil {
		return app.LedgerEvent{}, err
	}
	return event, nil
}

func (s *Store) AppendLedgerEventsConditionally(ctx context.Context, missionID string, build func([]app.LedgerEvent) ([]app.LedgerEvent, error)) ([]app.LedgerEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	events, err := listLedgerEventsTx(ctx, tx, missionID)
	if err != nil {
		return nil, err
	}
	toAppend, err := build(events)
	if err != nil {
		return nil, err
	}
	appended := make([]app.LedgerEvent, 0, len(toAppend))
	for _, event := range toAppend {
		committed, err := appendLedgerEventTx(ctx, tx, event)
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

func (s *Store) ListLedgerEvents(ctx context.Context, missionID string) ([]app.LedgerEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT event_id, mission_id, sequence, event_type, producer_type, producer_id,
       causation_event_id, correlation_id, payload_json, created_at
FROM plasma_ledger_events
WHERE mission_id = ?
ORDER BY sequence`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []app.LedgerEvent
	for rows.Next() {
		event, err := scanLedgerEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func listLedgerEventsTx(ctx context.Context, tx interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, missionID string) ([]app.LedgerEvent, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT event_id, mission_id, sequence, event_type, producer_type, producer_id,
       causation_event_id, correlation_id, payload_json, created_at
FROM plasma_ledger_events
WHERE mission_id = ?
ORDER BY sequence`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []app.LedgerEvent
	for rows.Next() {
		event, err := scanLedgerEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

type ledgerScanner interface {
	Scan(dest ...any) error
}

func scanLedgerEvent(scanner ledgerScanner) (app.LedgerEvent, error) {
	var event app.LedgerEvent
	var payload string
	var createdAt string
	if err := scanner.Scan(
		&event.EventID,
		&event.MissionID,
		&event.Sequence,
		&event.EventType,
		&event.Producer.Type,
		&event.Producer.ID,
		&event.CausationEventID,
		&event.CorrelationID,
		&payload,
		&createdAt); err != nil {
		return app.LedgerEvent{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return app.LedgerEvent{}, err
	}
	event.CreatedAt = parsed
	event.Payload = []byte(payload)
	return event, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		t = time.Now().UTC()
	}
	return t.UTC().Format(time.RFC3339Nano)
}
