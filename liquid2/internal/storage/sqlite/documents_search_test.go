package sqlite

import (
	"context"
	"testing"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func TestDocumentFTSBackfillsExistingRows(t *testing.T) {
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
	createLegacyDocumentForFTS(t, ctx, store)

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate remaining schema: %v", err)
	}
	assertFTSMatch(t, ctx, store, "pipeline", []string{"doc_1"})
}

func createLegacyDocumentForFTS(t *testing.T, ctx context.Context, store *Store) {
	t.Helper()
	statements := []string{
		`INSERT INTO folders (id, name, sort_order, created_at, updated_at)
		 VALUES ('folder_test', 'Test Folder', 0, 1000, 1000)`,
		`INSERT INTO documents (id, title, kind, folder_id, status, created_at, updated_at)
		 VALUES ('doc_1', 'Example', 'bookmark', 'folder_test', 'unread', 1000, 1000)`,
		`INSERT INTO document_contents (id, document_id, role, format, content, created_at)
		 VALUES ('content_1', 'doc_1', 'original', 'text', 'pipeline search body', 1000)`,
	}
	for _, statement := range statements {
		if _, err := store.DB().ExecContext(ctx, statement); err != nil {
			t.Fatalf("seed legacy fts row: %v", err)
		}
	}
}

func TestDocumentFTSTriggersTrackWrites(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()

	createTestDocument(t, ctx, q, "doc_1")
	_, err := q.UpdateDocumentMetadata(ctx, sqlitedb.UpdateDocumentMetadataParams{
		ID: "doc_1", Title: "SQLite Notes", FolderID: nullString("folder_test"),
		UpdatedAt: 2000,
	})
	if err != nil {
		t.Fatalf("update title: %v", err)
	}
	assertFTSMatch(t, ctx, store, "sqlite", []string{"doc_1"})

	_, err = q.CreateDocumentContent(ctx, sqlitedb.CreateDocumentContentParams{
		ID: "content_1", DocumentID: "doc_1", Role: "original", Format: "text",
		Content: "refresh pipeline body", CreatedAt: 3000,
	})
	if err != nil {
		t.Fatalf("create content: %v", err)
	}
	assertFTSMatch(t, ctx, store, "pipeline", []string{"doc_1"})

	if err := q.DeleteDocumentContents(ctx, "doc_1"); err != nil {
		t.Fatalf("delete contents: %v", err)
	}
	assertFTSMatch(t, ctx, store, "pipeline", nil)
}

func assertFTSMatch(t *testing.T, ctx context.Context, store *Store, query string, expected []string) {
	t.Helper()
	rows, err := store.DB().QueryContext(ctx, `
		SELECT document_id
		FROM documents_fts(?)
		ORDER BY document_id
	`, query)
	if err != nil {
		t.Fatalf("query fts: %v", err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan fts row: %v", err)
		}
		got = append(got, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate fts rows: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected matches %#v, got %#v", expected, got)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("expected matches %#v, got %#v", expected, got)
		}
	}
}
