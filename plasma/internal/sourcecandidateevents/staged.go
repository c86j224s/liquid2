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
	ExternalVersion   string `json:"external_version"`
	ExternalUpdatedAt string `json:"external_updated_at"`
}

// LatestStagedPayloadForURL은 normalized URL에 대응하는 최신 staged payload를 찾는다.
//
// 같은 URL이 여러 번 staging되면 sequence가 가장 큰 event를 택한다. artifact ID가
// 없는 event는 agent가 읽을 수 없으므로 false로 처리한다.
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
