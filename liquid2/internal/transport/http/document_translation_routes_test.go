package httptransport

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/translation"
)

type fakeDocumentTranslator struct {
	job   app.Job
	err   error
	input translation.EnqueueDocumentInput
}

func (translator *fakeDocumentTranslator) TranslateDocument(_ context.Context, input translation.EnqueueDocumentInput) (app.Job, error) {
	translator.input = input
	return translator.job, translator.err
}

func TestTranslateDocumentRoute(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	translator := &fakeDocumentTranslator{job: app.Job{
		ID: "job_1", Kind: app.JobKindTranslateDocument, Status: app.JobStatusQueued,
	}}
	router := NewRouter(service, WithDocumentTranslator(translator))

	response := serveJSON(router, http.MethodPost, "/api/v1/documents/doc_1/translate", `{
		"sourceContentId":"content_1",
		"targetLanguage":"ko"
	}`)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d: %s", response.Code, response.Body.String())
	}
	if translator.input.DocumentID != "doc_1" || translator.input.SourceContentID != "content_1" {
		t.Fatalf("unexpected translator input %#v", translator.input)
	}
	if !strings.Contains(response.Body.String(), `"kind":"translate_document"`) {
		t.Fatalf("expected translation job response, got %s", response.Body.String())
	}
}

func TestTranslateDocumentRouteErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{name: "missing", err: app.ErrNotFound, status: http.StatusNotFound},
		{name: "duplicate completed", err: app.ErrConflict, status: http.StatusConflict},
		{name: "duplicate queued", err: translation.ErrTranslationAlreadyQueued, status: http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := app.NewService()
			t.Cleanup(func() { _ = service.Close() })
			router := NewRouter(service, WithDocumentTranslator(&fakeDocumentTranslator{err: tc.err}))

			response := serveJSON(router, http.MethodPost, "/api/v1/documents/doc_1/translate", `{
				"sourceContentId":"content_1",
				"targetLanguage":"ko"
			}`)
			if response.Code != tc.status {
				t.Fatalf("expected status %d, got %d: %s", tc.status, response.Code, response.Body.String())
			}
		})
	}
}

func TestTranslateDocumentRouteUnavailable(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service)

	response := serveJSON(router, http.MethodPost, "/api/v1/documents/doc_1/translate", `{
		"sourceContentId":"content_1",
		"targetLanguage":"ko"
	}`)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", response.Code, response.Body.String())
	}
}
