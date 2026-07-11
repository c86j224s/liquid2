package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestRawArtifactRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newArtifactTestService(t, store)

	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_1",
		MissionID:  "mis_1",
		MediaType:  "text/plain",
		Filename:   "source.txt",
		Producer:   app.Producer{Type: "connector", ID: "liquid2"},
		Content:    []byte("snapshot body"),
	})
	if err != nil {
		t.Fatalf("CreateRawArtifact returned error: %v", err)
	}

	got, err := svc.GetRawArtifact(ctx, "art_1")
	if err != nil {
		t.Fatalf("GetRawArtifact returned error: %v", err)
	}
	if got.SHA256 != artifact.SHA256 || string(got.Content) != "snapshot body" {
		t.Fatalf("unexpected artifact round trip: %#v", got)
	}
	if !strings.HasPrefix(got.StorageURI, "plasma-artifact://mis_1/") {
		t.Fatalf("unexpected storage uri: %q", got.StorageURI)
	}
}

func TestSourceSnapshotReferencesArtifacts(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newArtifactTestService(t, store)
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_1",
		MissionID:  "mis_1",
		MediaType:  "application/json",
		Producer:   app.Producer{Type: "connector", ID: "liquid2"},
		Content:    []byte(`{"id":"doc_1"}`),
	})
	if err != nil {
		t.Fatalf("CreateRawArtifact returned error: %v", err)
	}

	snapshot, err := svc.CreateSourceSnapshot(ctx, app.CreateSourceSnapshotRequest{
		SnapshotID: "src_1",
		MissionID:  "mis_1",
		Connector: app.ConnectorRef{
			ConnectorID:      "liquid2",
			ConnectorType:    "liquid2",
			ExternalSourceID: "doc_1",
			ExternalURI:      "liquid2://documents/doc_1",
		},
		Title:       "Liquid2 source",
		ArtifactIDs: []string{"art_1"},
		ContentHash: app.ContentHash{Algorithm: "sha256", Value: artifact.SHA256},
		Locators:    []byte(`[{"locator_type":"text_position","start":0,"end":4,"artifact_id":"art_1"}]`),
	})
	if err != nil {
		t.Fatalf("CreateSourceSnapshot returned error: %v", err)
	}
	got, err := svc.GetSourceSnapshot(ctx, "src_1")
	if err != nil {
		t.Fatalf("GetSourceSnapshot returned error: %v", err)
	}
	if got.ContentHash.Value != artifact.SHA256 || len(got.ArtifactIDs) != 1 || got.ArtifactIDs[0] != "art_1" {
		t.Fatalf("unexpected snapshot: %#v", got)
	}
	if snapshot.Connector.ExternalURI != got.Connector.ExternalURI {
		t.Fatalf("connector reference was not preserved: %#v", got.Connector)
	}
}

func TestSourceSnapshotRejectsUnknownArtifact(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newArtifactTestService(t, store)
	_, err := svc.CreateSourceSnapshot(ctx, app.CreateSourceSnapshotRequest{
		SnapshotID:  "src_1",
		MissionID:   "mis_1",
		Connector:   app.ConnectorRef{ConnectorID: "liquid2", ConnectorType: "liquid2", ExternalSourceID: "doc_1"},
		ArtifactIDs: []string{"art_missing"},
	})
	if err == nil {
		t.Fatal("expected missing artifact error")
	}
}

