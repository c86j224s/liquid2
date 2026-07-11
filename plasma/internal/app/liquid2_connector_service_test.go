package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSearchLiquid2SourcesNormalizesAndDelegates(t *testing.T) {
	connector := &fakeLiquid2Connector{
		searchResult: Liquid2SourceSearchResult{
			Candidates: []Liquid2SourceCandidate{{
				Connector: ConnectorRef{ExternalSourceID: "doc_1"},
				Title:     " Result ",
			}},
		},
	}
	svc := NewService(fakeStore{})
	result, err := svc.SearchLiquid2Sources(context.Background(), connector, Liquid2SourceSearchRequest{
		MissionID: " mis_1 ",
		Query:     " storage ",
		Limit:     500,
		Filters:   Liquid2SourceFilters{Tag: " sqlite "},
	})
	if err != nil {
		t.Fatalf("SearchLiquid2Sources returned error: %v", err)
	}
	if connector.searchRequest.MissionID != "mis_1" || connector.searchRequest.Query != "storage" {
		t.Fatalf("request was not normalized: %#v", connector.searchRequest)
	}
	if connector.searchRequest.Limit != maxLiquid2SearchLimit {
		t.Fatalf("expected capped limit, got %d", connector.searchRequest.Limit)
	}
	if result.MissionID != "mis_1" || len(result.Candidates) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	candidate := result.Candidates[0]
	if candidate.Connector.ConnectorID != Liquid2ConnectorID ||
		candidate.Connector.ExternalURI != "liquid2://documents/doc_1" ||
		!candidate.CanSnapshot {
		t.Fatalf("candidate was not normalized: %#v", candidate)
	}
}

func TestSnapshotLiquid2SourcePersistsArtifactAndSnapshot(t *testing.T) {
	updatedAt := time.Date(2026, 6, 16, 4, 30, 0, 0, time.UTC)
	store := &liquid2SnapshotFakeStore{}
	connector := &fakeLiquid2Connector{
		document: Liquid2SourceDocument{
			Connector: ConnectorRef{ExternalSourceID: "doc_1", ExternalVersion: "42"},
			Title:     "Liquid2 document",
			SourceURI: "https://example.com/source",
			UpdatedAt: updatedAt,
			Contents: []Liquid2SourceContent{{
				ContentID: "content_1",
				Role:      "extracted",
				Format:    "markdown",
				Language:  "en",
				Content:   "hello research",
			}},
			Metadata: json.RawMessage(`{"kind":"scraped_article"}`),
		},
	}
	svc := NewService(store)
	result, err := svc.SnapshotLiquid2Source(context.Background(), connector, SnapshotLiquid2SourceRequest{
		MissionID:        "mis_1",
		ArtifactID:       "art_1",
		SnapshotID:       "src_1",
		ExternalSourceID: "doc_1",
		Reason:           "support claim",
	})
	if err != nil {
		t.Fatalf("SnapshotLiquid2Source returned error: %v", err)
	}
	if connector.readRequest.ExternalSourceID != "doc_1" {
		t.Fatalf("unexpected read request: %#v", connector.readRequest)
	}
	if result.Artifact.MediaType != Liquid2SnapshotMediaType ||
		result.Artifact.Producer.Type != "connector" ||
		result.Artifact.Producer.ID != Liquid2ConnectorID {
		t.Fatalf("unexpected artifact: %#v", result.Artifact)
	}
	if !strings.Contains(string(result.Artifact.Content), `"schema_version":"plasma.liquid2.snapshot.v1"`) ||
		!strings.Contains(string(result.Artifact.Content), `"content":"hello research"`) {
		t.Fatalf("artifact did not preserve Liquid2 material: %s", string(result.Artifact.Content))
	}
	if result.Snapshot.Connector.ExternalSourceID != "doc_1" ||
		result.Snapshot.Connector.ConnectorVersion != Liquid2HTTPConnectorV1 ||
		result.Snapshot.ExternalUpdatedAt != updatedAt {
		t.Fatalf("unexpected snapshot connector metadata: %#v", result.Snapshot)
	}
	if !strings.Contains(string(result.Snapshot.Locators), `"locator_type":"liquid2_content_range"`) {
		t.Fatalf("snapshot locators were not recorded: %s", string(result.Snapshot.Locators))
	}
}

