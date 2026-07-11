package feeds

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

type Refresher struct {
	documents *app.Service
	queue     jobs.Queue
	logger    *slog.Logger
}

type RefresherOption func(*Refresher)

func NewRefresher(documents *app.Service, queue jobs.Queue, options ...RefresherOption) *Refresher {
	if documents == nil {
		panic("feeds.NewRefresher: documents is nil")
	}
	refresher := &Refresher{
		documents: documents,
		queue:     queue,
		logger:    slog.Default().With("component", "feeds"),
	}
	for _, option := range options {
		option(refresher)
	}
	return refresher
}

func WithRefresherLogger(logger *slog.Logger) RefresherOption {
	return func(refresher *Refresher) {
		if logger != nil {
			refresher.logger = logger.With("component", "feeds")
		}
	}
}

func (refresher *Refresher) RefreshFeed(ctx context.Context, feedID string) (app.Job, error) {
	if refresher.queue == nil {
		return app.Job{}, ErrRefreshUnavailable
	}
	feed, err := refresher.documents.GetFeed(ctx, feedID)
	if err != nil {
		return app.Job{}, err
	}
	if !feed.Enabled {
		return app.Job{}, fmt.Errorf("%w: feed is disabled", app.ErrConflict)
	}
	payload, err := EncodePollFeedPayload(feed.ID)
	if err != nil {
		return app.Job{}, err
	}
	if err := refresher.ensureNoPending(ctx, payload); err != nil {
		return app.Job{}, err
	}
	job, err := refresher.queue.Enqueue(ctx, jobs.EnqueueRequest{
		Kind: jobs.KindPollFeed, PayloadJSON: payload,
	})
	if errors.Is(err, jobs.ErrJobConflict) {
		return app.Job{}, ErrRefreshAlreadyQueued
	}
	if err != nil {
		return app.Job{}, err
	}
	refresher.logger.DebugContext(ctx, "feed refresh enqueued",
		slog.String("operation", "feed_refresh_enqueue"),
		slog.String("feed_id", feed.ID),
		slog.String("job_id", job.ID),
	)
	return appJob(job), nil
}

func (refresher *Refresher) ensureNoPending(ctx context.Context, payload string) error {
	for _, status := range []string{jobs.StatusQueued, jobs.StatusRunning} {
		list, err := refresher.queue.List(ctx, jobs.Filters{
			Kind: jobs.KindPollFeed, Status: status,
		})
		if err != nil {
			return err
		}
		for _, job := range list {
			if samePayload(job.PayloadJSON, payload) {
				return ErrRefreshAlreadyQueued
			}
		}
	}
	return nil
}

func samePayload(left string, right string) bool {
	leftPayload, leftErr := DecodePollFeedPayload(left)
	rightPayload, rightErr := DecodePollFeedPayload(right)
	return leftErr == nil && rightErr == nil && leftPayload.FeedID == rightPayload.FeedID
}

func appJob(job jobs.Job) app.Job {
	return app.Job{
		ID: job.ID, Kind: job.Kind, Status: job.Status, PayloadJSON: job.PayloadJSON,
		Error: cloneString(job.Error), Attempts: job.Attempts, CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt, StartedAt: cloneInt64(job.StartedAt),
		FinishedAt: cloneInt64(job.FinishedAt),
	}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
