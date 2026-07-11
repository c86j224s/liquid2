package sqlite

import (
	"database/sql"
	"errors"
	"testing"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func TestInTxRollsBackOnError(t *testing.T) {
	store, ctx := newTestStore(t)
	expected := errors.New("stop transaction")

	err := store.InTx(ctx, func(q *sqlitedb.Queries) error {
		createTestDocument(t, ctx, q, "doc_tx")
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected transaction error, got %v", err)
	}

	if _, err := store.Queries().GetDocument(ctx, "doc_tx"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected document to be rolled back, got %v", err)
	}
}
