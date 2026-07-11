package sqlite

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const BackupSourceSQLite = "sqlite"

var (
	ErrBackupDestinationExists = errors.New("backup destination already exists")
	ErrBackupInvalidArtifactID = errors.New("backup artifact id is invalid")
	ErrBackupInMemorySource    = errors.New("sqlite backup requires a file-backed source")
)

type BackupResult struct {
	ID            string
	Filename      string
	CreatedAt     int64
	SourceType    string
	SchemaVersion int64
	SizeBytes     int64
	SHA256        string
}

func (s *Store) Backup(ctx context.Context, destination string) (BackupResult, error) {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return BackupResult{}, fmt.Errorf("backup destination is required")
	}
	if isInMemoryPath(s.path) {
		return BackupResult{}, ErrBackupInMemorySource
	}
	if err := ensureBackupDestination(destination); err != nil {
		return BackupResult{}, err
	}
	if _, err := backupArtifactID(destination); err != nil {
		return BackupResult{}, err
	}
	schemaVersion, err := s.queries.SchemaVersion(ctx)
	if err != nil {
		return BackupResult{}, fmt.Errorf("read schema version: %w", err)
	}
	if _, err = s.db.ExecContext(ctx, "VACUUM INTO "+sqliteQuote(destination)); err != nil {
		return BackupResult{}, fmt.Errorf("sqlite backup: %w", err)
	}
	if err = os.Chmod(destination, 0o600); err != nil {
		return BackupResult{}, fmt.Errorf("secure backup permissions: %w", err)
	}
	result, err := backupResult(destination, schemaVersion)
	if err != nil {
		return BackupResult{}, err
	}
	s.logger.LogAttrs(ctx, slog.LevelInfo, "sqlite backup created",
		slog.String("operation", "sqlite_backup"),
		slog.String("backup_id", result.ID),
		slog.Int64("schema_version", result.SchemaVersion),
		slog.Int64("size_bytes", result.SizeBytes),
	)
	return result, nil
}

func ensureBackupDestination(destination string) error {
	if info, err := os.Stat(destination); err == nil {
		if info.IsDir() {
			return fmt.Errorf("%w: directory", ErrBackupDestinationExists)
		}
		return ErrBackupDestinationExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func backupResult(path string, schemaVersion int64) (BackupResult, error) {
	size, checksum, err := fileSizeAndSHA256(path)
	if err != nil {
		return BackupResult{}, err
	}
	id, err := backupArtifactID(path)
	if err != nil {
		return BackupResult{}, err
	}
	filename := filepath.Base(path)
	return BackupResult{
		ID: id, Filename: filename, CreatedAt: time.Now().UnixMilli(), SourceType: BackupSourceSQLite,
		SchemaVersion: schemaVersion, SizeBytes: size, SHA256: checksum,
	}, nil
}

func backupArtifactID(path string) (string, error) {
	filename := filepath.Base(path)
	id := strings.TrimSuffix(filename, filepath.Ext(filename))
	if strings.TrimSpace(id) == "" || strings.HasPrefix(id, ".") {
		return "", ErrBackupInvalidArtifactID
	}
	return id, nil
}

func fileSizeAndSHA256(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return 0, "", err
	}
	return size, fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func sqliteQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func isInMemoryPath(path string) bool {
	normalized := strings.ToLower(strings.TrimSpace(path))
	return normalized == "" || normalized == ":memory:" || strings.Contains(normalized, "mode=memory")
}
