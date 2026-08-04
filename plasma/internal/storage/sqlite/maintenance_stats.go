package sqlite

import (
	"context"
	"fmt"
)

// ReadStorageStats는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func ReadStorageStats(ctx context.Context, dbPath string) (StorageStats, error) {
	path, err := normalizeStorageMaintenancePath(dbPath)
	if err != nil {
		return StorageStats{}, err
	}
	dbBytes, err := existingFileSize(path)
	if err != nil {
		return StorageStats{}, err
	}
	walPath := path + "-wal"
	walBytes, err := optionalFileSize(walPath)
	if err != nil {
		return StorageStats{}, err
	}
	shmPath := path + "-shm"
	shmBytes, err := optionalFileSize(shmPath)
	if err != nil {
		return StorageStats{}, err
	}
	db, err := openMaintenanceDB(path, 1000)
	if err != nil {
		return StorageStats{}, err
	}
	defer db.Close()

	stats := StorageStats{
		DBPath: path, DBBytes: dbBytes, WALPath: walPath, WALBytes: walBytes,
		SHMPath: shmPath, SHMBytes: shmBytes,
	}
	if err := db.QueryRowContext(ctx, `PRAGMA page_size`).Scan(&stats.PageSize); err != nil {
		return StorageStats{}, fmt.Errorf("read page_size: %w", err)
	}
	if err := db.QueryRowContext(ctx, `PRAGMA page_count`).Scan(&stats.PageCount); err != nil {
		return StorageStats{}, fmt.Errorf("read page_count: %w", err)
	}
	if err := db.QueryRowContext(ctx, `PRAGMA freelist_count`).Scan(&stats.FreelistCount); err != nil {
		return StorageStats{}, fmt.Errorf("read freelist_count: %w", err)
	}
	if err := db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&stats.JournalMode); err != nil {
		return StorageStats{}, fmt.Errorf("read journal_mode: %w", err)
	}
	if err := db.QueryRowContext(ctx, `PRAGMA auto_vacuum`).Scan(&stats.AutoVacuum); err != nil {
		return StorageStats{}, fmt.Errorf("read auto_vacuum: %w", err)
	}
	stats.ReclaimableBytes = stats.PageSize * stats.FreelistCount
	return stats, nil
}
