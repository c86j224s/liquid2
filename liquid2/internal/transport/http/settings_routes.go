package httptransport

import (
	"context"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/danielgtaylor/huma/v2"
)

type settingsOutput struct {
	// Body contains current user-managed app settings.
	Body app.AppSettings
}

type updateSettingsInput struct {
	// Body contains partial settings updates.
	Body app.UpdateSettingsInput
}

func registerSettingsRoutes(api huma.API, service *app.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "get-settings", Method: http.MethodGet,
		Path: "/api/v1/settings", Summary: "Get app settings",
		Tags: []string{"Settings"},
	}, func(ctx context.Context, _ *struct{}) (*settingsOutput, error) {
		settings, err := service.GetSettings(ctx)
		return &settingsOutput{Body: settings}, mapError(err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-settings", Method: http.MethodPatch,
		Path: "/api/v1/settings", Summary: "Update app settings",
		Tags: []string{"Settings"}, Errors: []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *updateSettingsInput) (*settingsOutput, error) {
		settings, err := service.UpdateSettings(ctx, input.Body)
		return &settingsOutput{Body: settings}, mapError(err)
	})
}
