package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
	httptransport "github.com/c86j224s/liquid2/internal/transport/http"
)

func TestNewBackupRunnerDisabledWithoutSQLiteOrDir(t *testing.T) {
	t.Setenv("LIQUID2_DB_PATH", "")
	t.Setenv("LIQUID2_BACKUP_DIR", t.TempDir())
	runner, err := newBackupRunner(testLogger(), nil)
	if err != nil || runner != nil {
		t.Fatalf("expected nil runner without SQLite, runner=%v err=%v", runner, err)
	}

	store := openMigratedStore(t, context.Background())
	t.Cleanup(func() { _ = store.Close() })
	t.Setenv("LIQUID2_DB_PATH", "liquid2.sqlite3")
	t.Setenv("LIQUID2_BACKUP_DIR", "")
	runner, err = newBackupRunner(testLogger(), store)
	if err != nil || runner != nil {
		t.Fatalf("expected nil runner without backup dir, runner=%v err=%v", runner, err)
	}

	parentFile := filepath.Join(t.TempDir(), "not-dir")
	if writeErr := os.WriteFile(parentFile, []byte("x"), 0o600); writeErr != nil {
		t.Fatalf("write parent file: %v", writeErr)
	}
	t.Setenv("LIQUID2_BACKUP_DIR", filepath.Join(parentFile, "backup"))
	runner, err = newBackupRunner(testLogger(), store)
	if err != nil || runner != nil {
		t.Fatalf("expected nil runner for unavailable backup dir, runner=%v err=%v", runner, err)
	}

	t.Setenv("LIQUID2_DB_PATH", ":memory:")
	t.Setenv("LIQUID2_BACKUP_DIR", t.TempDir())
	runner, err = newBackupRunner(testLogger(), store)
	if err != nil || runner != nil {
		t.Fatalf("expected nil runner for memory DB, runner=%v err=%v", runner, err)
	}
}

func TestSQLiteBackupRunnerCreatesAPISafeArtifact(t *testing.T) {
	ctx := context.Background()
	dbPath := seedBackupAPIDB(t, ctx)
	t.Setenv("LIQUID2_DB_PATH", dbPath)
	t.Setenv("LIQUID2_BACKUP_DIR", t.TempDir())
	store, err := sqlitestore.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err = store.Migrate(ctx); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	runner, err := newBackupRunner(testLogger(), store)
	if err != nil {
		t.Fatalf("new backup runner: %v", err)
	}
	sqliteRunner := runner.(*sqliteBackupRunner)
	sqliteRunner.clock = func() time.Time { return time.Date(2026, 6, 12, 1, 2, 3, 0, time.UTC) }

	artifact, err := runner.Backup(ctx)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if artifact.ID != "backup_20260612T010203000Z_1" || artifact.SourceType != "sqlite" {
		t.Fatalf("unexpected artifact identity: %#v", artifact)
	}
	if artifact.SchemaVersion != 13 || artifact.SizeBytes <= 0 || len(artifact.SHA256) != 64 {
		t.Fatalf("unexpected artifact metadata: %#v", artifact)
	}
	if artifact.DownloadURL != nil {
		t.Fatalf("expected nil download URL, got %#v", artifact.DownloadURL)
	}
	if strings.Contains(artifact.ID, string(filepath.Separator)) {
		t.Fatalf("artifact ID exposed path separator: %#v", artifact)
	}
}

func TestSQLiteBackupRunnerMapsMemoryStoreUnavailable(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open memory store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	runner := &sqliteBackupRunner{
		store: store, outputDir: t.TempDir(), logger: testLogger(), clock: time.Now,
	}
	_, err = runner.Backup(ctx)
	if !errors.Is(err, httptransport.ErrBackupUnavailable) {
		t.Fatalf("expected backup unavailable, got %v", err)
	}
}

func TestSQLiteBackupRunnerMapsDirectoryUnavailable(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t, ctx)
	t.Cleanup(func() { _ = store.Close() })
	parentFile := filepath.Join(t.TempDir(), "not-dir")
	if err := os.WriteFile(parentFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write parent file: %v", err)
	}
	runner := &sqliteBackupRunner{
		store: store, outputDir: filepath.Join(parentFile, "backup"), logger: testLogger(), clock: time.Now,
	}
	_, err := runner.Backup(ctx)
	if !errors.Is(err, httptransport.ErrBackupUnavailable) {
		t.Fatalf("expected backup unavailable, got %v", err)
	}
}

func seedBackupAPIDB(t *testing.T, ctx context.Context) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "liquid2.sqlite3")
	store, err := sqlitestore.Open(ctx, path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err = store.Migrate(ctx); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	_, err = store.Queries().CreateDocument(ctx, sqlitedb.CreateDocumentParams{
		ID: "doc_1", Title: "API Backup", Kind: "bookmark", Status: "unread",
		FolderID:  sql.NullString{String: "folder_default_inbox", Valid: true},
		CreatedAt: 1760000000000, UpdatedAt: 1760000000000,
	})
	if err != nil {
		t.Fatalf("seed document: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close seeded store: %v", err)
	}
	return path
}

func openMigratedStore(t *testing.T, ctx context.Context) *sqlitestore.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "liquid2.sqlite3")
	store, err := sqlitestore.Open(ctx, path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err = store.Migrate(ctx); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	return store
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
