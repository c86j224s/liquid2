package app

import (
	"database/sql"
	"strings"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

type sqliteDocumentSearchParams struct {
	includeDeleted           int64
	includeTrash             int64
	status                   sql.NullString
	kind                     sql.NullString
	folderID                 sql.NullString
	ratingMin                int64
	tag                      sql.NullString
	cursorID                 sql.NullString
	includeFolderDescendants int64
	limitRows                int64
}

func (reader sqliteReader) ListDocumentIDs(filters DocumentFilters) ([]string, *string, error) {
	return reader.tx.ListDocumentIDs(filters)
}

func (tx *sqliteTx) ListDocumentIDs(filters DocumentFilters) ([]string, *string, error) {
	filters, err := normalizeDocumentFilters(filters)
	if err != nil {
		return nil, nil, err
	}
	params := sqliteDocumentSearchParamsFor(filters)
	if filters.Cursor != "" {
		cursorID, err := decodeDocumentCursor(filters.Cursor)
		if err != nil {
			return nil, nil, err
		}
		params.cursorID = sql.NullString{String: cursorID, Valid: true}
	}
	var ids []string
	if filters.Query != "" {
		query, err := sqliteFTSQuery(filters.Query)
		if err != nil {
			return nil, nil, err
		}
		ids, err = tx.searchDocumentIDs(params, query, filters.Sort)
	} else {
		ids, err = tx.listDocumentIDs(params, filters.Sort)
	}
	tx.abort(err)
	page, nextCursor := sliceDocumentIDPage(ids, filters.Limit)
	return page, nextCursor, nil
}

func (tx *sqliteTx) listDocumentIDs(params sqliteDocumentSearchParams, sortValue string) ([]string, error) {
	switch sortValue {
	case DocumentSortCreatedDesc:
		return tx.q.ListDocumentIDsCreatedDesc(tx.ctx, params.createdDesc())
	case DocumentSortRatingDesc:
		return tx.q.ListDocumentIDsRatingDesc(tx.ctx, params.ratingDesc())
	default:
		return tx.q.ListDocumentIDsRecent(tx.ctx, params.recent())
	}
}

func (tx *sqliteTx) searchDocumentIDs(
	params sqliteDocumentSearchParams,
	query string,
	sortValue string,
) ([]string, error) {
	switch sortValue {
	case DocumentSortCreatedDesc:
		return tx.q.SearchDocumentIDsCreatedDesc(tx.ctx, params.searchCreatedDesc(query))
	case DocumentSortRatingDesc:
		return tx.q.SearchDocumentIDsRatingDesc(tx.ctx, params.searchRatingDesc(query))
	case DocumentSortRecent:
		return tx.q.SearchDocumentIDsRecent(tx.ctx, params.searchRecent(query))
	default:
		return tx.q.SearchDocumentIDsRelevance(tx.ctx, params.relevance(query))
	}
}

func sqliteDocumentSearchParamsFor(filters DocumentFilters) sqliteDocumentSearchParams {
	return sqliteDocumentSearchParams{
		includeDeleted:           sqliteBool(filters.IncludeDeleted),
		includeTrash:             sqliteBool(filters.IncludeTrash),
		status:                   sqliteOptionalString(filters.Status),
		kind:                     sqliteOptionalString(filters.Kind),
		folderID:                 sqliteOptionalString(filters.FolderID),
		ratingMin:                int64(filters.RatingMin),
		tag:                      sqliteOptionalString(filters.Tag),
		includeFolderDescendants: sqliteBool(filters.IncludeFolderDescendants),
		limitRows:                int64(filters.Limit + 1),
	}
}

func (params sqliteDocumentSearchParams) recent() sqlitedb.ListDocumentIDsRecentParams {
	return sqlitedb.ListDocumentIDsRecentParams{
		IncludeDeleted:           params.includeDeleted,
		IncludeTrash:             params.includeTrash,
		Status:                   params.status,
		Kind:                     params.kind,
		FolderID:                 params.folderID,
		RatingMin:                params.ratingMin,
		Tag:                      params.tag,
		CursorID:                 params.cursorID,
		IncludeFolderDescendants: params.includeFolderDescendants,
		LimitRows:                params.limitRows,
	}
}

func (params sqliteDocumentSearchParams) createdDesc() sqlitedb.ListDocumentIDsCreatedDescParams {
	return sqlitedb.ListDocumentIDsCreatedDescParams{
		IncludeDeleted:           params.includeDeleted,
		IncludeTrash:             params.includeTrash,
		Status:                   params.status,
		Kind:                     params.kind,
		FolderID:                 params.folderID,
		RatingMin:                params.ratingMin,
		Tag:                      params.tag,
		CursorID:                 params.cursorID,
		IncludeFolderDescendants: params.includeFolderDescendants,
		LimitRows:                params.limitRows,
	}
}

func (params sqliteDocumentSearchParams) ratingDesc() sqlitedb.ListDocumentIDsRatingDescParams {
	return sqlitedb.ListDocumentIDsRatingDescParams{
		IncludeDeleted:           params.includeDeleted,
		IncludeTrash:             params.includeTrash,
		Status:                   params.status,
		Kind:                     params.kind,
		FolderID:                 params.folderID,
		RatingMin:                params.ratingMin,
		Tag:                      params.tag,
		CursorID:                 params.cursorID,
		IncludeFolderDescendants: params.includeFolderDescendants,
		LimitRows:                params.limitRows,
	}
}

func (params sqliteDocumentSearchParams) relevance(query string) sqlitedb.SearchDocumentIDsRelevanceParams {
	return sqlitedb.SearchDocumentIDsRelevanceParams{
		Query:                    query,
		IncludeDeleted:           params.includeDeleted,
		IncludeTrash:             params.includeTrash,
		Status:                   params.status,
		Kind:                     params.kind,
		FolderID:                 params.folderID,
		RatingMin:                params.ratingMin,
		Tag:                      params.tag,
		CursorID:                 params.cursorID,
		IncludeFolderDescendants: params.includeFolderDescendants,
		LimitRows:                params.limitRows,
	}
}

func (params sqliteDocumentSearchParams) searchRecent(query string) sqlitedb.SearchDocumentIDsRecentParams {
	return sqlitedb.SearchDocumentIDsRecentParams(params.relevance(query))
}

func (params sqliteDocumentSearchParams) searchCreatedDesc(query string) sqlitedb.SearchDocumentIDsCreatedDescParams {
	return sqlitedb.SearchDocumentIDsCreatedDescParams(params.relevance(query))
}

func (params sqliteDocumentSearchParams) searchRatingDesc(query string) sqlitedb.SearchDocumentIDsRatingDescParams {
	return sqlitedb.SearchDocumentIDsRatingDescParams(params.relevance(query))
}

func sqliteFTSQuery(query string) (string, error) {
	tokens := queryTokens(query)
	if len(tokens) == 0 {
		return "", validation("query is required")
	}
	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		parts = append(parts, `"`+strings.ReplaceAll(token, `"`, `""`)+`"`)
	}
	return strings.Join(parts, " AND "), nil
}

func sqliteOptionalString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
