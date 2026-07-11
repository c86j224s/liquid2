package sqlite

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupCreatesReadableSQLiteCopy(t *testing.T) {
	store, ctx := newTestFileStore(t)
	q := store.Queries()
	createTestDocument(t, ctx, q, "doc_1")
	destination := filepath.Join(t.TempDir(), "backup.sqlite3")

	result, err := store.Backup(ctx, destination)
	if err != nil {
		t.Fatalf("backup sqlite store: %v", err)
	}
	if result.ID != "backup" || result.Filename != "backup.sqlite3" {
		t.Fatalf("unexpected backup identity: %#v", result)
	}
	if result.SourceType != BackupSourceSQLite || result.SchemaVersion != 13 {
		t.Fatalf("unexpected backup metadata: %#v", result)
	}
	if result.SizeBytes <= 0 || len(result.SHA256) != 64 {
		t.Fatalf("expected size and checksum, got %#v", result)
	}
	assertPrivateFileMode(t, destination)

	backupStore, err := Open(ctx, destination)
	if err != nil {
		t.Fatalf("open backup store: %v", err)
	}
	defer backupStore.Close()
	doc, err := backupStore.Queries().GetDocument(ctx, "doc_1")
	if err != nil {
		t.Fatalf("read backup document: %v", err)
	}
	if doc.Title != "Example" {
		t.Fatalf("unexpected backup document: %#v", doc)
	}
}

func TestBackupRejectsUnsafeSourcesAndDestinations(t *testing.T) {
	store, ctx := newTestFileStore(t)
	destination := filepath.Join(t.TempDir(), "backup.sqlite3")
	if err := writeTestFile(destination, "exists"); err != nil {
		t.Fatalf("seed existing destination: %v", err)
	}
	if _, err := store.Backup(ctx, destination); !errors.Is(err, ErrBackupDestinationExists) {
		t.Fatalf("expected existing destination error, got %v", err)
	}

	memoryStore, ctx := newTestStore(t)
	_, err := memoryStore.Backup(ctx, filepath.Join(t.TempDir(), "memory.sqlite3"))
	if !errors.Is(err, ErrBackupInMemorySource) {
		t.Fatalf("expected in-memory source error, got %v", err)
	}
	_, err = store.Backup(ctx, filepath.Join(t.TempDir(), ".sqlite3"))
	if !errors.Is(err, ErrBackupInvalidArtifactID) {
		t.Fatalf("expected invalid artifact id error, got %v", err)
	}
}

func TestBackupForcesPrivateFileModeInPermissiveDirectory(t *testing.T) {
	store, ctx := newTestFileStore(t)
	createTestDocument(t, ctx, store.Queries(), "doc_1")
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatalf("make permissive backup dir: %v", err)
	}
	destination := filepath.Join(dir, "backup.sqlite3")
	if _, err := store.Backup(ctx, destination); err != nil {
		t.Fatalf("backup sqlite store: %v", err)
	}
	assertPrivateFileMode(t, destination)
}

func assertPrivateFileMode(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat backup file: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("expected backup file mode 0600, got %o", mode)
	}
}
