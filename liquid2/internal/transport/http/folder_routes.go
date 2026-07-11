package httptransport

import (
	"context"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/danielgtaylor/huma/v2"
)

// folderIDInput carries a folder path parameter.
type folderIDInput struct {
	// ID is the target folder ID.
	ID string `path:"id" doc:"Folder ID"`
}

// folderBodyInput carries a folder create request.
type folderBodyInput struct {
	// Body contains folder fields.
	Body struct {
		// ParentID creates the folder under a parent when set.
		ParentID *string `json:"parentId,omitempty"`
		// Name is the folder display name.
		Name string `json:"name" minLength:"1" maxLength:"200"`
		// SortOrder controls sibling ordering.
		SortOrder int `json:"sortOrder"`
	}
}

// updateFolderInput carries a folder update request.
type updateFolderInput struct {
	// ID is the target folder ID.
	ID string `path:"id" doc:"Folder ID"`
	// Body contains mutable folder fields.
	Body struct {
		// ParentID moves the folder under a parent when set.
		ParentID *string `json:"parentId,omitempty"`
		// Name is the folder display name.
		Name string `json:"name" minLength:"1" maxLength:"200"`
		// SortOrder controls sibling ordering.
		SortOrder int `json:"sortOrder"`
	}
}

// deleteFolderInput carries folder deletion options.
type deleteFolderInput struct {
	// ID is the target folder ID.
	ID string `path:"id" doc:"Folder ID"`
	// DocumentAction controls how documents in the folder are handled.
	DocumentAction string `query:"documentAction" enum:"move_to_parent,move_to_uncategorized,reject_if_not_empty" required:"false"`
}

// folderListOutput returns the folder tree.
type folderListOutput struct {
	// Body is the folder list response body.
	Body struct {
		// Items contains root folders with nested children.
		Items []app.Folder `json:"items"`
	}
}

// folderOutput returns one folder.
type folderOutput struct {
	// Body is the folder response body.
	Body app.Folder
}

func registerFolderRoutes(api huma.API, service *app.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-folders", Method: http.MethodGet, Path: "/api/v1/folders",
		Summary: "List folder tree", Tags: []string{"Folders"},
	}, func(ctx context.Context, _ *struct{}) (*folderListOutput, error) {
		output := &folderListOutput{}
		folders, err := service.ListFolders(ctx)
		output.Body.Items = folders
		return output, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-folder", Method: http.MethodPost, Path: "/api/v1/folders",
		Summary: "Create folder", Tags: []string{"Folders"}, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusBadRequest, http.StatusConflict, http.StatusNotFound},
	}, func(ctx context.Context, input *folderBodyInput) (*folderOutput, error) {
		folder, err := service.CreateFolder(ctx, app.FolderInput{
			ParentID: input.Body.ParentID, Name: input.Body.Name, SortOrder: input.Body.SortOrder,
		})
		return &folderOutput{Body: folder}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-folder", Method: http.MethodPatch, Path: "/api/v1/folders/{id}",
		Summary: "Update folder", Tags: []string{"Folders"},
		Errors: []int{http.StatusBadRequest, http.StatusConflict, http.StatusNotFound},
	}, func(ctx context.Context, input *updateFolderInput) (*folderOutput, error) {
		folder, err := service.UpdateFolder(ctx, input.ID, app.FolderInput{
			ParentID: input.Body.ParentID, Name: input.Body.Name, SortOrder: input.Body.SortOrder,
		})
		return &folderOutput{Body: folder}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-folder", Method: http.MethodDelete, Path: "/api/v1/folders/{id}",
		Summary: "Delete folder", Tags: []string{"Folders"}, Errors: []int{http.StatusConflict, http.StatusNotFound},
	}, func(ctx context.Context, input *deleteFolderInput) (*struct{}, error) {
		action := input.DocumentAction
		if action == "" {
			action = "reject_if_not_empty"
		}
		return &struct{}{}, mapError(service.DeleteFolder(ctx, input.ID, action))
	})
}
