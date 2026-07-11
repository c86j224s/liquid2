package app

import sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"

func (tx *sqliteTx) DocumentNotes(documentID string) []DocumentNote {
	rows, err := tx.q.ListDocumentNotesAll(tx.ctx, documentID)
	tx.abort(err)
	notes := make([]DocumentNote, 0, len(rows))
	for _, row := range rows {
		notes = append(notes, sqliteNote(row))
	}
	return notes
}

func (tx *sqliteTx) Note(documentID string, noteID string) (DocumentNote, bool) {
	row, err := tx.q.GetDocumentNote(tx.ctx, sqlitedb.GetDocumentNoteParams{
		DocumentID: documentID, ID: noteID,
	})
	if tx.missing(err) {
		return DocumentNote{}, false
	}
	return sqliteNote(row), true
}

func (tx *sqliteTx) PutNote(note DocumentNote) {
	_, err := tx.q.UpsertDocumentNote(tx.ctx, sqlitedb.UpsertDocumentNoteParams{
		ID: note.ID, DocumentID: note.DocumentID, Body: note.Body, Format: note.Format,
		CreatedAt: note.CreatedAt, UpdatedAt: note.UpdatedAt, DeletedAt: sqliteNullInt64(note.DeletedAt),
	})
	tx.abort(err)
}
