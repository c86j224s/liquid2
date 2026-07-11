package sourceevents

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestBuildConnectorSourceSnapshottedPayloadPreservesContract(t *testing.T) {
	payload := BuildConnectorSourceSnapshottedPayload(ConnectorSourceSnapshottedPayloadRequest{
		SnapshotID:  " src_1 ",
		ArtifactIDs: []string{"art_1"},
		Connector: ConnectorRef{
			ConnectorID:      "liquid2",
			ConnectorType:    "liquid2",
			ExternalSourceID: "doc_1",
			ExternalURI:      "liquid2://documents/doc_1",
			ExternalVersion:  "42",
			ConnectorVersion: "plasma.liquid2.http.v1",
		},
		Reason: " support claim ",
	})

	assertPayloadMap(t, payload, map[string]any{
		"snapshot_id":  "src_1",
		"artifact_ids": []any{"art_1"},
		"reason":       "support claim",
		"connector": map[string]any{
			"connector_id":       "liquid2",
			"connector_type":     "liquid2",
			"external_source_id": "doc_1",
			"external_uri":       "liquid2://documents/doc_1",
			"external_version":   "42",
			"connector_version":  "plasma.liquid2.http.v1",
		},
	})
}

func TestBuildSourceSnapshottedPayloadPreservesGenericAndLiveFallbackContracts(t *testing.T) {
	generic := BuildSourceSnapshottedPayload(SourceSnapshottedPayloadRequest{
		SnapshotID:         "src_generic",
		ArtifactIDs:        []string{"art_1"},
		Connector:          ConnectorRef{ConnectorID: "url", ConnectorType: "url", ExternalSourceID: "https://example.com"},
		IncludeArtifactIDs: true,
	})
	assertPayloadMap(t, generic, map[string]any{
		"snapshot_id":  "src_generic",
		"artifact_ids": []any{"art_1"},
		"connector": map[string]any{
			"connector_id":       "url",
			"connector_type":     "url",
			"external_source_id": "https://example.com",
			"external_uri":       "",
			"external_version":   "",
			"connector_version":  "",
		},
	})

	live := BuildSourceSnapshottedPayload(SourceSnapshottedPayloadRequest{
		SnapshotID: "src_live",
		Connector:  ConnectorRef{ConnectorID: "media_url", ConnectorType: "media_url", ExternalURI: "https://example.com/video"},
	})
	assertPayloadMap(t, live, map[string]any{
		"snapshot_id": "src_live",
		"connector": map[string]any{
			"connector_id":       "media_url",
			"connector_type":     "media_url",
			"external_source_id": "",
			"external_uri":       "https://example.com/video",
			"external_version":   "",
			"connector_version":  "",
		},
	})
}

func TestBuildConfluenceUpdateSourceSnapshottedPayloadPreservesContract(t *testing.T) {
	payload := BuildConfluenceUpdateSourceSnapshottedPayload(ConfluenceUpdateSourceSnapshottedPayloadRequest{
		SnapshotID:         "src_new",
		ArtifactIDs:        []string{"art_new"},
		Connector:          ConnectorRef{ConnectorID: "confluence", ConnectorType: "confluence"},
		Reason:             " refresh ",
		PreviousSnapshotID: "src_old",
		PreviousVersion:    3,
		CloudID:            " cloud_1 ",
		PageID:             " 123 ",
	})

	assertPayloadMap(t, payload, map[string]any{
		"snapshot_id":          "src_new",
		"artifact_ids":         []any{"art_new"},
		"connector":            map[string]any{"connector_id": "confluence", "connector_type": "confluence", "external_source_id": "", "external_uri": "", "external_version": "", "connector_version": ""},
		"reason":               "refresh",
		"previous_snapshot_id": "src_old",
		"previous_version":     float64(3),
		"confluence_cloud_id":  "cloud_1",
		"confluence_page_id":   "123",
	})
}

func TestBuildUploadedFileSourceSnapshottedPayloadPreservesContract(t *testing.T) {
	payload := BuildUploadedFileSourceSnapshottedPayload(UploadedFileSourceSnapshottedPayloadRequest{
		SnapshotID:        "src_1",
		ArtifactIDs:       []string{"art_1"},
		Title:             " Research PDF ",
		OriginalFilename:  " report.pdf ",
		SanitizedFilename: "report.pdf",
		MediaType:         "application/pdf",
		ContentKind:       "pdf",
		SHA256:            "abc123",
		Deduplicated:      true,
	})

	assertPayloadMap(t, payload, map[string]any{
		"snapshot_id":        "src_1",
		"artifact_ids":       []any{"art_1"},
		"source_kind":        "file_upload",
		"title":              "Research PDF",
		"original_filename":  "report.pdf",
		"sanitized_filename": "report.pdf",
		"media_type":         "application/pdf",
		"content_kind":       "pdf",
		"sha256":             "abc123",
		"deduplicated":       true,
	})
}

func assertPayloadMap(t *testing.T, raw []byte, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("payload mismatch\n got: %#v\nwant: %#v", got, want)
	}
}
