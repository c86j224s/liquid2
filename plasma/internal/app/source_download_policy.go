package app

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"github.com/c86j224s/liquid2/plasma/internal/source/liquid2source"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidateevents"
)

func connectorSnapshotDownloadExcluded(connector sourcecontract.ConnectorRef) bool {
	connectorID := strings.ToLower(strings.TrimSpace(connector.ConnectorID))
	connectorType := strings.ToLower(strings.TrimSpace(connector.ConnectorType))
	return connectorID == confluencesource.ConfluenceConnectorID || connectorType == confluencesource.ConfluenceConnectorType ||
		connectorID == liquid2source.Liquid2ConnectorID || connectorType == liquid2source.Liquid2ConnectorType
}

func latestCandidateStagedEvent(events []ledger.Event, eventID string) (ledger.Event, sourcecandidateevents.StagedPayload, bool) {
	for _, event := range events {
		if event.EventID != eventID {
			continue
		}
		payload, ok := sourcecandidateevents.StagedPayloadFromEvent(sourceDownloadEvent(event))
		return event, payload, ok
	}
	return ledger.Event{}, sourcecandidateevents.StagedPayload{}, false
}

func sourceDownloadEvents(events []ledger.Event) []sourcecandidateevents.Event {
	converted := make([]sourcecandidateevents.Event, 0, len(events))
	for _, event := range events {
		converted = append(converted, sourceDownloadEvent(event))
	}
	return converted
}

func sourceDownloadEvent(event ledger.Event) sourcecandidateevents.Event {
	return sourcecandidateevents.Event{EventID: event.EventID, Sequence: event.Sequence, EventType: event.EventType, Payload: event.Payload, CreatedAt: event.CreatedAt}
}