func TestSnapshotLiquid2SourceWithEventBuildsSourceSnapshottedEvent(t *testing.T) {
	store := &liquid2SnapshotFakeStore{}
	connector := &fakeLiquid2Connector{
		document: Liquid2SourceDocument{
			Connector: ConnectorRef{ExternalSourceID: "doc_1", ExternalVersion: "42"},
			Title:     "Liquid2 document",
			Contents: []Liquid2SourceContent{{
				ContentID: "content_1",
				Role:      "extracted",
				Format:    "markdown",
				Content:   "hello research",
			}},
		},
	}
	svc := NewService(store)
	result, err := svc.SnapshotLiquid2SourceWithEvent(context.Background(), connector, SnapshotLiquid2SourceWithEventRequest{
		Snapshot: SnapshotLiquid2SourceRequest{
			MissionID:        "mis_1",
			ArtifactID:       "art_1",
			SnapshotID:       "src_1",
			ExternalSourceID: "doc_1",
			Reason:           " support claim ",
		},
		EventID:  "evt_1",
		Producer: Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		t.Fatalf("SnapshotLiquid2SourceWithEvent returned error: %v", err)
	}
	if result.Event.EventID != "evt_1" || result.Event.EventType != "source.snapshotted" ||
		result.Event.Producer.Type != "user" || result.Event.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected event shell: %#v", result.Event)
	}
	assertJSONPayloadIncludes(t, result.Event.Payload, map[string]any{
		"snapshot_id":  "src_1",
		"artifact_ids": []any{"art_1"},
		"reason":       "support claim",
		"connector": map[string]any{
			"connector_id":       Liquid2ConnectorID,
			"connector_type":     Liquid2ConnectorType,
			"external_source_id": "doc_1",
			"external_uri":       "liquid2://documents/doc_1",
			"external_version":   "42",
			"connector_version":  Liquid2HTTPConnectorV1,
		},
	})
}

func TestSnapshotLiquid2SourceAppliesContentRange(t *testing.T) {
	store := &liquid2SnapshotFakeStore{}
	connector := &fakeLiquid2Connector{
		document: Liquid2SourceDocument{
			Connector: ConnectorRef{ExternalSourceID: "doc_1"},
			Title:     "Liquid2 document",
			Contents: []Liquid2SourceContent{{
				ContentID: "content_1",
				Role:      "extracted",
				Format:    "text",
				Content:   "abcdef",
			}},
		},
	}
	svc := NewService(store)
	result, err := svc.SnapshotLiquid2Source(context.Background(), connector, SnapshotLiquid2SourceRequest{
		MissionID:        "mis_1",
		ArtifactID:       "art_1",
		SnapshotID:       "src_1",
		ExternalSourceID: "doc_1",
		ContentRanges:    []Liquid2ContentRange{{ContentID: "content_1", Start: 1, End: 4}},
	})
	if err != nil {
		t.Fatalf("SnapshotLiquid2Source returned error: %v", err)
	}
	if !strings.Contains(string(result.Artifact.Content), `"content":"bcd"`) ||
		strings.Contains(string(result.Artifact.Content), `"content":"abcdef"`) {
		t.Fatalf("artifact did not store the requested excerpt: %s", string(result.Artifact.Content))
	}
	if !strings.Contains(string(result.Snapshot.Locators), `"start":1`) ||
		!strings.Contains(string(result.Snapshot.Locators), `"end":4`) {
		t.Fatalf("locator did not record range: %s", string(result.Snapshot.Locators))
	}
}

