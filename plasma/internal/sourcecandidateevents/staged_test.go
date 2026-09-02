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
	payload, ok := LatestStagedPayloadForURL(events, "https://example.com/a", func(value string) (string, error) { return value, nil })
	if !ok || payload.ArtifactID != "art_new" {
		t.Fatalf("expected latest matching staged payload, got ok=%v payload=%#v", ok, payload)
	}
}

func TestLatestStagingTerminalForURLHonorsTerminalState(t *testing.T) {
	normalize := func(value string) (string, error) { return value, nil }
	cases := []struct{ name, latestType, wantArtifact string }{
		{"latest staged", StagedEventType, "art_new"},
		{"newer fetching", "source.candidate.staging_started", ""},
		{"newer failed", "source.candidate.staging_failed", ""},
		{"newer different staged", StagedEventType, "art_newest"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := []Event{stagedEvent(t, 1, "https://example.com/a", "art_old")}
			payload, _ := json.Marshal(StagedPayload{URL: "https://example.com/a", ArtifactID: tc.wantArtifact})
			events = append(events, Event{Sequence: 2, EventType: tc.latestType, Payload: payload})
			if tc.name == "newer different staged" {
				payload, _ = json.Marshal(StagedPayload{URL: "https://example.com/a", ArtifactID: "art_newest"})
				events[1].Payload = payload
			}
			got, state, ok := LatestStagingTerminalForURL(events, "https://example.com/a", normalize)
			if tc.wantArtifact == "" {
				if ok && state == StagedEventType {
					t.Fatalf("non-staged latest terminal returned staged: %#v", got)
				}
				return
			}
			if !ok || state != StagedEventType || got.ArtifactID != tc.wantArtifact {
				t.Fatalf("got payload=%#v state=%q ok=%v", got, state, ok)
			}
		})
	}
}

func TestLatestStagingTerminalForURLCorrelatesOutOfOrderAttempts(t *testing.T) {
	normalize := func(value string) (string, error) { return value, nil }
	startA := Event{EventID: "evt_start_a", Sequence: 1, EventType: "source.candidate.staging_started", Payload: stagedPayload(t, StagedPayload{URL: "https://example.com/a", StagingEventID: "evt_start_a"})}
	startB := Event{EventID: "evt_start_b", Sequence: 2, EventType: "source.candidate.staging_started", Payload: stagedPayload(t, StagedPayload{URL: "https://example.com/a", StagingEventID: "evt_start_b"})}
	stagedB := Event{EventID: "evt_stage_b", Sequence: 3, EventType: StagedEventType, Payload: stagedPayload(t, StagedPayload{URL: "https://example.com/a", ArtifactID: "art_b", StagingEventID: "evt_start_b"})}
	stagedALate := Event{EventID: "evt_stage_a", Sequence: 4, EventType: StagedEventType, Payload: stagedPayload(t, StagedPayload{URL: "https://example.com/a", ArtifactID: "art_a", StagingEventID: "evt_start_a"})}
	got, payload, state, ok := LatestStagingTerminalEventForURL([]Event{startA, startB, stagedB, stagedALate}, "https://example.com/a", normalize)
	if !ok || got.EventID != stagedB.EventID || payload.ArtifactID != "art_b" || state != StagedEventType {
		t.Fatalf("out-of-order attempt selected %#v %#v %q %v", got, payload, state, ok)
	}
}

func TestLatestStagingTerminalForURLRejectsUncorrelatedModernTerminal(t *testing.T) {
	normalize := func(value string) (string, error) { return value, nil }
	start := Event{EventID: "evt_start", Sequence: 1, EventType: "source.candidate.staging_started", Payload: stagedPayload(t, StagedPayload{URL: "https://example.com/a", StagingEventID: "evt_start"})}
	terminal := Event{EventID: "evt_stage", Sequence: 2, EventType: StagedEventType, Payload: stagedPayload(t, StagedPayload{URL: "https://example.com/a", ArtifactID: "art_uncorrelated"})}
	got, payload, state, ok := LatestStagingTerminalEventForURL([]Event{start, terminal}, "https://example.com/a", normalize)
	if !ok || got.EventID != start.EventID || payload.ArtifactID != "" || state != "source.candidate.staging_started" {
		t.Fatalf("uncorrelated terminal replaced open start: %#v %#v %q %v", got, payload, state, ok)
	}
}

func stagedPayload(t *testing.T, payload StagedPayload) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestLatestStagingTerminalForURLIgnoresUnrelatedURL(t *testing.T) {
	payload, _ := json.Marshal(StagedPayload{URL: "https://example.com/other", ArtifactID: "art_other"})
	_, _, ok := LatestStagingTerminalForURL([]Event{{Sequence: 1, EventType: StagedEventType, Payload: payload}}, "https://example.com/a", func(value string) (string, error) { return value, nil })
	if ok {
		t.Fatal("unrelated URL should not produce provenance")
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
