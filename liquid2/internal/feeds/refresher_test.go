package feeds

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
)

type fakeQueue struct {
	mu         sync.Mutex
	listed     []jobs.Job
	request    jobs.EnqueueRequest
	requests   []jobs.EnqueueRequest
	enqueueErr error
	enqueueCh  chan jobs.EnqueueRequest
}

func (queue *fakeQueue) Enqueue(ctx context.Context, request jobs.EnqueueRequest) (jobs.Job, error) {
	queue.mu.Lock()
	queue.request = request
	queue.requests = append(queue.requests, request)
	err := queue.enqueueErr
	queue.mu.Unlock()
	if queue.enqueueCh != nil {
		select {
		case queue.enqueueCh <- request:
		case <-ctx.Done():
			return jobs.Job{}, ctx.Err()
		}
	}
	if err != nil {
		return jobs.Job{}, err
	}
	return jobs.Job{ID: "job_1", Kind: request.Kind, Status: jobs.StatusQueued, PayloadJSON: request.PayloadJSON}, nil
}

func (queue *fakeQueue) List(_ context.Context, _ jobs.Filters) ([]jobs.Job, error) {
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

func (queue *fakeQueue) enqueuedRequests() []jobs.EnqueueRequest {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	requests := make([]jobs.EnqueueRequest, len(queue.requests))
	copy(requests, queue.requests)
	return requests
}

func TestRefresherEnqueuesPollFeedJob(t *testing.T) {
	ctx := context.Background()
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	feed, err := service.CreateFeed(ctx, app.CreateFeedInput{URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	queue := &fakeQueue{}
	refresher := NewRefresher(service, queue)
	job, err := refresher.RefreshFeed(ctx, feed.ID)
	if err != nil {
		t.Fatalf("refresh feed: %v", err)
	}
	if job.ID != "job_1" || queue.request.Kind != jobs.KindPollFeed {
		t.Fatalf("unexpected refresh job %#v request %#v", job, queue.request)
	}
}

func TestRefresherRejectsDuplicatePendingJob(t *testing.T) {
	ctx := context.Background()
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	feed, err := service.CreateFeed(ctx, app.CreateFeedInput{URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	payload, err := EncodePollFeedPayload(feed.ID)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	refresher := NewRefresher(service, &fakeQueue{listed: []jobs.Job{{
		ID: "job_1", Kind: jobs.KindPollFeed, Status: jobs.StatusQueued, PayloadJSON: payload,
	}}})
	_, err = refresher.RefreshFeed(ctx, feed.ID)
	if !errors.Is(err, ErrRefreshAlreadyQueued) {
		t.Fatalf("expected duplicate refresh error, got %v", err)
	}
}

func TestRefresherValidatesFeedState(t *testing.T) {
	ctx := context.Background()
	disabled := false
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	feed, err := service.CreateFeed(ctx, app.CreateFeedInput{
		URL: "https://example.com/feed.xml", Enabled: &disabled,
	})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	refresher := NewRefresher(service, &fakeQueue{})
	if _, err := refresher.RefreshFeed(ctx, "missing"); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("expected missing feed error, got %v", err)
	}
	if _, err := refresher.RefreshFeed(ctx, feed.ID); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("expected disabled feed conflict, got %v", err)
	}
}
