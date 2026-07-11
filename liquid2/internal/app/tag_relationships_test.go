package app

import (
	"context"
	"testing"
)

func TestReplaceDocumentTagsNormalizesIDs(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	doc, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Tagged"})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	first, err := service.CreateTag(ctx, "Zulu")
	if err != nil {
		t.Fatalf("create first tag: %v", err)
	}
	second, err := service.CreateTag(ctx, "Alpha")
	if err != nil {
		t.Fatalf("create second tag: %v", err)
	}

	detail, err := service.ReplaceDocumentTags(ctx, doc.Document.ID, []string{" " + first.ID + " ", "", first.ID, second.ID})
	if err != nil {
		t.Fatalf("replace tags: %v", err)
	}
	if len(detail.Tags) != 2 || detail.Tags[0].ID != second.ID || detail.Tags[1].ID != first.ID {
		t.Fatalf("unexpected normalized tags %#v", detail.Tags)
	}
}

func TestReplaceDocumentTagsDeletesUnreferencedRemovedTag(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
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

	detail, err := service.ReplaceDocumentTags(ctx, doc.Document.ID, []string{})
	if err != nil {
		t.Fatalf("clear tags: %v", err)
	}
	if len(detail.Tags) != 0 {
		t.Fatalf("expected no document tags, got %#v", detail.Tags)
	}
	assertTagMissing(t, ctx, service, tag.ID)
}

func TestReplaceDocumentTagsKeepsRemovedTagUsedByAnotherDocument(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	first, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "First"})
	if err != nil {
		t.Fatalf("create first document: %v", err)
	}
	second, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Second"})
	if err != nil {
		t.Fatalf("create second document: %v", err)
	}
	tag, err := service.CreateTag(ctx, "Shared")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if _, err = service.ReplaceDocumentTags(ctx, first.Document.ID, []string{tag.ID}); err != nil {
		t.Fatalf("assign first tag: %v", err)
	}
	if _, err = service.ReplaceDocumentTags(ctx, second.Document.ID, []string{tag.ID}); err != nil {
		t.Fatalf("assign second tag: %v", err)
	}

	if _, err = service.ReplaceDocumentTags(ctx, first.Document.ID, []string{}); err != nil {
		t.Fatalf("clear first tags: %v", err)
	}
	assertTagPresent(t, ctx, service, tag.ID)
}

func TestReplaceDocumentTagsDoesNotDeleteNeverAssignedTags(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	doc, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Tagged"})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	assigned, err := service.CreateTag(ctx, "Assigned")
	if err != nil {
		t.Fatalf("create assigned tag: %v", err)
	}
	unused, err := service.CreateTag(ctx, "Unused")
	if err != nil {
		t.Fatalf("create unused tag: %v", err)
	}
	if _, err = service.ReplaceDocumentTags(ctx, doc.Document.ID, []string{assigned.ID}); err != nil {
		t.Fatalf("assign tag: %v", err)
	}

	if _, err = service.ReplaceDocumentTags(ctx, doc.Document.ID, []string{}); err != nil {
		t.Fatalf("clear tags: %v", err)
	}
	assertTagMissing(t, ctx, service, assigned.ID)
	assertTagPresent(t, ctx, service, unused.ID)
}

func assertTagPresent(t *testing.T, ctx context.Context, service *Service, tagID string) {
	t.Helper()
	err := service.repo.View(ctx, func(tx RepositoryReader) error {
		if _, ok := tx.Tag(tagID); !ok {
			t.Fatalf("expected tag %q to remain", tagID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view tag: %v", err)
	}
}

func assertTagMissing(t *testing.T, ctx context.Context, service *Service, tagID string) {
	t.Helper()
	err := service.repo.View(ctx, func(tx RepositoryReader) error {
		if _, ok := tx.Tag(tagID); ok {
			t.Fatalf("expected tag %q to be deleted", tagID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view tag: %v", err)
	}
}
