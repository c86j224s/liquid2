package app

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"

	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
)

func TestSQLiteRepositoryViewDoesNotExposeWritableTx(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()

	err := service.repo.View(ctx, func(reader RepositoryReader) error {
		if _, ok := reader.(RepositoryTx); ok {
			return validation("view exposed writable repository tx")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view repository: %v", err)
	}
}

func TestSQLiteRepositoryCloseRejectsOperations(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, filepath.Join(t.TempDir(), "liquid2.db"), sqlitestore.WithLogger(slog.Default()))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close sqlite store: %v", err)
		}
	}()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate sqlite store: %v", err)
	}
	repo := NewSQLiteRepository(store)
	if err := repo.Close(); err != nil {
		t.Fatalf("close repository: %v", err)
	}
	err = repo.View(ctx, func(RepositoryReader) error { return nil })
	if !errors.Is(err, errRepositoryClosed) {
		t.Fatalf("expected repository closed error from view, got %v", err)
	}
	err = repo.Update(ctx, func(RepositoryTx) error { return nil })
	if !errors.Is(err, errRepositoryClosed) {
		t.Fatalf("expected repository closed error from update, got %v", err)
	}
}

func TestNewSQLiteRepositoryRequiresStore(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = NewSQLiteRepository(nil)
}
