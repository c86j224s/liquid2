package translation

import (
	"context"
	"errors"
	"log/slog"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

type EnqueueDocumentInput struct {
	DocumentID      string
	SourceContentID string
	TargetLanguage  string
}

type Enqueuer struct {
	documents *app.Service
	queue     jobs.Queue
	logger    *slog.Logger
}

type EnqueuerOption func(*Enqueuer)

func NewEnqueuer(documents *app.Service, queue jobs.Queue, options ...EnqueuerOption) *Enqueuer {
	if documents == nil {
		panic("translation.NewEnqueuer: documents is nil")
	}
	enqueuer := &Enqueuer{
		documents: documents,
		queue:     queue,
		logger:    slog.Default().With("component", "translation"),
	}
	for _, option := range options {
		option(enqueuer)
	}
	return enqueuer
}

func WithEnqueuerLogger(logger *slog.Logger) EnqueuerOption {
	return func(enqueuer *Enqueuer) {
		if logger != nil {
			enqueuer.logger = logger.With("component", "translation")
		}
	}
}

func (enqueuer *Enqueuer) TranslateDocument(ctx context.Context, input EnqueueDocumentInput) (app.Job, error) {
	if enqueuer.queue == nil {
		return app.Job{}, ErrTranslationUnavailable
	}
	prepared, err := enqueuer.documents.PrepareTranslation(ctx, input.DocumentID, app.PrepareTranslationInput{
		SourceContentID: input.SourceContentID,
		TargetLanguage:  input.TargetLanguage,
	})
	if err != nil {
		return app.Job{}, err
	}
	payload, err := EncodeTranslateDocumentPayload(
		input.DocumentID,
		input.SourceContentID,
		prepared.TargetLanguage,
	)
	if err != nil {
		return app.Job{}, err
	}
	if err := enqueuer.ensureNoPending(ctx, payload); err != nil {
		return app.Job{}, err
	}
	job, err := enqueuer.queue.Enqueue(ctx, jobs.EnqueueRequest{
		Kind: jobs.KindTranslateDocument, PayloadJSON: payload,
	})
	if errors.Is(err, jobs.ErrJobConflict) {
		return app.Job{}, ErrTranslationAlreadyQueued
	}
	if err != nil {
		return app.Job{}, err
	}
	enqueuer.logger.DebugContext(ctx, "translation enqueued",
		slog.String("operation", "document_translation_enqueue"),
		slog.String("document_id", input.DocumentID),
		slog.String("source_content_id", input.SourceContentID),
		slog.String("target_language", prepared.TargetLanguage),
		slog.String("job_id", job.ID),
	)
	return appJob(job), nil
}

func (enqueuer *Enqueuer) ensureNoPending(ctx context.Context, payload string) error {
	for _, status := range []string{jobs.StatusQueued, jobs.StatusRunning} {
		list, err := enqueuer.queue.List(ctx, jobs.Filters{
			Kind: jobs.KindTranslateDocument, Status: status,
		})
		if err != nil {
			return err
		}
		for _, job := range list {
			if samePayload(job.PayloadJSON, payload) {
				return ErrTranslationAlreadyQueued
			}
		}
	}
	return nil
}

func samePayload(left string, right string) bool {
	leftPayload, leftErr := DecodeTranslateDocumentPayload(left)
	rightPayload, rightErr := DecodeTranslateDocumentPayload(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return leftPayload.DocumentID == rightPayload.DocumentID &&
		leftPayload.SourceContentID == rightPayload.SourceContentID &&
		leftPayload.TargetLanguage == rightPayload.TargetLanguage
}
