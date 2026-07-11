package liquid2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestClientSearchLiquid2SourcesUsesPublicDocumentsEndpoint(t *testing.T) {
	var gotPath string
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [{
				"id": "doc_1",
				"title": "SQLite notes",
				"kind": "scraped_article",
				"canonicalUrl": "https://example.com/sqlite",
				"sourceUrl": "https://example.com/source",
				"language": "en",
				"status": "unread",
				"updatedAt": 1781583600000,
				"tags": ["sqlite", "storage"]
			}],
			"nextCursor": "next",
			"totalCount": 1
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	result, err := client.SearchLiquid2Sources(context.Background(), app.Liquid2SourceSearchRequest{
		MissionID: "mis_1",
		Query:     "sqlite",
		Limit:     5,
		Filters:   app.Liquid2SourceFilters{Tag: "storage", IncludeTrash: true},
	})
	if err != nil {
		t.Fatalf("SearchLiquid2Sources returned error: %v", err)
	}
	if gotPath != "/api/v1/documents" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	for _, want := range []string{"q=sqlite", "sort=relevance", "limit=5", "tag=storage", "includeTrash=true"} {
		if !strings.Contains(gotQuery, want) {
			t.Fatalf("query %q missing %q", gotQuery, want)
		}
	}
	if result.NextCursor != "next" || len(result.Candidates) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	candidate := result.Candidates[0]
	if candidate.Connector.ExternalSourceID != "doc_1" ||
		candidate.Connector.ExternalURI != "liquid2://documents/doc_1" ||
		candidate.SourceURI != "https://example.com/sqlite" ||
		!candidate.CanSnapshot {
		t.Fatalf("unexpected candidate: %#v", candidate)
	}
}

func TestClientReadLiquid2SourceUsesPublicDocumentDetailEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/documents/doc_1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"document": {
				"id": "doc_1",
				"title": "SQLite notes",
				"kind": "scraped_article",
				"folderId": "folder_research",
				"canonicalUrl": null,
				"sourceUrl": "https://example.com/source",
				"language": "en",
				"status": "read",
				"rating": 5,
				"createdAt": 1781580000000,
				"updatedAt": 1781583600000
			},
			"folderPath": [{"id": "folder_research", "name": "Research"}],
			"contents": [{
				"id": "content_1",
				"role": "extracted",
				"format": "markdown",
				"language": "en",
				"content": "# SQLite\n\nBody"
			}],
			"tags": [{"id": "tag_1", "name": "SQLite", "slug": "sqlite"}],
			"blobs": []
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, WithConnectorVersion("liquid2-http.test"))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	document, err := client.ReadLiquid2Source(context.Background(), app.Liquid2SourceReadRequest{
		ExternalSourceID: "doc_1",
	})
	if err != nil {
		t.Fatalf("ReadLiquid2Source returned error: %v", err)
	}
	if document.Connector.ConnectorID != app.Liquid2ConnectorID ||
		document.Connector.ConnectorVersion != "liquid2-http.test" ||
		document.SourceURI != "https://example.com/source" {
		t.Fatalf("unexpected connector metadata: %#v", document)
	}
	if len(document.Contents) != 1 ||
		document.Contents[0].ContentID != "content_1" ||
		document.Contents[0].Content != "# SQLite\n\nBody" {
		t.Fatalf("unexpected content: %#v", document.Contents)
	}
	if !strings.Contains(string(document.Metadata), `"slug":"sqlite"`) {
		t.Fatalf("metadata did not preserve tags: %s", string(document.Metadata))
	}
	if !strings.Contains(string(document.Metadata), `"folderPath":[{"id":"folder_research"`) {
		t.Fatalf("metadata did not preserve folder path: %s", string(document.Metadata))
	}
}

func TestClientReturnsHTTPStatusErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	_, err = client.ReadLiquid2Source(context.Background(), app.Liquid2SourceReadRequest{ExternalSourceID: "doc_missing"})
	if err == nil || !strings.Contains(err.Error(), "returned 404") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestClientReadRejectsMismatchedDocumentID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"document": {
				"id": "doc_other",
				"title": "Wrong document",
				"kind": "scraped_article",
				"canonicalUrl": null,
				"sourceUrl": null,
				"language": "en",
				"status": "read",
				"createdAt": 1781580000000,
				"updatedAt": 1781583600000
			},
			"folderPath": [],
			"contents": [],
			"tags": [],
			"blobs": []
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	_, err = client.ReadLiquid2Source(context.Background(), app.Liquid2SourceReadRequest{ExternalSourceID: "doc_1"})
	if err == nil || !strings.Contains(err.Error(), "document id mismatch") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	if _, err := NewClient("localhost:8080"); err == nil {
		t.Fatal("expected invalid base URL error")
	}
}

func TestNewClientRejectsNonHTTPBaseURL(t *testing.T) {
	if _, err := NewClient("ftp://example.com"); err == nil {
		t.Fatal("expected non-http base URL error")
	}
}
