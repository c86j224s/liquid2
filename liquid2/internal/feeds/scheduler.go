package feeds

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

const DefaultSchedulerConfigCheckInterval = time.Minute

type FeedSource interface {
	ListFeeds(context.Context) ([]app.Feed, error)
	GetSettings(context.Context) (app.AppSettings, error)
	SetFeedNextPollAt(context.Context, *int64) error
}

type Scheduler struct {
	source        FeedSource
	queue         jobs.Queue
	logger        *slog.Logger
	tickerFactory func(time.Duration) schedulerTicker
	stop          chan struct{}
	closeOnce     sync.Once
	runCancel     context.CancelFunc
	mu            sync.RWMutex
}

type SchedulerOption func(*Scheduler)

type SchedulerResult struct {
	SchedulerDisabled bool
	Scanned           int
	Enqueued          int
	SkippedDisabled   int
	SkippedPending    int
}

type schedulerTicker interface {
	C() <-chan time.Time
	Stop()
}

func NewScheduler(source FeedSource, queue jobs.Queue, options ...SchedulerOption) *Scheduler {
	if source == nil {
		panic("feeds.NewScheduler: source is nil")
	}
	if queue == nil {
		panic("feeds.NewScheduler: queue is nil")
	}
	scheduler := &Scheduler{
		source:        source,
		queue:         queue,
		logger:        slog.Default().With("component", "feeds"),
		tickerFactory: newSchedulerTicker,
		stop:          make(chan struct{}),
	}
	for _, option := range options {
		option(scheduler)
	}
	return scheduler
}

func WithSchedulerLogger(logger *slog.Logger) SchedulerOption {
	return func(scheduler *Scheduler) {
		if logger != nil {
			scheduler.logger = logger.With("component", "feeds")
		}
	}
}

func (scheduler *Scheduler) PollOnce(ctx context.Context) (SchedulerResult, error) {
	result := SchedulerResult{}
	settings, err := scheduler.source.GetSettings(ctx)
	if err != nil {
		return result, err
	}
	if !settings.FeedSchedulerEnabled {
		result.SchedulerDisabled = true
		scheduler.logger.DebugContext(ctx, "feed scheduler skipped",
			slog.String("operation", "feed_scheduler_poll"),
			slog.Bool("enabled", false),
		)
		return result, nil
	}
	feeds, err := scheduler.source.ListFeeds(ctx)
	if err != nil {
		return result, err
	}
	result.Scanned = len(feeds)
	for _, feed := range feeds {
		if !feed.Enabled {
			result.SkippedDisabled++
			continue
		}
		payload, err := EncodePollFeedPayload(feed.ID)
		if err != nil {
			return result, err
		}
		if _, err := scheduler.queue.Enqueue(ctx, jobs.EnqueueRequest{
			Kind: jobs.KindPollFeed, PayloadJSON: payload,
		}); errors.Is(err, jobs.ErrJobConflict) {
			result.SkippedPending++
			continue
		} else if err != nil {
			return result, err
		}
		result.Enqueued++
	}
	scheduler.logger.DebugContext(ctx, "feed scheduler poll completed",
		slog.String("operation", "feed_scheduler_poll"),
		slog.Int("scanned", result.Scanned),
		slog.Int("enqueued", result.Enqueued),
		slog.Int("skipped_disabled", result.SkippedDisabled),
		slog.Int("skipped_pending", result.SkippedPending),
	)
	return result, nil
}

func (scheduler *Scheduler) nextInterval(ctx context.Context) time.Duration {
	settings, err := scheduler.source.GetSettings(ctx)
	if err != nil {
		scheduler.logger.ErrorContext(ctx, "feed scheduler settings read failed",
			slog.String("operation", "feed_scheduler_settings"),
			slog.Any("error", err),
		)
		return DefaultSchedulerConfigCheckInterval
	}
	if !settings.FeedSchedulerEnabled {
		if settings.FeedNextPollAt != nil {
			scheduler.setNextPollAt(ctx, nil)
		}
		return DefaultSchedulerConfigCheckInterval
	}
	interval := time.Duration(settings.FeedPollIntervalSeconds) * time.Second
	nextAt := time.Now().Add(interval).UnixMilli()
	scheduler.setNextPollAt(ctx, &nextAt)
	return interval
}

func (scheduler *Scheduler) setNextPollAt(ctx context.Context, nextAt *int64) {
	if err := scheduler.source.SetFeedNextPollAt(ctx, nextAt); err != nil {
		scheduler.logger.WarnContext(ctx, "feed scheduler next poll update failed",
			slog.String("operation", "feed_scheduler_next_poll"),
			slog.Any("error", err),
		)
	}
}
