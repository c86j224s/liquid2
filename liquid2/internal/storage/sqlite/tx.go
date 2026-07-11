package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/c86j224s/liquid2/internal/logging"
	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func (s *Store) InTx(ctx context.Context, fn func(*sqlitedb.Queries) error) error {
	return s.inTx(ctx, "sqlite_transaction", func(_ *sql.Tx, q *sqlitedb.Queries) error {
		return fn(q)
	})
}

func (s *Store) inSQLTx(
	ctx context.Context,
	operation string,
	fn func(*sql.Tx) error,
	attrs ...slog.Attr,
) error {
	return s.inTx(ctx, operation, func(tx *sql.Tx, _ *sqlitedb.Queries) error {
		return fn(tx)
	}, attrs...)
}

func (s *Store) inTx(
	ctx context.Context,
	operation string,
	fn func(*sql.Tx, *sqlitedb.Queries) error,
	attrs ...slog.Attr,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "transaction begin failed", txAttrs(operation, err, attrs)...)
		return err
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.logger.LogAttrs(ctx, slog.LevelError, "transaction rollback failed", txAttrs(operation, err, attrs)...)
		}
	}()

	if err := fn(tx, s.queries.WithTx(tx)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, "transaction commit failed", txAttrs(operation, err, attrs)...)
		return err
	}
	committed = true
	s.logger.LogAttrs(ctx, logging.LevelTrace, "transaction committed", txAttrs(operation, nil, attrs)...)
	return nil
}

func txAttrs(operation string, err error, attrs []slog.Attr) []slog.Attr {
	items := make([]slog.Attr, 0, len(attrs)+2)
	items = append(items, slog.String("operation", operation))
	items = append(items, attrs...)
	if err != nil {
		items = append(items, slog.Any("error", err))
	}
	return items
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
