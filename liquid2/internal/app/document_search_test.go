package app

import (
	"context"
	"errors"
	"testing"
)

func TestServiceSearchComposesWithFilters(t *testing.T) {
	service := NewService(WithDemoSeed())
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()

	list, err := service.ListDocuments(ctx, DocumentFilters{
		Query: "api", Tag: "flutter",
	})
	if err != nil {
		t.Fatalf("search documents: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != "doc_demo_flutter" {
		t.Fatalf("expected flutter search result, got %#v", list.Items)
	}
}

func TestServiceSearchIncludesFolderDescendants(t *testing.T) {
	service := NewService(WithDemoSeed())
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()

	exact, err := service.ListDocuments(ctx, DocumentFilters{FolderID: defaultDocumentFolderID})
	if err != nil {
		t.Fatalf("list exact folder documents: %v", err)
	}
	if len(exact.Items) != 0 {
		t.Fatalf("expected no direct inbox documents, got %#v", exact.Items)
	}
	withChildren, err := service.ListDocuments(ctx, DocumentFilters{
		FolderID: defaultDocumentFolderID, IncludeFolderDescendants: true,
	})
	if err != nil {
		t.Fatalf("list descendant folder documents: %v", err)
	}
	if !hasDocument(withChildren.Items, "doc_demo_sqlite") ||
		!hasDocument(withChildren.Items, "doc_demo_go") ||
		hasDocument(withChildren.Items, "doc_demo_flutter") {
		t.Fatalf("unexpected descendant folder documents %#v", withChildren.Items)
	}
}

func TestServiceSearchSortAndCursor(t *testing.T) {
	service := NewService(WithDemoSeed())
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()

	first, err := service.ListDocuments(ctx, DocumentFilters{
		Sort: DocumentSortCreatedDesc, Limit: 1,
	})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(first.Items) != 1 || first.Items[0].ID != "doc_demo_flutter" || first.NextCursor == nil {
		t.Fatalf("unexpected first page %#v", first)
	}
	if first.TotalCount != 3 {
		t.Fatalf("expected total count 3, got %d", first.TotalCount)
	}
	second, err := service.ListDocuments(ctx, DocumentFilters{
		Sort: DocumentSortCreatedDesc, Limit: 1, Cursor: *first.NextCursor,
	})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != "doc_demo_go" {
		t.Fatalf("unexpected second page %#v", second)
	}
	if second.TotalCount != -1 {
		t.Fatalf("expected second page total count skipped, got %d", second.TotalCount)
	}

	rated, err := service.ListDocuments(ctx, DocumentFilters{Sort: DocumentSortRatingDesc})
	if err != nil {
		t.Fatalf("list rating sort: %v", err)
	}
	if len(rated.Items) < 2 || rated.Items[0].ID != "doc_demo_sqlite" || rated.Items[1].ID != "doc_demo_go" {
		t.Fatalf("unexpected rating sort %#v", rated.Items)
	}
}

func TestServiceSearchRejectsInvalidSort(t *testing.T) {
	service := NewService(WithDemoSeed())
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()

	if _, err := service.ListDocuments(ctx, DocumentFilters{Sort: "title_asc"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected invalid sort validation error, got %v", err)
	}
	if _, err := service.ListDocuments(ctx, DocumentFilters{Sort: DocumentSortRelevance}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected relevance validation error, got %v", err)
	}
}