func TestSnapshotLiquid2SourceRejectsReturnedSourceMismatch(t *testing.T) {
	connector := &fakeLiquid2Connector{
		document: Liquid2SourceDocument{
			Connector: ConnectorRef{ExternalSourceID: "doc_other"},
			Contents:  []Liquid2SourceContent{{ContentID: "content_1", Content: "abcdef"}},
		},
	}
	svc := NewService(&liquid2SnapshotFakeStore{})
	_, err := svc.SnapshotLiquid2Source(context.Background(), connector, SnapshotLiquid2SourceRequest{
		MissionID:        "mis_1",
		ArtifactID:       "art_1",
		SnapshotID:       "src_1",
		ExternalSourceID: "doc_1",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSnapshotLiquid2SourceRejectsProducerOverride(t *testing.T) {
	connector := &fakeLiquid2Connector{
		document: Liquid2SourceDocument{
			Connector: ConnectorRef{ExternalSourceID: "doc_1"},
			Contents:  []Liquid2SourceContent{{ContentID: "content_1", Content: "abcdef"}},
		},
	}
	svc := NewService(&liquid2SnapshotFakeStore{})
	_, err := svc.SnapshotLiquid2Source(context.Background(), connector, SnapshotLiquid2SourceRequest{
		MissionID:        "mis_1",
		ArtifactID:       "art_1",
		SnapshotID:       "src_1",
		ExternalSourceID: "doc_1",
		Producer:         Producer{Type: "user", ID: "ses_1"},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSnapshotLiquid2SourceRejectsUnlocatableContent(t *testing.T) {
	tests := []struct {
		name     string
		contents []Liquid2SourceContent
	}{
		{name: "empty list"},
		{name: "blank id", contents: []Liquid2SourceContent{{ContentID: " ", Content: "abcdef"}}},
		{name: "empty body", contents: []Liquid2SourceContent{{ContentID: "content_1", Content: " "}}},
		{name: "duplicate id", contents: []Liquid2SourceContent{
			{ContentID: "content_1", Content: "abcdef"},
			{ContentID: " content_1 ", Content: "ghijkl"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connector := &fakeLiquid2Connector{
				document: Liquid2SourceDocument{
					Connector: ConnectorRef{ExternalSourceID: "doc_1"},
					Contents:  tt.contents,
				},
			}
			svc := NewService(&liquid2SnapshotFakeStore{})
			_, err := svc.SnapshotLiquid2Source(context.Background(), connector, SnapshotLiquid2SourceRequest{
				MissionID:        "mis_1",
				ArtifactID:       "art_1",
				SnapshotID:       "src_1",
				ExternalSourceID: "doc_1",
			})
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestSnapshotLiquid2SourceRejectsUnknownContentRange(t *testing.T) {
	connector := &fakeLiquid2Connector{
		document: Liquid2SourceDocument{
			Connector: ConnectorRef{ExternalSourceID: "doc_1"},
			Contents:  []Liquid2SourceContent{{ContentID: "content_1", Content: "abcdef"}},
		},
	}
	svc := NewService(&liquid2SnapshotFakeStore{})
	_, err := svc.SnapshotLiquid2Source(context.Background(), connector, SnapshotLiquid2SourceRequest{
		MissionID:        "mis_1",
		ArtifactID:       "art_1",
		SnapshotID:       "src_1",
		ExternalSourceID: "doc_1",
		ContentRanges:    []Liquid2ContentRange{{ContentID: "missing", Start: 0, End: 1}},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

type fakeLiquid2Connector struct {
	searchRequest Liquid2SourceSearchRequest
	searchResult  Liquid2SourceSearchResult
	searchErr     error
	readRequest   Liquid2SourceReadRequest
	document      Liquid2SourceDocument
	readErr       error
}

func (f *fakeLiquid2Connector) SearchLiquid2Sources(_ context.Context, req Liquid2SourceSearchRequest) (Liquid2SourceSearchResult, error) {
	f.searchRequest = req
	if f.searchErr != nil {
		return Liquid2SourceSearchResult{}, f.searchErr
	}
	return f.searchResult, nil
}

func (f *fakeLiquid2Connector) ReadLiquid2Source(_ context.Context, req Liquid2SourceReadRequest) (Liquid2SourceDocument, error) {
	f.readRequest = req
	if f.readErr != nil {
		return Liquid2SourceDocument{}, f.readErr
	}
	return f.document, nil
}

type liquid2SnapshotFakeStore struct {
	fakeStore
	artifacts map[string]RawArtifact
	snapshots map[string]SourceSnapshot
	events    []LedgerEvent
}

func (f *liquid2SnapshotFakeStore) CreateRawArtifact(_ context.Context, artifact RawArtifact) error {
	if f.artifacts == nil {
		f.artifacts = map[string]RawArtifact{}
	}
	f.artifacts[artifact.ArtifactID] = artifact
	return nil
}

func (f *liquid2SnapshotFakeStore) GetRawArtifact(_ context.Context, artifactID string) (RawArtifact, error) {
	if artifact, ok := f.artifacts[artifactID]; ok {
		return artifact, nil
	}
	return RawArtifact{}, errors.New("missing artifact")
}

func (f *liquid2SnapshotFakeStore) CreateSourceSnapshot(_ context.Context, snapshot SourceSnapshot) error {
	if f.snapshots == nil {
		f.snapshots = map[string]SourceSnapshot{}
	}
	f.snapshots[snapshot.SnapshotID] = snapshot
	return nil
}

func (f *liquid2SnapshotFakeStore) GetSourceSnapshot(_ context.Context, snapshotID string) (SourceSnapshot, error) {
	if snapshot, ok := f.snapshots[snapshotID]; ok {
		return snapshot, nil
	}
	return SourceSnapshot{}, errors.New("missing snapshot")
}

func (f *liquid2SnapshotFakeStore) CommitAtomicWrite(_ context.Context, write AtomicWrite) (AtomicWriteResult, error) {
	if f.artifacts == nil {
		f.artifacts = map[string]RawArtifact{}
	}
	if f.snapshots == nil {
		f.snapshots = map[string]SourceSnapshot{}
	}
	for i, event := range write.Events {
		event.Sequence = int64(len(f.events) + 1)
		f.events = append(f.events, event)
		write.Events[i] = event
	}
	for _, artifact := range write.RawArtifacts {
		f.artifacts[artifact.ArtifactID] = artifact
	}
	for _, snapshot := range write.SourceSnapshots {
		f.snapshots[snapshot.SnapshotID] = snapshot
	}
	return AtomicWriteResult{Events: write.Events}, nil
}
