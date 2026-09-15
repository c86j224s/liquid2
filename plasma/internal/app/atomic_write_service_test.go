package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

type atomicSourceStore struct {
	fakeStore
	artifacts map[string]artifactcontract.Raw
	writes    []AtomicWrite
	commitErr error
}

func (s *atomicSourceStore) GetRawArtifact(_ context.Context, id string) (artifactcontract.Raw, error) {
	if artifact, ok := s.artifacts[id]; ok {
		return artifact, nil
	}
	return artifactcontract.Raw{}, errors.New("missing artifact")
}

func (s *atomicSourceStore) CommitAtomicWrite(_ context.Context, write AtomicWrite) (AtomicWriteResult, error) {
	if s.commitErr != nil {
		return AtomicWriteResult{}, s.commitErr
	}
	s.writes = append(s.writes, write)
	for i := range write.Events {
		write.Events[i].Sequence = int64(i + 1)
	}
	return AtomicWriteResult{Events: write.Events}, nil
}

func atomicEvent(eventType string) ledger.AppendRequest {
	return ledger.AppendRequest{EventID: "evt_1", MissionID: "mis_1", EventType: eventType, Producer: ledger.Producer{Type: "test", ID: "atomic"}}
}

func atomicSnapshot(ids []string) source.CreateRequest {
	return source.CreateRequest{SnapshotID: "src_1", MissionID: "mis_1", Connector: source.ConnectorRef{ConnectorID: "connector", ConnectorType: "test", ExternalSourceID: "external"}, ArtifactIDs: ids}
}

