package app

import "github.com/c86j224s/liquid2/plasma/internal/ledgerstate"

func ledgerStateEventsFromApp(events []LedgerEvent) []ledgerstate.Event {
	converted := make([]ledgerstate.Event, 0, len(events))
	for _, event := range events {
		converted = append(converted, ledgerstate.Event{
			EventID:   event.EventID,
			Sequence:  event.Sequence,
			EventType: event.EventType,
			Payload:   event.Payload,
			CreatedAt: event.CreatedAt,
		})
	}
	return converted
}
