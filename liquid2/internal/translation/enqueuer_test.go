package translation

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

type fakeQueue struct {
	mu       sync.Mutex
	listed   []jobs.Job
	request  jobs.EnqueueRequest
	requests []jobs.EnqueueRequest
	err      error
}

func (queue *fakeQueue) Enqueue(_ context.Context, request jobs.EnqueueRequest) (jobs.Job, error) {
	queue.mu.Lock()
	queue.request = request
	queue.requests = append(queue.requests, request)
	err := queue.err
	queue.mu.Unlock()
	if err != nil {
		return jobs.Job{}, err
	}
	return jobs.Job{ID: "job_1", Kind: request.Kind, Status: jobs.StatusQueued, PayloadJSON: request.PayloadJSON}, nil
}

func (queue *fakeQueue) List(context.Context, jobs.Filters) ([]jobs.Job, error) {
	return queue.listed, nil
}

func (queue *fakeQueue) Claim(context.Context, []string) (jobs.Job, bool, error) {
	return jobs.Job{}, false, nil
}
func (queue *fakeQueue) Complete(context.Context, string) (jobs.Job, error) { return jobs.Job{}, nil }
func (queue *fakeQueue) Fail(context.Context, string, string) (jobs.Job, error) {
	return jobs.Job{}, nil
}
func (queue *fakeQueue) Requeue(context.Context, string, string) (jobs.Job, error) {
	return jobs.Job{}, nil
}
func (queue *fakeQueue) RecoverRunning(context.Context, string) error { return nil }
func (queue *fakeQueue) Job(context.Context, string) (jobs.Job, bool, error) {
	return jobs.Job{}, false, nil
}

func TestEnqueuerEnqueuesTranslateDocumentJob(t *testing.T) {
	ctx := context.Background()
	service, source := newTranslationSource(t, app.ContentFormatText)
	queue := &fakeQueue{}
	enqueuer := NewEnqueuer(service, queue)

	job, err := enqueuer.TranslateDocument(ctx, EnqueueDocumentInput{
		DocumentID: source.Document.ID, SourceContentID: source.Contents[0].ID,
		TargetLanguage: "KO",
	})
	if err != nil {
		t.Fatalf("enqueue translation: %v", err)
	}
	if job.ID != "job_1" || queue.request.Kind != jobs.KindTranslateDocument {
		t.Fatalf("unexpected job %#v request %#v", job, queue.request)
	}
	payload, err := DecodeTranslateDocumentPayload(queue.request.PayloadJSON)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.TargetLanguage != "ko" || payload.DocumentID != source.Document.ID {
		t.Fatalf("unexpected payload %#v", payload)
	}
}

func TestEnqueuerRejectsDuplicatePendingJob(t *testing.T) {
	ctx := context.Background()
	service, source := newTranslationSource(t, app.ContentFormatText)
	payload, err := EncodeTranslateDocumentPayload(source.Document.ID, source.Contents[0].ID, "ko")
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	queue := &fakeQueue{listed: []jobs.Job{{
		ID: "job_1", Kind: jobs.KindTranslateDocument, Status: jobs.StatusQueued,
		PayloadJSON: payload,
	}}}
	enqueuer := NewEnqueuer(service, queue)

	_, err = enqueuer.TranslateDocument(ctx, EnqueueDocumentInput{
		DocumentID: source.Document.ID, SourceContentID: source.Contents[0].ID,
		TargetLanguage: "ko",
	})
	if !errors.Is(err, ErrTranslationAlreadyQueued) {
		t.Fatalf("expected queued duplicate error, got %v", err)
	}
	if len(queue.requests) != 0 {
		t.Fatalf("duplicate job should not enqueue: %#v", queue.requests)
	}
}

func TestEnqueuerValidatesSourceBeforeEnqueue(t *testing.T) {
	ctx := context.Background()
	service, source := newTranslationSource(t, app.ContentFormatText)
	queue := &fakeQueue{}
	enqueuer := NewEnqueuer(service, queue)

	_, err := enqueuer.TranslateDocument(ctx, EnqueueDocumentInput{
		DocumentID: source.Document.ID, SourceContentID: "missing", TargetLanguage: "ko",
	})
	if !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("expected source not found, got %v", err)
	}
	if len(queue.requests) != 0 {
		t.Fatalf("invalid source should not enqueue: %#v", queue.requests)
	}
}

func TestEnqueuerRequiresQueue(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	enqueuer := NewEnqueuer(service, nil)
	_, err := enqueuer.TranslateDocument(context.Background(), EnqueueDocumentInput{})
	if !errors.Is(err, ErrTranslationUnavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
}
