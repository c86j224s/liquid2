package app

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

func sourceCandidateSnapshotsFromApp(snapshots []SourceSnapshot) []sourcecandidateevents.Snapshot {
	converted := make([]sourcecandidateevents.Snapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		converted = append(converted, sourcecandidateevents.Snapshot{
			ArtifactIDs: append([]string(nil), snapshot.ArtifactIDs...),
		})
	}
	return converted
}
