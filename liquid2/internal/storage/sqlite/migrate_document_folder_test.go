package sqlite

import (
	"context"
	"testing"
)

func TestMigrateRequiresDocumentFolder(t *testing.T) {
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
		INSERT INTO documents (
		  id, title, kind, status, created_at, updated_at
		) VALUES (
		  'doc_legacy', 'Legacy', 'bookmark', 'unread', 1000, 1000
		)
	`)
	if err != nil {
		t.Fatalf("seed legacy document: %v", err)
	}

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var folderID string
	err = store.DB().QueryRowContext(
		ctx,
		"SELECT folder_id FROM documents WHERE id = 'doc_legacy'",
	).Scan(&folderID)
	if err != nil {
		t.Fatalf("read migrated folder: %v", err)
	}
	if folderID == "" {
		t.Fatal("expected migrated document folder")
	}
	assertDocumentFolderOnDelete(t, ctx, store, "RESTRICT")
	_, err = store.DB().ExecContext(ctx, "DELETE FROM folders WHERE id = ?", folderID)
	if err == nil {
		t.Fatal("expected folder delete with documents to fail")
	}
	_, err = store.DB().ExecContext(ctx, `
		INSERT INTO documents (
		  id, title, kind, status, created_at, updated_at
		) VALUES (
		  'doc_invalid', 'Invalid', 'bookmark', 'unread', 2000, 2000
		)
	`)
	if err == nil {
		t.Fatal("expected insert without folder to fail")
	}
	_, err = store.DB().ExecContext(ctx, `
		UPDATE documents SET folder_id = NULL WHERE id = 'doc_legacy'
	`)
	if err == nil {
		t.Fatal("expected folder clear to fail")
	}
}

func assertDocumentFolderOnDelete(t *testing.T, ctx context.Context, store *Store, want string) {
	t.Helper()
	rows, err := store.DB().QueryContext(ctx, "PRAGMA foreign_key_list(documents)")
	if err != nil {
		t.Fatalf("foreign key list: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, seq int
		var tableName, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &tableName, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatalf("scan foreign key: %v", err)
		}
		if tableName == "folders" && from == "folder_id" {
			if onDelete != want {
				t.Fatalf("expected document folder on_delete %s, got %s", want, onDelete)
			}
			return
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("foreign key rows: %v", err)
	}
	t.Fatal("expected document folder foreign key")
}
