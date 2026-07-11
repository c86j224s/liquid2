package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"
)

func TestMemoryRepositoryUpdateErrorRollsBack(t *testing.T) {
	repo := newMemoryRepository(memoryRepositoryConfig{
		logger: slog.Default().With("component", "test"),
		now:    func() int64 { return 1760000000000 },
	})
	t.Cleanup(func() { _ = repo.Close() })
	ctx := context.Background()

	err := repo.Update(ctx, func(tx RepositoryTx) error {
		tx.PutDocument(documentRecord{
			meta: DocumentMetadata{
				ID: "doc_abort", Title: "Abort", Kind: DocumentKindBookmark,
				Status: DocumentStatusUnread, CreatedAt: tx.Now(), UpdatedAt: tx.Now(),
			},
			contents: []DocumentContent{},
			blobs:    []BlobMetadata{},
			blobData: map[string][]byte{},
			tagIDs:   []string{},
		})
		return validation("abort")
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}

	err = repo.View(ctx, func(tx RepositoryReader) error {
		if _, ok := tx.Document("doc_abort"); ok {
			return fmt.Errorf("aborted document was committed")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view repository: %v", err)
	}
}

func TestMemoryRepositoryViewDoesNotExposeWritableTx(t *testing.T) {
	repo := newMemoryRepository(memoryRepositoryConfig{
		logger: slog.Default().With("component", "test"),
		now:    func() int64 { return 1760000000000 },
	})
	t.Cleanup(func() { _ = repo.Close() })

	err := repo.View(context.Background(), func(reader RepositoryReader) error {
		if _, ok := reader.(RepositoryTx); ok {
			return fmt.Errorf("view exposed writable repository tx")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view repository: %v", err)
	}
}

func TestMemoryRepositoryAcceptedUpdateIgnoresLaterContextCancel(t *testing.T) {
	repo := newMemoryRepository(memoryRepositoryConfig{
		logger: slog.Default().With("component", "test"),
		now:    func() int64 { return 1760000000000 },
	})
	t.Cleanup(func() { _ = repo.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	release := make(chan struct{})
	errs := make(chan error, 1)

	go func() {
		errs <- repo.Update(ctx, func(tx RepositoryTx) error {
			close(started)
			<-release
			tx.PutDocument(testDocumentRecord(tx, "doc_ctx_cancel"))
			return nil
		})
	}()
	<-started
	cancel()
	close(release)
	if err := <-errs; err != nil {
		t.Fatalf("accepted update returned error: %v", err)
	}

	err := repo.View(context.Background(), func(tx RepositoryReader) error {
		if _, ok := tx.Document("doc_ctx_cancel"); !ok {
			return fmt.Errorf("accepted update was not committed")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view repository: %v", err)
	}
}

func TestMemoryRepositoryCloseRejectsOperations(t *testing.T) {
	repo := newMemoryRepository(memoryRepositoryConfig{
		logger: slog.Default().With("component", "test"),
		now:    func() int64 { return 1760000000000 },
	})
	if err := repo.Close(); err != nil {
		t.Fatalf("close repository: %v", err)
	}
	err := repo.View(context.Background(), func(RepositoryReader) error {
		return nil
	})
	if !errors.Is(err, errRepositoryClosed) {
		t.Fatalf("expected repository closed error, got %v", err)
	}
}

func TestMemoryRepositoryCloseRejectsOperationsDuringShutdown(t *testing.T) {
	repo := newMemoryRepository(memoryRepositoryConfig{
		logger: slog.Default().With("component", "test"),
		now:    func() int64 { return 1760000000000 },
	})
	started := make(chan struct{})
	release := make(chan struct{})
	updateErrs := make(chan error, 1)
	closeErrs := make(chan error, 1)

	go func() {
		updateErrs <- repo.Update(context.Background(), func(tx RepositoryTx) error {
			close(started)
			<-release
			tx.PutDocument(testDocumentRecord(tx, "doc_close"))
			return nil
		})
	}()
	<-started
	go func() {
		closeErrs <- repo.Close()
	}()
	waitMemoryRepositoryClosing(t, repo)

	err := repo.View(context.Background(), func(RepositoryReader) error {
		return nil
	})
	if !errors.Is(err, errRepositoryClosed) {
		t.Fatalf("expected repository closed error during shutdown, got %v", err)
	}
	close(release)
	if err := <-updateErrs; err != nil {
		t.Fatalf("accepted update returned error: %v", err)
	}
	if err := <-closeErrs; err != nil {
		t.Fatalf("close repository: %v", err)
	}
}

func waitMemoryRepositoryClosing(t *testing.T, repo *memoryRepository) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		repo.mu.RLock()
		closing := repo.closing
		repo.mu.RUnlock()
		if closing {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("repository did not enter closing state")
}

func testDocumentRecord(tx RepositoryTx, id string) documentRecord {
	now := tx.Now()
	return documentRecord{
		meta: DocumentMetadata{
			ID: id, Title: "Test", Kind: DocumentKindBookmark,
			Status: DocumentStatusUnread, CreatedAt: now, UpdatedAt: now,
		},
		contents: []DocumentContent{},
		blobs:    []BlobMetadata{},
		blobData: map[string][]byte{},
		tagIDs:   []string{},
	}
}
