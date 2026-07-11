package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/ingest"
)

func TestIngestionRoutesCreateDocuments(t *testing.T) {
	service := app.NewService(app.WithClock(func() int64 { return 1760000000000 }))
	router := ingestionTestRouter(service)

	response := serveJSON(router, http.MethodPost, "/api/v1/documents/bookmark", `{"url":"https://example.com/a"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected bookmark status 201, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"kind":"bookmark"`) {
		t.Fatalf("expected bookmark document, got %s", response.Body.String())
	}

	response = serveJSON(router, http.MethodPost, "/api/v1/documents/scrape", `{"url":"https://example.com/a"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected scrape status 201, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"content":"Readable body"`) {
		t.Fatalf("expected scraped content, got %s", response.Body.String())
	}
}

func TestIngestionRoutesRejectUnsafeBookmarkURL(t *testing.T) {
	router := ingestionTestRouter(app.NewService())

	response := serveJSON(router, http.MethodPost, "/api/v1/documents/bookmark", `{"url":"http://127.0.0.1/a"}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "unsafe url") {
		t.Fatalf("expected unsafe URL error, got %s", response.Body.String())
	}
}

func TestScrapeRouteHidesFetchFailureDetails(t *testing.T) {
	service := app.NewService()
	ingestion := ingest.NewService(service, ingest.WithFetcher(failingFetcher{}))
	router := NewRouter(service, WithIngestion(ingestion))

	response := serveJSON(router, http.MethodPost, "/api/v1/documents/scrape", `{"url":"https://example.com/a?token=secret"}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "token=secret") ||
		strings.Contains(response.Body.String(), "connection refused") {
		t.Fatalf("expected sanitized error response, got %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "fetch failed") {
		t.Fatalf("expected fetch failure classification, got %s", response.Body.String())
	}
}

func TestRescrapeDocumentRoute(t *testing.T) {
	service := app.NewService(app.WithClock(func() int64 { return 1760000000000 }))
	router := ingestionTestRouter(service)
	detail, err := service.CreateScrapedDocument(context.Background(), app.ScrapedDocumentInput{
		URL: "https://example.com/final", SourceURL: "https://example.com/start",
		Title: "Article", Content: "Old body",
	})
	if err != nil {
		t.Fatalf("create scraped document: %v", err)
	}

	response := serveJSON(
		router,
		http.MethodPost,
		"/api/v1/documents/"+detail.Document.ID+"/rescrape",
		"",
	)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var rescraped app.DocumentDetail
	if err := json.NewDecoder(response.Body).Decode(&rescraped); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if rescraped.Contents[0].Content != "Readable body" {
		t.Fatalf("expected refreshed content, got %#v", rescraped.Contents)
	}
}

func TestUploadDocumentRoute(t *testing.T) {
	router := ingestionTestRouter(app.NewService(app.WithClock(func() int64 { return 1760000000000 })))

	body, contentType := multipartBody(t, "note.txt", "text/plain", "Stored body")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"mimeType":"text/plain"`) {
		t.Fatalf("expected blob metadata, got %s", response.Body.String())
	}
}

func TestUploadDocumentRouteRejectsUnsupportedMedia(t *testing.T) {
	router := ingestionTestRouter(app.NewService())

	body, contentType := multipartBody(t, "image.png", "image/png", "not a real png")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status 415, got %d: %s", response.Code, response.Body.String())
	}
}

func TestUploadDocumentRouteRejectsMissingFile(t *testing.T) {
	router := ingestionTestRouter(app.NewService())

	body, contentType := multipartFields(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", response.Code, response.Body.String())
	}
}

func TestUploadDocumentRouteRejectsOversizedFile(t *testing.T) {
	router := ingestionTestRouter(app.NewService())

	body, contentType := multipartBody(
		t,
		"large.txt",
		"text/plain",
		strings.Repeat("x", ingest.MaxUploadBytes+1),
	)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/upload", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d: %s", response.Code, response.Body.String())
	}
}

func multipartBody(t *testing.T, filename string, mimeType string, content string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "Uploaded note"); err != nil {
		t.Fatalf("write title: %v", err)
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", mimeType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return &body, writer.FormDataContentType()
}

func multipartFields(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "Uploaded note"); err != nil {
		t.Fatalf("write title: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return &body, writer.FormDataContentType()
}
