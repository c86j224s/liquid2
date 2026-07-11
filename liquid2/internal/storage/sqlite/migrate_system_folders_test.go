package sqlite

import (
	"context"
	"strings"
	"testing"
)

func TestMigrateAddsSystemFolderRoles(t *testing.T) {
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
	for _, migration := range []string{
		"0001_init.sql",
		"0002_app_sequence.sql",
		"0003_active_job_dedupe.sql",
		"0004_documents_fts.sql",
		"0005_document_versions.sql",
		"0006_document_list_indexes.sql",
		"0007_document_folder_required.sql",
	} {
		version, err := migrationVersion(migration)
		if err != nil {
			t.Fatalf("parse migration: %v", err)
		}
		if err := store.applyMigration(ctx, version, migration); err != nil {
			t.Fatalf("apply %s: %v", migration, err)
		}
	}
	_, err = store.DB().ExecContext(ctx, `
			INSERT INTO folders (
			  id, parent_id, name, sort_order, created_at, updated_at
			) VALUES
			(
			  'folder_existing_feeds', NULL, 'Feeds', 10, 1000, 1000
			),
			(
			  'folder_existing_trash', NULL, 'Trash', 20, 1000, 1000
			),
			(
			  'folder_existing_feeds_child', 'folder_existing_feeds', 'Child',
			  30, 1000, 1000
			)
		`)
	if err != nil {
		t.Fatalf("seed existing folders: %v", err)
	}
	_, err = store.DB().ExecContext(ctx, `
			INSERT INTO documents (
			  id, title, kind, folder_id, status, created_at, updated_at
			) VALUES (
			  'doc_existing_feed', 'Feed doc', 'rss_item',
			  'folder_existing_feeds', 'unread', 1000, 1000
			);

			INSERT INTO feeds (
			  id, url, folder_id, enabled, created_at, updated_at
			) VALUES (
			  'feed_existing', 'https://example.com/feed.xml',
			  'folder_existing_feeds', 1, 1000, 1000
			);
		`)
	if err != nil {
		t.Fatalf("seed folder references: %v", err)
	}

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var trashCount int
	err = store.DB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM folders WHERE system_role = 'trash'
	`).Scan(&trashCount)
	if err != nil {
		t.Fatalf("count trash roles: %v", err)
	}
	if trashCount != 1 {
		t.Fatalf("expected one trash folder, got %d", trashCount)
	}
	var trashID string
	err = store.DB().QueryRowContext(ctx, `
		SELECT id FROM folders WHERE system_role = 'trash'
	`).Scan(&trashID)
	if err != nil {
		t.Fatalf("read trash id: %v", err)
	}
	if trashID != "folder_existing_trash" {
		t.Fatalf("expected existing trash promoted, got %q", trashID)
	}
	var feedsID string
	err = store.DB().QueryRowContext(ctx, `
		SELECT id FROM folders WHERE system_role = 'feeds'
	`).Scan(&feedsID)
	if err != nil {
		t.Fatalf("read feeds id: %v", err)
	}
	if feedsID != "folder_existing_feeds" {
		t.Fatalf("expected existing feeds promoted, got %q", feedsID)
	}
	assertFolderReference(t, ctx, store, "documents", "doc_existing_feed", "folder_existing_feeds")
	assertFolderReference(t, ctx, store, "feeds", "feed_existing", "folder_existing_feeds")
	var parentID string
	err = store.DB().QueryRowContext(ctx, `
		SELECT parent_id FROM folders WHERE id = 'folder_existing_feeds_child'
	`).Scan(&parentID)
	if err != nil {
		t.Fatalf("read feeds child parent: %v", err)
	}
	if parentID != "folder_existing_feeds" {
		t.Fatalf("expected child folder parent preserved, got %q", parentID)
	}
	assertNoForeignKeyCheckRows(t, ctx, store)
	assertFolderSystemRoleCheckIncludesFeeds(t, ctx, store)
}

func assertFolderReference(t *testing.T, ctx context.Context, store *Store, table string, id string, want string) {
	t.Helper()
	var folderID string
	err := store.DB().QueryRowContext(ctx, `
		SELECT folder_id FROM `+table+` WHERE id = ?
	`, id).Scan(&folderID)
	if err != nil {
		t.Fatalf("read %s folder reference: %v", table, err)
	}
	if folderID != want {
		t.Fatalf("expected %s folder reference %q, got %q", table, want, folderID)
	}
}

func assertNoForeignKeyCheckRows(t *testing.T, ctx context.Context, store *Store) {
	t.Helper()
	var violations int
	rows, err := store.DB().QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		t.Fatalf("foreign key check: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		violations++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("foreign key check rows: %v", err)
	}
	if violations != 0 {
		t.Fatalf("expected no foreign key violations, got %d", violations)
	}
}

func assertFolderSystemRoleCheckIncludesFeeds(t *testing.T, ctx context.Context, store *Store) {
	t.Helper()
	var schema string
	err := store.DB().QueryRowContext(ctx, `
		SELECT sql FROM sqlite_master
		WHERE type = 'table' AND name = 'folders'
	`).Scan(&schema)
	if err != nil {
		t.Fatalf("read folders schema: %v", err)
	}
	if !strings.Contains(schema, "'feeds'") {
		t.Fatalf("expected folders system_role check to include feeds: %s", schema)
	}
}
