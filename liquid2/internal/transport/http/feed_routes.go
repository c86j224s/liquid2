package httptransport

import (
	"context"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	feedrefresh "github.com/c86j224s/liquid2/internal/feeds"
	"github.com/danielgtaylor/huma/v2"
)

type feedIDInput struct {
	// ID is the target feed ID.
	ID string `path:"id" doc:"Feed ID"`
}

type createFeedInput struct {
	// Body contains feed fields.
	Body struct {
		// URL is the RSS or Atom feed URL.
		URL string `json:"url" minLength:"1" maxLength:"2048"`
		// Title is an optional display title.
		Title *string `json:"title,omitempty" maxLength:"200"`
		// FolderID is rejected on create; feed folders are assigned automatically.
		FolderID *string `json:"folderId,omitempty"`
		// Enabled controls scheduled polling.
		Enabled *bool `json:"enabled,omitempty"`
	}
}

type updateFeedInput struct {
	// ID is the target feed ID.
	ID string `path:"id" doc:"Feed ID"`
	// Body contains mutable feed fields.
	Body struct {
		// URL replaces the RSS or Atom feed URL when set.
		URL *string `json:"url,omitempty" minLength:"1" maxLength:"2048"`
		// Title replaces or clears the display title when set.
		Title *string `json:"title,omitempty" maxLength:"200"`
		// FolderID replaces or clears the target folder when set.
		FolderID *string `json:"folderId,omitempty"`
		// Enabled controls scheduled polling when set.
		Enabled *bool `json:"enabled,omitempty"`
	}
}

type feedListOutput struct {
	// Body is the feed list response body.
	Body struct {
		// Items contains RSS subscriptions.
		Items []app.Feed `json:"items"`
	}
}

type feedOutput struct {
	// Body is the feed response body.
	Body app.Feed
}

type feedRefreshOutput struct {
	// Body is the refresh response body.
	Body struct {
		// Job contains the enqueued feed refresh job.
		Job app.Job `json:"job"`
	}
}

func registerFeedRoutes(api huma.API, service *app.Service, refresher FeedRefresher) {
	huma.Register(api, huma.Operation{
		OperationID: "list-feeds", Method: http.MethodGet, Path: "/api/v1/feeds",
		Summary: "List feeds", Tags: []string{"Feeds"},
	}, func(ctx context.Context, _ *struct{}) (*feedListOutput, error) {
		output := &feedListOutput{}
		feeds, err := service.ListFeeds(ctx)
		output.Body.Items = feeds
		return output, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-feed", Method: http.MethodPost, Path: "/api/v1/feeds",
		Summary: "Create feed", Tags: []string{"Feeds"}, DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusBadRequest, http.StatusConflict, http.StatusNotFound},
	}, func(ctx context.Context, input *createFeedInput) (*feedOutput, error) {
		feed, err := service.CreateFeed(ctx, app.CreateFeedInput{
			URL: input.Body.URL, Title: input.Body.Title,
			FolderID: input.Body.FolderID, Enabled: input.Body.Enabled,
		})
		return &feedOutput{Body: feed}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-feed", Method: http.MethodPatch, Path: "/api/v1/feeds/{id}",
		Summary: "Update feed", Tags: []string{"Feeds"},
		Errors: []int{http.StatusBadRequest, http.StatusConflict, http.StatusNotFound},
	}, func(ctx context.Context, input *updateFeedInput) (*feedOutput, error) {
		feed, err := service.UpdateFeed(ctx, input.ID, app.UpdateFeedInput{
			URL: input.Body.URL, Title: input.Body.Title,
			FolderID: input.Body.FolderID, Enabled: input.Body.Enabled,
		})
		return &feedOutput{Body: feed}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-feed", Method: http.MethodDelete, Path: "/api/v1/feeds/{id}",
		Summary: "Delete feed", Tags: []string{"Feeds"}, Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *feedIDInput) (*struct{}, error) {
		return &struct{}{}, mapError(service.DeleteFeed(ctx, input.ID))
	})

	huma.Register(api, huma.Operation{
		OperationID: "refresh-feed", Method: http.MethodPost, Path: "/api/v1/feeds/{id}/refresh",
		Summary: "Refresh feed", Tags: []string{"Feeds"}, DefaultStatus: http.StatusAccepted,
		Errors: []int{http.StatusNotFound, http.StatusConflict, http.StatusServiceUnavailable},
	}, func(ctx context.Context, input *feedIDInput) (*feedRefreshOutput, error) {
		if refresher == nil {
			return nil, mapError(feedrefresh.ErrRefreshUnavailable)
		}
		job, err := refresher.RefreshFeed(ctx, input.ID)
		if err != nil {
			return nil, mapError(err)
		}
		output := &feedRefreshOutput{}
		output.Body.Job = job
		return output, nil
	})
}
