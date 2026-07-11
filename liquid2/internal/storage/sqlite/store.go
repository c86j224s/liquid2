package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
	_ "modernc.org/sqlite"
)

type Store struct {
	db      *sql.DB
	path    string
	logger  *slog.Logger
	queries *sqlitedb.Queries
}

const maxOpenConnections = 1

type Option func(*storeConfig)

type storeConfig struct {
	logger *slog.Logger
}

func Open(ctx context.Context, path string, options ...Option) (*Store, error) {
	config := storeConfig{
		logger: slog.Default().With("component", "sqlite"),
	}
	for _, option := range options {
		option(&config)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		config.logger.ErrorContext(ctx, "sqlite open failed",
			slog.String("operation", "sqlite_open"),
			slog.Any("error", err),
		)
		return nil, err
	}
	// Keep PRAGMA state and in-memory databases bound to one SQLite connection.
	db.SetMaxOpenConns(maxOpenConnections)

	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		config.logger.ErrorContext(ctx, "sqlite foreign keys pragma failed",
			slog.String("operation", "sqlite_open"),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	config.logger.DebugContext(ctx, "sqlite store opened",
		slog.String("operation", "sqlite_open"),
		slog.Int("max_open_connections", maxOpenConnections),
	)
	return newStore(path, db, config.logger), nil
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *storeConfig) {
		if logger != nil {
			config.logger = logger.With("component", "sqlite")
		}
	}
}

func newStore(path string, db *sql.DB, logger *slog.Logger) *Store {
	return &Store{
		db:      db,
		path:    path,
		logger:  logger,
		queries: sqlitedb.New(db),
	}
}

func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		s.logger.Error("sqlite close failed",
			slog.String("operation", "sqlite_close"),
			slog.Any("error", err),
		)
		return err
	}
	s.logger.Debug("sqlite store closed", slog.String("operation", "sqlite_close"))
	return nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Queries() *sqlitedb.Queries {
	return s.queries
}
