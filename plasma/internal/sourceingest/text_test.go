package sourceingest

import (
	"context"
	"testing"
)

func TestCreateTextSourceWithEventPreservesPayload(t *testing.T) {
	store := &sourceCandidateServiceStore{}
	result, err := CreateTextSourceWithEvent(context.Background(), store, CreateTextSourceWithEventRequest{
		MissionID:  "mis_1",
		ArtifactID: "art_text",
		SnapshotID: "src_text",
		EventID:    "evt_text",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
		Source: TextSourceContent{
			Title:   "Pasted note",
			Content: "source body",
		},
	})
	if err != nil {
		t.Fatalf("CreateTextSourceWithEvent returned error: %v", err)
	}
	if result.Artifact.Filename != "pasted-note.txt" {
		t.Fatalf("unexpected text artifact filename: %#v", result.Artifact)
	}
	if result.Snapshot.Connector.ExternalSourceID != "manual:src_text" ||
		result.Snapshot.Connector.ExternalURI != "" ||
		string(result.Snapshot.Locators) != `[{"locator_type":"full_text"}]` {
		t.Fatalf("unexpected text snapshot: %#v", result.Snapshot)
	}
	assertJSONPayload(t, result.Event.Payload, map[string]any{
		"snapshot_id":  "src_text",
		"artifact_ids": []any{"art_text"},
		"source_kind":  "text",
		"title":        "Pasted note",
	})
}

func TestCreateTextSourceWithEventPreservesExternalURI(t *testing.T) {
	store := &sourceCandidateServiceStore{}
	result, err := CreateTextSourceWithEvent(context.Background(), store, CreateTextSourceWithEventRequest{
		MissionID:  "mis_1",
		ArtifactID: "art_text",
		SnapshotID: "src_text",
		EventID:    "evt_text",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
		Source: TextSourceContent{
			Content:     "source body",
			ExternalURI: "https://example.com/source",
		},
	})
	if err != nil {
		t.Fatalf("CreateTextSourceWithEvent returned error: %v", err)
	}
	if result.Snapshot.Title != "Pasted text source" ||
		result.Snapshot.Connector.ExternalSourceID != "https://example.com/source" ||
		result.Snapshot.Connector.ExternalURI != "https://example.com/source" {
		t.Fatalf("unexpected external URI snapshot: %#v", result.Snapshot)
	}
}
