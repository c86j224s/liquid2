package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

const (
	JobQueued    = "queued"
	JobRunning   = "running"
	JobCompleted = "completed"
	JobFailed    = "failed"
)

var ErrInvalidJobTransition = errors.New("invalid job transition")

func (s *Store) TransitionJob(
	ctx context.Context,
	id string,
	next string,
	now int64,
	message string,
) (sqlitedb.Job, error) {
	var updated sqlitedb.Job
	err := s.InTx(ctx, func(q *sqlitedb.Queries) error {
		current, err := q.GetJob(ctx, id)
		if err != nil {
			s.logger.ErrorContext(ctx, "job lookup failed",
				slog.String("operation", "job_transition"),
				slog.String("job_id", id),
				slog.String("next_status", next),
				slog.Any("error", err),
			)
			return err
		}
		if !canTransition(current.Status, next) {
			s.logger.WarnContext(ctx, "invalid job transition",
				slog.String("operation", "job_transition"),
				slog.String("job_id", id),
				slog.String("job_kind", current.Kind),
				slog.String("job_status", current.Status),
				slog.String("next_status", next),
			)
			return fmt.Errorf("%w: %s -> %s", ErrInvalidJobTransition, current.Status, next)
		}

		params := transitionParams(current, next, now, message)
		updated, err = q.UpdateJobState(ctx, params)
		if err != nil {
			s.logger.ErrorContext(ctx, "job transition update failed",
				slog.String("operation", "job_transition"),
				slog.String("job_id", id),
				slog.String("job_kind", current.Kind),
				slog.String("job_status", current.Status),
				slog.String("next_status", next),
				slog.Any("error", err),
			)
		}
		return err
	})
	if err == nil {
		s.logger.DebugContext(ctx, "job transitioned",
			slog.String("operation", "job_transition"),
			slog.String("job_id", updated.ID),
			slog.String("job_kind", updated.Kind),
			slog.String("job_status", updated.Status),
			slog.Int64("attempt", updated.Attempts),
		)
	}
	return updated, err
}

func (s *Store) RecoverRunningJobs(ctx context.Context, now int64, message string) error {
	err := s.queries.RecoverRunningJobs(ctx, sqlitedb.RecoverRunningJobsParams{
		Status:     JobFailed,
		Error:      nullString(message),
		UpdatedAt:  now,
		FinishedAt: sql.NullInt64{Int64: now, Valid: true},
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "running job recovery failed",
			slog.String("operation", "job_recovery"),
			slog.Any("error", err),
		)
		return err
	}
	s.logger.DebugContext(ctx, "running job recovery applied",
		slog.String("operation", "job_recovery"),
		slog.String("job_status", JobFailed),
	)
	return nil
}

func canTransition(current string, next string) bool {
	switch current {
	case JobQueued:
		return next == JobRunning
	case JobRunning:
		return next == JobCompleted || next == JobFailed || next == JobQueued
	case JobFailed:
		return next == JobQueued
	default:
		return false
	}
}

func transitionParams(job sqlitedb.Job, next string, now int64, message string) sqlitedb.UpdateJobStateParams {
	params := sqlitedb.UpdateJobStateParams{
		ID:         job.ID,
		Status:     next,
		Error:      job.Error,
		Attempts:   job.Attempts,
		UpdatedAt:  now,
		StartedAt:  job.StartedAt,
		FinishedAt: job.FinishedAt,
	}

	if next == JobRunning {
		params.Attempts++
		params.StartedAt = sql.NullInt64{Int64: now, Valid: true}
	}
	if next == JobCompleted || next == JobFailed {
		params.FinishedAt = sql.NullInt64{Int64: now, Valid: true}
	}
	if message != "" {
		params.Error = sql.NullString{String: message, Valid: true}
	}
	return params
}
