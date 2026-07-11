package feeds

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

func TestSchedulerPollOnceEnqueuesEnabledFeeds(t *testing.T) {
	ctx := context.Background()
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	enableScheduler(t, ctx, service)
	enabled, err := service.CreateFeed(ctx, app.CreateFeedInput{URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("create enabled feed: %v", err)
	}
	disabledValue := false
	if _, err := service.CreateFeed(ctx, app.CreateFeedInput{
		URL: "https://example.com/disabled.xml", Enabled: &disabledValue,
	}); err != nil {
		t.Fatalf("create disabled feed: %v", err)
	}
	queue := &fakeQueue{}
	result, err := NewScheduler(service, queue).PollOnce(ctx)
	if err != nil {
		t.Fatalf("poll once: %v", err)
	}
	if result.Scanned != 2 || result.Enqueued != 1 || result.SkippedDisabled != 1 {
		t.Fatalf("unexpected result %#v", result)
	}
	requests := queue.enqueuedRequests()
	if len(requests) != 1 || requests[0].Kind != jobs.KindPollFeed {
		t.Fatalf("unexpected requests %#v", requests)
	}
	payload, err := DecodePollFeedPayload(requests[0].PayloadJSON)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.FeedID != enabled.ID {
		t.Fatalf("expected feed %q, got %q", enabled.ID, payload.FeedID)
	}
}

func TestSchedulerPollOnceSkipsDuplicateActiveJobs(t *testing.T) {
	ctx := context.Background()
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	enableScheduler(t, ctx, service)
	if _, err := service.CreateFeed(ctx, app.CreateFeedInput{URL: "https://example.com/feed.xml"}); err != nil {
		t.Fatalf("create feed: %v", err)
	}
	queue := &fakeQueue{enqueueErr: jobs.ErrJobConflict}
	result, err := NewScheduler(service, queue).PollOnce(ctx)
	if err != nil {
		t.Fatalf("poll once: %v", err)
	}
	if result.Enqueued != 0 || result.SkippedPending != 1 {
		t.Fatalf("unexpected duplicate result %#v", result)
	}
}

func TestSchedulerPollOnceReturnsQueueErrors(t *testing.T) {
	ctx := context.Background()
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	enableScheduler(t, ctx, service)
	if _, err := service.CreateFeed(ctx, app.CreateFeedInput{URL: "https://example.com/feed.xml"}); err != nil {
		t.Fatalf("create feed: %v", err)
	}
	expected := errors.New("queue unavailable")
	queue := &fakeQueue{enqueueErr: expected}
	_, err := NewScheduler(service, queue).PollOnce(ctx)
	if !errors.Is(err, expected) {
		t.Fatalf("expected queue error, got %v", err)
	}
}

func TestSchedulerRunStopsOnClose(t *testing.T) {
	ctx := context.Background()
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	enableScheduler(t, ctx, service)
	if _, err := service.CreateFeed(ctx, app.CreateFeedInput{URL: "https://example.com/feed.xml"}); err != nil {
		t.Fatalf("create feed: %v", err)
	}
	ticker := newManualTicker()
	tickerReady := make(chan struct{})
	queue := &fakeQueue{enqueueCh: make(chan jobs.EnqueueRequest, 1)}
	scheduler := NewScheduler(service, queue, withSchedulerTicker(func(time.Duration) schedulerTicker {
		select {
		case <-tickerReady:
		default:
			close(tickerReady)
		}
		return ticker
	}))
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx) }()
	<-tickerReady
	ticker.tick()
	select {
	case <-queue.enqueueCh:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not enqueue after tick")
	}
	if err := scheduler.Close(); err != nil {
		t.Fatalf("close scheduler: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop")
	}
	if !ticker.stopped() {
		t.Fatal("ticker was not stopped")
	}
}

func TestSchedulerPollOnceSkipsWhenDisabled(t *testing.T) {
	ctx := context.Background()
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	if _, err := service.CreateFeed(ctx, app.CreateFeedInput{URL: "https://example.com/feed.xml"}); err != nil {
		t.Fatalf("create feed: %v", err)
	}
	queue := &fakeQueue{}
	result, err := NewScheduler(service, queue).PollOnce(ctx)
	if err != nil {
		t.Fatalf("poll once: %v", err)
	}
	if !result.SchedulerDisabled || len(queue.enqueuedRequests()) != 0 {
		t.Fatalf("expected disabled scheduler without enqueue, got %#v", result)
	}
}

func enableScheduler(t *testing.T, ctx context.Context, service *app.Service) {
	t.Helper()
	enabled := true
	if _, err := service.UpdateSettings(ctx, app.UpdateSettingsInput{
		FeedSchedulerEnabled: &enabled,
	}); err != nil {
		t.Fatalf("enable scheduler: %v", err)
	}
}

type manualTicker struct {
	ch           chan time.Time
	stoppedValue bool
}

func newManualTicker() *manualTicker {
	return &manualTicker{ch: make(chan time.Time, 1)}
}

func (ticker *manualTicker) C() <-chan time.Time {
	return ticker.ch
}

func (ticker *manualTicker) Stop() {
	ticker.stoppedValue = true
}

func (ticker *manualTicker) tick() {
	ticker.ch <- time.Now()
}

func (ticker *manualTicker) stopped() bool {
	return ticker.stoppedValue
}
