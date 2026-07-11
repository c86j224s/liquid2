package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

type sqliteRepository struct {
	store  *sqlitestore.Store
	logger *slog.Logger
	now    func() int64
	mu     sync.RWMutex
	closed bool
}

type sqliteRepositoryConfig struct {
	logger *slog.Logger
	now    func() int64
}

type SQLiteRepositoryOption func(*sqliteRepositoryConfig)

type sqliteAbort struct {
	err error
}

func NewSQLiteRepository(store *sqlitestore.Store, options ...SQLiteRepositoryOption) Repository {
	if store == nil {
		panic("app.NewSQLiteRepository: store is nil")
	}
	config := sqliteRepositoryConfig{logger: slog.Default().With("component", "app"), now: unixMillis}
	for _, option := range options {
		option(&config)
	}
	return &sqliteRepository{store: store, logger: config.logger.With("repository", "sqlite"), now: config.now}
}

func WithSQLiteRepositoryLogger(logger *slog.Logger) SQLiteRepositoryOption {
	return func(config *sqliteRepositoryConfig) {
		if logger != nil {
			config.logger = logger.With("component", "app")
		}
	}
}

func WithSQLiteRepositoryClock(clock func() int64) SQLiteRepositoryOption {
	return func(config *sqliteRepositoryConfig) {
		if clock != nil {
			config.now = clock
		}
	}
}

func (repo *sqliteRepository) View(ctx context.Context, fn func(RepositoryReader) error) error {
	return repo.run(ctx, func(tx *sqliteTx) error {
		return fn(sqliteReader{tx: tx})
	})
}

func (repo *sqliteRepository) Update(ctx context.Context, fn func(RepositoryTx) error) error {
	return repo.run(ctx, func(tx *sqliteTx) error {
		return fn(tx)
	})
}

func (repo *sqliteRepository) Close() error {
	repo.mu.Lock()
	repo.closed = true
	repo.mu.Unlock()
	return nil
}

func (repo *sqliteRepository) run(ctx context.Context, fn func(*sqliteTx) error) error {
	repo.mu.RLock()
	if repo.closed {
		repo.mu.RUnlock()
		return errRepositoryClosed
	}
	defer repo.mu.RUnlock()

	err := repo.store.InTx(ctx, func(q *sqlitedb.Queries) (err error) {
		tx := sqliteTx{ctx: ctx, q: q, now: repo.now}
		defer func() {
			if recovered := recover(); recovered != nil {
				if abort, ok := recovered.(sqliteAbort); ok {
					err = abort.err
					return
				}
				err = fmt.Errorf("sqlite repository operation panic: %v", recovered)
			}
		}()
		return fn(&tx)
	})
	if err != nil && !expectedRepositoryError(err) {
		repo.logger.ErrorContext(ctx, "sqlite repository operation failed",
			slog.String("operation", "sqlite_repository"),
			slog.Any("error", err),
		)
	}
	return err
}
