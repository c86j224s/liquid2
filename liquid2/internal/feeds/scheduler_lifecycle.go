package feeds

import (
	"context"
	"log/slog"
	"time"
)

func (scheduler *Scheduler) Run(ctx context.Context) error {
	scheduler.logger.InfoContext(ctx, "feed scheduler starting",
		slog.String("operation", "feed_scheduler_start"),
	)
	defer scheduler.logger.InfoContext(ctx, "feed scheduler stopped",
		slog.String("operation", "feed_scheduler_stop"),
	)
	runCtx, cancel := context.WithCancel(ctx)
	scheduler.setRunCancel(cancel)
	defer func() {
		scheduler.setRunCancel(nil)
		cancel()
	}()
	for {
		interval := scheduler.nextInterval(ctx)
		ticker := scheduler.tickerFactory(interval)
		select {
		case <-ctx.Done():
			ticker.Stop()
			return ctx.Err()
		case <-scheduler.stop:
			ticker.Stop()
			return nil
		case <-ticker.C():
			ticker.Stop()
			if _, err := scheduler.PollOnce(runCtx); err != nil {
				if runCtx.Err() != nil {
					return scheduler.runErr(runCtx)
				}
				scheduler.logger.ErrorContext(ctx, "feed scheduler poll failed",
					slog.String("operation", "feed_scheduler_poll"),
					slog.Any("error", err),
				)
			}
		}
	}
}

func (scheduler *Scheduler) Close() error {
	scheduler.closeOnce.Do(func() {
		close(scheduler.stop)
		scheduler.cancelRun()
	})
	return nil
}

func (scheduler *Scheduler) setRunCancel(cancel context.CancelFunc) {
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	scheduler.runCancel = cancel
}

func (scheduler *Scheduler) cancelRun() {
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	if scheduler.runCancel != nil {
		scheduler.runCancel()
	}
}

func (scheduler *Scheduler) runErr(ctx context.Context) error {
	select {
	case <-scheduler.stop:
		return nil
	default:
		return ctx.Err()
	}
}

func withSchedulerTicker(factory func(time.Duration) schedulerTicker) SchedulerOption {
	return func(scheduler *Scheduler) {
		if factory != nil {
			scheduler.tickerFactory = factory
		}
	}
}