func TestSourceSnapshotRejectsHashMismatch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newArtifactTestService(t, store)
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_1",
		MissionID:  "mis_1",
		MediaType:  "text/plain",
		Producer:   app.Producer{Type: "connector", ID: "liquid2"},
		Content:    []byte("snapshot body"),
	}); err != nil {
		t.Fatalf("CreateRawArtifact returned error: %v", err)
	}
	_, err := svc.CreateSourceSnapshot(ctx, app.CreateSourceSnapshotRequest{
		SnapshotID:  "src_1",
		MissionID:   "mis_1",
		Connector:   app.ConnectorRef{ConnectorID: "liquid2", ConnectorType: "liquid2", ExternalSourceID: "doc_1"},
		ArtifactIDs: []string{"art_1"},
		ContentHash: app.ContentHash{Algorithm: "sha256", Value: strings.Repeat("b", 64)},
	})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestLiveLocalPathSourceSnapshotAllowsZeroArtifacts(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newArtifactTestService(t, store)
	snapshot, err := svc.CreateSourceSnapshot(ctx, app.CreateSourceSnapshotRequest{
		SnapshotID: "src_live",
		MissionID:  "mis_1",
		Connector: app.ConnectorRef{
			ConnectorID:      "local_path",
			ConnectorType:    app.SourceConnectorTypeLocalPath,
			ExternalSourceID: "docs:guide.md",
			ConnectorVersion: "plasma.local_path.v1",
		},
		Title:       "Guide",
		ArtifactIDs: nil,
		Locators:    json.RawMessage(`[{"locator_type":"local_path","root_id":"docs","relative_path":"guide.md","path_kind":"file"}]`),
		Access:      app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicyLiveReference},
	})
	if err != nil {
		t.Fatalf("CreateSourceSnapshot returned error: %v", err)
	}
	if len(snapshot.ArtifactIDs) != 0 {
		t.Fatalf("expected zero artifacts, got %#v", snapshot.ArtifactIDs)
	}
	if snapshot.ContentHash.Algorithm != "none" || snapshot.ContentHash.Value != "" {
		t.Fatalf("live reference must not store empty artifact hash as content hash: %#v", snapshot.ContentHash)
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "/") && strings.Contains(string(encoded), "Users") {
		t.Fatalf("live source snapshot leaked an absolute path: %s", string(encoded))
	}
	got, err := svc.GetSourceSnapshot(ctx, "src_live")
	if err != nil {
		t.Fatalf("GetSourceSnapshot returned error: %v", err)
	}
	if got.Access.RetrievalPolicy != app.SourceRetrievalPolicyLiveReference || got.Connector.ConnectorType != app.SourceConnectorTypeLocalPath {
		t.Fatalf("unexpected live source round trip: %#v", got)
	}
}

func TestZeroArtifactSnapshotRequiresLiveLocalPath(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newArtifactTestService(t, store)
	_, err := svc.CreateSourceSnapshot(ctx, app.CreateSourceSnapshotRequest{
		SnapshotID: "src_bad",
		MissionID:  "mis_1",
		Connector: app.ConnectorRef{
			ConnectorID:      "manual",
			ConnectorType:    "text",
			ExternalSourceID: "manual:src_bad",
		},
		Access: app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
	})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSourceRemovedAndRestoredProjection(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newArtifactTestService(t, store)
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_1",
		MissionID:  "mis_1",
		MediaType:  "text/plain",
		Producer:   app.Producer{Type: "user", ID: "test"},
		Content:    []byte("body"),
	})
	if err != nil {
		t.Fatalf("CreateRawArtifact returned error: %v", err)
	}
	if _, err := svc.CreateSourceSnapshot(ctx, app.CreateSourceSnapshotRequest{
		SnapshotID:  "src_1",
		MissionID:   "mis_1",
		Connector:   app.ConnectorRef{ConnectorID: "manual", ConnectorType: "text", ExternalSourceID: "manual:src_1"},
		ArtifactIDs: []string{artifact.ArtifactID},
	}); err != nil {
		t.Fatalf("CreateSourceSnapshot returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_removed",
		MissionID: "mis_1",
		EventType: app.SourceRemovedEvent,
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   json.RawMessage(`{"snapshot_id":"src_1","reason":"wrong source"}`),
	}); err != nil {
		t.Fatalf("AppendEvent source.removed returned error: %v", err)
	}
	active, err := svc.ListSourceSnapshots(ctx, "mis_1")
	if err != nil {
		t.Fatalf("ListSourceSnapshots returned error: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("removed source should be hidden by default: %#v", active)
	}
	withRemoved, err := svc.ListSourceSnapshotsWithState(ctx, app.ListSourceSnapshotsRequest{MissionID: "mis_1", IncludeRemoved: true})
	if err != nil {
		t.Fatalf("ListSourceSnapshotsWithState returned error: %v", err)
	}
	if len(withRemoved) != 1 || !withRemoved[0].State.Removed || withRemoved[0].State.RemovedEventID != "evt_removed" {
		t.Fatalf("expected removed source state, got %#v", withRemoved)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_restored",
		MissionID: "mis_1",
		EventType: app.SourceRestoredEvent,
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   json.RawMessage(`{"snapshot_id":"src_1"}`),
	}); err != nil {
		t.Fatalf("AppendEvent source.restored returned error: %v", err)
	}
	active, err = svc.ListSourceSnapshots(ctx, "mis_1")
	if err != nil {
		t.Fatalf("ListSourceSnapshots returned error: %v", err)
	}
	if len(active) != 1 || active[0].State.Removed || active[0].State.RestoredEventID != "evt_restored" {
		t.Fatalf("expected restored active source, got %#v", active)
	}
}

func newArtifactTestService(t *testing.T, store *Store) *app.Service {
	t.Helper()
	svc := app.NewService(store)
	if _, err := svc.CreateMission(context.Background(), app.CreateMissionRequest{MissionID: "mis_1", Title: "Mission"}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	return svc
}
