package sqlite

import (
	"errors"
)

var (
	// ErrStorageMaintenanceFileBackedRequired는 in-memory/DSN DB에 파일 기반 compact를
	// 수행하려 할 때 반환된다.
	ErrStorageMaintenanceFileBackedRequired = errors.New("sqlite storage maintenance requires a file-backed database path")
	// ErrStorageMaintenanceDestinationExists는 compact output을 덮어쓰지 않기 위한 오류다.
	ErrStorageMaintenanceDestinationExists = errors.New("sqlite storage compact destination already exists")
	// ErrStorageMaintenanceOfflineRequired는 안전한 offline compact를 위해 실행 중 DB를
	// 멈춰야 함을 알리는 오류다.
	ErrStorageMaintenanceOfflineRequired = errors.New("sqlite database is busy; stop Plasma before compacting storage")
)

// StorageStats는 SQLite DB 본체와 WAL/SHM 파일의 현재 크기와 vacuum 관련 지표다.
type StorageStats struct {
	DBPath           string `json:"db_path"`
	DBBytes          int64  `json:"db_bytes"`
	WALPath          string `json:"wal_path"`
	WALBytes         int64  `json:"wal_bytes"`
	SHMPath          string `json:"shm_path"`
	SHMBytes         int64  `json:"shm_bytes"`
	PageSize         int64  `json:"page_size"`
	PageCount        int64  `json:"page_count"`
	FreelistCount    int64  `json:"freelist_count"`
	ReclaimableBytes int64  `json:"reclaimable_bytes"`
	JournalMode      string `json:"journal_mode"`
	AutoVacuum       int64  `json:"auto_vacuum"`
}

// StorageCompactResult는 compact 또는 replace maintenance 실행 결과와 전후 통계를
// 함께 담는다.
type StorageCompactResult struct {
	DBPath         string       `json:"db_path"`
	OutputPath     string       `json:"output_path,omitempty"`
	Replaced       bool         `json:"replaced"`
	DryRun         bool         `json:"dry_run"`
	BackupPaths    []string     `json:"backup_paths,omitempty"`
	Original       StorageStats `json:"original"`
	Compacted      StorageStats `json:"compacted"`
	SavedBytes     int64        `json:"saved_bytes"`
	IntegrityCheck string       `json:"integrity_check"`
}

// TotalBytes는 DB 본체와 WAL/SHM 파일 크기를 합산한다.
func (stats StorageStats) TotalBytes() int64 {
	return stats.DBBytes + stats.WALBytes + stats.SHMBytes
}
