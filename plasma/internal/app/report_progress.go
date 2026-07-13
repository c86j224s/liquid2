package app

import "github.com/c86j224s/liquid2/plasma/internal/ledgerstate"

// ReportProgress is the typed application read model consumed by transports.
type ReportProgress = ledgerstate.ReportProgress

func ReportProgressFromEvents(events []LedgerEvent) ReportProgress {
	stateEvents := make([]ledgerstate.Event, 0, len(events))
	for _, event := range events {
		stateEvents = append(stateEvents, ledgerstate.Event{
			EventID: event.EventID, Sequence: event.Sequence, EventType: event.EventType,
			Payload: event.Payload, CreatedAt: event.CreatedAt,
		})
	}
	return ledgerstate.ProjectReportProgress(stateEvents)
}
