package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func normalizeStorageMaintenancePath(dbPath string) (string, error) {
	path := strings.TrimSpace(dbPath)
	normalized := strings.ToLower(path)
	if path == "" || path == ":memory:" || strings.HasPrefix(normalized, "file:") {
		return "", ErrStorageMaintenanceFileBackedRequired
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%w: directory", ErrStorageMaintenanceFileBackedRequired)
	}
	return abs, nil
}

func normalizeStorageCompactDestination(sourcePath string, outputPath string) (string, error) {
	destination := strings.TrimSpace(outputPath)
	if destination == "" {
		return "", errors.New("compact output path is required")
	}
	if err := ensureParentDir(destination); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	if abs == sourcePath {
		return "", errors.New("compact output must differ from source database")
	}
	if info, err := os.Stat(abs); err == nil {
		if info.IsDir() {
			return "", fmt.Errorf("%w: directory", ErrStorageMaintenanceDestinationExists)
		}
		return "", ErrStorageMaintenanceDestinationExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return abs, nil
}

func openMaintenanceDB(path string, busyTimeoutMS int) (*sql.DB, error) {
	dsn := appendSQLiteParam(path, fmt.Sprintf("_pragma=busy_timeout(%d)", busyTimeoutMS))
	dsn = appendSQLiteParam(dsn, "_pragma=foreign_keys(1)")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func ensureExclusiveMaintenanceWindow(ctx context.Context, path string) error {
	db, err := openMaintenanceDB(path, 1)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `BEGIN EXCLUSIVE`); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageMaintenanceOfflineRequired, err)
	}
	_, _ = db.ExecContext(ctx, `ROLLBACK`)
	return nil
}

func vacuumInto(ctx context.Context, sourcePath string, destinationPath string) error {
	db, err := openMaintenanceDB(sourcePath, 1)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, "VACUUM INTO "+quoteSQLiteString(destinationPath)); err != nil {
		return fmt.Errorf("sqlite vacuum into: %w", err)
	}
	return nil
}

func quoteSQLiteString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
