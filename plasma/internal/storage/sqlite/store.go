package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, dbPath string) (*Store, error) {
	path := strings.TrimSpace(dbPath)
	if err := ensureParentDir(path); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, err
	}

	store := &Store{db: db}
	if err := store.bootstrap(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Health(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) MigrationVersions(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT version FROM plasma_schema_migrations ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []string
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *Store) bootstrap(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS plasma_schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
)`); err != nil {
		return err
	}

	names, err := migrationNames()
	if err != nil {
		return err
	}
	for _, name := range names {
		applied, err := s.migrationApplied(ctx, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := s.applyMigration(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) migrationApplied(ctx context.Context, version string) (bool, error) {
	var existing string
	err := s.db.QueryRowContext(ctx,
		`SELECT version FROM plasma_schema_migrations WHERE version = ?`, version).Scan(&existing)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

func (s *Store) applyMigration(ctx context.Context, name string) error {
	sqlText, err := migrationFiles.ReadFile(filepath.ToSlash(filepath.Join("migrations", name)))
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, string(sqlText)); err != nil {
		return fmt.Errorf("apply %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO plasma_schema_migrations (version, applied_at)
		 VALUES (?, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))`, name); err != nil {
		return err
	}
	return tx.Commit()
}

func sqliteDSN(dbPath string) string {
	dsn := strings.TrimSpace(dbPath)
	if dsn == "" || dsn == ":memory:" {
		dsn = "file:plasma-memory?mode=memory&cache=shared"
	}
	if !strings.Contains(dsn, "_pragma=foreign_keys") {
		dsn = appendSQLiteParam(dsn, "_pragma=foreign_keys(1)")
	}
	if !strings.Contains(dsn, "_pragma=journal_mode") {
		dsn = appendSQLiteParam(dsn, "_pragma=journal_mode(WAL)")
	}
	if !strings.Contains(dsn, "_pragma=busy_timeout") {
		dsn = appendSQLiteParam(dsn, "_pragma=busy_timeout(5000)")
	}
	if !strings.Contains(dsn, "_txlock") {
		dsn = appendSQLiteParam(dsn, "_txlock=immediate")
	}
	return dsn
}

func appendSQLiteParam(dsn string, param string) string {
	if strings.Contains(dsn, "?") {
		return dsn + "&" + param
	}
	return dsn + "?" + param
}

func migrationNames() ([]string, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func ensureParentDir(dbPath string) error {
	if dbPath == "" || dbPath == ":memory:" || strings.HasPrefix(dbPath, "file:") {
		return nil
	}
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
