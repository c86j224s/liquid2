package app

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteRepositoryDeletesUnreferencedRemovedTag(t *testing.T) {
	ctx := context.Background()
	service, closeService := newSQLiteService(t, ctx, filepath.Join(t.TempDir(), "liquid2.db"))
	defer closeService()
	doc, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Tagged"})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	tag, err := service.CreateTag(ctx, "Ephemeral")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if _, err = service.ReplaceDocumentTags(ctx, doc.Document.ID, []string{tag.ID}); err != nil {
		t.Fatalf("assign tag: %v", err)
	}

	if _, err = service.ReplaceDocumentTags(ctx, doc.Document.ID, []string{}); err != nil {
		t.Fatalf("clear tags: %v", err)
	}
	assertTagMissing(t, ctx, service, tag.ID)
}
