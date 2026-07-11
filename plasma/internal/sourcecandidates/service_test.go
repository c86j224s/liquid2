package sourcecandidates

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestSourceCandidateStageRejectsUnrelatedRawArtifactReuse(t *testing.T) {
	ctx := context.Background()
	content := []byte("same body")
	sum := sha256.Sum256(content)
	sha := hex.EncodeToString(sum[:])
	store := &sourceCandidateServiceStore{
		artifacts: map[string]app.RawArtifact{
			"art_report": {
				ArtifactID: "art_report",
				MissionID:  "mis_1",
				MediaType:  "text/markdown; charset=utf-8",
				SHA256:     sha,
				Content:    content,
				Producer:   app.Producer{Type: "agent_session", ID: "ses_report"},
			},
		},
	}

	err := Stage(ctx, store, SourceCandidateStageRequest{
		Job: SourceCandidateStagingJob{
			MissionID:       "mis_1",
			ProposalEventID: "evt_proposed",
			Candidate: SourceCandidateProposal{
				URL:    "https://example.com/source",
				Title:  "Example",
				Reason: "Candidate body",
				State:  "proposed",
			},
			Producer:       app.Producer{Type: "agent_session", ID: "ses_1"},
			StartedEventID: "evt_started",
		},
		Fetcher: func(context.Context, string) (SourceCandidateFetched, error) {
			return SourceCandidateFetched{
				Content:         content,
				MediaType:       "text/plain; charset=utf-8",
				Title:           "Example",
				ByteSize:        int64(len(content)),
				TextLength:      len(content),
				TextLengthKnown: true,
			}, nil
		},
		NewArtifactID: func(string) string { return "art_candidate" },
		NewEventID:    sourceCandidateTestEventID,
	})
	if err != nil {
		t.Fatalf("StageSourceCandidate returned error: %v", err)
	}
	if _, ok := store.artifacts["art_candidate"]; !ok {
		t.Fatalf("expected new candidate artifact, got %#v", store.artifacts)
	}
	if store.stagedArtifactID() != "art_candidate" {
		t.Fatalf("unrelated raw artifact must not be reused, events=%#v", store.events)
	}

	store.events = append(store.events, app.LedgerEvent{
		EventID:   "evt_existing_staged",
		MissionID: "mis_1",
		Sequence:  int64(len(store.events) + 1),
		EventType: "source.candidate.staged",
		Payload: mustMarshalJSON(map[string]any{
			"url":         "https://example.com/source",
			"artifact_id": "art_report",
		}),
	})
	delete(store.artifacts, "art_candidate")

	err = Stage(ctx, store, SourceCandidateStageRequest{
		Job: SourceCandidateStagingJob{
			MissionID:       "mis_1",
			ProposalEventID: "evt_proposed_2",
			Candidate: SourceCandidateProposal{
				URL:    "https://example.com/source",
				Title:  "Example",
				Reason: "Candidate body",
				State:  "proposed",
			},
			Producer:       app.Producer{Type: "agent_session", ID: "ses_1"},
			StartedEventID: "evt_started_2",
		},
		Fetcher: func(context.Context, string) (SourceCandidateFetched, error) {
			return SourceCandidateFetched{
				Content:         content,
				MediaType:       "text/plain; charset=utf-8",
				Title:           "Example",
				ByteSize:        int64(len(content)),
				TextLength:      len(content),
				TextLengthKnown: true,
			}, nil
		},
		NewArtifactID: func(string) string { return "art_should_not_create" },
		NewEventID:    sourceCandidateTestEventID,
	})
	if err != nil {
		t.Fatalf("StageSourceCandidate reuse returned error: %v", err)
	}
	if _, ok := store.artifacts["art_should_not_create"]; ok {
		t.Fatalf("expected staged source candidate artifact reuse, got new artifact %#v", store.artifacts["art_should_not_create"])
	}
	if got := store.stagedArtifactID(); got != "art_report" {
		t.Fatalf("expected staged source candidate artifact reuse, got %q events=%#v", got, store.events)
	}
}

func TestSourceCandidateDecisionEventsPreservePayload(t *testing.T) {
	store := &sourceCandidateServiceStore{}
	rejected, err := Reject(context.Background(), store, SourceCandidateDecisionRequest{
		EventID:   "evt_reject",
		MissionID: "mis_1",
		URL:       "HTTPS://Example.com/source#fragment",
		Reason:    "이미 더 좋은 공식 문서를 소스로 붙였습니다.",
		Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		t.Fatalf("RejectSourceCandidate returned error: %v", err)
	}
	if rejected.EventType != "source.candidate.rejected" ||
		rejected.Producer.Type != "user" || rejected.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected reject event shell: %#v", rejected)
	}
	assertJSONPayloadIncludes(t, rejected.Payload, map[string]any{
		"kind":   "source_candidate_rejected",
		"url":    "https://example.com/source",
		"reason": "이미 더 좋은 공식 문서를 소스로 붙였습니다.",
	})

	restored, err := Restore(context.Background(), store, SourceCandidateDecisionRequest{
		EventID:   "evt_restore",
		MissionID: "mis_1",
		URL:       "https://example.com/source#ignored",
		Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		t.Fatalf("RestoreSourceCandidate returned error: %v", err)
	}
	if restored.EventType != "source.candidate.restored" ||
		restored.Producer.Type != "user" || restored.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected restore event shell: %#v", restored)
	}
	assertJSONPayloadIncludes(t, restored.Payload, map[string]any{
		"kind":   "source_candidate_restored",
		"url":    "https://example.com/source",
		"reason": defaultSourceCandidateRestoreReason,
	})
}

