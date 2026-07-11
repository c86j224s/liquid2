package jobs

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunnerRetriesPanicAsParentDecision(t *testing.T) {
	queue := &fakeQueue{
		hasClaim: true,
		claimJob: Job{ID: "job_1", Kind: KindPollFeed, Status: StatusRunning,
			Attempts: 1},
	}
	runner := NewRunner(queue, WithMaxAttempts(2), WithHandler(KindPollFeed, func(context.Context, Job) error {
		panic("boom")
	}))

	ran, err := runner.RunOnce(context.Background())
	if err != nil || !ran {
		t.Fatalf("run once: ran=%v err=%v", ran, err)
	}
	if queue.requeuedID != "job_1" || queue.requeueMessage != "job worker panicked" {
		t.Fatalf("expected retry after panic, got id=%q message=%q", queue.requeuedID, queue.requeueMessage)
	}
	if queue.failedID != "" {
		t.Fatalf("did not expect failed job, got %q", queue.failedID)
	}
}

func TestRunnerFailsAfterMaxAttempts(t *testing.T) {
	queue := &fakeQueue{
		hasClaim: true,
		claimJob: Job{ID: "job_1", Kind: KindPollFeed, Status: StatusRunning,
			Attempts: 2},
	}
	runner := NewRunner(queue, WithMaxAttempts(2), WithHandler(KindPollFeed, func(context.Context, Job) error {
		return errors.New("provider failed")
	}))

	ran, err := runner.RunOnce(context.Background())
	if err != nil || !ran {
		t.Fatalf("run once: ran=%v err=%v", ran, err)
	}
	if queue.failedID != "job_1" || queue.failedMessage != "job failed" {
		t.Fatalf("expected failed job, got id=%q message=%q", queue.failedID, queue.failedMessage)
	}
	if queue.requeuedID != "" {
		t.Fatalf("did not expect retry, got %q", queue.requeuedID)
	}
}

func TestRunnerRequeuesCanceledJob(t *testing.T) {
	queue := &fakeQueue{
		hasClaim: true,
		claimJob: Job{ID: "job_1", Kind: KindPollFeed, Status: StatusRunning,
			Attempts: 1},
	}
	started := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	runner := NewRunner(queue, WithMaxAttempts(2), WithHandler(KindPollFeed, func(ctx context.Context, _ Job) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}))

	done := make(chan error, 1)
	go func() {
		_, err := runner.RunOnce(ctx)
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run once returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled job did not return")
	}
	if queue.requeuedID != "job_1" || queue.requeueMessage != "job canceled" {
		t.Fatalf("expected canceled job retry, got id=%q message=%q", queue.requeuedID, queue.requeueMessage)
	}
}

func TestRunnerRecoverRunning(t *testing.T) {
	queue := &fakeQueue{}
	runner := NewRunner(queue)

	if err := runner.RecoverRunning(context.Background()); err != nil {
		t.Fatalf("recover running: %v", err)
	}
	if queue.recoverMessage != "startup recovery" {
		t.Fatalf("unexpected recovery message %q", queue.recoverMessage)
	}
}

func TestRunnerCloseStopsLoop(t *testing.T) {
	queue := &fakeQueue{}
	runner := NewRunner(queue, WithIdleDelay(time.Millisecond), WithHandler(KindPollFeed, func(context.Context, Job) error {
		return nil
	}))
	done := make(chan error, 1)
	go func() { done <- runner.Run(context.Background()) }()

	if err := runner.Close(); err != nil {
		t.Fatalf("close runner: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runner did not stop")
	}
}

func TestRunnerCloseCancelsRunningJob(t *testing.T) {
	queue := &fakeQueue{
		hasClaim: true,
		claimJob: Job{ID: "job_1", Kind: KindPollFeed, Status: StatusRunning,
			Attempts: 1},
	}
	started := make(chan struct{})
	runner := NewRunner(queue, WithIdleDelay(time.Millisecond), WithHandler(KindPollFeed, func(ctx context.Context, _ Job) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}))
	done := make(chan error, 1)
	go func() { done <- runner.Run(context.Background()) }()
	<-started

	if err := runner.Close(); err != nil {
		t.Fatalf("close runner: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runner did not stop running job")
	}
	if queue.requeuedID != "job_1" || queue.requeueMessage != "job canceled" {
		t.Fatalf("expected canceled job retry, got id=%q message=%q", queue.requeuedID, queue.requeueMessage)
	}
}

func TestRunnerClosePersistsCanceledJobWithSQLite(t *testing.T) {
	ctx := context.Background()
	queue, closeQueue := newSQLiteQueue(t, ctx)
	defer closeQueue()
	if _, err := queue.Enqueue(ctx, EnqueueRequest{ID: "job_1", Kind: KindPollFeed, PayloadJSON: "{}"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	started := make(chan struct{})
	runner := NewRunner(queue,
		WithIdleDelay(time.Millisecond),
		WithTransitionTimeout(time.Second),
		WithHandler(KindPollFeed, func(ctx context.Context, _ Job) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		}),
	)
	done := make(chan error, 1)
	go func() { done <- runner.Run(context.Background()) }()
	<-started

	if err := runner.Close(); err != nil {
		t.Fatalf("close runner: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runner did not stop running sqlite job")
	}
	job, ok, err := queue.Job(context.Background(), "job_1")
	if err != nil || !ok {
		t.Fatalf("job: ok=%v err=%v", ok, err)
	}
	if job.Status != StatusQueued || job.Error == nil || *job.Error != "job canceled" {
		t.Fatalf("expected canceled job requeued, got %#v", job)
	}
}
