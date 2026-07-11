package sourcecandidateevents

import (
	"encoding/json"
	"testing"
)

func TestOpenStagedArtifactIDsExcludesApprovedSnapshots(t *testing.T) {
	events := []Event{
		stagedEvent(t, 1, "https://example.com/a", "art_open"),
		stagedEvent(t, 2, "https://example.com/b", "art_attached"),
	}
	snapshots := []Snapshot{{ArtifactIDs: []string{"art_attached"}}}

	open := OpenStagedArtifactIDs(events, snapshots)
	if _, ok := open["art_open"]; !ok {
		t.Fatalf("expected open staged artifact to remain: %#v", open)
	}
	if _, ok := open["art_attached"]; ok {
		t.Fatalf("expected attached artifact to be removed: %#v", open)
	}
}

func TestLatestStagedPayloadForURLUsesLatestMatchingSequence(t *testing.T) {
	events := []Event{
		stagedEvent(t, 2, "https://example.com/a", "art_new"),
		stagedEvent(t, 1, "https://example.com/a", "art_old"),
		stagedEvent(t, 3, "https://example.com/b", "art_other"),
	}

	payload, ok := LatestStagedPayloadForURL(events, "https://example.com/a", func(value string) (string, error) {
		return value, nil
	})
	if !ok || payload.ArtifactID != "art_new" {
		t.Fatalf("expected latest matching staged payload, got ok=%v payload=%#v", ok, payload)
	}
}

func stagedEvent(t *testing.T, sequence int64, url string, artifactID string) Event {
	t.Helper()
	payload, err := json.Marshal(StagedPayload{URL: url, ArtifactID: artifactID})
	if err != nil {
		t.Fatalf("marshal staged payload: %v", err)
	}
	return Event{Sequence: sequence, EventType: StagedEventType, Payload: payload}
}
