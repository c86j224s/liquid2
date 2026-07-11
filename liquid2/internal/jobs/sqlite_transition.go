package jobs

import (
	"context"
	"database/sql"
	"errors"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func (queue *SQLiteQueue) transition(ctx context.Context, id string, next string, message string) (Job, error) {
	var job Job
	err := queue.store.InTx(ctx, func(q *sqlitedb.Queries) error {
		current, err := q.GetJob(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrJobNotFound
		}
		if err != nil {
			return err
		}
		if !canTransition(current.Status, next) {
			return ErrInvalidTransition
		}
		row, err := q.UpdateJobState(ctx, transitionParams(current, next, queue.now(), message))
		if err != nil {
			return err
		}
		job = sqliteJob(row)
		return nil
	})
	return job, err
}

func canTransition(current string, next string) bool {
	switch current {
	case StatusQueued:
		return next == StatusRunning
	case StatusRunning:
		return next == StatusCompleted || next == StatusFailed || next == StatusQueued
	case StatusFailed:
		return next == StatusQueued
	default:
		return false
	}
}

func transitionParams(job sqlitedb.Job, next string, now int64, message string) sqlitedb.UpdateJobStateParams {
	params := sqlitedb.UpdateJobStateParams{
		ID: job.ID, Status: next, Error: job.Error, Attempts: job.Attempts,
		UpdatedAt: now, StartedAt: job.StartedAt, FinishedAt: job.FinishedAt,
	}
	switch next {
	case StatusQueued:
		params.Error = nullString(message)
		params.StartedAt = sql.NullInt64{}
		params.FinishedAt = sql.NullInt64{}
	case StatusCompleted:
		params.Error = sql.NullString{}
		params.FinishedAt = nullInt64(now)
	case StatusFailed:
		params.Error = nullString(message)
		params.FinishedAt = nullInt64(now)
	}
	return params
}
