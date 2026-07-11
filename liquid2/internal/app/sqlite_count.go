package app

import sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"

func (reader sqliteReader) CountDocuments(filters DocumentFilters) (int, error) {
	return reader.tx.CountDocuments(filters)
}

func (tx *sqliteTx) CountDocuments(filters DocumentFilters) (int, error) {
	filters, err := normalizeDocumentFilters(filters)
	if err != nil {
		return 0, err
	}
	params := sqliteDocumentSearchParamsFor(filters)
	var count int64
	if filters.Query != "" {
		query, err := sqliteFTSQuery(filters.Query)
		if err != nil {
			return 0, err
		}
		count, err = tx.q.CountSearchDocumentIDs(tx.ctx, params.searchCount(query))
	} else {
		count, err = tx.q.CountDocumentIDs(tx.ctx, params.count())
	}
	tx.abort(err)
	return int(count), err
}

func (params sqliteDocumentSearchParams) count() sqlitedb.CountDocumentIDsParams {
	return sqlitedb.CountDocumentIDsParams{
		IncludeDeleted:           params.includeDeleted,
		IncludeTrash:             params.includeTrash,
		Status:                   params.status,
		Kind:                     params.kind,
		FolderID:                 params.folderID,
		RatingMin:                params.ratingMin,
		Tag:                      params.tag,
		IncludeFolderDescendants: params.includeFolderDescendants,
	}
}

func (params sqliteDocumentSearchParams) searchCount(query string) sqlitedb.CountSearchDocumentIDsParams {
	return sqlitedb.CountSearchDocumentIDsParams{
		Query:                    query,
		IncludeDeleted:           params.includeDeleted,
		IncludeTrash:             params.includeTrash,
		Status:                   params.status,
		Kind:                     params.kind,
		FolderID:                 params.folderID,
		RatingMin:                params.ratingMin,
		Tag:                      params.tag,
		IncludeFolderDescendants: params.includeFolderDescendants,
	}
}
