package sqlite

import (
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
)

func reportrunEvents(events []ledger.Event) []reportrun.Event {
	out := make([]reportrun.Event, 0, len(events))
	for _, event := range events {
		out = append(out, reportrun.Event{
			EventID: event.EventID, MissionID: event.MissionID,
			Sequence: event.Sequence, EventType: event.EventType, Payload: event.Payload,
			CreatedAt: event.CreatedAt,
		})
	}
	return out
}

func containsReportLedgerEvent(events []ledger.Event) bool {
	for _, event := range events {
		if reportrun.IsReportEventType(event.EventType) {
			return true
		}
	}
	return false
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
