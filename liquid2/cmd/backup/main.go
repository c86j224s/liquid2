package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	liquidconfig "github.com/c86j224s/liquid2/internal/config"
	"github.com/c86j224s/liquid2/internal/logging"
	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
)

type backupConfig struct {
	dbPath    string
	outputDir string
	filename  string
	logLevel  string
	logFormat string
}

type backupResponse struct {
	Backup backupArtifact `json:"backup"`
}

type backupArtifact struct {
	ID            string `json:"id"`
	CreatedAt     int64  `json:"createdAt"`
	SourceType    string `json:"sourceType"`
	SchemaVersion int64  `json:"schemaVersion"`
	SizeBytes     int64  `json:"sizeBytes"`
	SHA256        string `json:"sha256"`
	Filename      string `json:"fileName"`
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	config, err := parseFlags(args, stderr)
	if err != nil {
		return 2
	}
	logger, err := logging.New(stderr, logging.Config{Level: config.logLevel, Format: config.logFormat})
	if err != nil {
		fmt.Fprintf(stderr, "configure logger: %v\n", err)
		return 2
	}
	if err := validateConfig(config); err != nil {
		logger.Error("backup configuration invalid", slog.Any("error", err))
		return 2
	}
	result, err := createBackup(ctx, logger, config)
	if err != nil {
		logger.Error("backup failed", slog.Any("error", err))
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(backupResponse{Backup: backupArtifactFromResult(result)}); err != nil {
		logger.Error("write backup response failed", slog.Any("error", err))
		return 1
	}
	return 0
}

func parseFlags(args []string, stderr io.Writer) (backupConfig, error) {
	config := backupConfig{
		dbPath:    os.Getenv("LIQUID2_DB_PATH"),
		outputDir: os.Getenv("LIQUID2_BACKUP_DIR"),
		logLevel:  getenv("LIQUID2_LOG_LEVEL", "info"),
		logFormat: getenv("LIQUID2_LOG_FORMAT", logging.FormatText),
	}
	flags := flag.NewFlagSet("backup", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&config.dbPath, "db", config.dbPath, "SQLite database path")
	flags.StringVar(&config.outputDir, "out-dir", config.outputDir, "backup output directory")
	flags.StringVar(&config.filename, "filename", "", "backup filename")
	flags.StringVar(&config.logLevel, "log-level", config.logLevel, "log level")
	flags.StringVar(&config.logFormat, "log-format", config.logFormat, "log format")
	if err := flags.Parse(args); err != nil {
		return backupConfig{}, err
	}
	loaded, err := liquidconfig.Load(liquidconfig.Args{
		liquidconfig.KeyDBPath:    config.dbPath,
		liquidconfig.KeyBackupDir: config.outputDir,
		liquidconfig.KeyLogLevel:  config.logLevel,
		liquidconfig.KeyLogFormat: config.logFormat,
	})
	if err != nil {
		return backupConfig{}, err
	}
	config.dbPath = loaded.Value(liquidconfig.KeyDBPath, "")
	config.outputDir = loaded.Value(liquidconfig.KeyBackupDir, "")
	config.logLevel = loaded.Value(liquidconfig.KeyLogLevel, "info")
	config.logFormat = loaded.Value(liquidconfig.KeyLogFormat, logging.FormatText)
	return config, nil
}

func validateConfig(config backupConfig) error {
	if config.dbPath == "" {
		return fmt.Errorf("database path is required")
	}
	if config.outputDir == "" {
		return fmt.Errorf("backup output directory is required")
	}
	if isMemoryDBPath(config.dbPath) {
		return sqlitestore.ErrBackupInMemorySource
	}
	info, err := os.Stat(config.dbPath)
	if err != nil {
		return fmt.Errorf("database path is not readable: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("database path must be a file")
	}
	if config.filename != "" && filepath.Base(config.filename) != config.filename {
		return fmt.Errorf("backup filename must not contain path separators")
	}
	if config.filename != "" && backupArtifactID(config.filename) == "" {
		return sqlitestore.ErrBackupInvalidArtifactID
	}
	return nil
}

func createBackup(ctx context.Context, logger *slog.Logger, config backupConfig) (sqlitestore.BackupResult, error) {
	if err := os.MkdirAll(config.outputDir, 0o700); err != nil {
		return sqlitestore.BackupResult{}, fmt.Errorf("create backup directory: %w", err)
	}
	store, err := sqlitestore.Open(ctx, config.dbPath, sqlitestore.WithLogger(logger))
	if err != nil {
		return sqlitestore.BackupResult{}, fmt.Errorf("open sqlite store: %w", err)
	}
	defer store.Close()
	filename := config.filename
	if filename == "" {
		filename = defaultBackupFilename(time.Now().UTC())
	}
	result, err := store.Backup(ctx, filepath.Join(config.outputDir, filename))
	if err != nil {
		return sqlitestore.BackupResult{}, err
	}
	logger.Info("backup command completed",
		slog.String("operation", "backup_command"),
		slog.String("backup_id", result.ID),
		slog.Int64("schema_version", result.SchemaVersion),
		slog.Int64("size_bytes", result.SizeBytes),
	)
	return result, nil
}

func backupArtifactFromResult(result sqlitestore.BackupResult) backupArtifact {
	return backupArtifact{
		ID: result.ID, CreatedAt: result.CreatedAt, SourceType: result.SourceType,
		SchemaVersion: result.SchemaVersion, SizeBytes: result.SizeBytes,
		SHA256: result.SHA256, Filename: result.Filename,
	}
}

func defaultBackupFilename(now time.Time) string {
	return "backup_" + now.Format("20060102T150405000Z") + ".sqlite3"
}

func backupArtifactID(filename string) string {
	id := filename[:len(filename)-len(filepath.Ext(filename))]
	if id == "" || id[0] == '.' {
		return ""
	}
	return id
}

func isMemoryDBPath(path string) bool {
	normalized := strings.ToLower(strings.TrimSpace(path))
	return normalized == ":memory:" || strings.Contains(normalized, "mode=memory")
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
