package feeds

import (
	"context"
	"errors"
	"log/slog"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

type Pipeline struct {
	documents *app.Service
	stages    []jobs.Stage
	logger    *slog.Logger
}

type PipelineOption func(*Pipeline)

func NewPipeline(documents *app.Service, fetcher Fetcher, parser Parser, options ...PipelineOption) *Pipeline {
	if documents == nil {
		panic("feeds.NewPipeline: documents is nil")
	}
	if fetcher == nil {
		fetcher = NewHTTPFetcher()
	}
	if parser == nil {
		parser = NewGofeedParser()
	}
	normalizer := NewNormalizer()
	pipeline := &Pipeline{
		documents: documents,
		stages: []jobs.Stage{
			NewFetchStage(fetcher),
			NewParseStage(parser),
			NewNormalizeStage(normalizer),
			NewImportStage(documents),
		},
		logger: slog.Default().With("component", "feeds"),
	}
	for _, option := range options {
		option(pipeline)
	}
	return pipeline
}

func WithLogger(logger *slog.Logger) PipelineOption {
	return func(pipeline *Pipeline) {
		if logger != nil {
			pipeline.logger = logger.With("component", "feeds")
		}
	}
}

func (pipeline *Pipeline) Handle(ctx context.Context, job jobs.Job) error {
	feed, ok, err := pipeline.feedForJob(ctx, job)
	if err != nil || !ok {
		return err
	}
	data := any(feed)
	for _, stage := range pipeline.stages {
		output, err := stage.Run(ctx, jobs.StageInput{Job: job, Data: data})
		if err != nil {
			pipeline.logger.WarnContext(ctx, "feed stage failed",
				slog.String("operation", "feed_refresh"),
				slog.String("stage", stage.Name()),
				slog.String("job_id", job.ID),
				slog.String("feed_id", feed.ID),
				slog.String("error_kind", errorKind(err)),
			)
			return safeStageError(err)
		}
		data = output.Data
	}
	if result, ok := data.(app.FeedImportResult); ok {
		pipeline.logger.DebugContext(ctx, "feed refresh completed",
			slog.String("operation", "feed_refresh"),
			slog.String("job_id", job.ID),
			slog.String("feed_id", feed.ID),
			slog.Int("imported", result.Imported),
			slog.Int("skipped", result.Skipped),
		)
	}
	return nil
}

func (pipeline *Pipeline) feedForJob(ctx context.Context, job jobs.Job) (app.Feed, bool, error) {
	payload, err := DecodePollFeedPayload(job.PayloadJSON)
	if err != nil {
		return app.Feed{}, false, err
	}
	feed, err := pipeline.documents.GetFeed(ctx, payload.FeedID)
	if errors.Is(err, app.ErrNotFound) {
		pipeline.logger.WarnContext(ctx, "feed refresh skipped for missing feed",
			slog.String("operation", "feed_refresh"),
			slog.String("job_id", job.ID),
			slog.String("feed_id", payload.FeedID),
		)
		return app.Feed{}, false, nil
	}
	if err != nil {
		return app.Feed{}, false, err
	}
	if !feed.Enabled {
		pipeline.logger.WarnContext(ctx, "feed refresh skipped for disabled feed",
			slog.String("operation", "feed_refresh"),
			slog.String("job_id", job.ID),
			slog.String("feed_id", feed.ID),
		)
		return app.Feed{}, false, nil
	}
	return feed, true, nil
}
