package app

// DocumentNote is a user-authored note attached to a document.
type DocumentNote struct {
	// ID is the stable note identifier.
	ID string `json:"id"`
	// DocumentID points to the owning document.
	DocumentID string `json:"documentId"`
	// Body is the note text.
	Body string `json:"body"`
	// Format identifies the note body format.
	Format string `json:"format"`
	// CreatedAt is the creation timestamp in Unix milliseconds.
	CreatedAt int64 `json:"createdAt"`
	// UpdatedAt is the last update timestamp in Unix milliseconds.
	UpdatedAt int64 `json:"updatedAt"`
	// DeletedAt is set when the note has been soft-deleted.
	DeletedAt *int64 `json:"deletedAt"`
}

// NoteList is the list response for document notes.
type NoteList struct {
	// Items contains the notes for the document.
	Items []DocumentNote `json:"items"`
}