func TestAtomicSourceSnapshotWritesExpectedBundlesAndDefaultPayloads(t *testing.T) {
	t.Run("new artifact", func(t *testing.T) {
		store := &atomicSourceStore{}
		result, err := NewService(store).CreateSourceSnapshotWithEvent(context.Background(), source.CreateSourceSnapshotWithEventRequest{
			Artifact: artifactcontract.CreateRequest{ArtifactID: "art_1", MissionID: "mis_1", MediaType: "text/plain", Producer: ledger.Producer{Type: "test", ID: "atomic"}, Content: []byte("body")},
			Snapshot: atomicSnapshot(nil), Event: atomicEvent(sourceevents.SourceSnapshottedEventType),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(store.writes) != 1 || len(store.writes[0].Events) != 1 || len(store.writes[0].RawArtifacts) != 1 || len(store.writes[0].SourceSnapshots) != 1 {
			t.Fatalf("unexpected write: %#v", store.writes)
		}
		if !strings.Contains(string(result.Event.Payload), "art_1") || !strings.Contains(string(result.Event.Payload), "src_1") {
			t.Fatalf("default payload=%s", result.Event.Payload)
		}
	})

	t.Run("existing artifact", func(t *testing.T) {
		store := &atomicSourceStore{artifacts: map[string]artifactcontract.Raw{"art_1": {ArtifactID: "art_1", MissionID: "mis_1", SHA256: strings.Repeat("a", 64)}}}
		result, err := NewService(store).CreateExistingArtifactSourceSnapshotWithEvent(context.Background(), source.CreateExistingArtifactSourceSnapshotWithEventRequest{Snapshot: atomicSnapshot([]string{"art_1"}), Event: atomicEvent(sourceevents.SourceSnapshottedEventType)})
		if err != nil {
			t.Fatal(err)
		}
		if len(store.writes) != 1 || len(store.writes[0].RawArtifacts) != 0 || len(store.writes[0].SourceSnapshots) != 1 || !strings.Contains(string(result.Event.Payload), "art_1") {
			t.Fatalf("unexpected write/result: %#v %#v", store.writes, result)
		}
	})

	t.Run("live reference", func(t *testing.T) {
		store := &atomicSourceStore{}
		req := atomicSnapshot(nil)
		req.Connector = source.ConnectorRef{ConnectorID: "local", ConnectorType: source.ConnectorTypeLocalPath, ExternalSourceID: "root:file.txt"}
		req.Locators = json.RawMessage(`{"locator_type":"local_path","root_id":"root","relative_path":"file.txt","path_kind":"file"}`)
		req.Access.RetrievalPolicy = source.RetrievalPolicyLiveReference
		result, err := NewService(store).CreateLiveSourceSnapshotWithEvent(context.Background(), source.CreateLiveSourceSnapshotWithEventRequest{Snapshot: req, Event: atomicEvent(sourceevents.SourceSnapshottedEventType)})
		if err != nil {
			t.Fatal(err)
		}
		if len(store.writes) != 1 || len(store.writes[0].RawArtifacts) != 0 || len(result.Snapshot.ArtifactIDs) != 0 || result.Snapshot.ContentHash != (source.ContentHash{Algorithm: "none", Value: ""}) {
			t.Fatalf("unexpected live result/write: %#v %#v", result, store.writes)
		}
		if strings.Contains(string(result.Event.Payload), "artifact_ids") {
			t.Fatalf("live default payload invented artifact IDs: %s", result.Event.Payload)
		}
	})
}

func TestAtomicSourceSnapshotValidationDoesNotCommit(t *testing.T) {
	cases := []struct {
		name string
		call func(*Service) error
	}{
		{"new wrong event", func(s *Service) error {
			_, err := s.CreateSourceSnapshotWithEvent(context.Background(), source.CreateSourceSnapshotWithEventRequest{Artifact: artifactcontract.CreateRequest{ArtifactID: "art_1", MissionID: "mis_1", MediaType: "text/plain", Producer: ledger.Producer{Type: "test", ID: "atomic"}, Content: []byte("body")}, Snapshot: atomicSnapshot(nil), Event: atomicEvent("wrong")})
			return err
		}},
		{"existing invalid snapshot", func(s *Service) error {
			_, err := s.CreateExistingArtifactSourceSnapshotWithEvent(context.Background(), source.CreateExistingArtifactSourceSnapshotWithEventRequest{Snapshot: atomicSnapshot(nil), Event: atomicEvent(sourceevents.SourceSnapshottedEventType)})
			return err
		}},
		{"live wrong event", func(s *Service) error {
			req := atomicSnapshot(nil)
			req.Connector = source.ConnectorRef{ConnectorID: "local", ConnectorType: source.ConnectorTypeLocalPath, ExternalSourceID: "root:file.txt"}
			req.Locators = json.RawMessage(`{"locator_type":"local_path","root_id":"root","relative_path":"file.txt","path_kind":"file"}`)
			req.Access.RetrievalPolicy = source.RetrievalPolicyLiveReference
			_, err := s.CreateLiveSourceSnapshotWithEvent(context.Background(), source.CreateLiveSourceSnapshotWithEventRequest{Snapshot: req, Event: atomicEvent("wrong")})
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &atomicSourceStore{artifacts: map[string]artifactcontract.Raw{"art_1": {ArtifactID: "art_1", MissionID: "mis_1", SHA256: strings.Repeat("a", 64)}}}
			if err := tc.call(NewService(store)); err == nil {
				t.Fatal("expected validation error")
			}
			if len(store.writes) != 0 {
				t.Fatalf("validation committed writes: %#v", store.writes)
			}
		})
	}
}

func TestAtomicSourceSnapshotCommitFailureDoesNotReportCommittedState(t *testing.T) {
	want := errors.New("commit failed")
	store := &atomicSourceStore{commitErr: want}
	_, err := NewService(store).CreateSourceSnapshotWithEvent(context.Background(), source.CreateSourceSnapshotWithEventRequest{
		Artifact: artifactcontract.CreateRequest{ArtifactID: "art_1", MissionID: "mis_1", MediaType: "text/plain", Producer: ledger.Producer{Type: "test", ID: "atomic"}, Content: []byte("body")}, Snapshot: atomicSnapshot(nil), Event: atomicEvent(sourceevents.SourceSnapshottedEventType),
	})
	if !errors.Is(err, want) || len(store.writes) != 0 {
		t.Fatalf("err=%v writes=%#v", err, store.writes)
	}
}
