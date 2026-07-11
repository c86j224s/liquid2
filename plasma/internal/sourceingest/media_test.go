package sourceingest

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCreateFetchedMediaURLSourceWithEventStoresImageArtifact(t *testing.T) {
	store := &sourceCandidateServiceStore{}
	result, err := CreateFetchedMediaURLSourceWithEvent(context.Background(), store, CreateFetchedMediaURLSourceRequest{
		MissionID:   "mis_1",
		URL:         "https://example.com/image.png",
		Title:       "Example image",
		License:     "CC-BY",
		Attribution: "Example author",
		ArtifactID:  "art_image",
		SnapshotID:  "src_image",
		EventID:     "evt_image",
		Producer:    Producer{Type: "user", ID: "plasma-ui"},
		Fetched: FetchedMediaSource{
			Content:   []byte("fake-png-bytes"),
			MediaType: "image/png",
			MediaKind: MediaKindImage,
			ByteSize:  int64(len("fake-png-bytes")),
			Width:     640,
			Height:    480,
		},
	})
	if err != nil {
		t.Fatalf("CreateFetchedMediaURLSourceWithEvent returned error: %v", err)
	}
	if !result.HasArtifact || result.Artifact.Filename != "example-image.png" {
		t.Fatalf("expected image artifact result, got %#v", result)
	}
	locator := oneMediaLocator(t, result.Snapshot.Locators)
	if locator.MediaKind != MediaKindImage ||
		locator.InspectionSupport != "metadata_only_until_vision_engine_configured" ||
		locator.SHA256 == "" ||
		locator.License != "CC-BY" ||
		locator.Attribution != "Example author" {
		t.Fatalf("unexpected image locator: %#v", locator)
	}
	assertJSONPayload(t, result.Event.Payload, map[string]any{
		"snapshot_id":  "src_image",
		"artifact_ids": []any{"art_image"},
		"source_kind":  SourceConnectorTypeMediaURL,
		"media_kind":   MediaKindImage,
		"title":        "Example image",
		"url":          "https://example.com/image.png",
	})
}

func TestCreateFetchedMediaURLSourceWithEventStoresAudioLiveReference(t *testing.T) {
	store := &sourceCandidateServiceStore{}
	result, err := CreateFetchedMediaURLSourceWithEvent(context.Background(), store, CreateFetchedMediaURLSourceRequest{
		MissionID:  "mis_1",
		URL:        "https://example.com/sound.mp3",
		Title:      "Example audio",
		SnapshotID: "src_audio",
		EventID:    "evt_audio",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
		Fetched: FetchedMediaSource{
			MediaType: "audio/mpeg",
			MediaKind: MediaKindAudio,
			ByteSize:  12345,
		},
	})
	if err != nil {
		t.Fatalf("CreateFetchedMediaURLSourceWithEvent returned error: %v", err)
	}
	if result.HasArtifact || len(result.Snapshot.ArtifactIDs) != 0 {
		t.Fatalf("expected live reference without artifact, got %#v", result)
	}
	if result.Snapshot.Access.RetrievalPolicy != SourceRetrievalPolicyLiveReference {
		t.Fatalf("expected live reference access, got %#v", result.Snapshot.Access)
	}
	locator := oneMediaLocator(t, result.Snapshot.Locators)
	if locator.MediaKind != MediaKindAudio || locator.InspectionSupport != "inspect_unsupported" || locator.License != "unknown" {
		t.Fatalf("unexpected audio locator: %#v", locator)
	}
	assertJSONPayload(t, result.Event.Payload, map[string]any{
		"snapshot_id": "src_audio",
		"source_kind": SourceConnectorTypeMediaURL,
		"media_kind":  MediaKindAudio,
		"title":       "Example audio",
		"url":         "https://example.com/sound.mp3",
	})
}

func oneMediaLocator(t *testing.T, raw json.RawMessage) MediaLocator {
	t.Helper()
	var locators []MediaLocator
	if err := json.Unmarshal(raw, &locators); err != nil {
		t.Fatalf("unmarshal locators: %v", err)
	}
	if len(locators) != 1 {
		t.Fatalf("expected one locator, got %#v", locators)
	}
	return locators[0]
}
