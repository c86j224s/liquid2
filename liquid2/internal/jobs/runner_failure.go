package jobs

import (
	"context"
	"log/slog"
)

func (runner *Runner) runClaimed(ctx context.Context, job Job, handler Handler) error {
	runner.logger.DebugContext(ctx, "job started",
		slog.String("operation", "job_start"),
		slog.String("job_id", job.ID),
		slog.String("job_kind", job.Kind),
		slog.Int64("attempt", job.Attempts),
	)
	event := runWorker(ctx, job, handler)
	if !event.failed() {
		transitionCtx, cancel := runner.transitionContext()
		defer cancel()
		_, err := runner.queue.Complete(transitionCtx, job.ID)
		if err == nil {
			runner.logger.DebugContext(ctx, "job completed",
				slog.String("operation", "job_complete"),
				slog.String("job_id", job.ID),
				slog.String("job_kind", job.Kind),
			)
		}
		return err
	}
	return runner.recordFailure(ctx, job, event)
}

func (runner *Runner) recordFailure(ctx context.Context, job Job, event workerEvent) error {
	message := event.safeMessage()
	attrs := []slog.Attr{
		slog.String("operation", "job_fail"),
		slog.String("job_id", job.ID),
		slog.String("job_kind", job.Kind),
		slog.Int64("attempt", job.Attempts),
	}
	if event.err != nil {
		attrs = append(attrs, slog.Any("error", event.err))
	}
	if event.panicValue != nil {
		attrs = append(attrs,
			slog.Any("panic", event.panicValue),
			slog.String("stack", string(event.stack)),
		)
	}
	transitionCtx, cancel := runner.transitionContext()
	defer cancel()
	if job.Attempts < runner.maxAttempts {
		_, err := runner.queue.Requeue(transitionCtx, job.ID, message)
		runner.logger.LogAttrs(ctx, slog.LevelWarn, "job retry queued", attrs...)
		return err
	}
	_, err := runner.queue.Fail(transitionCtx, job.ID, message)
	runner.logger.LogAttrs(ctx, slog.LevelError, "job failed", attrs...)
	return err
}

func (runner *Runner) transitionContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), runner.transitionTimeout)
}
