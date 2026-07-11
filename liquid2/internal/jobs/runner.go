package jobs

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"
)

const (
	defaultIdleDelay         = 100 * time.Millisecond
	defaultMaxAttempts       = 3
	defaultTransitionTimeout = 5 * time.Second
)

type Runner struct {
	queue             Queue
	logger            *slog.Logger
	handlers          map[string]Handler
	idleDelay         time.Duration
	transitionTimeout time.Duration
	maxAttempts       int64
	stop              chan struct{}
	closeOnce         sync.Once
	runCancel         context.CancelFunc
	mu                sync.RWMutex
}

type RunnerOption func(*Runner)

func NewRunner(queue Queue, options ...RunnerOption) *Runner {
	if queue == nil {
		panic("jobs.NewRunner: queue is nil")
	}
	runner := &Runner{
		queue:             queue,
		logger:            slog.Default().With("component", "jobs"),
		handlers:          map[string]Handler{},
		idleDelay:         defaultIdleDelay,
		transitionTimeout: defaultTransitionTimeout,
		maxAttempts:       defaultMaxAttempts,
		stop:              make(chan struct{}),
	}
	for _, option := range options {
		option(runner)
	}
	return runner
}

func WithLogger(logger *slog.Logger) RunnerOption {
	return func(runner *Runner) {
		if logger != nil {
			runner.logger = logger.With("component", "jobs")
		}
	}
}

func WithIdleDelay(delay time.Duration) RunnerOption {
	return func(runner *Runner) {
		if delay > 0 {
			runner.idleDelay = delay
		}
	}
}

func WithTransitionTimeout(timeout time.Duration) RunnerOption {
	return func(runner *Runner) {
		if timeout > 0 {
			runner.transitionTimeout = timeout
		}
	}
}

func WithMaxAttempts(attempts int64) RunnerOption {
	return func(runner *Runner) {
		if attempts > 0 {
			runner.maxAttempts = attempts
		}
	}
}

func WithHandler(kind string, handler Handler) RunnerOption {
	return func(runner *Runner) {
		runner.Register(kind, handler)
	}
}

func (runner *Runner) Register(kind string, handler Handler) {
	if kind == "" || handler == nil {
		return
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.handlers[kind] = handler
}

func (runner *Runner) RecoverRunning(ctx context.Context) error {
	runner.logger.DebugContext(ctx, "recovering running jobs", slog.String("operation", "job_recovery"))
	return runner.queue.RecoverRunning(ctx, "startup recovery")
}

func (runner *Runner) Run(ctx context.Context) error {
	runner.logger.InfoContext(ctx, "job runner starting", slog.String("operation", "job_runner_start"))
	defer runner.logger.InfoContext(ctx, "job runner stopped", slog.String("operation", "job_runner_stop"))
	runCtx, cancel := context.WithCancel(ctx)
	runner.setRunCancel(cancel)
	defer func() {
		runner.setRunCancel(nil)
		cancel()
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-runner.stop:
			return nil
		default:
		}
		ran, err := runner.RunOnce(runCtx)
		if err != nil {
			return err
		}
		if !ran && !runner.wait(runCtx) {
			select {
			case <-runner.stop:
				return nil
			default:
				return runCtx.Err()
			}
		}
	}
}

func (runner *Runner) Close() error {
	runner.closeOnce.Do(func() {
		close(runner.stop)
		runner.cancelRun()
	})
	return nil
}

func (runner *Runner) RunOnce(ctx context.Context) (bool, error) {
	handlers := runner.handlerSnapshot()
	if len(handlers) == 0 {
		return false, nil
	}
	job, ok, err := runner.queue.Claim(ctx, handlerKinds(handlers))
	if err != nil || !ok {
		return false, err
	}
	handler, ok := handlers[job.Kind]
	if !ok {
		_, failErr := runner.queue.Fail(ctx, job.ID, "job handler not registered")
		return true, failErr
	}
	return true, runner.runClaimed(ctx, job, handler)
}

func handlerKinds(handlers map[string]Handler) []string {
	kinds := make([]string, 0, len(handlers))
	for kind := range handlers {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

func (runner *Runner) handlerSnapshot() map[string]Handler {
	runner.mu.RLock()
	defer runner.mu.RUnlock()
	handlers := make(map[string]Handler, len(runner.handlers))
	for kind, handler := range runner.handlers {
		handlers[kind] = handler
	}
	return handlers
}
