package sourcecandidateevents

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// StagedEventType은 승인 전 source 후보의 fetched artifact가 준비됐음을 나타낸다.
const StagedEventType = "source.candidate.staged"

// Event는 staged candidate projection이 필요로 하는 장부 event의 최소 필드다.
type Event struct {
	EventID   string
	Sequence  int64
	EventType string
	Payload   json.RawMessage
	CreatedAt time.Time
}

// Snapshot은 staged artifact가 이미 승인된 source snapshot에 포함됐는지 판정하기
// 위한 최소 view다.
type Snapshot struct {
	ArtifactIDs []string
}

// StagedPayload는 source.candidate.staged event의 stable payload다.
//
// ArtifactID가 있어야 대화 agent가 승인 전 후보를 읽을 수 있다. 이 payload만으로
// source가 승인됐다고 해석하면 안 된다.
type StagedPayload struct {
	URL               string `json:"url"`
	Title             string `json:"title"`
	ProposalEventID   string `json:"proposal_event_id"`
	ArtifactID        string `json:"artifact_id"`
	MediaKind         string `json:"media_kind"`
	ExternalVersion   string `json:"external_version"`
	ExternalUpdatedAt string `json:"external_updated_at"`
	StagingEventID    string `json:"staging_event_id"`
	Width             int    `json:"width"`
	Height            int    `json:"height"`
}

// LatestStagingTerminalEventForURL correlates terminal events to the newest
// staging attempt. Terminal payloads with staging_event_id are authoritative;
// legacy streams without starts retain sequence-based behavior.
func LatestStagingTerminalEventForURL(events []Event, normalizedURL string, normalize func(string) (string, error)) (Event, StagedPayload, string, bool) {
	if normalize == nil {
		return Event{}, StagedPayload{}, "", false
	}
	ordered := append([]Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Sequence < ordered[j].Sequence })
	starts := map[string]Event{}
	for _, event := range ordered {
		if event.EventType != "source.candidate.staging_started" {
			continue
		}
		var payload StagedPayload
		if json.Unmarshal(event.Payload, &payload) != nil {
			continue
		}
		existing, err := normalize(payload.URL)
		if err == nil && existing == normalizedURL {
			starts[existing] = event
		}
	}
	latestStart := starts[normalizedURL]
	var selected Event
	var selectedPayload StagedPayload
	selectedState := ""
	found := false
	for _, event := range ordered {
		if event.EventType != "source.candidate.staging_started" && event.EventType != StagedEventType && event.EventType != "source.candidate.staging_failed" {
			continue
		}
		var payload StagedPayload
		if json.Unmarshal(event.Payload, &payload) != nil {
			continue
		}
		existing, err := normalize(payload.URL)
		if err != nil || existing != normalizedURL {
			continue
		}
		if event.EventType != "source.candidate.staging_started" && latestStart.EventID != "" {
			attemptID := strings.TrimSpace(payload.StagingEventID)
			if attemptID == "" || attemptID != latestStart.EventID {
				continue
			}
		}
		if !found || event.Sequence >= selected.Sequence {
			payload.StagingEventID = firstNonEmptyStagingID(payload.StagingEventID, event.EventID)
			selected, selectedPayload, selectedState, found = event, payload, event.EventType, true
		}
	}
	return selected, selectedPayload, selectedState, found
}

func firstNonEmptyStagingID(payloadID, eventID string) string {
	if strings.TrimSpace(payloadID) != "" {
		return strings.TrimSpace(payloadID)
	}
	return strings.TrimSpace(eventID)
}

// LatestStagingTerminalForURL returns the latest staging lifecycle payload.
func LatestStagingTerminalForURL(events []Event, normalizedURL string, normalize func(string) (string, error)) (StagedPayload, string, bool) {
	_, payload, state, found := LatestStagingTerminalEventForURL(events, normalizedURL, normalize)
	return payload, state, found
}

// LatestStagedPayloadForURL returns the latest staged payload only when its
// lifecycle remains staged, not fetching or failed.
func LatestStagedPayloadForURL(events []Event, normalizedURL string, normalize func(string) (string, error)) (StagedPayload, bool) {
	payload, state, found := LatestStagingTerminalForURL(events, normalizedURL, normalize)
	if !found || state != StagedEventType || strings.TrimSpace(payload.ArtifactID) == "" {
		return StagedPayload{}, false
	}
	return payload, true
}

// StagedPayloadFromEvent는 source.candidate.staged event에서 payload를 안전하게 읽는다.
func StagedPayloadFromEvent(event Event) (StagedPayload, bool) {
	if event.EventType != StagedEventType {
		return StagedPayload{}, false
	}
	var payload StagedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return StagedPayload{}, false
	}
	if strings.TrimSpace(payload.ArtifactID) == "" {
		return StagedPayload{}, false
	}
	return payload, true
}

// OpenStagedArtifactIDs는 아직 승인된 snapshot에 흡수되지 않은 staged artifact ID
// 집합을 계산한다.
func OpenStagedArtifactIDs(events []Event, snapshots []Snapshot) map[string]struct{} {
	staged := map[string]struct{}{}
	for _, event := range events {
		payload, ok := StagedPayloadFromEvent(event)
		if !ok {
			continue
		}
		staged[strings.TrimSpace(payload.ArtifactID)] = struct{}{}
	}
	for _, snapshot := range snapshots {
		for _, artifactID := range snapshot.ArtifactIDs {
			delete(staged, strings.TrimSpace(artifactID))
		}
	}
	return staged
}

// IsOpenStagedArtifact는 artifact가 여전히 승인 전 후보로 남아 있는지 판정한다.
func IsOpenStagedArtifact(events []Event, snapshots []Snapshot, artifactID string) bool {
	_, ok := OpenStagedArtifactIDs(events, snapshots)[strings.TrimSpace(artifactID)]
	return ok
}
