package httptransport

import (
	"context"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/danielgtaylor/huma/v2"
)

// documentIDInput carries a document path parameter.
type documentIDInput struct {
	// ID is the target document ID.
	ID string `path:"id" doc:"Document ID"`
}

// listDocumentsInput carries document list query filters.
type listDocumentsInput struct {
	// Query filters documents by plain text search.
	Query string `query:"q" maxLength:"256" required:"false"`
	// Status filters documents by read-state status.
	Status string `query:"status" enum:"unread,read" required:"false"`
	// FolderID filters documents to a specific folder.
	FolderID string `query:"folderId" required:"false"`
	// IncludeFolderDescendants includes child folders when folderId is set.
	IncludeFolderDescendants bool `query:"includeFolderDescendants" required:"false"`
	// Tag filters documents by tag slug.
	Tag string `query:"tag" required:"false"`
	// RatingMin filters documents by minimum rating.
	RatingMin int `query:"ratingMin" minimum:"1" maximum:"5" required:"false"`
	// Kind filters documents by document kind.
	Kind string `query:"kind" enum:"bookmark,scraped_article,uploaded_file,rss_item" required:"false"`
	// Sort controls document list ordering.
	Sort string `query:"sort" enum:"relevance,recent,created_desc,rating_desc" required:"false"`
	// IncludeDeleted includes soft-deleted documents when true.
	IncludeDeleted bool `query:"includeDeleted" required:"false"`
	// IncludeTrash includes documents assigned to the Trash folder.
	IncludeTrash bool `query:"includeTrash" required:"false"`
	// Limit caps the number of returned documents.
	Limit int `query:"limit" minimum:"1" maximum:"100" required:"false"`
	// Cursor identifies the next page to fetch.
	Cursor string `query:"cursor" required:"false"`
}

// documentListOutput returns a document page.
type documentListOutput struct {
	// Body is the document list response body.
	Body app.DocumentList
}

// documentDetailOutput returns a full document.
type documentDetailOutput struct {
	// Body is the document detail response body.
	Body app.DocumentDetail
}

// updateDocumentInput carries a document update request.
type updateDocumentInput struct {
	// ID is the target document ID.
	ID string `path:"id" doc:"Document ID"`
	// Body contains mutable document fields.
	Body struct {
		// Title replaces the document title when set.
		Title *string `json:"title,omitempty" minLength:"1" maxLength:"300"`
		// FolderID moves the document to a folder when set.
		FolderID *string `json:"folderId,omitempty"`
	}
}

// ratingInput carries a document rating request.
type ratingInput struct {
	// ID is the target document ID.
	ID string `path:"id" doc:"Document ID"`
	// Body contains the rating value.
	Body struct {
		// Rating is the optional user rating from 1 to 5.
		Rating *int `json:"rating,omitempty" minimum:"1" maximum:"5"`
	}
}

// replaceTagsInput carries a full document tag replacement request.
type replaceTagsInput struct {
	// ID is the target document ID.
	ID string `path:"id" doc:"Document ID"`
	// Body contains replacement tag IDs.
	Body struct {
		// TagIDs replaces all tags assigned to the document.
		TagIDs []string `json:"tagIds" doc:"Replacement tag IDs"`
	}
}

// deletedOutput returns soft-delete metadata.
type deletedOutput struct {
	// Body is the deletion response body.
	Body struct {
		// Deleted is true when the soft-delete completed.
		Deleted bool `json:"deleted"`
		// DeletedAt is the deletion timestamp in Unix milliseconds.
		DeletedAt int64 `json:"deletedAt"`
	}
}

