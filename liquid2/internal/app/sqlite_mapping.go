package app

import (
	"database/sql"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func sqliteDocumentParams(record documentRecord) sqlitedb.UpsertDocumentParams {
	meta := record.meta
	return sqlitedb.UpsertDocumentParams{
		ID: meta.ID, Title: meta.Title, Kind: meta.Kind,
		FolderID: sqliteNullString(meta.FolderID), CanonicalUrl: sqliteNullString(meta.CanonicalURL),
		SourceUrl: sqliteNullString(meta.SourceURL), Language: sqliteNullString(meta.Language),
		Status: meta.Status, Rating: sqliteNullInt(meta.Rating), CreatedAt: meta.CreatedAt,
		UpdatedAt: meta.UpdatedAt, ReadAt: sqliteNullInt64(meta.ReadAt), DeletedAt: sqliteNullInt64(meta.DeletedAt),
	}
}

func sqliteDocumentMeta(row sqlitedb.Document) DocumentMetadata {
	return DocumentMetadata{
		ID: row.ID, Title: row.Title, Kind: row.Kind, FolderID: sqliteStringPtr(row.FolderID),
		CanonicalURL: sqliteStringPtr(row.CanonicalUrl), SourceURL: sqliteStringPtr(row.SourceUrl),
		Language: sqliteStringPtr(row.Language), Status: row.Status, Rating: sqliteIntPtr(row.Rating),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, ReadAt: sqliteInt64Ptr(row.ReadAt),
		DeletedAt: sqliteInt64Ptr(row.DeletedAt),
	}
}

func sqliteFolder(row sqlitedb.Folder) Folder {
	return Folder{
		ID: row.ID, ParentID: sqliteStringPtr(row.ParentID), Name: row.Name,
		SystemRole: sqliteStringPtr(row.SystemRole), SortOrder: int(row.SortOrder),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		Children: []Folder{},
	}
}

func sqliteTag(row sqlitedb.Tag) Tag {
	return Tag{ID: row.ID, Name: row.Name, Slug: row.Slug, CreatedAt: row.CreatedAt}
}

func sqliteFeed(row sqlitedb.Feed) Feed {
	return Feed{
		ID: row.ID, URL: row.Url, Title: sqliteStringPtr(row.Title),
		FolderID: sqliteStringPtr(row.FolderID), Enabled: row.Enabled == 1,
		LastCheckedAt: sqliteInt64Ptr(row.LastCheckedAt), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func sqliteFeedItem(row sqlitedb.FeedItem) FeedItem {
	return FeedItem{
		ID: row.ID, FeedID: row.FeedID, DocumentID: row.DocumentID,
		GUID: sqliteStringPtr(row.Guid), URL: row.Url, CanonicalURL: sqliteStringPtr(row.CanonicalUrl),
		ContentHash: sqliteStringPtr(row.ContentHash), PublishedAt: sqliteInt64Ptr(row.PublishedAt),
		CreatedAt: row.CreatedAt,
	}
}

func sqliteJob(row sqlitedb.Job) Job {
	return Job{
		ID: row.ID, Kind: row.Kind, Status: row.Status, PayloadJSON: row.PayloadJson,
		Error: sqliteStringPtr(row.Error), Attempts: row.Attempts, CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt, StartedAt: sqliteInt64Ptr(row.StartedAt),
		FinishedAt: sqliteInt64Ptr(row.FinishedAt),
	}
}

func sqliteNote(row sqlitedb.DocumentNote) DocumentNote {
	return DocumentNote{
		ID: row.ID, DocumentID: row.DocumentID, Body: row.Body, Format: row.Format,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, DeletedAt: sqliteInt64Ptr(row.DeletedAt),
	}
}

func sqliteBool(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func sqliteNullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func sqliteStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	cloned := value.String
	return &cloned
}

func sqliteNullInt(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func sqliteIntPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	cloned := int(value.Int64)
	return &cloned
}

func sqliteNullInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func sqliteInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	cloned := value.Int64
	return &cloned
}
