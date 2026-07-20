package sqlite

import (
	"errors"
)

var (
	ErrStorageMaintenanceFileBackedRequired = errors.New("sqlite storage maintenance requires a file-backed database path")
	ErrStorageMaintenanceDestinationExists  = errors.New("sqlite storage compact destination already exists")
	ErrStorageMaintenanceOfflineRequired    = errors.New("sqlite database is busy; stop Plasma before compacting storage")
)

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

func (stats StorageStats) TotalBytes() int64 {
	return stats.DBBytes + stats.WALBytes + stats.SHMBytes
}
