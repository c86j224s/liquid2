package translation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

func TestPipelineReturnsSafeProviderPanicFailure(t *testing.T) {
	ctx := context.Background()
	service, source := newTranslationSource(t, app.ContentFormatText)
	provider := &fakeProvider{panicValue: "raw provider body with secret"}
	pipeline := NewPipeline(service, provider)
	job := translateJob(t, source.Document.ID, source.Contents[0].ID, "ko")

	err := pipeline.Handle(ctx, job)
	if !errors.Is(err, ErrProviderFailed) {
		t.Fatalf("expected provider failure, got %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("provider panic leaked raw detail: %v", err)
	}
	detail, getErr := service.GetDocument(ctx, source.Document.ID)
	if getErr != nil {
		t.Fatalf("get document: %v", getErr)
	}
	if len(detail.Contents) != 1 {
		t.Fatalf("provider panic should not mutate contents: %#v", detail.Contents)
	}
}

func TestPipelineSkipsProviderForDuplicateTranslation(t *testing.T) {
	ctx := context.Background()
	service, source := newTranslationSource(t, app.ContentFormatText)
	if _, err := service.AppendTranslatedContent(ctx, source.Document.ID, app.AppendTranslationInput{
		SourceContentID: source.Contents[0].ID, TargetLanguage: "ko", Content: "Translated",
	}); err != nil {
		t.Fatalf("append translation: %v", err)
	}
	provider := &fakeProvider{result: Result{Content: "Second translation"}}
	pipeline := NewPipeline(service, provider)
	payload, err := EncodeTranslateDocumentPayload(source.Document.ID, source.Contents[0].ID, "ko")
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	job := jobs.Job{ID: "job_1", Kind: jobs.KindTranslateDocument, PayloadJSON: payload}

	err = pipeline.Handle(ctx, job)
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("expected duplicate conflict, got %v", err)
	}
	if len(provider.requests) != 0 {
		t.Fatalf("duplicate translation should not call provider: %#v", provider.requests)
	}
}
