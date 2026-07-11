package httptransport

import (
	"context"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/danielgtaylor/huma/v2"
)

// tagBodyInput carries a tag create request.
type tagBodyInput struct {
	// Body contains tag fields.
	Body struct {
		// Name is the tag display name.
		Name string `json:"name" minLength:"1" maxLength:"100"`
	}
}

// tagListOutput returns all tags.
type tagListOutput struct {
	// Body is the tag list response body.
	Body struct {
		// Items contains tags sorted by implementation order.
		Items []app.Tag `json:"items"`
	}
}

// tagOutput returns one tag.
type tagOutput struct {
	// Body is the tag response body.
	Body app.Tag
}

func registerTagRoutes(api huma.API, service *app.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-tags", Method: http.MethodGet, Path: "/api/v1/tags",
		Summary: "List tags", Tags: []string{"Tags"},
	}, func(ctx context.Context, _ *struct{}) (*tagListOutput, error) {
		output := &tagListOutput{}
		tags, err := service.ListTags(ctx)
		output.Body.Items = tags
		return output, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-tag", Method: http.MethodPost, Path: "/api/v1/tags",
		Summary: "Create tag", Tags: []string{"Tags"}, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusBadRequest, http.StatusConflict},
	}, func(ctx context.Context, input *tagBodyInput) (*tagOutput, error) {
		tag, err := service.CreateTag(ctx, input.Body.Name)
		return &tagOutput{Body: tag}, mapError(err)
	})
}
