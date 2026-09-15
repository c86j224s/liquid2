package app

import (
	"context"
	"errors"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"strings"
	"testing"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func TestCreateRawArtifactComputesHashAndLogicalURI(t *testing.T) {
	store := &artifactFakeStore{}
	svc := NewService(store)
	artifact, err := svc.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID: "art_1",
		MissionID:  "mis_1",
		MediaType:  "text/plain",
		Filename:   "source.txt",
		Producer:   ledger.Producer{Type: "connector", ID: "liquid2"},
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
	_, err := svc.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID:     "art_1",
		MissionID:      "mis_1",
		MediaType:      "text/plain",
		Producer:       ledger.Producer{Type: "connector", ID: "liquid2"},
		Content:        []byte("hello"),
		ExpectedSHA256: "bad",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSourceSnapshotRejectsCrossMissionArtifact(t *testing.T) {
	store := &artifactFakeStore{
		artifacts: map[string]artifactcontract.Raw{
			"art_1": {ArtifactID: "art_1", MissionID: "mis_other", SHA256: strings.Repeat("a", 64)},
		},
	}
	svc := NewService(store)
	_, err := svc.CreateSourceSnapshot(context.Background(), sourcecontract.CreateRequest{
		SnapshotID:  "src_1",
		MissionID:   "mis_1",
		Connector:   sourcecontract.ConnectorRef{ConnectorID: "liquid2", ConnectorType: "liquid2", ExternalSourceID: "doc_1"},
		ArtifactIDs: []string{"art_1"},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSourceSnapshotRejectsDuplicateArtifacts(t *testing.T) {
	store := &artifactFakeStore{
		artifacts: map[string]artifactcontract.Raw{
			"art_1": {ArtifactID: "art_1", MissionID: "mis_1", SHA256: strings.Repeat("a", 64)},
		},
	}
	svc := NewService(store)
	_, err := svc.CreateSourceSnapshot(context.Background(), sourcecontract.CreateRequest{
		SnapshotID:  "src_1",
		MissionID:   "mis_1",
		Connector:   sourcecontract.ConnectorRef{ConnectorID: "liquid2", ConnectorType: "liquid2", ExternalSourceID: "doc_1"},
		ArtifactIDs: []string{"art_1", " art_1 "},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

type artifactFakeStore struct {
	fakeStore
	artifacts map[string]artifactcontract.Raw
}

func (f *artifactFakeStore) CreateRawArtifact(_ context.Context, artifact artifactcontract.Raw) error {
	if f.artifacts == nil {
		f.artifacts = map[string]artifactcontract.Raw{}
	}
	f.artifacts[artifact.ArtifactID] = artifact
	return nil
}

func (f *artifactFakeStore) GetRawArtifact(_ context.Context, artifactID string) (artifactcontract.Raw, error) {
	if artifact, ok := f.artifacts[artifactID]; ok {
		return artifact, nil
	}
	return artifactcontract.Raw{}, errors.New("missing artifact")
}

func (f *artifactFakeStore) CreateSourceSnapshot(context.Context, sourcecontract.Snapshot) error {
	return nil
}

func (f *artifactFakeStore) GetSourceSnapshot(context.Context, string) (sourcecontract.Snapshot, error) {
	return sourcecontract.Snapshot{}, nil
}
