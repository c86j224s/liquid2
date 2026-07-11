package app

import "context"

func (s *Service) ListDocumentVersions(ctx context.Context, documentID string) ([]DocumentVersion, error) {
	return withView(ctx, s, func(tx RepositoryReader) ([]DocumentVersion, error) {
		if _, ok := tx.Document(documentID); !ok {
			return nil, notFound("document")
		}
		return cloneDocumentVersions(tx.DocumentVersions(documentID)), nil
	})
}

func recordDocumentVersion(tx RepositoryTx, doc documentRecord, mutationKind string, now int64) {
	versions := tx.DocumentVersions(doc.meta.ID)
	tx.PutDocumentVersion(DocumentVersion{
		ID:           tx.NextID("docver"),
		DocumentID:   doc.meta.ID,
		Sequence:     int64(len(versions) + 1),
		MutationKind: mutationKind,
		Title:        doc.meta.Title,
		Contents:     cloneDocumentContents(doc.contents),
		Metadata:     cloneDocumentMetadata(doc.meta),
		CreatedAt:    now,
	})
}
