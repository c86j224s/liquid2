package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	liquidconfig "github.com/c86j224s/liquid2/internal/config"
	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func TestRunCreatesBackupArtifact(t *testing.T) {
	isolateConfig(t)
	ctx := context.Background()
	dbPath := seedBackupCommandDB(t, ctx)
	outDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(ctx, []string{
		"-db", dbPath,
		"-out-dir", outDir,
		"-filename", "test-backup.sqlite3",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got %d stderr=%s", code, stderr.String())
	}
	var response backupResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode backup response: %v output=%s", err, stdout.String())
	}
	backup := response.Backup
	if backup.ID != "test-backup" || backup.Filename != "test-backup.sqlite3" {
		t.Fatalf("unexpected backup identity: %#v", backup)
	}
	if backup.SchemaVersion != 13 || backup.SizeBytes <= 0 || len(backup.SHA256) != 64 {
		t.Fatalf("unexpected backup metadata: %#v", backup)
	}
	backupPath := filepath.Join(outDir, "test-backup.sqlite3")
	store, err := sqlitestore.Open(ctx, backupPath)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer store.Close()
	doc, err := store.Queries().GetDocument(ctx, "doc_1")
	if err != nil {
		t.Fatalf("read backup document: %v", err)
	}
	if doc.Title != "Command Backup" {
		t.Fatalf("unexpected backup document: %#v", doc)
	}
}

func TestRunRejectsExistingBackupFile(t *testing.T) {
	isolateConfig(t)
	ctx := context.Background()
	dbPath := seedBackupCommandDB(t, ctx)
	outDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outDir, "exists.sqlite3"), []byte("exists"), 0o600); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}
	var stderr bytes.Buffer
	code := run(ctx, []string{
		"-db", dbPath,
		"-out-dir", outDir,
		"-filename", "exists.sqlite3",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("expected backup failure, got %d stderr=%s", code, stderr.String())
	}
}

func TestRunRejectsInvalidConfig(t *testing.T) {
	isolateConfig(t)
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"-out-dir", t.TempDir()}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("expected usage failure, got %d stderr=%s", code, stderr.String())
	}
}

func TestRunRejectsInMemoryDatabaseClearly(t *testing.T) {
	isolateConfig(t)
	var stderr bytes.Buffer
	code := run(context.Background(), []string{
		"-db", ":memory:",
		"-out-dir", t.TempDir(),
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("expected usage failure, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "file-backed source") {
		t.Fatalf("expected file-backed source error, got %s", stderr.String())
	}
}

func TestRunRejectsEmptyBackupArtifactID(t *testing.T) {
	isolateConfig(t)
	dbPath := seedBackupCommandDB(t, context.Background())
	var stderr bytes.Buffer
	code := run(context.Background(), []string{
		"-db", dbPath,
		"-out-dir", t.TempDir(),
		"-filename", ".sqlite3",
	}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("expected usage failure, got %d stderr=%s", code, stderr.String())
	}
}

func isolateConfig(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv(liquidconfig.RuntimeModeEnv, liquidconfig.RuntimeModeRelease)
	t.Chdir(t.TempDir())
}

func seedBackupCommandDB(t *testing.T, ctx context.Context) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "liquid2.db")
	store, err := sqlitestore.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate sqlite store: %v", err)
	}
	_, err = store.Queries().CreateDocument(ctx, sqlitedb.CreateDocumentParams{
		ID:        "doc_1",
		Title:     "Command Backup",
		Kind:      "bookmark",
		FolderID:  sql.NullString{String: "folder_default_inbox", Valid: true},
		Status:    "unread",
		CreatedAt: 1000,
		UpdatedAt: 1000,
		Rating:    sql.NullInt64{},
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	return dbPath
}
