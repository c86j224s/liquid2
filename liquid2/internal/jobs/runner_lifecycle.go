package jobs

import (
	"context"
	"time"
)

func (runner *Runner) setRunCancel(cancel context.CancelFunc) {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.runCancel = cancel
}

func (runner *Runner) cancelRun() {
	runner.mu.RLock()
	defer runner.mu.RUnlock()
	if runner.runCancel != nil {
		runner.runCancel()
	}
}

func (runner *Runner) wait(ctx context.Context) bool {
	timer := time.NewTimer(runner.idleDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-runner.stop:
		return false
	case <-timer.C:
		return true
	}
}
