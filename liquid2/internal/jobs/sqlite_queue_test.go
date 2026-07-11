package jobs

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
)

func TestSQLiteQueueLifecycle(t *testing.T) {
	ctx := context.Background()
	queue, closeQueue := newSQLiteQueue(t, ctx)
	defer closeQueue()

	enqueued, err := queue.Enqueue(ctx, EnqueueRequest{ID: "job_1", Kind: KindPollFeed, PayloadJSON: "{}"})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	again, err := queue.Enqueue(ctx, EnqueueRequest{ID: "job_1", Kind: KindPollFeed, PayloadJSON: "{}"})
	if err != nil {
		t.Fatalf("enqueue existing: %v", err)
	}
	if again.ID != enqueued.ID || again.Status != StatusQueued {
		t.Fatalf("unexpected idempotent enqueue result %#v", again)
	}

	claimed, ok, err := queue.Claim(ctx, []string{KindPollFeed})
	if err != nil || !ok {
		t.Fatalf("claim: ok=%v err=%v", ok, err)
	}
	if claimed.Status != StatusRunning || claimed.Attempts != 1 || claimed.StartedAt == nil {
		t.Fatalf("unexpected claimed job %#v", claimed)
	}
	completed, err := queue.Complete(ctx, claimed.ID)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if completed.Status != StatusCompleted || completed.FinishedAt == nil {
		t.Fatalf("unexpected completed job %#v", completed)
	}
}

func TestSQLiteQueueGeneratesJobID(t *testing.T) {
	ctx := context.Background()
	queue, closeQueue := newSQLiteQueue(t, ctx)
	defer closeQueue()

	enqueued, err := queue.Enqueue(ctx, EnqueueRequest{Kind: KindPollFeed, PayloadJSON: "{}"})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if !strings.HasPrefix(enqueued.ID, "job_") || !strings.HasSuffix(enqueued.ID, "_1") || enqueued.Status != StatusQueued {
		t.Fatalf("unexpected generated job %#v", enqueued)
	}
}

func TestSQLiteQueueDetectsConflictingEnqueue(t *testing.T) {
	ctx := context.Background()
	queue, closeQueue := newSQLiteQueue(t, ctx)
	defer closeQueue()

	if _, err := queue.Enqueue(ctx, EnqueueRequest{ID: "job_1", Kind: KindPollFeed, PayloadJSON: "{}"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_, err := queue.Enqueue(ctx, EnqueueRequest{ID: "job_1", Kind: KindPollFeed, PayloadJSON: `{"next":true}`})
	if !errors.Is(err, ErrJobConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestSQLiteQueueRejectsDuplicateActivePayload(t *testing.T) {
	ctx := context.Background()
	queue, closeQueue := newSQLiteQueue(t, ctx)
	defer closeQueue()

	if _, err := queue.Enqueue(ctx, EnqueueRequest{Kind: KindPollFeed, PayloadJSON: `{"feedId":"feed_1"}`}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_, err := queue.Enqueue(ctx, EnqueueRequest{Kind: KindPollFeed, PayloadJSON: `{"feedId":"feed_1"}`})
	if !errors.Is(err, ErrJobConflict) {
		t.Fatalf("expected queued duplicate conflict, got %v", err)
	}
	claimed, ok, err := queue.Claim(ctx, []string{KindPollFeed})
	if err != nil || !ok {
		t.Fatalf("claim: ok=%v err=%v", ok, err)
	}
	_, err = queue.Enqueue(ctx, EnqueueRequest{Kind: KindPollFeed, PayloadJSON: `{"feedId":"feed_1"}`})
	if !errors.Is(err, ErrJobConflict) {
		t.Fatalf("expected running duplicate conflict, got %v", err)
	}
	if _, err := queue.Complete(ctx, claimed.ID); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if _, err := queue.Enqueue(ctx, EnqueueRequest{Kind: KindPollFeed, PayloadJSON: `{"feedId":"feed_1"}`}); err != nil {
		t.Fatalf("enqueue after completion: %v", err)
	}
}

func TestSQLiteQueueClaimsOnlyRequestedKinds(t *testing.T) {
	ctx := context.Background()
	queue, closeQueue := newSQLiteQueue(t, ctx)
	defer closeQueue()

	if _, err := queue.Enqueue(ctx, EnqueueRequest{ID: "job_1", Kind: KindPollFeed, PayloadJSON: "{}"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, ok, err := queue.Claim(ctx, []string{KindScrapeURL}); err != nil || ok {
		t.Fatalf("expected no claim for unrequested kind, ok=%v err=%v", ok, err)
	}
	claimed, ok, err := queue.Claim(ctx, []string{KindPollFeed})
	if err != nil || !ok || claimed.ID != "job_1" {
		t.Fatalf("expected requested kind claim, job=%#v ok=%v err=%v", claimed, ok, err)
	}
}

func TestSQLiteQueueRejectsTooManyClaimKinds(t *testing.T) {
	ctx := context.Background()
	queue, closeQueue := newSQLiteQueue(t, ctx)
	defer closeQueue()

	_, _, err := queue.Claim(ctx, []string{
		KindPollFeed, KindScrapeURL, KindTranslateDocument, KindExtractUploadText, "future_kind",
	})
	if !errors.Is(err, ErrTooManyClaimKinds) {
		t.Fatalf("expected too many claim kinds error, got %v", err)
	}
}

func TestSQLiteQueueRecoverRunning(t *testing.T) {
	ctx := context.Background()
	queue, closeQueue := newSQLiteQueue(t, ctx)
	defer closeQueue()

	if _, err := queue.Enqueue(ctx, EnqueueRequest{ID: "job_1", Kind: KindPollFeed, PayloadJSON: "{}"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, _, err := queue.Claim(ctx, []string{KindPollFeed}); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := queue.RecoverRunning(ctx, "startup recovery"); err != nil {
		t.Fatalf("recover running: %v", err)
	}
	job, ok, err := queue.Job(ctx, "job_1")
	if err != nil || !ok {
		t.Fatalf("job: ok=%v err=%v", ok, err)
	}
	if job.Status != StatusFailed || job.Error == nil || *job.Error != "startup recovery" {
		t.Fatalf("unexpected recovered job %#v", job)
	}
}

func newSQLiteQueue(t *testing.T, ctx context.Context) (*SQLiteQueue, func()) {
	t.Helper()
	store, err := sqlitestore.Open(ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	queue := NewSQLiteQueue(store, WithSQLiteClock(func() int64 { return 1760000000000 }))
	return queue, func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	}
}
