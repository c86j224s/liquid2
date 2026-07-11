package app

import (
	"context"
	"testing"
)

func TestMoveDocumentToTrashIsIdempotent(t *testing.T) {
	now := int64(1760000000000)
	service := NewService(WithClock(func() int64 { return now }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	detail, err := service.CreateDocument(ctx, CreateDocumentInput{Title: "Discard"})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	now += 1000
	detail, err = service.MoveDocumentToTrash(ctx, detail.Document.ID)
	if err != nil {
		t.Fatalf("move to trash: %v", err)
	}
	updatedAt := detail.Document.UpdatedAt
	now += 1000
	detail, err = service.MoveDocumentToTrash(ctx, detail.Document.ID)
	if err != nil {
		t.Fatalf("move to trash again: %v", err)
	}
	if detail.Document.UpdatedAt != updatedAt {
		t.Fatalf("expected updated_at to stay %d, got %d", updatedAt, detail.Document.UpdatedAt)
	}
}
