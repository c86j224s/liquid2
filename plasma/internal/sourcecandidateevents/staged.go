package sourcecandidateevents

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

const StagedEventType = "source.candidate.staged"

type Event struct {
	EventID   string
	Sequence  int64
	EventType string
	Payload   json.RawMessage
	CreatedAt time.Time
}

type Snapshot struct {
	ArtifactIDs []string
}

type StagedPayload struct {
	URL               string `json:"url"`
	Title             string `json:"title"`
	ProposalEventID   string `json:"proposal_event_id"`
	ArtifactID        string `json:"artifact_id"`
	ExternalVersion   string `json:"external_version"`
	ExternalUpdatedAt string `json:"external_updated_at"`
}

func LatestStagedPayloadForURL(events []Event, normalizedURL string, normalize func(string) (string, error)) (StagedPayload, bool) {
	if normalize == nil {
		return StagedPayload{}, false
	}
	ordered := append([]Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Sequence < ordered[j].Sequence
	})
	var selected StagedPayload
	found := false
	for _, event := range ordered {
		payload, ok := StagedPayloadFromEvent(event)
		if !ok {
			continue
		}
		existing, err := normalize(payload.URL)
		if err != nil || existing != normalizedURL {
			continue
		}
		selected = payload
		found = true
	}
	if !found || strings.TrimSpace(selected.ArtifactID) == "" {
		return StagedPayload{}, false
	}
	return selected, true
}

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

func IsOpenStagedArtifact(events []Event, snapshots []Snapshot, artifactID string) bool {
	_, ok := OpenStagedArtifactIDs(events, snapshots)[strings.TrimSpace(artifactID)]
	return ok
}
