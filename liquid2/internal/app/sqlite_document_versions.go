package app

import (
	"encoding/json"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func (tx *sqliteTx) DocumentVersions(documentID string) []DocumentVersion {
	rows, err := tx.q.ListDocumentVersions(tx.ctx, documentID)
	tx.abort(err)
	versions := make([]DocumentVersion, 0, len(rows))
	for _, row := range rows {
		versions = append(versions, tx.documentVersion(row))
	}
	return versions
}

func (tx *sqliteTx) PutDocumentVersion(version DocumentVersion) {
	_, err := tx.q.CreateDocumentVersion(tx.ctx, tx.documentVersionParams(version))
	tx.abort(err)
}

func (tx *sqliteTx) documentVersion(row sqlitedb.DocumentVersion) DocumentVersion {
	contents := tx.decodeDocumentVersionContents(row.ContentSnapshotJson)
	var metadata DocumentMetadata
	tx.abort(json.Unmarshal([]byte(row.MetadataSnapshotJson), &metadata))
	return DocumentVersion{
		ID: row.ID, DocumentID: row.DocumentID, Sequence: row.Sequence,
		MutationKind: row.MutationKind, Title: row.Title,
		Contents: contents, Metadata: metadata, CreatedAt: row.CreatedAt,
	}
}

func (tx *sqliteTx) documentVersionParams(version DocumentVersion) sqlitedb.CreateDocumentVersionParams {
	contents := tx.encodeDocumentVersionContents(version.Contents)
	metadata, err := json.Marshal(version.Metadata)
	tx.abort(err)
	return sqlitedb.CreateDocumentVersionParams{
		ID: version.ID, DocumentID: version.DocumentID, Sequence: version.Sequence,
		MutationKind: version.MutationKind, Title: version.Title,
		ContentSnapshotJson: contents, MetadataSnapshotJson: string(metadata),
		CreatedAt: version.CreatedAt,
	}
}

type documentVersionContentSnapshot struct {
	ID              string  `json:"id"`
	Role            string  `json:"role"`
	Format          string  `json:"format"`
	Language        *string `json:"language"`
	Content         string  `json:"content"`
	SourceContentID *string `json:"sourceContentId"`
}

func (tx *sqliteTx) encodeDocumentVersionContents(contents []DocumentContent) string {
	if contents == nil {
		return "null"
	}
	snapshot := make([]documentVersionContentSnapshot, len(contents))
	for i, content := range contents {
		snapshot[i] = documentVersionContentSnapshot{
			ID: content.ID, Role: content.Role, Format: content.Format,
			Language: cloneString(content.Language), Content: content.Content,
			SourceContentID: cloneString(content.SourceContentID),
		}
	}
	data, err := json.Marshal(snapshot)
	tx.abort(err)
	return string(data)
}

func (tx *sqliteTx) decodeDocumentVersionContents(value string) []DocumentContent {
	var snapshot []documentVersionContentSnapshot
	tx.abort(json.Unmarshal([]byte(value), &snapshot))
	if snapshot == nil {
		return nil
	}
	contents := make([]DocumentContent, len(snapshot))
	for i, content := range snapshot {
		contents[i] = DocumentContent{
			ID: content.ID, Role: content.Role, Format: content.Format,
			Language: cloneString(content.Language), Content: content.Content,
			SourceContentID: cloneString(content.SourceContentID),
		}
	}
	return contents
}
