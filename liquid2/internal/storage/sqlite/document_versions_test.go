package sqlite

import (
	"testing"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func TestDocumentVersionQueries(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()
	createTestDocument(t, ctx, q, "doc_1")

	_, err := q.CreateDocumentVersion(ctx, sqlitedb.CreateDocumentVersionParams{
		ID: "ver_1", DocumentID: "doc_1", Sequence: 1, MutationKind: "title",
		Title: "Original", ContentSnapshotJson: "[]",
		MetadataSnapshotJson: `{"id":"doc_1","title":"Original"}`, CreatedAt: 2000,
	})
	if err != nil {
		t.Fatalf("create document version: %v", err)
	}
	_, err = q.CreateDocumentVersion(ctx, sqlitedb.CreateDocumentVersionParams{
		ID: "ver_2", DocumentID: "doc_1", Sequence: 2, MutationKind: "content",
		Title: "Renamed", ContentSnapshotJson: "[]",
		MetadataSnapshotJson: `{"id":"doc_1","title":"Renamed"}`, CreatedAt: 3000,
	})
	if err != nil {
		t.Fatalf("create second document version: %v", err)
	}

	versions, err := q.ListDocumentVersions(ctx, "doc_1")
	if err != nil {
		t.Fatalf("list document versions: %v", err)
	}
	if len(versions) != 2 || versions[0].Sequence != 1 || versions[1].Sequence != 2 {
		t.Fatalf("expected versions ordered by sequence, got %#v", versions)
	}
}

func TestDocumentVersionConstraints(t *testing.T) {
	store, ctx := newTestStore(t)
	q := store.Queries()
	createTestDocument(t, ctx, q, "doc_1")

	_, err := q.CreateDocumentVersion(ctx, sqlitedb.CreateDocumentVersionParams{
		ID: "ver_bad", DocumentID: "doc_1", Sequence: 1, MutationKind: "rating",
		Title: "Original", ContentSnapshotJson: "[]",
		MetadataSnapshotJson: `{"id":"doc_1","title":"Original"}`, CreatedAt: 2000,
	})
	if err == nil {
		t.Fatal("expected invalid mutation kind constraint error")
	}
}
