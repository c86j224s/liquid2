package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func newTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()

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
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store, ctx
}

func newTestFileStore(t *testing.T) (*Store, context.Context) {
	t.Helper()

	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	if err != nil {
		t.Fatalf("open file store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store, ctx
}

func writeTestFile(path string, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o600)
}

func createTestDocument(t *testing.T, ctx context.Context, q *sqlitedb.Queries, id string) {
	t.Helper()
	createTestFolder(t, ctx, q, "folder_test")

	_, err := q.CreateDocument(ctx, sqlitedb.CreateDocumentParams{
		ID:        id,
		Title:     "Example",
		Kind:      "bookmark",
		FolderID:  nullString("folder_test"),
		Status:    "unread",
		CreatedAt: 1000,
		UpdatedAt: 1000,
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
}

func createTestFolder(t *testing.T, ctx context.Context, q *sqlitedb.Queries, id string) {
	t.Helper()
	if _, err := q.GetFolder(ctx, id); err == nil {
		return
	} else if err != sql.ErrNoRows {
		t.Fatalf("get test folder: %v", err)
	}
	_, err := q.CreateFolder(ctx, sqlitedb.CreateFolderParams{
		ID: id, Name: "Test Folder", CreatedAt: 1000, UpdatedAt: 1000,
	})
	if err != nil {
		t.Fatalf("create test folder: %v", err)
	}
}

func validHash() string {
	return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}

func nullInt(value int64) sql.NullInt64 {
	return sql.NullInt64{Int64: value, Valid: true}
}
