package sqlite

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type renamedStorageFile struct {
	from string
	to   string
}

func CompactStorageReplace(ctx context.Context, dbPath string) (StorageCompactResult, error) {
	sourcePath, err := normalizeStorageMaintenancePath(dbPath)
	if err != nil {
		return StorageCompactResult{}, err
	}
	tempPath := compactTempPath(sourcePath)
	result, err := CompactStorageTo(ctx, sourcePath, tempPath)
	if err != nil {
		return StorageCompactResult{}, err
	}
	if err := ensureExclusiveMaintenanceWindow(ctx, sourcePath); err != nil {
		_ = os.Remove(tempPath)
		return StorageCompactResult{}, err
	}
	backups, err := renameStorageFilesToBackups(sourcePath, time.Now().UTC())
	if err != nil {
		_ = os.Remove(tempPath)
		return StorageCompactResult{}, err
	}
	if err := os.Rename(tempPath, sourcePath); err != nil {
		if restoreErr := restoreStorageBackups(backups); restoreErr != nil {
			return StorageCompactResult{}, fmt.Errorf("replace compacted database: %w; restore backups: %v", err, restoreErr)
		}
		return StorageCompactResult{}, fmt.Errorf("replace compacted database: %w", err)
	}
	replacedStats, err := ReadStorageStats(ctx, sourcePath)
	if err != nil {
		return StorageCompactResult{}, err
	}
	result.OutputPath = sourcePath
	result.Replaced = true
	result.BackupPaths = renamedBackupPaths(backups)
	result.Compacted = replacedStats
	result.SavedBytes = result.Original.TotalBytes() - result.Compacted.TotalBytes()
	return result, nil
}

func compactTempPath(sourcePath string) string {
	dir := filepath.Dir(sourcePath)
	base := filepath.Base(sourcePath)
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	return filepath.Join(dir, "."+base+".compact-"+stamp+".tmp")
}

func renameStorageFilesToBackups(dbPath string, timestamp time.Time) ([]renamedStorageFile, error) {
	stamp := timestamp.Format("20060102T150405Z")
	candidates := []string{dbPath, dbPath + "-wal", dbPath + "-shm"}
	var renamed []renamedStorageFile
	for _, path := range candidates {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			_ = restoreStorageBackups(renamed)
			return nil, err
		}
		if info.IsDir() {
			_ = restoreStorageBackups(renamed)
			return nil, fmt.Errorf("storage sidecar is a directory: %s", path)
		}
		backup := path + ".backup-" + stamp
		if _, err := os.Stat(backup); err == nil {
			_ = restoreStorageBackups(renamed)
			return nil, fmt.Errorf("%w: %s", ErrStorageMaintenanceDestinationExists, backup)
		} else if !os.IsNotExist(err) {
			_ = restoreStorageBackups(renamed)
			return nil, err
		}
		if err := os.Rename(path, backup); err != nil {
			_ = restoreStorageBackups(renamed)
			return nil, err
		}
		renamed = append(renamed, renamedStorageFile{from: path, to: backup})
	}
	return renamed, nil
}

func restoreStorageBackups(backups []renamedStorageFile) error {
	for index := len(backups) - 1; index >= 0; index-- {
		if err := os.Rename(backups[index].to, backups[index].from); err != nil {
			return err
		}
	}
	return nil
}

func renamedBackupPaths(backups []renamedStorageFile) []string {
	paths := make([]string, 0, len(backups))
	for _, backup := range backups {
		paths = append(paths, backup.to)
	}
	return paths
}

func existingFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if info.IsDir() {
		return 0, fmt.Errorf("storage path is a directory: %s", path)
	}
	return info.Size(), nil
}

func optionalFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if info.IsDir() {
		return 0, fmt.Errorf("storage sidecar is a directory: %s", path)
	}
	return info.Size(), nil
}

func sourceFileMode(path string) fs.FileMode {
	info, err := os.Stat(path)
	if err != nil {
		return 0o600
	}
	mode := info.Mode().Perm()
	if mode == 0 {
		return 0o600
	}
	return mode
}
