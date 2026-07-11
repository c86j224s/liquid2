package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
)

func TestDocumentRoutes(t *testing.T) {
	service := app.NewService(app.WithClock(func() int64 { return 1760000000000 }))
	doc, err := service.CreateDocument(context.Background(), app.CreateDocumentInput{
		Title: "Example", Kind: app.DocumentKindBookmark,
	})
	if err != nil {
		t.Fatalf("seed document: %v", err)
	}
	router := NewRouter(service)

	response := serveJSON(router, http.MethodGet, "/api/v1/documents/"+doc.Document.ID, "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"title":"Example"`) {
		t.Fatalf("expected document detail, got %s", response.Body.String())
	}

	response = serveJSON(router, http.MethodPatch, "/api/v1/documents/"+doc.Document.ID, `{"title":"Updated"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"title":"Updated"`) {
		t.Fatalf("expected updated title, got %s", response.Body.String())
	}
}

func TestAPIValidationFailure(t *testing.T) {
	router := NewRouter(app.NewService())
	response := serveJSON(router, http.MethodGet, "/api/v1/documents?ratingMin=9", "")

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected validation status 422, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "expected number") {
		t.Fatalf("expected validation detail, got %s", response.Body.String())
	}
}

func TestDocumentSearchQueryValidation(t *testing.T) {
	router := NewRouter(app.NewService())
	response := serveJSON(router, http.MethodGet, "/api/v1/documents?sort=title_asc", "")

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected validation status 422, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "expected value to be one of") {
		t.Fatalf("expected enum validation detail, got %s", response.Body.String())
	}
}

func TestDocumentNotesRoute(t *testing.T) {
	service := app.NewService(app.WithClock(func() int64 { return 1760000000000 }))
	doc, err := service.CreateDocument(context.Background(), app.CreateDocumentInput{Title: "Example"})
	if err != nil {
		t.Fatalf("seed document: %v", err)
	}
	router := NewRouter(service)

	response := serveJSON(router, http.MethodPost, "/api/v1/documents/"+doc.Document.ID+"/notes", `{"body":"Remember","format":"text"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"body":"Remember"`) {
		t.Fatalf("expected note body, got %s", response.Body.String())
	}
}

func TestOpenAPIRouteInventory(t *testing.T) {
	router := NewRouter(app.NewService())
	response := serveJSON(router, http.MethodGet, "/openapi-3.0.json", "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var spec struct {
		OpenAPI string                    `json:"openapi"`
		Paths   map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	for _, path := range []string{
		"/healthz", "/api/v1/documents", "/api/v1/documents/{id}",
		"/api/v1/documents/{id}/move-to-trash", "/api/v1/documents/{id}/notes",
		"/api/v1/documents/{id}/rescrape", "/api/v1/documents/scrape-translate",
		"/api/v1/folders", "/api/v1/tags",
		"/api/v1/feeds", "/api/v1/feeds/{id}", "/api/v1/jobs", "/api/v1/jobs/{id}",
		"/api/v1/backup", "/api/v1/export", "/api/v1/exports/{id}",
	} {
		if _, ok := spec.Paths[path]; !ok {
			t.Fatalf("expected OpenAPI path %s", path)
		}
	}
	params := spec.Paths["/api/v1/documents"]["get"].(map[string]any)["parameters"].([]any)
	for _, name := range []string{"q", "sort", "includeFolderDescendants", "includeTrash"} {
		if !hasOpenAPIParameter(params, name) {
			t.Fatalf("expected document list OpenAPI parameter %s", name)
		}
	}
}

func TestMoveDocumentToTrashRoute(t *testing.T) {
	service := app.NewService(app.WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	doc, err := service.CreateDocument(context.Background(), app.CreateDocumentInput{
		Title: "Discard", Kind: app.DocumentKindBookmark,
	})
	if err != nil {
		t.Fatalf("seed document: %v", err)
	}
	router := NewRouter(service)

	response := serveJSON(router, http.MethodPost, "/api/v1/documents/"+doc.Document.ID+"/move-to-trash", "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var detail app.DocumentDetail
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if detail.Document.FolderID == nil {
		t.Fatalf("expected trash folder id, got %#v", detail.Document)
	}

	response = serveJSON(router, http.MethodGet, "/api/v1/documents", "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), doc.Document.ID) {
		t.Fatalf("expected default list to hide trash document, got %s", response.Body.String())
	}

	response = serveJSON(router, http.MethodGet, "/api/v1/documents?folderId="+*detail.Document.FolderID, "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected trash list status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), doc.Document.ID) {
		t.Fatalf("expected folder list to include trash document, got %s", response.Body.String())
	}
}

func hasOpenAPIParameter(params []any, name string) bool {
	for _, item := range params {
		param, ok := item.(map[string]any)
		if ok && param["name"] == name {
			return true
		}
	}
	return false
}

func serveJSON(handler http.Handler, method string, path string, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
