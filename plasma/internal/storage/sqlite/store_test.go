package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"

	_ "modernc.org/sqlite"
)

func TestOpenCreatesSeparatePlasmaDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "plasma.db")
	store, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	versions, err := store.MigrationVersions(context.Background())
	if err != nil {
		t.Fatalf("MigrationVersions returned error: %v", err)
	}
	if len(versions) != 9 ||
		versions[0] != "0001_bootstrap.sql" ||
		versions[1] != "0002_mission_ledger.sql" ||
		versions[2] != "0003_mission_projection.sql" ||
		versions[3] != "0004_source_snapshots.sql" ||
		versions[4] != "0005_research_records.sql" ||
		versions[5] != "0006_report_canvas.sql" ||
		versions[6] != "0007_confluence_connections.sql" ||
		versions[7] != "0008_mission_activity_list.sql" ||
		versions[8] != "0009_app_settings.sql" {
		t.Fatalf("unexpected migration versions: %#v", versions)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "plasma.db")
	for i := 0; i < 2; i++ {
		store, err := Open(context.Background(), dbPath)
		if err != nil {
			t.Fatalf("Open run %d returned error: %v", i, err)
		}
		store.Close()
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM plasma_schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != 9 {
		t.Fatalf("expected nine migration rows, got %d", count)
	}
}

func TestStorePersistsModelDefaults(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	defaults := app.ModelDefaults{
		WorkflowGoalModel:           "gpt-5.5",
		WorkflowGoalReasoningEffort: "high",
	}
	if err := store.SaveModelDefaults(ctx, defaults); err != nil {
		t.Fatalf("SaveModelDefaults returned error: %v", err)
	}
	got, err := store.GetModelDefaults(ctx)
	if err != nil {
		t.Fatalf("GetModelDefaults returned error: %v", err)
	}
	if got != defaults {
		t.Fatalf("unexpected defaults: %#v", got)
	}
}

func TestForeignKeysEnabledOnPooledConnections(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	store.db.SetMaxOpenConns(2)

	first, err := store.db.Conn(ctx)
	if err != nil {
		t.Fatalf("first conn: %v", err)
	}
	defer first.Close()
	second, err := store.db.Conn(ctx)
	if err != nil {
		t.Fatalf("second conn: %v", err)
	}

	var enabled int
	if err := second.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&enabled); err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if enabled != 1 {
		t.Fatalf("expected foreign_keys enabled on pooled connection, got %d", enabled)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("close second conn: %v", err)
	}

	_, err = store.AppendLedgerEvent(ctx, app.LedgerEvent{
		EventID:   "evt_1",
		MissionID: "mis_missing",
		EventType: "mission.created",
		Producer:  app.Producer{Type: "user", ID: "ses_1"},
		Payload:   []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected pooled connection to enforce mission foreign key")
	}
}

func TestAsyncWriterPragmasEnabled(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	var journalMode string
	if err := store.db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("expected WAL journal mode, got %q", journalMode)
	}

	var busyTimeout int
	if err := store.db.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout < 5000 {
		t.Fatalf("expected busy_timeout >= 5000, got %d", busyTimeout)
	}
}

func TestSQLiteDSNUsesImmediateTransactions(t *testing.T) {
	dsn := sqliteDSN(filepath.Join(t.TempDir(), "plasma.db"))
	if !strings.Contains(dsn, "_txlock=immediate") {
		t.Fatalf("expected immediate transaction lock in DSN, got %q", dsn)
	}
}
