package sourceingest

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

type sourceCandidateServiceStore struct {
	events    []LedgerEvent
	artifacts map[string]RawArtifact
	sources   []SourceSnapshot
}

func (s *sourceCandidateServiceStore) ListEvents(_ context.Context, missionID string) ([]LedgerEvent, error) {
	events := make([]LedgerEvent, 0, len(s.events))
	for _, event := range s.events {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

func (s *sourceCandidateServiceStore) ListRawArtifacts(_ context.Context, missionID string) ([]RawArtifact, error) {
	artifacts := make([]RawArtifact, 0, len(s.artifacts))
	for _, artifact := range s.artifacts {
		if artifact.MissionID == missionID {
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts, nil
}

func (s *sourceCandidateServiceStore) GetRawArtifact(_ context.Context, artifactID string) (RawArtifact, error) {
	if artifact, ok := s.artifacts[artifactID]; ok {
		return artifact, nil
	}
	return RawArtifact{}, ErrInvalidInput
}

func (s *sourceCandidateServiceStore) ListSourceSnapshots(_ context.Context, missionID string) ([]SourceSnapshot, error) {
	sources := make([]SourceSnapshot, 0, len(s.sources))
	for _, source := range s.sources {
		if source.MissionID == missionID {
			sources = append(sources, source)
		}
	}
	return sources, nil
}

func (s *sourceCandidateServiceStore) CreateSourceSnapshotWithEvent(_ context.Context, req CreateSourceSnapshotWithEventRequest) (SourceSnapshotWithEventResult, error) {
	artifact := rawArtifactFromRequest(req.Artifact)
	snapshot := sourceSnapshotFromRequest(req.Snapshot)
	if len(snapshot.ArtifactIDs) == 0 {
		snapshot.ArtifactIDs = []string{artifact.ArtifactID}
	}
	if artifact.SHA256 != "" {
		snapshot.ContentHash.Algorithm = "sha256"
		snapshot.ContentHash.Value = artifact.SHA256
	}
	event := ledgerEventFromRequest(req.Event, len(s.events)+1)
	s.storeArtifact(artifact)
	s.sources = append(s.sources, snapshot)
	s.events = append(s.events, event)
	return SourceSnapshotWithEventResult{Artifact: artifact, Snapshot: snapshot, Event: event}, nil
}

func (s *sourceCandidateServiceStore) CreateExistingArtifactSourceSnapshotWithEvent(_ context.Context, req CreateExistingArtifactSourceSnapshotWithEventRequest) (ExistingArtifactSourceSnapshotWithEventResult, error) {
	snapshot := sourceSnapshotFromRequest(req.Snapshot)
	event := ledgerEventFromRequest(req.Event, len(s.events)+1)
	s.sources = append(s.sources, snapshot)
	s.events = append(s.events, event)
	return ExistingArtifactSourceSnapshotWithEventResult{Snapshot: snapshot, Event: event}, nil
}

func (s *sourceCandidateServiceStore) CreateLiveSourceSnapshotWithEvent(_ context.Context, req CreateLiveSourceSnapshotWithEventRequest) (LiveSourceSnapshotWithEventResult, error) {
	snapshot := sourceSnapshotFromRequest(req.Snapshot)
	event := ledgerEventFromRequest(req.Event, len(s.events)+1)
	s.sources = append(s.sources, snapshot)
	s.events = append(s.events, event)
	return LiveSourceSnapshotWithEventResult{Snapshot: snapshot, Event: event}, nil
}

func (s *sourceCandidateServiceStore) AppendEvent(_ context.Context, req AppendEventRequest) (LedgerEvent, error) {
	event := ledgerEventFromRequest(req, len(s.events)+1)
	s.events = append(s.events, event)
	return event, nil
}

func (s *sourceCandidateServiceStore) storeArtifact(artifact RawArtifact) {
	if s.artifacts == nil {
		s.artifacts = map[string]RawArtifact{}
	}
	s.artifacts[artifact.ArtifactID] = artifact
}

func rawArtifactFromRequest(req CreateRawArtifactRequest) RawArtifact {
	sha := req.ExpectedSHA256
	if sha == "" {
		sha = sha256HexBytes(req.Content)
	}
	return RawArtifact{
		ArtifactID: req.ArtifactID,
		MissionID:  req.MissionID,
		MediaType:  req.MediaType,
		ByteSize:   int64(len(req.Content)),
		SHA256:     sha,
		Filename:   req.Filename,
		Producer:   req.Producer,
		CreatedAt:  time.Now().UTC(),
		Content:    req.Content,
	}
}

func sourceSnapshotFromRequest(req CreateSourceSnapshotRequest) SourceSnapshot {
	return SourceSnapshot{
		SnapshotID:        req.SnapshotID,
		MissionID:         req.MissionID,
		Connector:         req.Connector,
		Title:             req.Title,
		ExternalUpdatedAt: req.ExternalUpdatedAt,
		ArtifactIDs:       req.ArtifactIDs,
		ContentHash:       req.ContentHash,
		Locators:          req.Locators,
		Access:            req.Access,
	}
}

func ledgerEventFromRequest(req AppendEventRequest, sequence int) LedgerEvent {
	return LedgerEvent{
		EventID:          req.EventID,
		MissionID:        req.MissionID,
		Sequence:         int64(sequence),
		EventType:        req.EventType,
		Producer:         req.Producer,
		CausationEventID: req.CausationEventID,
		CorrelationID:    req.CorrelationID,
		Payload:          req.Payload,
		CreatedAt:        time.Now().UTC(),
	}
}

func assertJSONPayload(t *testing.T, raw []byte, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("payload mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func assertJSONPayloadIncludes(t *testing.T, raw []byte, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	for key, value := range want {
		if !reflect.DeepEqual(got[key], value) {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got[key], value, got)
		}
	}
}
