package jobs

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

type SQLiteQueue struct {
	store  *sqlitestore.Store
	logger *slog.Logger
	now    func() int64
}

const maxClaimKinds = 4

type SQLiteQueueOption func(*SQLiteQueue)

func NewSQLiteQueue(store *sqlitestore.Store, options ...SQLiteQueueOption) *SQLiteQueue {
	if store == nil {
		panic("jobs.NewSQLiteQueue: store is nil")
	}
	queue := &SQLiteQueue{store: store, logger: slog.Default().With("component", "jobs"), now: unixMillis}
	for _, option := range options {
		option(queue)
	}
	return queue
}

func WithSQLiteLogger(logger *slog.Logger) SQLiteQueueOption {
	return func(queue *SQLiteQueue) {
		if logger != nil {
			queue.logger = logger.With("component", "jobs")
		}
	}
}

func WithSQLiteClock(clock func() int64) SQLiteQueueOption {
	return func(queue *SQLiteQueue) {
		if clock != nil {
			queue.now = clock
		}
	}
}

func (queue *SQLiteQueue) Enqueue(ctx context.Context, request EnqueueRequest) (Job, error) {
	if err := request.validate(); err != nil {
		return Job{}, err
	}
	var job Job
	err := queue.store.InTx(ctx, func(q *sqlitedb.Queries) error {
		id, err := queue.jobID(ctx, q, request.ID)
		if err != nil {
			return err
		}
		row, err := q.EnqueueJob(ctx, sqlitedb.EnqueueJobParams{
			ID: id, Kind: request.Kind, PayloadJson: request.PayloadJSON, Now: queue.now(),
		})
		if errors.Is(err, sql.ErrNoRows) {
			existing, getErr := q.GetJob(ctx, id)
			if getErr != nil {
				return getErr
			}
			if existing.Kind != request.Kind || existing.PayloadJson != request.PayloadJSON {
				return ErrJobConflict
			}
			row = existing
		} else if err != nil {
			return mapEnqueueError(err)
		}
		job = sqliteJob(row)
		return nil
	})
	if err == nil {
		queue.logger.DebugContext(ctx, "job enqueued", slog.String("operation", "job_enqueue"), slog.String("job_id", job.ID), slog.String("job_kind", job.Kind))
	}
	return job, err
}

func (queue *SQLiteQueue) Claim(ctx context.Context, kinds []string) (Job, bool, error) {
	if len(kinds) == 0 {
		return Job{}, false, nil
	}
	var job Job
	params, err := claimParams(queue.now(), kinds)
	if err != nil {
		return Job{}, false, err
	}
	err = queue.store.InTx(ctx, func(q *sqlitedb.Queries) error {
		row, err := q.ClaimQueuedJob(ctx, params)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		job = sqliteJob(row)
		return nil
	})
	if err != nil || job.ID == "" {
		return Job{}, false, err
	}
	queue.logger.DebugContext(ctx, "job claimed", slog.String("operation", "job_claim"), slog.String("job_id", job.ID), slog.String("job_kind", job.Kind), slog.Int64("attempt", job.Attempts))
	return job, true, nil
}

func claimParams(now int64, kinds []string) (sqlitedb.ClaimQueuedJobParams, error) {
	if len(kinds) > maxClaimKinds {
		return sqlitedb.ClaimQueuedJobParams{}, ErrTooManyClaimKinds
	}
	params := sqlitedb.ClaimQueuedJobParams{Now: now}
	if len(kinds) > 0 {
		params.Kind1 = nullString(kinds[0])
	}
	if len(kinds) > 1 {
		params.Kind2 = nullString(kinds[1])
	}
	if len(kinds) > 2 {
		params.Kind3 = nullString(kinds[2])
	}
	if len(kinds) > 3 {
		params.Kind4 = nullString(kinds[3])
	}
	return params, nil
}

func (queue *SQLiteQueue) Complete(ctx context.Context, id string) (Job, error) {
	return queue.transition(ctx, id, StatusCompleted, "")
}

func (queue *SQLiteQueue) Fail(ctx context.Context, id string, message string) (Job, error) {
	return queue.transition(ctx, id, StatusFailed, message)
}

func (queue *SQLiteQueue) Requeue(ctx context.Context, id string, message string) (Job, error) {
	return queue.transition(ctx, id, StatusQueued, message)
}

func (queue *SQLiteQueue) RecoverRunning(ctx context.Context, message string) error {
	now := queue.now()
	err := queue.store.InTx(ctx, func(q *sqlitedb.Queries) error {
		return q.RecoverRunningJobs(ctx, sqlitedb.RecoverRunningJobsParams{
			Status: StatusFailed, Error: nullString(message), UpdatedAt: now,
			FinishedAt: nullInt64(now),
		})
	})
	if err == nil {
		queue.logger.WarnContext(ctx, "running jobs recovered", slog.String("operation", "job_recovery"), slog.String("job_status", StatusFailed))
	}
	return err
}

func (queue *SQLiteQueue) Job(ctx context.Context, id string) (Job, bool, error) {
	row, err := queue.store.Queries().GetJob(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, false, nil
	}
	if err != nil {
		return Job{}, false, err
	}
	return sqliteJob(row), true, nil
}

func (queue *SQLiteQueue) List(ctx context.Context, filters Filters) ([]Job, error) {
	rows, err := queue.store.Queries().ListJobs(ctx, sqlitedb.ListJobsParams{
		Status: nullString(filterString(filters.Status)), Kind: nullString(filterString(filters.Kind)),
		Limit: jobLimit(filters.Limit),
	})
	if err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(rows))
	for _, row := range rows {
		jobs = append(jobs, sqliteJob(row))
	}
	return jobs, nil
}
