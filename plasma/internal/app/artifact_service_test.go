package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCreateRawArtifactComputesHashAndLogicalURI(t *testing.T) {
	store := &artifactFakeStore{}
	svc := NewService(store)
	artifact, err := svc.CreateRawArtifact(context.Background(), CreateRawArtifactRequest{
		ArtifactID: "art_1",
		MissionID:  "mis_1",
		MediaType:  "text/plain",
		Filename:   "source.txt",
		Producer:   Producer{Type: "connector", ID: "liquid2"},
		Content:    []byte("hello"),
	})
	if err != nil {
		t.Fatalf("CreateRawArtifact returned error: %v", err)
	}
	if artifact.ByteSize != 5 || artifact.SHA256 == "" {
		t.Fatalf("unexpected artifact metadata: %#v", artifact)
	}
	if !strings.HasPrefix(artifact.StorageURI, "plasma-artifact://mis_1/") {
		t.Fatalf("expected logical Plasma artifact URI, got %q", artifact.StorageURI)
	}
	if strings.Contains(artifact.StorageURI, "/Users/") {
		t.Fatalf("storage uri exposes local path: %q", artifact.StorageURI)
	}
}

func TestCreateRawArtifactRejectsHashMismatch(t *testing.T) {
	svc := NewService(&artifactFakeStore{})
	_, err := svc.CreateRawArtifact(context.Background(), CreateRawArtifactRequest{
		ArtifactID:     "art_1",
		MissionID:      "mis_1",
		MediaType:      "text/plain",
		Producer:       Producer{Type: "connector", ID: "liquid2"},
		Content:        []byte("hello"),
		ExpectedSHA256: "bad",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSourceSnapshotRejectsCrossMissionArtifact(t *testing.T) {
	store := &artifactFakeStore{
		artifacts: map[string]RawArtifact{
			"art_1": {ArtifactID: "art_1", MissionID: "mis_other", SHA256: strings.Repeat("a", 64)},
		},
	}
	svc := NewService(store)
	_, err := svc.CreateSourceSnapshot(context.Background(), CreateSourceSnapshotRequest{
		SnapshotID:  "src_1",
		MissionID:   "mis_1",
		Connector:   ConnectorRef{ConnectorID: "liquid2", ConnectorType: "liquid2", ExternalSourceID: "doc_1"},
		ArtifactIDs: []string{"art_1"},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSourceSnapshotRejectsDuplicateArtifacts(t *testing.T) {
	store := &artifactFakeStore{
		artifacts: map[string]RawArtifact{
			"art_1": {ArtifactID: "art_1", MissionID: "mis_1", SHA256: strings.Repeat("a", 64)},
		},
	}
	svc := NewService(store)
	_, err := svc.CreateSourceSnapshot(context.Background(), CreateSourceSnapshotRequest{
		SnapshotID:  "src_1",
		MissionID:   "mis_1",
		Connector:   ConnectorRef{ConnectorID: "liquid2", ConnectorType: "liquid2", ExternalSourceID: "doc_1"},
		ArtifactIDs: []string{"art_1", " art_1 "},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

type artifactFakeStore struct {
	fakeStore
	artifacts map[string]RawArtifact
}

func (f *artifactFakeStore) CreateRawArtifact(_ context.Context, artifact RawArtifact) error {
	if f.artifacts == nil {
		f.artifacts = map[string]RawArtifact{}
	}
	f.artifacts[artifact.ArtifactID] = artifact
	return nil
}

func (f *artifactFakeStore) GetRawArtifact(_ context.Context, artifactID string) (RawArtifact, error) {
	if artifact, ok := f.artifacts[artifactID]; ok {
		return artifact, nil
	}
	return RawArtifact{}, errors.New("missing artifact")
}

func (f *artifactFakeStore) CreateSourceSnapshot(context.Context, SourceSnapshot) error {
	return nil
}

func (f *artifactFakeStore) GetSourceSnapshot(context.Context, string) (SourceSnapshot, error) {
	return SourceSnapshot{}, nil
}
