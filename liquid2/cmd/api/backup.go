package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
	httptransport "github.com/c86j224s/liquid2/internal/transport/http"
)

const backupCreateAttempts = 3

type sqliteBackupRunner struct {
	store     *sqlitestore.Store
	outputDir string
	logger    *slog.Logger
	clock     func() time.Time
	sequence  atomic.Uint64
}

func newBackupRunner(logger *slog.Logger, store *sqlitestore.Store) (httptransport.BackupRunner, error) {
	outputDir := strings.TrimSpace(getenv("LIQUID2_BACKUP_DIR", ""))
	if store == nil || outputDir == "" || isBackupMemoryDBPath(getenv("LIQUID2_DB_PATH", "")) {
		return nil, nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	if err := os.MkdirAll(outputDir, 0o700); err != nil {
		logger.Error("backup directory unavailable", slog.String("component", "api"), slog.Any("error", err))
		return nil, nil
	}
	logger.Info("backup API enabled", slog.String("component", "api"), slog.String("operation", "backup_api_start"))
	return &sqliteBackupRunner{
		store: store, outputDir: outputDir, logger: logger.With("component", "api"), clock: time.Now,
	}, nil
}

func (runner *sqliteBackupRunner) Backup(ctx context.Context) (httptransport.BackupArtifact, error) {
	if err := os.MkdirAll(runner.outputDir, 0o700); err != nil {
		runner.logger.ErrorContext(ctx, "backup directory unavailable",
			slog.String("operation", "backup_api"),
			slog.Any("error", err),
		)
		return httptransport.BackupArtifact{}, fmt.Errorf("%w: backup directory", httptransport.ErrBackupUnavailable)
	}
	var lastCollision error
	for attempt := 0; attempt < backupCreateAttempts; attempt++ {
		filename := runner.nextFilename()
		result, err := runner.store.Backup(ctx, filepath.Join(runner.outputDir, filename))
		if errors.Is(err, sqlitestore.ErrBackupDestinationExists) {
			lastCollision = err
			continue
		}
		if errors.Is(err, sqlitestore.ErrBackupInMemorySource) {
			return httptransport.BackupArtifact{}, httptransport.ErrBackupUnavailable
		}
		if err != nil {
			return httptransport.BackupArtifact{}, err
		}
		runner.logger.InfoContext(ctx, "backup API completed",
			slog.String("operation", "backup_api"),
			slog.String("backup_id", result.ID),
			slog.Int64("schema_version", result.SchemaVersion),
			slog.Int64("size_bytes", result.SizeBytes),
		)
		return backupArtifactFromResult(result), nil
	}
	return httptransport.BackupArtifact{}, fmt.Errorf("backup destination collision: %w", lastCollision)
}

func (runner *sqliteBackupRunner) nextFilename() string {
	seq := runner.sequence.Add(1)
	return fmt.Sprintf("backup_%s_%d.sqlite3", runner.clock().UTC().Format("20060102T150405000Z"), seq)
}

func backupArtifactFromResult(result sqlitestore.BackupResult) httptransport.BackupArtifact {
	return httptransport.BackupArtifact{
		ID: result.ID, CreatedAt: result.CreatedAt, SourceType: result.SourceType,
		SchemaVersion: result.SchemaVersion, SizeBytes: result.SizeBytes, SHA256: result.SHA256,
	}
}

func isBackupMemoryDBPath(path string) bool {
	normalized := strings.ToLower(strings.TrimSpace(path))
	return normalized == ":memory:" || strings.Contains(normalized, "mode=memory")
}
