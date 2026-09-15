package researchcatalog

import (
	"encoding/json"
	"reflect"
	"testing"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func TestSummarizeSourceSnapshotLocalMediaAndUploadPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		snapshot source.Snapshot
		want     ObjectSummary
	}{
		{
			name: "local path",
			snapshot: source.Snapshot{
				SnapshotID: "src_local", MissionID: "mis_1", Title: "Local", ArtifactIDs: []string{"art_1", ""},
				Connector: source.ConnectorRef{ConnectorType: source.ConnectorTypeLocalPath, ExternalURI: "fallback"},
				Access:    source.Access{RetrievalPolicy: source.RetrievalPolicySnapshotOnly},
				Locators:  json.RawMessage(`{"locator_type":"local_path","root_id":"root-1","relative_path":"docs/a.md","path_kind":"file"}`),
				State:     source.State{Removed: true},
			},
			want: ObjectSummary{ObjectKind: ObjectSourceSnapshot, ObjectID: "src_local", MissionID: "mis_1", Summary: "Local", Refs: []ObjectRef{{ObjectKind: ObjectRawArtifact, ObjectID: "art_1"}, {ObjectKind: ObjectRawArtifact, ObjectID: ""}}, Metadata: map[string]any{"connector_type": "local_path", "retrieval_policy": "snapshot_only", "state": "active", "removed": true, "root_id": "root-1", "relative_path": "docs/a.md", "path_kind": "file"}},
		},
		{
			name:     "media",
			snapshot: source.Snapshot{SnapshotID: "src_media", MissionID: "mis_1", Connector: source.ConnectorRef{ConnectorType: source.ConnectorTypeMediaURL}, Locators: json.RawMessage(`[{"locator_type":"media","media_kind":"image","mime_type":"image/png","byte_size":9,"width":3,"height":4,"canonical_url":"https://example.com/canonical","source_page_url":"https://example.com/page","direct_media_url":"https://example.com/file","license":"MIT","attribution":"A","inspection_support":"full"}]`)},
			want:     ObjectSummary{ObjectKind: ObjectSourceSnapshot, ObjectID: "src_media", MissionID: "mis_1", Summary: "src_media", Refs: []ObjectRef{}, Metadata: map[string]any{"connector_type": "media_url", "retrieval_policy": "", "state": "active", "removed": false, "media_kind": "image", "mime_type": "image/png", "byte_size": int64(9), "width": 3, "height": 4, "canonical_url": "https://example.com/canonical", "source_page_url": "https://example.com/page", "direct_media_url": "https://example.com/file", "license": "MIT", "attribution": "A", "inspection_support": "full"}},
		},
		{
			name:     "upload PDF overwrites shared keys",
			snapshot: source.Snapshot{SnapshotID: "src_upload", MissionID: "mis_1", Connector: source.ConnectorRef{ConnectorType: source.ConnectorTypeFileUpload}, Locators: json.RawMessage(`{"locator_type":"file_upload","original_filename":"old.pdf","sanitized_filename":"new.pdf","mime_type":"application/pdf","media_type":"application/octet-stream","byte_size":11,"content_kind":"pdf","url":"https://example.com/file.pdf","sha256":"abc"}`)},
			want:     ObjectSummary{ObjectKind: ObjectSourceSnapshot, ObjectID: "src_upload", MissionID: "mis_1", Summary: "src_upload", Refs: []ObjectRef{}, Metadata: map[string]any{"connector_type": "file_upload", "retrieval_policy": "", "state": "active", "removed": false, "locator_type": "pdf_document", "original_filename": "old.pdf", "sanitized_filename": "new.pdf", "mime_type": "application/pdf", "media_type": "application/octet-stream", "byte_size": float64(11), "content_kind": "pdf", "sha256": "abc", "filename": "new.pdf", "url": "https://example.com/file.pdf"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeSourceSnapshot(tt.snapshot)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SummarizeSourceSnapshot = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSummarizeSourceSnapshotInvalidLocatorKeepsBaseMetadata(t *testing.T) {
	got := SummarizeSourceSnapshot(source.Snapshot{SnapshotID: "src_1", MissionID: "mis_1", Connector: source.ConnectorRef{ConnectorType: source.ConnectorTypeLocalPath}, Locators: json.RawMessage(`{"not":"a locator"}`)})
	want := map[string]any{"connector_type": source.ConnectorTypeLocalPath, "retrieval_policy": "", "state": source.StateActive, "removed": false}
	if !reflect.DeepEqual(got.Metadata, want) {
		t.Fatalf("metadata = %#v, want %#v", got.Metadata, want)
	}
}

func TestSummarizeRawArtifactUsesReadKindAndFallbackTitle(t *testing.T) {
	artifact := artifactcontract.Raw{ArtifactID: "art_1", MissionID: "mis_1", MediaType: "application/octet-stream", ByteSize: 4}
	want := ObjectSummary{ObjectKind: ObjectRawArtifact, ObjectID: "art_1", MissionID: "mis_1", Summary: "application/octet-stream", Metadata: map[string]any{"byte_size": int64(4), "media_type": "application/octet-stream", "read_kind": "caller-kind"}}
	if got := SummarizeRawArtifact(artifact, "caller-kind"); !reflect.DeepEqual(got, want) {
		t.Fatalf("SummarizeRawArtifact = %#v, want %#v", got, want)
	}
}

func TestSummarizeLedgerEventMalformedPayloadAndOrderedDedupe(t *testing.T) {
	event := ledger.Event{EventID: "evt_1", MissionID: "mis_1", Sequence: 7, EventType: "source.saved", Payload: json.RawMessage(`{"snapshot_id":" src_1 ","artifact_id":"art_1","artifact_ids":["art_2","", "art_1", " art_2 "]}`)}
	want := ObjectSummary{ObjectKind: ObjectLedgerEvent, ObjectID: "evt_1", MissionID: "mis_1", Summary: "#7 source.saved", Refs: []ObjectRef{{ObjectKind: ObjectRawArtifact, ObjectID: "art_1"}, {ObjectKind: ObjectRawArtifact, ObjectID: "art_2"}, {ObjectKind: ObjectSourceSnapshot, ObjectID: "src_1"}}, Metadata: map[string]any{"event_type": "source.saved", "sequence": int64(7)}}
	if got := SummarizeLedgerEvent(event); !reflect.DeepEqual(got, want) {
		t.Fatalf("SummarizeLedgerEvent = %#v, want %#v", got, want)
	}
	malformed := SummarizeLedgerEvent(ledger.Event{EventID: "evt_bad", Payload: json.RawMessage("not-json")})
	if malformed.Refs == nil || len(malformed.Refs) != 0 {
		t.Fatalf("malformed refs = %#v, want non-nil empty", malformed.Refs)
	}
}

func TestDedupeRefsEmptyFilteringOrderAndNilSemantics(t *testing.T) {
	got := DedupeRefs([]ObjectRef{{ObjectKind: "z", ObjectID: "2"}, {}, {ObjectKind: "a", ObjectID: "2"}, {ObjectKind: "z", ObjectID: "1"}, {ObjectKind: "a", ObjectID: "2"}})
	want := []ObjectRef{{ObjectKind: "a", ObjectID: "2"}, {ObjectKind: "z", ObjectID: "1"}, {ObjectKind: "z", ObjectID: "2"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DedupeRefs = %#v, want %#v", got, want)
	}
	if got := DedupeRefs(nil); got == nil || len(got) != 0 {
		t.Fatalf("DedupeRefs(nil) = %#v, want non-nil empty", got)
	}
}
