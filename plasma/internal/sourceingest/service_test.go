package sourceingest

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCreateFetchedURLSourceWithEventPreservesPayload(t *testing.T) {
	store := &sourceCandidateServiceStore{}
	result, err := CreateFetchedURLSourceWithEvent(context.Background(), store, CreateFetchedURLSourceRequest{
		MissionID:  "mis_1",
		URL:        "https://example.com/article",
		Title:      "Article",
		ArtifactID: "art_url",
		SnapshotID: "src_url",
		EventID:    "evt_url",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
		Fetched: FetchedURLSource{
			Content:           []byte("<html><title>Article</title></html>"),
			MediaType:         "text/html; charset=utf-8",
			Title:             "Fetched title",
			ExternalVersion:   "etag=v1",
			ExternalUpdatedAt: time.Date(2026, 7, 1, 2, 3, 4, 0, time.UTC),
		},
		FetchedAt: time.Date(2026, 7, 2, 3, 4, 5, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateFetchedURLSourceWithEvent returned error: %v", err)
	}
	if result.Artifact.Filename != "article.html" || result.Snapshot.Connector.ConnectorType != "url" {
		t.Fatalf("unexpected URL source result: %#v", result)
	}
	assertJSONPayload(t, result.Event.Payload, map[string]any{
		"snapshot_id":  "src_url",
		"artifact_ids": []any{"art_url"},
		"source_kind":  "url",
		"title":        "Article",
		"url":          "https://example.com/article",
	})
}

func TestCreateFetchedURLSourceWithEventRecordsBrowserRenderCandidateDiagnostic(t *testing.T) {
	store := &sourceCandidateServiceStore{}
	result, err := CreateFetchedURLSourceWithEvent(context.Background(), store, CreateFetchedURLSourceRequest{
		MissionID:  "mis_1",
		URL:        "https://example.com/app",
		Title:      "App",
		ArtifactID: "art_url",
		SnapshotID: "src_url",
		EventID:    "evt_url",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
		Fetched: FetchedURLSource{
			Content:   []byte(sourceIngestBrowserRenderHTMLFixture()),
			MediaType: "text/html; charset=utf-8",
		},
	})
	if err != nil {
		t.Fatalf("CreateFetchedURLSourceWithEvent returned error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(result.Event.Payload, &payload); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	diagnosis, ok := payload["browser_render_candidate"].(map[string]any)
	if !ok || diagnosis["candidate"] != true {
		t.Fatalf("expected browser render candidate diagnosis, got %#v", payload)
	}
}

func TestCreateFetchedURLSourceWithEventRecordsBrowserRenderMetadata(t *testing.T) {
	renderedAt := time.Date(2026, 7, 10, 1, 2, 3, 0, time.UTC)
	store := &sourceCandidateServiceStore{}
	result, err := CreateFetchedURLSourceWithEvent(context.Background(), store, CreateFetchedURLSourceRequest{
		MissionID:                      "mis_1",
		URL:                            "https://example.com/app",
		Title:                          "Rendered",
		ArtifactID:                     "art_url",
		SnapshotID:                     "src_url",
		EventID:                        "evt_url",
		Producer:                       Producer{Type: "user", ID: "plasma-ui"},
		SourceCandidateProposalEventID: "evt_candidate_proposed",
		Fetched: FetchedURLSource{
			Content:            []byte("<!doctype html><html><body><main>Rendered article body</main></body></html>"),
			MediaType:          "text/html; charset=utf-8",
			Title:              "Rendered",
			RetrievalMethod:    "browser_render",
			FinalURL:           "https://example.com/app?loaded=1",
			RenderedAt:         renderedAt,
			RawFetchSHA256:     strings.Repeat("a", 64),
			RawFetchArtifactID: "art_candidate_raw",
		},
	})
	if err != nil {
		t.Fatalf("CreateFetchedURLSourceWithEvent returned error: %v", err)
	}
	assertJSONPayloadIncludes(t, result.Event.Payload, map[string]any{
		"retrieval_method":                   "browser_render",
		"final_url":                          "https://example.com/app?loaded=1",
		"rendered_at":                        renderedAt.Format(time.RFC3339Nano),
		"raw_fetch_sha256":                   strings.Repeat("a", 64),
		"raw_fetch_artifact_id":              "art_candidate_raw",
		"source_candidate_proposal_event_id": "evt_candidate_proposed",
	})
	var locators []map[string]string
	if err := json.Unmarshal(result.Snapshot.Locators, &locators); err != nil {
		t.Fatalf("locators are not JSON: %v", err)
	}
	if len(locators) != 1 || locators[0]["retrieval_method"] != "browser_render" ||
		locators[0]["final_url"] != "https://example.com/app?loaded=1" ||
		locators[0]["rendered_at"] != renderedAt.Format(time.RFC3339Nano) {
		t.Fatalf("expected browser render locator metadata, got %#v", locators)
	}
}

func TestBuildSourceSnapshotFailureAppendRequestPreservesPayloadContract(t *testing.T) {
	req := BuildSourceSnapshotFailureAppendRequest(SourceSnapshotFailureAppendRequest{
		EventID:    "evt_failed",
		MissionID:  "mis_1",
		SourceKind: "url",
		URL:        "https://example.com/locked",
		Message:    "텍스트 소스 가져오기 실패: HTTP 401",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
	})
	if req.EventID != "evt_failed" || req.MissionID != "mis_1" ||
		req.EventType != "source.snapshot_failed" || req.Producer.Type != "user" || req.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected source snapshot failure event request: %#v", req)
	}
	assertJSONPayload(t, req.Payload, map[string]any{
		"kind":        "source_snapshot_failed",
		"source_kind": "url",
		"url":         "https://example.com/locked",
		"message":     "텍스트 소스 가져오기 실패: HTTP 401",
	})
}

func TestCreateStagedURLSourceWithEventPreservesReusePayload(t *testing.T) {
	store := &sourceCandidateServiceStore{artifacts: map[string]RawArtifact{
		"art_staged": {
			ArtifactID: "art_staged",
			MissionID:  "mis_1",
			MediaType:  "text/plain; charset=utf-8",
			SHA256:     strings.Repeat("a", 64),
			Content:    []byte("candidate"),
			CreatedAt:  time.Date(2026, 7, 2, 3, 4, 5, 0, time.UTC),
			Producer:   Producer{Type: "agent_session", ID: "ses_1"},
		},
	}}
	result, err := CreateStagedURLSourceWithEvent(context.Background(), store, CreateStagedURLSourceRequest{
		MissionID:  "mis_1",
		URL:        "https://example.com/candidate",
		Title:      "",
		SnapshotID: "src_staged",
		EventID:    "evt_staged_url",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
		Staged: StagedSourceCandidate{
			URL:             "https://example.com/candidate",
			Title:           "Candidate",
			ProposalEventID: "evt_proposed",
			Artifact:        store.artifacts["art_staged"],
			ExternalVersion: "etag=v1",
		},
	})
	if err != nil {
		t.Fatalf("CreateStagedURLSourceWithEvent returned error: %v", err)
	}
	if !result.ReusedSourceCandidate || result.Artifact.ArtifactID != "art_staged" {
		t.Fatalf("expected staged artifact reuse, got %#v", result)
	}
	assertJSONPayload(t, result.Event.Payload, map[string]any{
		"snapshot_id":                        "src_staged",
		"artifact_ids":                       []any{"art_staged"},
		"source_kind":                        "url",
		"title":                              "Candidate",
		"url":                                "https://example.com/candidate",
		"source_candidate_proposal_event_id": "evt_proposed",
		"source_candidate_artifact_reused":   true,
	})
}

func sourceIngestBrowserRenderHTMLFixture() string {
	return `<html><head><title>Client App</title>` +
		strings.Repeat(`<script src="/assets/chunk.js"></script>`, 5) +
		`<script>window.__INITIAL_STATE__={};` + strings.Repeat("bundle();", 300) + `</script>` +
		`</head><body><div id="root" data-reactroot=""></div><noscript>Please enable JavaScript to view this page.</noscript></body></html>`
}

func TestCreateFetchedPDFURLSourceWithEventPreservesPayload(t *testing.T) {
	pdf := sourceIngestTestPDF(t, []string{"PDF Source"})
	store := &sourceCandidateServiceStore{}
	result, err := CreateFetchedPDFURLSourceWithEvent(context.Background(), store, CreateFetchedPDFURLSourceRequest{
		MissionID:  "mis_1",
		URL:        "https://example.com/source.pdf",
		Title:      "PDF Source",
		ArtifactID: "art_pdf",
		SnapshotID: "src_pdf",
		EventID:    "evt_pdf",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
		Fetched: FetchedURLSource{
			Content:         pdf,
			MediaType:       "application/pdf",
			ByteSize:        int64(len(pdf)),
			PageCount:       3,
			TextLength:      123,
			TextLengthKnown: true,
		},
	})
	if err != nil {
		t.Fatalf("CreateFetchedPDFURLSourceWithEvent returned error: %v", err)
	}
	if result.Artifact.Filename != "pdf-source.pdf" || result.Snapshot.Connector.ConnectorType != SourceConnectorTypePDFURL {
		t.Fatalf("unexpected PDF source result: %#v", result)
	}
	assertJSONPayload(t, result.Event.Payload, map[string]any{
		"snapshot_id":       "src_pdf",
		"artifact_ids":      []any{"art_pdf"},
		"source_kind":       SourceConnectorTypePDFURL,
		"title":             "PDF Source",
		"url":               "https://example.com/source.pdf",
		"page_count":        float64(3),
		"text_length":       float64(123),
		"text_length_known": true,
	})
}

func TestCreateStagedPDFURLSourceWithEventPreservesReusePayload(t *testing.T) {
	pdf := sourceIngestTestPDF(t, []string{"Staged PDF Source"})
	store := &sourceCandidateServiceStore{artifacts: map[string]RawArtifact{
		"art_staged_pdf": {
			ArtifactID: "art_staged_pdf",
			MissionID:  "mis_1",
			MediaType:  "application/pdf",
			SHA256:     sha256HexBytes(pdf),
			Content:    pdf,
			CreatedAt:  time.Date(2026, 7, 2, 3, 4, 5, 0, time.UTC),
			Producer:   Producer{Type: "agent_session", ID: "ses_1"},
		},
	}}
	result, err := CreateStagedPDFURLSourceWithEvent(context.Background(), store, CreateStagedPDFURLSourceRequest{
		MissionID:  "mis_1",
		URL:        "https://example.com/candidate.pdf",
		SnapshotID: "src_staged_pdf",
		EventID:    "evt_staged_pdf",
		Producer:   Producer{Type: "user", ID: "plasma-ui"},
		Staged: StagedSourceCandidate{
			URL:             "https://example.com/candidate.pdf",
			Title:           "Staged PDF",
			ProposalEventID: "evt_pdf_proposed",
			Artifact:        store.artifacts["art_staged_pdf"],
		},
	})
	if err != nil {
		t.Fatalf("CreateStagedPDFURLSourceWithEvent returned error: %v", err)
	}
	if !result.ReusedSourceCandidate || result.Artifact.ArtifactID != "art_staged_pdf" {
		t.Fatalf("expected staged PDF artifact reuse, got %#v", result)
	}
	assertJSONPayload(t, result.Event.Payload, map[string]any{
		"snapshot_id":                        "src_staged_pdf",
		"artifact_ids":                       []any{"art_staged_pdf"},
		"source_kind":                        SourceConnectorTypePDFURL,
		"title":                              "Staged PDF",
		"url":                                "https://example.com/candidate.pdf",
		"page_count":                         float64(1),
		"text_length":                        float64(0),
		"text_length_known":                  false,
		"source_candidate_proposal_event_id": "evt_pdf_proposed",
		"source_candidate_artifact_reused":   true,
	})
}
