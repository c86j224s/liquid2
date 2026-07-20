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

	_ "modernc.org/sqlite"
)

func TestRunStorageStatsJSON(t *testing.T) {
	dbPath := createCLIReclaimableStorageDB(t)
	var out, errOut bytes.Buffer

	code := run(context.Background(), []string{"storage", "stats", "-db", dbPath, "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run returned %d, stderr %q", code, errOut.String())
	}
	var payload struct {
		Storage struct {
			DBPath           string `json:"db_path"`
			FreelistCount    int64  `json:"freelist_count"`
			ReclaimableBytes int64  `json:"reclaimable_bytes"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("parse json: %v\n%s", err, out.String())
	}
	if payload.Storage.DBPath != dbPath {
		t.Fatalf("unexpected db path %q", payload.Storage.DBPath)
	}
	if payload.Storage.FreelistCount <= 0 || payload.Storage.ReclaimableBytes <= 0 {
		t.Fatalf("expected reclaimable stats, got %#v", payload.Storage)
	}
}

func TestRunStorageCompactOutput(t *testing.T) {
	dbPath := createCLIReclaimableStorageDB(t)
	outputPath := filepath.Join(t.TempDir(), "compact.db")
	var out, errOut bytes.Buffer

	code := run(context.Background(), []string{"storage", "compact", "-db", dbPath, "-output", outputPath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run returned %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "compacted storage") || !strings.Contains(out.String(), "replaced           false") {
		t.Fatalf("unexpected output %q", out.String())
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected compacted output: %v", err)
	}
}

func TestRunStorageCompactDryRunJSON(t *testing.T) {
	dbPath := createCLIReclaimableStorageDB(t)
	var out, errOut bytes.Buffer

	code := run(context.Background(), []string{"storage", "compact", "-db", dbPath, "-dry-run", "-json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run returned %d, stderr %q", code, errOut.String())
	}
	var payload struct {
		Compact struct {
			DryRun     bool   `json:"dry_run"`
			Replaced   bool   `json:"replaced"`
			OutputPath string `json:"output_path"`
			SavedBytes int64  `json:"saved_bytes"`
		} `json:"compact"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("parse json: %v\n%s", err, out.String())
	}
	if !payload.Compact.DryRun || payload.Compact.Replaced || payload.Compact.OutputPath != "" {
		t.Fatalf("unexpected dry-run payload: %#v", payload.Compact)
	}
	if payload.Compact.SavedBytes <= 0 {
		t.Fatalf("expected dry-run saved bytes, got %#v", payload.Compact)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(dbPath), ".*.compact-*.tmp"))
	if err != nil {
		t.Fatalf("glob temp outputs: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("dry-run left temporary output: %v", matches)
	}
}

func TestRunStorageCompactDryRunRejectsOutput(t *testing.T) {
	dbPath := createCLIReclaimableStorageDB(t)
	var out, errOut bytes.Buffer

	code := run(context.Background(), []string{"storage", "compact", "-db", dbPath, "-dry-run", "-output", filepath.Join(t.TempDir(), "out.db")}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run returned %d, stderr %q stdout %q", code, errOut.String(), out.String())
	}
	if !strings.Contains(errOut.String(), "--dry-run cannot be combined") {
		t.Fatalf("unexpected stderr %q", errOut.String())
	}
}

func TestRunStorageCompactReplace(t *testing.T) {
	dbPath := createCLIReclaimableStorageDB(t)
	var out, errOut bytes.Buffer

	code := run(context.Background(), []string{"storage", "compact", "-db", dbPath, "-replace"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run returned %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "replaced           true") || !strings.Contains(out.String(), "backup") {
		t.Fatalf("unexpected output %q", out.String())
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected replaced db: %v", err)
	}
}

func TestRunStorageCompactRequiresOutputOrReplace(t *testing.T) {
	dbPath := createCLIReclaimableStorageDB(t)
	var out, errOut bytes.Buffer

	code := run(context.Background(), []string{"storage", "compact", "-db", dbPath}, &out, &errOut)
	if code != 2 {
		t.Fatalf("run returned %d, stderr %q stdout %q", code, errOut.String(), out.String())
	}
	if !strings.Contains(errOut.String(), "--output") || !strings.Contains(errOut.String(), "--replace") {
		t.Fatalf("unexpected stderr %q", errOut.String())
	}
}

func createCLIReclaimableStorageDB(t *testing.T) string {
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
