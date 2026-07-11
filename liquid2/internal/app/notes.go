package app

import (
	"context"
	"log/slog"
	"strings"
)

// CreateNoteInput contains note fields accepted from API requests.
type CreateNoteInput struct {
	// Body is the note text.
	Body string
	// Format identifies the note body format.
	Format string
}

func (s *Service) ListDocumentNotes(ctx context.Context, documentID string) (NoteList, error) {
	return withView(ctx, s, func(tx RepositoryReader) (NoteList, error) {
		if _, ok := tx.Document(documentID); !ok {
			return NoteList{}, notFound("document")
		}
		items := []DocumentNote{}
		for _, note := range tx.DocumentNotes(documentID) {
			if note.DeletedAt == nil {
				items = append(items, cloneNote(note))
			}
		}
		return NoteList{Items: items}, nil
	})
}

func (s *Service) CreateDocumentNote(ctx context.Context, documentID string, input CreateNoteInput) (DocumentNote, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentNote, error) {
		doc, ok := tx.Document(documentID)
		if !ok {
			return DocumentNote{}, notFound("document")
		}
		if doc.meta.DeletedAt != nil {
			return DocumentNote{}, notFound("document")
		}
		if err := validateNoteInput(input); err != nil {
			return DocumentNote{}, err
		}
		now := tx.Now()
		note := DocumentNote{
			ID: tx.NextID("note"), DocumentID: documentID, Body: strings.TrimSpace(input.Body),
			Format: input.Format, CreatedAt: now, UpdatedAt: now,
		}
		tx.PutNote(note)
		s.logger.DebugContext(ctx, "document note created", slog.String("operation", "document_note_create"), slog.String("document_id", documentID), slog.String("note_id", note.ID))
		return cloneNote(note), nil
	})
}

func (s *Service) UpdateDocumentNote(ctx context.Context, documentID string, noteID string, input CreateNoteInput) (DocumentNote, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentNote, error) {
		note, err := requireNote(tx, documentID, noteID)
		if err != nil {
			return DocumentNote{}, err
		}
		if err := validateNoteInput(input); err != nil {
			return DocumentNote{}, err
		}
		note.Body = strings.TrimSpace(input.Body)
		note.Format = input.Format
		note.UpdatedAt = tx.Now()
		tx.PutNote(note)
		s.logger.DebugContext(ctx, "document note updated", slog.String("operation", "document_note_update"), slog.String("document_id", documentID), slog.String("note_id", noteID))
		return cloneNote(note), nil
	})
}

func (s *Service) DeleteDocumentNote(ctx context.Context, documentID string, noteID string) (int64, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (int64, error) {
		note, err := requireNote(tx, documentID, noteID)
		if err != nil {
			return 0, err
		}
		now := tx.Now()
		note.DeletedAt = &now
		note.UpdatedAt = now
		tx.PutNote(note)
		s.logger.DebugContext(ctx, "document note deleted", slog.String("operation", "document_note_delete"), slog.String("document_id", documentID), slog.String("note_id", noteID))
		return now, nil
	})
}

func requireNote(tx RepositoryReader, documentID string, noteID string) (DocumentNote, error) {
	if _, ok := tx.Document(documentID); !ok {
		return DocumentNote{}, notFound("document")
	}
	note, ok := tx.Note(documentID, noteID)
	if !ok || note.DeletedAt != nil {
		return DocumentNote{}, notFound("note")
	}
	return note, nil
}

func validateNoteInput(input CreateNoteInput) error {
	if strings.TrimSpace(input.Body) == "" {
		return validation("note body is required")
	}
	if input.Format != "text" && input.Format != "markdown" {
		return validation("note format must be text or markdown")
	}
	return nil
}

func cloneNote(note DocumentNote) DocumentNote {
	note.DeletedAt = cloneInt64(note.DeletedAt)
	return note
}
