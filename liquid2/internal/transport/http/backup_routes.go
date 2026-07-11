package httptransport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

var ErrBackupUnavailable = errors.New("backup unavailable")

type createBackupInput struct {
	// Body is intentionally empty. Backup storage paths come from server config.
	Body struct{}
}

type BackupArtifact struct {
	ID            string  `json:"id"`
	CreatedAt     int64   `json:"createdAt"`
	SourceType    string  `json:"sourceType"`
	SchemaVersion int64   `json:"schemaVersion"`
	SizeBytes     int64   `json:"sizeBytes"`
	SHA256        string  `json:"sha256"`
	DownloadURL   *string `json:"downloadUrl"`
}

type backupOutput struct {
	// Body is the backup response body.
	Body struct {
		// Backup contains API-safe backup artifact metadata.
		Backup BackupArtifact `json:"backup"`
	}
}

func registerBackupRoutes(api huma.API, runner BackupRunner, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "create-backup", Method: http.MethodPost, Path: "/api/v1/backup",
		Summary: "Create SQLite backup", Tags: []string{"Backup"},
		Errors: []int{http.StatusServiceUnavailable},
	}, func(ctx context.Context, _ *createBackupInput) (*backupOutput, error) {
		if runner == nil {
			return nil, huma.Error503ServiceUnavailable(ErrBackupUnavailable.Error())
		}
		artifact, err := runner.Backup(ctx)
		if err != nil {
			if errors.Is(err, ErrBackupUnavailable) {
				return nil, huma.Error503ServiceUnavailable(ErrBackupUnavailable.Error())
			}
			logger.ErrorContext(ctx, "backup request failed",
				slog.String("operation", "backup_api"),
				slog.Any("error", err),
			)
			return nil, huma.Error500InternalServerError("internal error")
		}
		output := &backupOutput{}
		output.Body.Backup = artifact
		return output, nil
	})
}
