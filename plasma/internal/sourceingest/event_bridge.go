package sourceingest

import "github.com/c86j224s/liquid2/plasma/internal/sourcecandidateevents"

func sourceCandidateEventsFromApp(events []LedgerEvent) []sourcecandidateevents.Event {
	converted := make([]sourcecandidateevents.Event, 0, len(events))
	for _, event := range events {
		converted = append(converted, sourcecandidateevents.Event{
			EventID:   event.EventID,
			Sequence:  event.Sequence,
			EventType: event.EventType,
			Payload:   event.Payload,
			CreatedAt: event.CreatedAt,
		})
	}
	return converted
}

func normalizeSourceCandidateURL(value string) (string, error) {
	normalized, _, err := normalizeSourceIngestURL(value)
	return normalized, err
}
