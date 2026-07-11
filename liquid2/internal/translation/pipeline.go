package translation

import (
	"context"
	"log/slog"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

type Pipeline struct {
	stages []jobs.Stage
	logger *slog.Logger
}

type PipelineOption func(*Pipeline)

func NewPipeline(documents *app.Service, provider Provider, options ...PipelineOption) *Pipeline {
	if documents == nil {
		panic("translation.NewPipeline: documents is nil")
	}
	if provider == nil {
		panic("translation.NewPipeline: provider is nil")
	}
	pipeline := &Pipeline{
		stages: []jobs.Stage{
			NewLoadSourceStage(documents),
			NewTranslateStage(provider),
			NewPersistStage(documents),
		},
		logger: slog.Default().With("component", "translation"),
	}
	for _, option := range options {
		option(pipeline)
	}
	return pipeline
}

func WithLogger(logger *slog.Logger) PipelineOption {
	return func(pipeline *Pipeline) {
		if logger != nil {
			pipeline.logger = logger.With("component", "translation")
		}
	}
}

func (pipeline *Pipeline) Handle(ctx context.Context, job jobs.Job) error {
	payload, err := DecodeTranslateDocumentPayload(job.PayloadJSON)
	if err != nil {
		return safeStageError(err)
	}
	data := any(payload)
	for _, stage := range pipeline.stages {
		output, err := stage.Run(ctx, jobs.StageInput{Job: job, Data: data})
		if err != nil {
			pipeline.logger.WarnContext(ctx, "translation stage failed",
				slog.String("operation", "document_translation"),
				slog.String("stage", stage.Name()),
				slog.String("job_id", job.ID),
				slog.String("document_id", payload.DocumentID),
				slog.String("source_content_id", payload.SourceContentID),
				slog.String("target_language", payload.TargetLanguage),
				slog.String("error_kind", errorKind(err)),
			)
			return safeStageError(err)
		}
		data = output.Data
	}
	pipeline.logger.DebugContext(ctx, "translation completed",
		slog.String("operation", "document_translation"),
		slog.String("job_id", job.ID),
		slog.String("document_id", payload.DocumentID),
		slog.String("source_content_id", payload.SourceContentID),
		slog.String("target_language", payload.TargetLanguage),
	)
	return nil
}
