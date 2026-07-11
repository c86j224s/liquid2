package app

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteRepositorySearchUsesDocumentIDBoundary(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()

	if err := service.SeedDemo(ctx); err != nil {
		t.Fatalf("seed demo: %v", err)
	}
	list, err := service.ListDocuments(ctx, DocumentFilters{Query: "1mb", Tag: "backend"})
	if err != nil {
		t.Fatalf("search sqlite documents: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != "doc_demo_sqlite" {
		t.Fatalf("expected sqlite search result, got %#v", list.Items)
	}
	list, err = service.ListDocuments(ctx, DocumentFilters{
		Query: "1mb", Tag: "backend", Sort: DocumentSortRecent,
	})
	if err != nil {
		t.Fatalf("search sqlite documents with recent sort: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != "doc_demo_sqlite" {
		t.Fatalf("expected sorted sqlite search result, got %#v", list.Items)
	}
}

func TestSQLiteRepositoryListUsesCursorAfterLimitedFirstPage(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()

	if err := service.SeedDemo(ctx); err != nil {
		t.Fatalf("seed demo: %v", err)
	}
	first, err := service.ListDocuments(ctx, DocumentFilters{
		Sort: DocumentSortCreatedDesc, Limit: 1,
	})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(first.Items) != 1 || first.Items[0].ID != "doc_demo_flutter" ||
		first.NextCursor == nil {
		t.Fatalf("unexpected first page %#v", first)
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
}

func TestSQLiteRepositorySearchUsesCursorAfterLimitedFirstPage(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()

	if err := service.SeedDemo(ctx); err != nil {
		t.Fatalf("seed demo: %v", err)
	}
	first, err := service.ListDocuments(ctx, DocumentFilters{Query: "상태", Limit: 1})
	if err != nil {
		t.Fatalf("search first page: %v", err)
	}
	if len(first.Items) != 1 || first.NextCursor == nil {
		t.Fatalf("unexpected first search page %#v", first)
	}
	second, err := service.ListDocuments(ctx, DocumentFilters{
		Query: "상태", Limit: 1, Cursor: *first.NextCursor,
	})
	if err != nil {
		t.Fatalf("search second page: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID {
		t.Fatalf("unexpected second search page %#v after %#v", second, first)
	}
	if second.TotalCount != -1 {
		t.Fatalf("expected second page total count skipped, got %d", second.TotalCount)
	}
}
