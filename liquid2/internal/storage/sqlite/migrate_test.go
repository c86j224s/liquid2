package sqlite

import (
	"context"
	"testing"
)

func TestMigrateAppliesInitialSchema(t *testing.T) {
	store, ctx := newTestStore(t)

	version, err := store.Queries().SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("schema version: %v", err)
	}
	if version != 13 {
		t.Fatalf("expected schema version 13, got %d", version)
	}
	assertIndexExists(t, ctx, store, "document_tags_tag_idx")
}

func assertIndexExists(t *testing.T, ctx context.Context, store *Store, name string) {
	t.Helper()
	var count int
	err := store.DB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'index' AND name = ?
	`, name).Scan(&count)
	if err != nil {
		t.Fatalf("count index %s: %v", name, err)
	}
	if count != 1 {
		t.Fatalf("expected index %s to exist", name)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	store, ctx := newTestStore(t)

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestMigrateDeduplicatesActiveJobsBeforeIndex(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	if err := store.applyMigration(ctx, 1, "0001_init.sql"); err != nil {
		t.Fatalf("apply migration 1: %v", err)
	}
	if err := store.applyMigration(ctx, 2, "0002_app_sequence.sql"); err != nil {
		t.Fatalf("apply migration 2: %v", err)
	}
	seedDuplicateActiveJobs(t, ctx, store)

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate duplicate active jobs: %v", err)
	}
	var active int
	err = store.DB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM jobs
		WHERE kind = 'poll_feed'
		  AND payload_json = '{"feedId":"feed_1"}'
		  AND status IN ('queued', 'running')
	`).Scan(&active)
	if err != nil {
		t.Fatalf("count active jobs: %v", err)
	}
	if active != 1 {
		t.Fatalf("expected one active duplicate survivor, got %d", active)
	}
	var failed int
	err = store.DB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM jobs
		WHERE error = 'deduplicated active job during migration'
	`).Scan(&failed)
	if err != nil {
		t.Fatalf("count failed jobs: %v", err)
	}
	if failed != 2 {
		t.Fatalf("expected two deduplicated jobs, got %d", failed)
	}
}

func seedDuplicateActiveJobs(t *testing.T, ctx context.Context, store *Store) {
	t.Helper()
	_, err := store.DB().ExecContext(ctx, `
		INSERT INTO jobs (
		  id, kind, status, payload_json, attempts, created_at, updated_at
		) VALUES
		  ('job_1', 'poll_feed', 'queued', '{"feedId":"feed_1"}', 0, 1000, 1000),
		  ('job_2', 'poll_feed', 'queued', '{"feedId":"feed_1"}', 0, 2000, 2000),
		  ('job_3', 'poll_feed', 'running', '{"feedId":"feed_1"}', 1, 3000, 3000),
		  ('job_4', 'poll_feed', 'queued', '{"feedId":"feed_2"}', 0, 4000, 4000)
	`)
	if err != nil {
		t.Fatalf("seed duplicate active jobs: %v", err)
	}
}
