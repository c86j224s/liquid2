package app

import (
	"context"
	"log/slog"
	"time"
)

// Service coordinates document-library application behavior.
type Service struct {
	// logger records application-level events.
	logger *slog.Logger
	// repo stores and retrieves document-library state.
	repo Repository
}

// serviceConfig contains construction-time service options.
type serviceConfig struct {
	// logger is the base application logger.
	logger *slog.Logger
	// now returns the current Unix timestamp in milliseconds.
	now func() int64
	// seeds populate initial in-memory state for explicit local workflows.
	seeds []repositorySeed
	// repo overrides the default memory repository.
	repo Repository
}

// Option customizes Service construction.
type Option func(*serviceConfig)

func NewService(options ...Option) *Service {
	config := serviceConfig{
		logger: slog.Default().With("component", "app"),
		now:    unixMillis,
	}
	for _, option := range options {
		option(&config)
	}
	repo := config.repo
	if repo == nil {
		repo = newMemoryRepository(memoryRepositoryConfig{
			logger: config.logger, now: config.now, seeds: config.seeds,
		})
	}
	service := &Service{
		logger: config.logger,
		repo:   repo,
	}
	return service
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *serviceConfig) {
		if logger != nil {
			config.logger = logger.With("component", "app")
		}
	}
}

func WithClock(clock func() int64) Option {
	return func(config *serviceConfig) {
		if clock != nil {
			config.now = clock
		}
	}
}

func WithRepository(repo Repository) Option {
	return func(config *serviceConfig) {
		if repo != nil {
			config.repo = repo
		}
	}
}

func (s *Service) Health(ctx context.Context) Health {
	s.logger.DebugContext(ctx, "health checked", slog.String("operation", "health"))
	return Health{OK: true}
}

func (s *Service) Close() error {
	return s.repo.Close()
}

func withUpdate[T any](
	ctx context.Context,
	service *Service,
	fn func(RepositoryTx) (T, error),
) (T, error) {
	return withRepository(ctx, service.repo.Update, fn)
}

func withView[T any](
	ctx context.Context,
	service *Service,
	fn func(RepositoryReader) (T, error),
) (T, error) {
	return withRepository(ctx, service.repo.View, fn)
}

func withRepository[T any, Tx any](
	ctx context.Context,
	run func(context.Context, func(Tx) error) error,
	fn func(Tx) (T, error),
) (T, error) {
	var value T
	if err := run(ctx, func(tx Tx) error {
		var domainErr error
		value, domainErr = fn(tx)
		return domainErr
	}); err != nil {
		var zero T
		return zero, err
	}
	return value, nil
}

func unixMillis() int64 {
	return time.Now().UnixMilli()
}
