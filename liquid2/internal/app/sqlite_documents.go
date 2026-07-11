package app

import (
	"crypto/sha256"
	"fmt"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func (tx *sqliteTx) Document(id string) (documentRecord, bool) {
	row, err := tx.q.GetDocument(tx.ctx, id)
	if tx.missing(err) {
		return documentRecord{}, false
	}
	return tx.documentRecord(row), true
}

func (tx *sqliteTx) Documents() []documentRecord {
	rows, err := tx.q.ListDocuments(tx.ctx)
	tx.abort(err)
	records := make([]documentRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, tx.documentRecord(row))
	}
	return records
}

func (tx *sqliteTx) PutDocument(record documentRecord) {
	_, err := tx.q.UpsertDocument(tx.ctx, sqliteDocumentParams(record))
	tx.abort(err)
	tx.replaceDocumentContents(record)
	tx.replaceDocumentBlobs(record)
	tx.replaceDocumentTags(record)
}

func (tx *sqliteTx) documentRecord(row sqlitedb.Document) documentRecord {
	blobs, blobData := tx.documentBlobs(row.ID)
	return documentRecord{
		meta:     sqliteDocumentMeta(row),
		contents: tx.documentContents(row.ID),
		blobs:    blobs,
		blobData: blobData,
		tagIDs:   tx.documentTagIDs(row.ID),
	}
}

func (tx *sqliteTx) documentContents(documentID string) []DocumentContent {
	rows, err := tx.q.ListDocumentContents(tx.ctx, documentID)
	tx.abort(err)
	if len(rows) == 0 {
		return nil
	}
	contents := make([]DocumentContent, 0, len(rows))
	for _, row := range rows {
		contents = append(contents, DocumentContent{
			ID: row.ID, Role: row.Role, Format: row.Format,
			Language: sqliteStringPtr(row.Language), Content: row.Content,
			SourceContentID: sqliteStringPtr(row.SourceContentID),
		})
	}
	return contents
}

func (tx *sqliteTx) replaceDocumentContents(record documentRecord) {
	createdAtByID := tx.contentCreatedAtByID(record.meta.ID)
	tx.abort(tx.q.DeleteDocumentContents(tx.ctx, record.meta.ID))
	for _, content := range record.contents {
		createdAt := record.meta.UpdatedAt
		if existing, ok := createdAtByID[content.ID]; ok {
			createdAt = existing
		}
		_, err := tx.q.CreateDocumentContent(tx.ctx, sqlitedb.CreateDocumentContentParams{
			ID: content.ID, DocumentID: record.meta.ID, Role: content.Role, Format: content.Format,
			Language: sqliteNullString(content.Language), Content: content.Content,
			SourceContentID: sqliteNullString(content.SourceContentID), CreatedAt: createdAt,
		})
		tx.abort(err)
	}
}

func (tx *sqliteTx) contentCreatedAtByID(documentID string) map[string]int64 {
	rows, err := tx.q.ListDocumentContents(tx.ctx, documentID)
	tx.abort(err)
	createdAtByID := map[string]int64{}
	for _, row := range rows {
		createdAtByID[row.ID] = row.CreatedAt
	}
	return createdAtByID
}

func (tx *sqliteTx) documentBlobs(documentID string) ([]BlobMetadata, map[string][]byte) {
	rows, err := tx.q.ListDocumentBlobs(tx.ctx, documentID)
	tx.abort(err)
	blobs := make([]BlobMetadata, 0, len(rows))
	data := map[string][]byte{}
	for _, row := range rows {
		blobs = append(blobs, BlobMetadata{
			ID: row.ID, Filename: row.Filename, MimeType: row.MimeType,
			Size: row.Size, CreatedAt: row.CreatedAt,
		})
		data[row.ID] = append([]byte(nil), row.Data...)
	}
	return blobs, data
}

func (tx *sqliteTx) replaceDocumentBlobs(record documentRecord) {
	tx.abort(tx.q.DeleteDocumentBlobs(tx.ctx, record.meta.ID))
	for _, blob := range record.blobs {
		data := append([]byte(nil), record.blobData[blob.ID]...)
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		_, err := tx.q.CreateBlob(tx.ctx, sqlitedb.CreateBlobParams{
			ID: blob.ID, DocumentID: record.meta.ID, Filename: blob.Filename,
			MimeType: blob.MimeType, Size: blob.Size, Sha256: hash,
			Data: data, CreatedAt: blob.CreatedAt,
		})
		tx.abort(err)
	}
}

func (tx *sqliteTx) documentTagIDs(documentID string) []string {
	tags, err := tx.q.ListDocumentTags(tx.ctx, documentID)
	tx.abort(err)
	ids := make([]string, 0, len(tags))
	for _, tag := range tags {
		ids = append(ids, tag.ID)
	}
	return ids
}

func (tx *sqliteTx) replaceDocumentTags(record documentRecord) {
	tx.abort(tx.q.DeleteDocumentTags(tx.ctx, record.meta.ID))
	seen := map[string]struct{}{}
	for _, tagID := range record.tagIDs {
		if _, ok := seen[tagID]; ok {
			continue
		}
		seen[tagID] = struct{}{}
		tx.abort(tx.q.AssignDocumentTag(tx.ctx, sqlitedb.AssignDocumentTagParams{
			DocumentID: record.meta.ID, TagID: tagID,
		}))
	}
}