func TestSourceCandidateDecisionURLValidationPreservesWebErrorMessages(t *testing.T) {
	store := &sourceCandidateServiceStore{}

	_, err := Reject(context.Background(), store, SourceCandidateDecisionRequest{
		EventID:   "evt_reject",
		MissionID: "mis_1",
		URL:       "ftp://example.com/source",
		Producer:  app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err == nil {
		t.Fatalf("RejectSourceCandidate returned nil error")
	}
	if got := err.Error(); !strings.Contains(got, "source URL must use http or https") ||
		strings.Contains(got, "source candidate URL") {
		t.Fatalf("expected previous Web source URL error wording, got %q", got)
	}
}

type sourceCandidateServiceStore struct {
	events    []app.LedgerEvent
	artifacts map[string]app.RawArtifact
	sources   []app.SourceSnapshot
}

func (s *sourceCandidateServiceStore) ListEvents(_ context.Context, missionID string) ([]app.LedgerEvent, error) {
	events := make([]app.LedgerEvent, 0, len(s.events))
	for _, event := range s.events {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

func (s *sourceCandidateServiceStore) ListRawArtifacts(_ context.Context, missionID string) ([]app.RawArtifact, error) {
	artifacts := make([]app.RawArtifact, 0, len(s.artifacts))
	for _, artifact := range s.artifacts {
		if artifact.MissionID == missionID {
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts, nil
}

func (s *sourceCandidateServiceStore) GetRawArtifact(_ context.Context, artifactID string) (app.RawArtifact, error) {
	if artifact, ok := s.artifacts[artifactID]; ok {
		return artifact, nil
	}
	return app.RawArtifact{}, fmt.Errorf("missing artifact %s", artifactID)
}

func (s *sourceCandidateServiceStore) ListSourceSnapshotsWithState(_ context.Context, req app.ListSourceSnapshotsRequest) ([]app.SourceSnapshot, error) {
	sources := make([]app.SourceSnapshot, 0, len(s.sources))
	for _, source := range s.sources {
		if source.MissionID == req.MissionID {
			sources = append(sources, source)
		}
	}
	return sources, nil
}

func (s *sourceCandidateServiceStore) AppendEvent(_ context.Context, req app.AppendEventRequest) (app.LedgerEvent, error) {
	event := app.LedgerEvent{
		EventID:          req.EventID,
		MissionID:        req.MissionID,
		Sequence:         int64(len(s.events) + 1),
		EventType:        req.EventType,
		Producer:         req.Producer,
		CausationEventID: req.CausationEventID,
		CorrelationID:    req.CorrelationID,
		Payload:          req.Payload,
		CreatedAt:        time.Now().UTC(),
	}
	s.events = append(s.events, event)
	return event, nil
}

func (s *sourceCandidateServiceStore) CreateRawArtifactWithEvent(_ context.Context, req app.CreateRawArtifactRequest, build func(app.RawArtifact) app.AppendEventRequest) (app.RawArtifact, app.LedgerEvent, error) {
	if s.artifacts == nil {
		s.artifacts = map[string]app.RawArtifact{}
	}
	artifact := app.RawArtifact{
		ArtifactID: req.ArtifactID,
		MissionID:  req.MissionID,
		MediaType:  req.MediaType,
		ByteSize:   int64(len(req.Content)),
		SHA256:     req.ExpectedSHA256,
		Filename:   req.Filename,
		Producer:   req.Producer,
		CreatedAt:  time.Now().UTC(),
		Content:    append([]byte(nil), req.Content...),
	}
	s.artifacts[artifact.ArtifactID] = artifact
	eventReq := build(artifact)
	event, err := s.AppendEvent(context.Background(), eventReq)
	return artifact, event, err
}

func (s *sourceCandidateServiceStore) stagedArtifactID() string {
	for index := len(s.events) - 1; index >= 0; index-- {
		event := s.events[index]
		if event.EventType != "source.candidate.staged" {
			continue
		}
		var payload struct {
			ArtifactID string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err == nil {
			return payload.ArtifactID
		}
	}
	return ""
}

func sourceCandidateTestEventID(prefix string) string {
	return prefix + "_test"
}
