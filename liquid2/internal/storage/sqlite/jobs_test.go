package sqlite

import (
	"errors"
	"testing"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func TestJobTransitions(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()

	_, err := q.CreateJob(ctx, sqlitedb.CreateJobParams{
		ID: "job_1", Kind: "poll_feed", Status: JobQueued,
		PayloadJson: "{}", CreatedAt: 1000, UpdatedAt: 1000,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	running, err := store.TransitionJob(ctx, "job_1", JobRunning, 2000, "")
	if err != nil {
		t.Fatalf("transition running: %v", err)
	}
	if running.Attempts != 1 || !running.StartedAt.Valid {
		t.Fatalf("unexpected running job state: %+v", running)
	}

	done, err := store.TransitionJob(ctx, "job_1", JobCompleted, 3000, "")
	if err != nil {
		t.Fatalf("transition completed: %v", err)
	}
	if done.Status != JobCompleted || !done.FinishedAt.Valid {
		t.Fatalf("unexpected completed job state: %+v", done)
	}
	if !done.StartedAt.Valid || done.StartedAt.Int64 != running.StartedAt.Int64 {
		t.Fatalf("expected completed job to preserve started_at, got %+v", done.StartedAt)
	}

	_, err = store.TransitionJob(ctx, "job_1", JobRunning, 4000, "")
	if !errors.Is(err, ErrInvalidJobTransition) {
		t.Fatalf("expected invalid transition error, got %v", err)
	}
}

func TestRecoverRunningJobs(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()

	_, err := q.CreateJob(ctx, sqlitedb.CreateJobParams{
		ID: "job_1", Kind: "poll_feed", Status: JobRunning,
		PayloadJson: "{}", Attempts: 1, CreatedAt: 1000, UpdatedAt: 1000,
		StartedAt: nullInt(1000),
	})
	if err != nil {
		t.Fatalf("create running job: %v", err)
	}
	if err := store.RecoverRunningJobs(ctx, 2000, "startup recovery"); err != nil {
		t.Fatalf("recover running jobs: %v", err)
	}

	job, err := q.GetJob(ctx, "job_1")
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if job.Status != JobFailed || !job.FinishedAt.Valid || !job.Error.Valid {
		t.Fatalf("unexpected recovered job: %+v", job)
	}
}
