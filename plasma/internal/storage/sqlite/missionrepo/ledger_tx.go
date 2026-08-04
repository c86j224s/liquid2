package missionrepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// AppendLedgerEventTx appends a ledger event inside a caller-owned transaction.
//
// The caller owns commit and rollback. The event sequence is allocated from the
// current transaction snapshot and returned on the committed event value.
func AppendLedgerEventTx(ctx context.Context, tx *sql.Tx, event ledger.Event) (ledger.Event, error) {
	var sequence int64
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(MAX(sequence), 0) + 1
FROM plasma_ledger_events
WHERE mission_id = ?`, event.MissionID).Scan(&sequence); err != nil {
		return ledger.Event{}, err
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
		sqlitevalue.FormatTime(event.CreatedAt)); err != nil {
		return ledger.Event{}, fmt.Errorf("append ledger event: %w", err)
	}
	return event, nil
}

// ListLedgerEventsTx reads ledger events inside a caller-owned transaction or queryer.
func ListLedgerEventsTx(ctx context.Context, tx interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, missionID string) ([]ledger.Event, error) {
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

	var events []ledger.Event
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

func scanLedgerEvent(scanner ledgerScanner) (ledger.Event, error) {
	var event ledger.Event
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
		return ledger.Event{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return ledger.Event{}, err
	}
	event.CreatedAt = parsed
	event.Payload = []byte(payload)
	return event, nil
}
