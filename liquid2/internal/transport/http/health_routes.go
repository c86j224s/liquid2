package httptransport

import (
	"context"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/danielgtaylor/huma/v2"
)

// healthOutput returns process health.
type healthOutput struct {
	// Body is the health response body.
	Body app.Health
}

func registerHealth(api huma.API, service *app.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "get-health",
		Method:      http.MethodGet,
		Path:        "/healthz",
		Summary:     "Check process health",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, _ *struct{}) (*healthOutput, error) {
		return &healthOutput{Body: service.Health(ctx)}, nil
	})
}
