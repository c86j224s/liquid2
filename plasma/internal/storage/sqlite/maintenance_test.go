package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestReadStorageStatsReportsFreelist(t *testing.T) {
	ctx := context.Background()
	dbPath := createReclaimableStorageDB(t)

	stats, err := ReadStorageStats(ctx, dbPath)
	if err != nil {
		t.Fatalf("ReadStorageStats returned error: %v", err)
	}
	if stats.DBPath != dbPath {
		t.Fatalf("unexpected db path %q", stats.DBPath)
	}
	if stats.DBBytes <= 0 || stats.PageSize <= 0 || stats.PageCount <= 0 {
		t.Fatalf("unexpected size stats: %#v", stats)
	}
	if stats.FreelistCount <= 0 {
		t.Fatalf("expected reclaimable freelist pages, got %#v", stats)
	}
	if stats.ReclaimableBytes != stats.PageSize*stats.FreelistCount {
		t.Fatalf("unexpected reclaimable bytes: %#v", stats)
	}
}

func TestCompactStorageToCreatesVerifiedCopy(t *testing.T) {
	ctx := context.Background()
	dbPath := createReclaimableStorageDB(t)
	outputPath := filepath.Join(t.TempDir(), "plasma.compact.db")

	result, err := CompactStorageTo(ctx, dbPath, outputPath)
	if err != nil {
		t.Fatalf("CompactStorageTo returned error: %v", err)
	}
	if result.Replaced {
		t.Fatal("copy compact should not replace source")
	}
	if result.OutputPath != outputPath {
		t.Fatalf("unexpected output path %q", result.OutputPath)
	}
	if result.IntegrityCheck != "ok" {
		t.Fatalf("unexpected integrity check %q", result.IntegrityCheck)
	}
	if result.Compacted.FreelistCount != 0 {
		t.Fatalf("expected compacted copy to have no freelist pages, got %#v", result.Compacted)
	}
	if result.SavedBytes <= 0 {
		t.Fatalf("expected compact to save bytes, got %#v", result)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected compacted output: %v", err)
	}

	sourceStats, err := ReadStorageStats(ctx, dbPath)
	if err != nil {
		t.Fatalf("read source stats: %v", err)
	}
	if sourceStats.FreelistCount <= 0 {
		t.Fatalf("source should remain unchanged, got %#v", sourceStats)
	}
}

func TestCompactStorageDryRunRemovesTemporaryCopy(t *testing.T) {
	ctx := context.Background()
	dbPath := createReclaimableStorageDB(t)

	result, err := CompactStorageDryRun(ctx, dbPath)
	if err != nil {
		t.Fatalf("CompactStorageDryRun returned error: %v", err)
	}
	if !result.DryRun || result.Replaced || result.OutputPath != "" {
		t.Fatalf("unexpected dry-run result: %#v", result)
	}
	if result.Compacted.FreelistCount != 0 {
		t.Fatalf("expected dry-run compacted stats, got %#v", result.Compacted)
	}
	if result.SavedBytes <= 0 {
		t.Fatalf("expected dry-run to report saved bytes, got %#v", result)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(dbPath), ".*.compact-*.tmp"))
	if err != nil {
		t.Fatalf("glob temp outputs: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("dry-run left temporary output: %v", matches)
	}
	sourceStats, err := ReadStorageStats(ctx, dbPath)
	if err != nil {
		t.Fatalf("read source stats: %v", err)
	}
	if sourceStats.FreelistCount <= 0 {
		t.Fatalf("source should remain unchanged, got %#v", sourceStats)
	}
}

func TestCompactStorageToRejectsInvalidInputs(t *testing.T) {
	ctx := context.Background()
	dbPath := createReclaimableStorageDB(t)
	outputPath := filepath.Join(t.TempDir(), "exists.db")
	if err := os.WriteFile(outputPath, []byte("exists"), 0o600); err != nil {
		t.Fatalf("write destination: %v", err)
	}

	if _, err := CompactStorageTo(ctx, ":memory:", filepath.Join(t.TempDir(), "out.db")); !errors.Is(err, ErrStorageMaintenanceFileBackedRequired) {
		t.Fatalf("expected file-backed error, got %v", err)
	}
	if _, err := CompactStorageTo(ctx, dbPath, outputPath); !errors.Is(err, ErrStorageMaintenanceDestinationExists) {
		t.Fatalf("expected destination exists error, got %v", err)
	}
}

func TestCompactStorageToRequiresOfflineDatabase(t *testing.T) {
	ctx := context.Background()
	dbPath := createReclaimableStorageDB(t)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `BEGIN EXCLUSIVE`); err != nil {
		t.Fatalf("begin exclusive: %v", err)
	}
	defer conn.ExecContext(ctx, `ROLLBACK`)

	_, err = CompactStorageTo(ctx, dbPath, filepath.Join(t.TempDir(), "out.db"))
	if !errors.Is(err, ErrStorageMaintenanceOfflineRequired) {
		t.Fatalf("expected offline-required error, got %v", err)
	}
}

func TestCompactStorageReplaceBacksUpAndReplaces(t *testing.T) {
	ctx := context.Background()
	dbPath := createReclaimableStorageDB(t)

	result, err := CompactStorageReplace(ctx, dbPath)
	if err != nil {
		t.Fatalf("CompactStorageReplace returned error: %v", err)
	}
	if !result.Replaced {
		t.Fatal("expected replace result")
	}
	if result.OutputPath != dbPath {
		t.Fatalf("unexpected output path %q", result.OutputPath)
	}
	if len(result.BackupPaths) == 0 {
		t.Fatalf("expected backup paths, got %#v", result)
	}
	for _, backup := range result.BackupPaths {
		if _, err := os.Stat(backup); err != nil {
			t.Fatalf("expected backup %s: %v", backup, err)
		}
	}
	stats, err := ReadStorageStats(ctx, dbPath)
	if err != nil {
		t.Fatalf("read replaced stats: %v", err)
	}
	if stats.FreelistCount != 0 {
		t.Fatalf("expected replaced db to be compacted, got %#v", stats)
	}
	if result.SavedBytes <= 0 {
		t.Fatalf("expected replace to save bytes, got %#v", result)
	}
}

func createReclaimableStorageDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE blobs (id INTEGER PRIMARY KEY, body BLOB NOT NULL)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	for i := 0; i < 80; i++ {
		if _, err := db.Exec(`INSERT INTO blobs (body) VALUES (zeroblob(8192))`); err != nil {
			t.Fatalf("insert blob: %v", err)
		}
	}
	if _, err := db.Exec(`DELETE FROM blobs WHERE id <= 60`); err != nil {
		t.Fatalf("delete blobs: %v", err)
	}
	return dbPath
}
