package httptransport

import (
	"context"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/danielgtaylor/huma/v2"
)

// noteIDInput carries document and note path parameters.
type noteIDInput struct {
	// ID is the target document ID.
	ID string `path:"id" doc:"Document ID"`
	// NoteID is the target note ID.
	NoteID string `path:"noteId" doc:"Note ID"`
}

// noteListInput carries the document path parameter for note listing.
type noteListInput struct {
	// ID is the target document ID.
	ID string `path:"id" doc:"Document ID"`
}

// noteBodyInput carries a note create request.
type noteBodyInput struct {
	// ID is the target document ID.
	ID string `path:"id" doc:"Document ID"`
	// Body contains note fields.
	Body struct {
		// Body is the note text.
		Body string `json:"body" minLength:"1" maxLength:"10000"`
		// Format identifies the note body format.
		Format string `json:"format" enum:"text,markdown"`
	}
}

// updateNoteInput carries a note update request.
type updateNoteInput struct {
	// ID is the target document ID.
	ID string `path:"id" doc:"Document ID"`
	// NoteID is the target note ID.
	NoteID string `path:"noteId" doc:"Note ID"`
	// Body contains mutable note fields.
	Body struct {
		// Body is the note text.
		Body string `json:"body" minLength:"1" maxLength:"10000"`
		// Format identifies the note body format.
		Format string `json:"format" enum:"text,markdown"`
	}
}

// noteListOutput returns notes for a document.
type noteListOutput struct {
	// Body is the note list response body.
	Body app.NoteList
}

// noteOutput returns one note.
type noteOutput struct {
	// Body is the note response body.
	Body app.DocumentNote
}

func registerNoteRoutes(api huma.API, service *app.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-document-notes", Method: http.MethodGet, Path: "/api/v1/documents/{id}/notes",
		Summary: "List document notes", Tags: []string{"Document Notes"}, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *noteListInput) (*noteListOutput, error) {
		notes, err := service.ListDocumentNotes(ctx, input.ID)
		return &noteListOutput{Body: notes}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-document-note", Method: http.MethodPost, Path: "/api/v1/documents/{id}/notes",
		Summary: "Create document note", Tags: []string{"Document Notes"},
		DefaultStatus: http.StatusCreated, Errors: []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *noteBodyInput) (*noteOutput, error) {
		note, err := service.CreateDocumentNote(ctx, input.ID, app.CreateNoteInput{
			Body: input.Body.Body, Format: input.Body.Format,
		})
		return &noteOutput{Body: note}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-document-note", Method: http.MethodPatch, Path: "/api/v1/documents/{id}/notes/{noteId}",
		Summary: "Update document note", Tags: []string{"Document Notes"},
		Errors: []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *updateNoteInput) (*noteOutput, error) {
		note, err := service.UpdateDocumentNote(ctx, input.ID, input.NoteID, app.CreateNoteInput{
			Body: input.Body.Body, Format: input.Body.Format,
		})
		return &noteOutput{Body: note}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-document-note", Method: http.MethodDelete, Path: "/api/v1/documents/{id}/notes/{noteId}",
		Summary: "Soft-delete document note", Tags: []string{"Document Notes"}, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *noteIDInput) (*deletedOutput, error) {
		deletedAt, err := service.DeleteDocumentNote(ctx, input.ID, input.NoteID)
		output := &deletedOutput{}
		output.Body.Deleted = true
		output.Body.DeletedAt = deletedAt
		return output, mapError(err)
	})
}