func registerDocumentRoutes(api huma.API, service *app.Service, translator DocumentTranslator) {
	huma.Register(api, huma.Operation{
		OperationID: "list-documents", Method: http.MethodGet, Path: "/api/v1/documents",
		Summary: "List documents", Tags: []string{"Documents"}, Errors: []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *listDocumentsInput) (*documentListOutput, error) {
		filters := app.DocumentFilters{
			Query: input.Query, Status: input.Status, FolderID: input.FolderID,
			IncludeFolderDescendants: input.IncludeFolderDescendants,
			Tag:                      input.Tag, RatingMin: input.RatingMin, Kind: input.Kind,
			Sort: input.Sort, IncludeDeleted: input.IncludeDeleted,
			IncludeTrash: input.IncludeTrash,
			Limit:        input.Limit, Cursor: input.Cursor,
		}
		list, err := service.ListDocuments(ctx, filters)
		return &documentListOutput{Body: list}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-document", Method: http.MethodGet, Path: "/api/v1/documents/{id}",
		Summary: "Get document detail", Tags: []string{"Documents"}, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *documentIDInput) (*documentDetailOutput, error) {
		detail, err := service.GetDocument(ctx, input.ID)
		return &documentDetailOutput{Body: detail}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-document", Method: http.MethodPatch, Path: "/api/v1/documents/{id}",
		Summary: "Update document metadata", Tags: []string{"Documents"},
		Errors: []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *updateDocumentInput) (*documentDetailOutput, error) {
		detail, err := service.UpdateDocument(ctx, input.ID, app.UpdateDocumentInput{
			Title: input.Body.Title, FolderID: input.Body.FolderID,
		})
		return &documentDetailOutput{Body: detail}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-document", Method: http.MethodDelete, Path: "/api/v1/documents/{id}",
		Summary: "Soft-delete document", Tags: []string{"Documents"}, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *documentIDInput) (*deletedOutput, error) {
		deletedAt, err := service.DeleteDocument(ctx, input.ID)
		output := &deletedOutput{}
		output.Body.Deleted = true
		output.Body.DeletedAt = deletedAt
		return output, mapError(err)
	})

	registerDocumentStateRoutes(api, service)
	registerDocumentTranslationRoutes(api, translator)
}

func registerDocumentStateRoutes(api huma.API, service *app.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "mark-document-read", Method: http.MethodPost, Path: "/api/v1/documents/{id}/mark-read",
		Summary: "Mark document read", Tags: []string{"Documents"}, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *documentIDInput) (*documentDetailOutput, error) {
		detail, err := service.MarkDocumentRead(ctx, input.ID)
		return &documentDetailOutput{Body: detail}, mapError(err)
	})
	huma.Register(api, huma.Operation{
		OperationID: "mark-document-unread", Method: http.MethodPost, Path: "/api/v1/documents/{id}/mark-unread",
		Summary: "Mark document unread", Tags: []string{"Documents"}, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *documentIDInput) (*documentDetailOutput, error) {
		detail, err := service.MarkDocumentUnread(ctx, input.ID)
		return &documentDetailOutput{Body: detail}, mapError(err)
	})
	huma.Register(api, huma.Operation{
		OperationID: "move-document-to-trash", Method: http.MethodPost, Path: "/api/v1/documents/{id}/move-to-trash",
		Summary: "Move document to trash", Tags: []string{"Documents"}, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *documentIDInput) (*documentDetailOutput, error) {
		detail, err := service.MoveDocumentToTrash(ctx, input.ID)
		return &documentDetailOutput{Body: detail}, mapError(err)
	})
	huma.Register(api, huma.Operation{
		OperationID: "set-document-rating", Method: http.MethodPut, Path: "/api/v1/documents/{id}/rating",
		Summary: "Set document rating", Tags: []string{"Documents"},
		Errors: []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *ratingInput) (*documentDetailOutput, error) {
		detail, err := service.SetDocumentRating(ctx, input.ID, input.Body.Rating)
		return &documentDetailOutput{Body: detail}, mapError(err)
	})
	huma.Register(api, huma.Operation{
		OperationID: "replace-document-tags", Method: http.MethodPut, Path: "/api/v1/documents/{id}/tags",
		Summary: "Replace document tags", Tags: []string{"Documents", "Tags"},
		Errors: []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *replaceTagsInput) (*documentDetailOutput, error) {
		detail, err := service.ReplaceDocumentTags(ctx, input.ID, input.Body.TagIDs)
		return &documentDetailOutput{Body: detail}, mapError(err)
	})
}
