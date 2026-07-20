package sqlite

import (
	"context"
	"errors"
	"fmt"
	"os"
)

func CompactStorageTo(ctx context.Context, dbPath string, outputPath string) (StorageCompactResult, error) {
	sourcePath, err := normalizeStorageMaintenancePath(dbPath)
	if err != nil {
		return StorageCompactResult{}, err
	}
	destinationPath, err := normalizeStorageCompactDestination(sourcePath, outputPath)
	if err != nil {
		return StorageCompactResult{}, err
	}
	if err := ensureExclusiveMaintenanceWindow(ctx, sourcePath); err != nil {
		return StorageCompactResult{}, err
	}
	original, err := ReadStorageStats(ctx, sourcePath)
	if err != nil {
		return StorageCompactResult{}, err
	}
	if err := vacuumInto(ctx, sourcePath, destinationPath); err != nil {
		_ = os.Remove(destinationPath)
		return StorageCompactResult{}, err
	}
	if err := os.Chmod(destinationPath, sourceFileMode(sourcePath)); err != nil {
		_ = os.Remove(destinationPath)
		return StorageCompactResult{}, fmt.Errorf("set compacted database permissions: %w", err)
	}
	integrity, err := VerifyStorage(ctx, destinationPath)
	if err != nil {
		_ = os.Remove(destinationPath)
		return StorageCompactResult{}, err
	}
	compacted, err := ReadStorageStats(ctx, destinationPath)
	if err != nil {
		_ = os.Remove(destinationPath)
		return StorageCompactResult{}, err
	}
	return StorageCompactResult{
		DBPath: sourcePath, OutputPath: destinationPath, Original: original, Compacted: compacted,
		SavedBytes: original.TotalBytes() - compacted.TotalBytes(), IntegrityCheck: integrity,
	}, nil
}

func CompactStorageDryRun(ctx context.Context, dbPath string) (StorageCompactResult, error) {
	sourcePath, err := normalizeStorageMaintenancePath(dbPath)
	if err != nil {
		return StorageCompactResult{}, err
	}
	tempPath := compactTempPath(sourcePath)
	result, err := CompactStorageTo(ctx, sourcePath, tempPath)
	if err != nil {
		return StorageCompactResult{}, err
	}
	if err := os.Remove(tempPath); err != nil {
		return StorageCompactResult{}, fmt.Errorf("remove dry-run compacted database: %w", err)
	}
	result.OutputPath = ""
	result.DryRun = true
	return result, nil
}

func VerifyStorage(ctx context.Context, dbPath string) (string, error) {
	path, err := normalizeStorageMaintenancePath(dbPath)
	if err != nil {
		return "", err
	}
	db, err := openMaintenanceDB(path, 1000)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var integrity string
	if err := db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return "", fmt.Errorf("run integrity_check: %w", err)
	}
	if integrity != "ok" {
		return integrity, fmt.Errorf("sqlite integrity_check failed: %s", integrity)
	}
	rows, err := db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return "", fmt.Errorf("run foreign_key_check: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return "", errors.New("sqlite foreign_key_check failed")
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("read foreign_key_check: %w", err)
	}
	return integrity, nil
}
