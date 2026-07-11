package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const foreignKeysOffDirective = "-- liquid2:foreign_keys_off"

func (s *Store) Migrate(ctx context.Context) error {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		s.logger.ErrorContext(ctx, "read migrations failed",
			slog.String("operation", "sqlite_migrate"),
			slog.Any("error", err),
		)
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	current, err := s.currentVersion(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "read schema version failed",
			slog.String("operation", "sqlite_migrate"),
			slog.Any("error", err),
		)
		return err
	}

	for _, entry := range entries {
		version, err := migrationVersion(entry.Name())
		if err != nil {
			s.logger.ErrorContext(ctx, "parse migration version failed",
				slog.String("operation", "sqlite_migrate"),
				slog.String("migration", entry.Name()),
				slog.Any("error", err),
			)
			return err
		}
		if version <= current {
			continue
		}
		if err := s.applyMigration(ctx, version, entry.Name()); err != nil {
			return err
		}
		current = version
	}

	s.logger.DebugContext(ctx, "sqlite migrations complete",
		slog.String("operation", "sqlite_migrate"),
		slog.Int64("schema_version", current),
	)
	return nil
}

func (s *Store) currentVersion(ctx context.Context) (int64, error) {
	var version int64
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err == nil {
		return version, nil
	}
	if strings.Contains(err.Error(), "no such table") {
		return 0, nil
	}
	return 0, err
}

func (s *Store) applyMigration(ctx context.Context, version int64, name string) error {
	body, err := migrationFiles.ReadFile(path.Join("migrations", name))
	if err != nil {
		s.logger.ErrorContext(ctx, "read migration failed",
			slog.String("operation", "sqlite_migrate"),
			slog.String("migration", name),
			slog.Any("error", err),
		)
		return err
	}

	run := func() error {
		return s.inSQLTx(ctx, "sqlite_migrate", func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, string(body)); err != nil {
				s.logger.ErrorContext(ctx, "migration apply failed",
					slog.String("operation", "sqlite_migrate"),
					slog.String("migration", name),
					slog.Any("error", err),
				)
				return fmt.Errorf("apply migration %s: %w", name, err)
			}
			if migrationNeedsForeignKeysOff(body) {
				if err := assertNoForeignKeyViolations(ctx, tx); err != nil {
					return fmt.Errorf("validate migration %s foreign keys: %w", name, err)
				}
			}
			if _, err := tx.ExecContext(
				ctx,
				"INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, unixepoch() * 1000)",
				version,
				name,
			); err != nil {
				s.logger.ErrorContext(ctx, "record migration version failed",
					slog.String("operation", "sqlite_migrate"),
					slog.String("migration", name),
					slog.Any("error", err),
				)
				return err
			}
			return nil
		}, slog.String("migration", name))
	}
	var applyErr error
	if migrationNeedsForeignKeysOff(body) {
		applyErr = s.withForeignKeysDisabled(ctx, run)
	} else {
		applyErr = run()
	}
	if applyErr != nil {
		return applyErr
	}
	s.logger.DebugContext(ctx, "migration applied",
		slog.String("operation", "sqlite_migrate"),
		slog.String("migration", name),
		slog.Int64("schema_version", version),
	)
	return nil
}

func (s *Store) withForeignKeysDisabled(ctx context.Context, fn func() error) (err error) {
	var enabled int
	if err := s.db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil {
		return fmt.Errorf("read foreign_keys pragma: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("disable foreign_keys pragma: %w", err)
	}
	defer func() {
		if enabled == 0 {
			return
		}
		if _, restoreErr := s.db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); restoreErr != nil && err == nil {
			err = fmt.Errorf("restore foreign_keys pragma: %w", restoreErr)
		}
	}()
	return fn()
}

func assertNoForeignKeyViolations(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		var table string
		var rowID sql.NullInt64
		var parent string
		var fkID int
		if err := rows.Scan(&table, &rowID, &parent, &fkID); err != nil {
			return err
		}
		return fmt.Errorf("foreign key violation table=%s rowid=%v parent=%s fkid=%d", table, rowID, parent, fkID)
	}
	return rows.Err()
}

func migrationNeedsForeignKeysOff(body []byte) bool {
	return strings.Contains(string(body), foreignKeysOffDirective)
}

func migrationVersion(name string) (int64, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("invalid migration name %q", name)
	}
	return strconv.ParseInt(prefix, 10, 64)
}
