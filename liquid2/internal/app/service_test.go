package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

func TestServiceHealth(t *testing.T) {
	service := NewService()
	t.Cleanup(func() { _ = service.Close() })

	health := service.Health(context.Background())
	if !health.OK {
		t.Fatal("expected healthy service")
	}
}

func TestServiceDemoSeedPopulatesReviewData(t *testing.T) {
	service := NewService(WithDemoSeed())
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()

	list, err := service.ListDocuments(ctx, DocumentFilters{})
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(list.Items) != 3 {
		t.Fatalf("expected 3 demo documents, got %d", len(list.Items))
	}
	if !hasDocument(list.Items, "doc_demo_sqlite") {
		t.Fatal("expected sqlite demo document")
	}

	detail, err := service.GetDocument(ctx, "doc_demo_sqlite")
	if err != nil {
		t.Fatalf("get demo document: %v", err)
	}
	if len(detail.Contents) != 1 || detail.Contents[0].Content == "" {
		t.Fatalf("expected demo content, got %#v", detail.Contents)
	}
	if detail.Contents[0].Language == nil || *detail.Contents[0].Language != "ko" {
		t.Fatalf("expected demo content language, got %#v", detail.Contents[0].Language)
	}
	*detail.Contents[0].Language = "mutated"
	detail, err = service.GetDocument(ctx, "doc_demo_sqlite")
	if err != nil {
		t.Fatalf("get demo document again: %v", err)
	}
	if detail.Contents[0].Language == nil || *detail.Contents[0].Language != "ko" {
		t.Fatalf("expected isolated demo content language, got %#v", detail.Contents[0].Language)
	}
	if len(detail.Tags) != 2 {
		t.Fatalf("expected demo tags, got %d", len(detail.Tags))
	}

	notes, err := service.ListDocumentNotes(ctx, "doc_demo_sqlite")
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if len(notes.Items) != 1 {
		t.Fatalf("expected demo note, got %d", len(notes.Items))
	}

	folders, err := service.ListFolders(ctx)
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	if len(folders) != 3 || len(folders[0].Children)+len(folders[1].Children) != 1 {
		t.Fatalf("expected nested demo folder tree, got %#v", folders)
	}
	if folders[0].Name != "Inbox" || folders[0].Children[0].Name != "Research" || folders[1].Name != "Feeds" {
		t.Fatalf("expected stable demo folder ordering, got %#v", folders)
	}
	if folders[2].Name != trashDocumentFolderName || folders[2].SystemRole == nil || *folders[2].SystemRole != FolderSystemRoleTrash {
		t.Fatalf("expected stable trash folder, got %#v", folders)
	}

	filtered, err := service.ListDocuments(ctx, DocumentFilters{Tag: "flutter"})
	if err != nil {
		t.Fatalf("filter documents: %v", err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].ID != "doc_demo_flutter" {
		t.Fatalf("expected flutter demo document, got %#v", filtered.Items)
	}
}

func TestServiceSerializesConcurrentDocumentCreation(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	const count = 32

	var wg sync.WaitGroup
	ids := make(chan string, count)
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doc, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "doc"})
			if err != nil {
				errs <- err
				return
			}
			ids <- doc.Document.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)

	for err := range errs {
		t.Fatalf("create document: %v", err)
	}
	seen := map[string]struct{}{}
	for id := range ids {
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate document id: %s", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != count {
		t.Fatalf("expected %d ids, got %d", count, len(seen))
	}
}

func TestServiceReturnsDomainErrorThroughRepository(t *testing.T) {
	repo := &recordingRepository{
		inner: newMemoryRepository(memoryRepositoryConfig{
			logger: slog.Default().With("component", "test"),
			now:    func() int64 { return 1760000000000 },
		}),
	}
	service := NewService(WithRepository(repo))
	t.Cleanup(func() { _ = service.Close() })

	_, err := service.CreateDocument(context.Background(), CreateDocumentInput{})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if !errors.Is(repo.updateErr, ErrValidation) {
		t.Fatalf("expected repository callback validation error, got %v", repo.updateErr)
	}
}

func TestMemoryRepositoryDoesNotTreatErrorPanicAsDomainError(t *testing.T) {
	service := NewService()
	t.Cleanup(func() { _ = service.Close() })
	boom := errors.New("boom")

	err := service.repo.Update(context.Background(), func(RepositoryTx) error {
		panic(boom)
	})
	if err == nil || !strings.Contains(err.Error(), "memory repository operation panic: boom") {
		t.Fatalf("expected wrapped panic error, got %v", err)
	}
	if errors.Is(err, boom) {
		t.Fatalf("expected panic error not to preserve the original error identity")
	}
}

type recordingRepository struct {
	inner     Repository
	updateErr error
}

func (repo *recordingRepository) View(ctx context.Context, fn func(RepositoryReader) error) error {
	return repo.inner.View(ctx, fn)
}

func (repo *recordingRepository) Update(ctx context.Context, fn func(RepositoryTx) error) error {
	return repo.inner.Update(ctx, func(tx RepositoryTx) error {
		err := fn(tx)
		repo.updateErr = err
		return err
	})
}

func (repo *recordingRepository) Close() error {
	return repo.inner.Close()
}

func hasDocument(items []DocumentSummary, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
