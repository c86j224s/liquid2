package httptransport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/danielgtaylor/huma/v2"
)

var (
	ErrExportUnavailable = errors.New("export unavailable")
	ErrExportNotFound    = errors.New("export not found")
)

type ExportRequest struct {
	DocumentIDs []string
}

type exportIDInput struct {
	ID string `path:"id" doc:"Export artifact ID"`
}

type createExportInput struct {
	Body struct {
		DocumentIDs  *[]string `json:"documentIds,omitempty" minItems:"1" nullable:"true"`
		IncludeBlobs *bool     `json:"includeBlobs,omitempty"`
	}
}

type ExportArtifact struct {
	ID              string  `json:"id"`
	CreatedAt       int64   `json:"createdAt"`
	ManifestVersion int     `json:"manifestVersion"`
	DocumentCount   int     `json:"documentCount"`
	BlobCount       int     `json:"blobCount"`
	SizeBytes       int64   `json:"sizeBytes"`
	SHA256          string  `json:"sha256"`
	DownloadURL     *string `json:"downloadUrl"`
}

type exportOutput struct {
	Body struct {
		Export ExportArtifact `json:"export"`
	}
}

func registerExportRoutes(api huma.API, runner ExportRunner, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "create-export", Method: http.MethodPost, Path: "/api/v1/export",
		Summary: "Create markdown export", Tags: []string{"Export"},
		Errors: []int{http.StatusBadRequest, http.StatusNotFound, http.StatusServiceUnavailable},
	}, func(ctx context.Context, input *createExportInput) (*exportOutput, error) {
		if runner == nil {
			return nil, huma.Error503ServiceUnavailable(ErrExportUnavailable.Error())
		}
		if input.Body.IncludeBlobs != nil && !*input.Body.IncludeBlobs {
			return nil, huma.Error400BadRequest("includeBlobs=false is not supported")
		}
		request := ExportRequest{}
		if input.Body.DocumentIDs != nil {
			request.DocumentIDs = append([]string(nil), (*input.Body.DocumentIDs)...)
		}
		artifact, err := runner.Export(ctx, request)
		return exportResponse(artifact), mapExportRouteError(ctx, logger, "export_create_api", err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-export", Method: http.MethodGet, Path: "/api/v1/exports/{id}",
		Summary: "Get export metadata", Tags: []string{"Export"},
		Errors: []int{http.StatusNotFound, http.StatusServiceUnavailable},
	}, func(ctx context.Context, input *exportIDInput) (*exportOutput, error) {
		if runner == nil {
			return nil, huma.Error503ServiceUnavailable(ErrExportUnavailable.Error())
		}
		artifact, err := runner.GetExport(ctx, input.ID)
		return exportResponse(artifact), mapExportRouteError(ctx, logger, "export_get_api", err)
	})
}

func exportResponse(artifact ExportArtifact) *exportOutput {
	output := &exportOutput{}
	output.Body.Export = artifact
	return output
}

func mapExportRouteError(ctx context.Context, logger *slog.Logger, operation string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrExportUnavailable):
		return huma.Error503ServiceUnavailable(ErrExportUnavailable.Error())
	case errors.Is(err, ErrExportNotFound), errors.Is(err, app.ErrNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, app.ErrValidation):
		return huma.Error400BadRequest(err.Error())
	default:
		logger.ErrorContext(ctx, "export request failed",
			slog.String("operation", operation),
			slog.Any("error", err),
		)
		return huma.Error500InternalServerError("internal error")
	}
}
