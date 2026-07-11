package app

import (
	"context"
	"testing"
)

func TestDocumentViewsIncludeFolderPath(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()

	root, err := service.CreateFolder(ctx, FolderInput{Name: "Inbox", SortOrder: 1})
	if err != nil {
		t.Fatalf("create root folder: %v", err)
	}
	child, err := service.CreateFolder(ctx, FolderInput{
		Name: "Research", ParentID: &root.ID, SortOrder: 1,
	})
	if err != nil {
		t.Fatalf("create child folder: %v", err)
	}
	document, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Paper"})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	if _, err = service.UpdateDocument(ctx, document.Document.ID, UpdateDocumentInput{
		FolderID: &child.ID,
	}); err != nil {
		t.Fatalf("move document: %v", err)
	}

	list, err := service.ListDocuments(ctx, DocumentFilters{})
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	assertFolderPath(t, list.Items[0].FolderPath, []string{"Inbox", "Research"})
	detail, err := service.GetDocument(ctx, document.Document.ID)
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	assertFolderPath(t, detail.FolderPath, []string{"Inbox", "Research"})
}

func assertFolderPath(t *testing.T, path []FolderBreadcrumb, names []string) {
	t.Helper()
	if len(path) != len(names) {
		t.Fatalf("expected folder path %v, got %#v", names, path)
	}
	for index, name := range names {
		if path[index].Name != name {
			t.Fatalf("expected folder path %v, got %#v", names, path)
		}
	}
}
