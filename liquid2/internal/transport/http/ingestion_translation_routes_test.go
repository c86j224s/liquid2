package httptransport

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/translation"
)

func TestScrapeTranslateRoute(t *testing.T) {
	service := app.NewService(app.WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	translator := &fakeDocumentTranslator{job: app.Job{
		ID: "job_1", Kind: app.JobKindTranslateDocument, Status: app.JobStatusQueued,
	}}
	router := ingestionTestRouterWithTranslator(service, translator)

	response := serveJSON(router, http.MethodPost, "/api/v1/documents/scrape-translate", `{
		"url":"https://example.com/a",
		"targetLanguage":"KO"
	}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", response.Code, response.Body.String())
	}
	if translator.input.DocumentID == "" || translator.input.SourceContentID == "" {
		t.Fatalf("expected translator input from scraped document, got %#v", translator.input)
	}
	if translator.input.TargetLanguage != "ko" {
		t.Fatalf("expected normalized language, got %q", translator.input.TargetLanguage)
	}
	body := response.Body.String()
	if !strings.Contains(body, `"content":"Readable body"`) ||
		!strings.Contains(body, `"kind":"translate_document"`) {
		t.Fatalf("expected scraped document and translation job, got %s", body)
	}
}

func TestScrapeTranslateRouteRequiresTranslatorBeforeScrape(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := ingestionTestRouter(service)

	response := serveJSON(router, http.MethodPost, "/api/v1/documents/scrape-translate", `{
		"url":"https://example.com/a",
		"targetLanguage":"ko"
	}`)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", response.Code, response.Body.String())
	}
	assertNoDocuments(t, service)
}

func TestScrapeTranslateRouteValidatesLanguageBeforeScrape(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := ingestionTestRouterWithTranslator(service, &fakeDocumentTranslator{})

	response := serveJSON(router, http.MethodPost, "/api/v1/documents/scrape-translate", `{
		"url":"https://example.com/a",
		"targetLanguage":"??"
	}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", response.Code, response.Body.String())
	}
	assertNoDocuments(t, service)
}

func TestScrapeTranslateRouteKeepsDocumentWhenEnqueueFails(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	translator := &fakeDocumentTranslator{err: translation.ErrTranslationAlreadyQueued}
	router := ingestionTestRouterWithTranslator(service, translator)

	response := serveJSON(router, http.MethodPost, "/api/v1/documents/scrape-translate", `{
		"url":"https://example.com/a",
		"targetLanguage":"ko"
	}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", response.Code, response.Body.String())
	}
	list, err := service.ListDocuments(context.Background(), app.DocumentFilters{})
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected scraped document to remain after enqueue failure, got %d", len(list.Items))
	}
}

func assertNoDocuments(t *testing.T, service *app.Service) {
	t.Helper()
	list, err := service.ListDocuments(context.Background(), app.DocumentFilters{})
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("expected no documents, got %d", len(list.Items))
	}
}
