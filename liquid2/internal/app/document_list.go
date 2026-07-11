package app

import "context"

func (s *Service) ListDocuments(ctx context.Context, filters DocumentFilters) (DocumentList, error) {
	filters, err := normalizeDocumentFilters(filters)
	if err != nil {
		return DocumentList{}, err
	}
	return withView(ctx, s, func(tx RepositoryReader) (DocumentList, error) {
		totalCount := -1
		if filters.Cursor == "" {
			count, err := tx.CountDocuments(filters)
			if err != nil {
				return DocumentList{}, err
			}
			totalCount = count
		}
		ids, nextCursor, err := tx.ListDocumentIDs(filters)
		if err != nil {
			return DocumentList{}, err
		}
		items := make([]DocumentSummary, 0, len(ids))
		for _, id := range ids {
			if doc, ok := tx.Document(id); ok {
				items = append(items, documentSummary(tx, doc))
			}
		}
		return DocumentList{Items: items, NextCursor: nextCursor, TotalCount: totalCount}, nil
	})
}
