package translation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

type fakeProvider struct {
	result     Result
	err        error
	panicValue any
	requests   []Request
}

func (provider *fakeProvider) Translate(_ context.Context, request Request) (Result, error) {
	provider.requests = append(provider.requests, request)
	if provider.panicValue != nil {
		panic(provider.panicValue)
	}
	if provider.err != nil {
		return Result{}, provider.err
	}
	return provider.result, nil
}

func TestPipelineAppendsTranslationWithProviderResult(t *testing.T) {
	ctx := context.Background()
	service, source := newTranslationSource(t, app.ContentFormatMarkdown)
	provider := &fakeProvider{result: Result{Content: "Translated body"}}
	pipeline := NewPipeline(service, provider)
	job := translateJob(t, source.Document.ID, source.Contents[0].ID, "KO")

	if err := pipeline.Handle(ctx, job); err != nil {
		t.Fatalf("handle translation job: %v", err)
	}

	if len(provider.requests) != 1 {
		t.Fatalf("expected one provider request, got %d", len(provider.requests))
	}
	request := provider.requests[0]
	if request.Text != "Original body" || request.Format != app.ContentFormatMarkdown {
		t.Fatalf("unexpected provider request %#v", request)
	}
	detail, err := service.GetDocument(ctx, source.Document.ID)
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	if len(detail.Contents) != 2 {
		t.Fatalf("expected source plus translation, got %#v", detail.Contents)
	}
	translation := detail.Contents[1]
	if translation.Content != "Translated body" || translation.Format != app.ContentFormatMarkdown {
		t.Fatalf("unexpected translation %#v", translation)
	}
	if translation.Language == nil || *translation.Language != "ko" {
		t.Fatalf("unexpected translation language %#v", translation.Language)
	}
}

func TestPipelineReturnsSafeProviderFailure(t *testing.T) {
	ctx := context.Background()
	service, source := newTranslationSource(t, app.ContentFormatText)
	provider := &fakeProvider{err: errors.New("raw provider body with secret")}
	pipeline := NewPipeline(service, provider)
	job := translateJob(t, source.Document.ID, source.Contents[0].ID, "ko")

	err := pipeline.Handle(ctx, job)
	if !errors.Is(err, ErrProviderFailed) {
		t.Fatalf("expected provider failure, got %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("provider error leaked raw detail: %v", err)
	}
	detail, getErr := service.GetDocument(ctx, source.Document.ID)
	if getErr != nil {
		t.Fatalf("get document: %v", getErr)
	}
	if len(detail.Contents) != 1 {
		t.Fatalf("provider failure should not mutate contents: %#v", detail.Contents)
	}
}

func TestPipelineSkipsProviderForMissingSourceContent(t *testing.T) {
	ctx := context.Background()
	service, source := newTranslationSource(t, app.ContentFormatText)
	provider := &fakeProvider{result: Result{Content: "Translated body"}}
	pipeline := NewPipeline(service, provider)
	job := translateJob(t, source.Document.ID, "missing", "ko")

	err := pipeline.Handle(ctx, job)
	if !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if len(provider.requests) != 0 {
		t.Fatalf("missing source should not call provider: %#v", provider.requests)
	}
}

func TestPipelineSkipsProviderForInvalidTargetLanguage(t *testing.T) {
	ctx := context.Background()
	service, source := newTranslationSource(t, app.ContentFormatText)
	provider := &fakeProvider{result: Result{Content: "Translated body"}}
	pipeline := NewPipeline(service, provider)
	job := jobs.Job{
		ID:   "job_1",
		Kind: jobs.KindTranslateDocument,
		PayloadJSON: `{"documentId":"` + source.Document.ID +
			`","sourceContentId":"` + source.Contents[0].ID + `","targetLanguage":"??"}`,
	}

	err := pipeline.Handle(ctx, job)
	if !errors.Is(err, ErrInvalidJobPayload) {
		t.Fatalf("expected invalid payload, got %v", err)
	}
	if len(provider.requests) != 0 {
		t.Fatalf("invalid language should not call provider: %#v", provider.requests)
	}
}

func newTranslationSource(t *testing.T, format string) (*app.Service, app.DocumentDetail) {
	t.Helper()
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	source, err := service.CreateScrapedDocument(context.Background(), app.ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Article", Content: "Original body", Format: format,
	})
	if err != nil {
		t.Fatalf("create source document: %v", err)
	}
	return service, source
}

func translateJob(t *testing.T, documentID, sourceContentID, targetLanguage string) jobs.Job {
	t.Helper()
	payload, err := EncodeTranslateDocumentPayload(documentID, sourceContentID, targetLanguage)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	return jobs.Job{ID: "job_1", Kind: jobs.KindTranslateDocument, PayloadJSON: payload}
}
