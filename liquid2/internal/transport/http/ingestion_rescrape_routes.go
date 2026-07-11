package httptransport

import (
	"context"
	"net/http"

	"github.com/c86j224s/liquid2/internal/ingest"
	"github.com/danielgtaylor/huma/v2"
)

func registerRescrapeRoute(api huma.API, service *ingest.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "rescrape-document", Method: http.MethodPost,
		Path: "/api/v1/documents/{id}/rescrape", Summary: "Re-scrape document",
		Tags: []string{"Ingestion"},
		Errors: []int{
			http.StatusBadRequest, http.StatusNotFound,
			http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType,
		},
	}, func(ctx context.Context, input *documentIDInput) (*documentDetailOutput, error) {
		detail, err := service.Rescrape(ctx, input.ID)
		return &documentDetailOutput{Body: detail}, mapError(err)
	})
}
