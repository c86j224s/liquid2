package sqlite

import (
	"testing"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func TestDocumentNotes(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()
	createTestDocument(t, ctx, q, "doc_1")

	note, err := q.CreateDocumentNote(ctx, sqlitedb.CreateDocumentNoteParams{
		ID:         "note_1",
		DocumentID: "doc_1",
		Body:       "Remember this",
		Format:     "text",
		CreatedAt:  1000,
		UpdatedAt:  1000,
	})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if note.Body != "Remember this" {
		t.Fatalf("unexpected note body %q", note.Body)
	}

	notes, err := q.ListDocumentNotes(ctx, "doc_1")
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}

	if _, err := q.SoftDeleteDocumentNote(ctx, sqlitedb.SoftDeleteDocumentNoteParams{
		ID:         "note_1",
		DocumentID: "doc_1",
		DeletedAt:  nullInt(2000),
		UpdatedAt:  2000,
	}); err != nil {
		t.Fatalf("soft delete note: %v", err)
	}
	notes, err = q.ListDocumentNotes(ctx, "doc_1")
	if err != nil {
		t.Fatalf("list notes after delete: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("expected deleted note to be hidden, got %d", len(notes))
	}
}
