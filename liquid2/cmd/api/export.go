package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/exporter"
	sqlitestore "github.com/c86j224s/liquid2/internal/storage/sqlite"
	httptransport "github.com/c86j224s/liquid2/internal/transport/http"
)

const exportCreateAttempts = 3

type apiExportRunner struct {
	service   *app.Service
	store     *sqlitestore.Store
	outputDir string
	logger    *slog.Logger
	clock     func() time.Time
	sequence  atomic.Uint64
}

func newExportRunner(logger *slog.Logger, service *app.Service, store *sqlitestore.Store) httptransport.ExportRunner {
	outputDir := strings.TrimSpace(getenv("LIQUID2_EXPORT_DIR", ""))
	if service == nil || outputDir == "" {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	if err := ensureExportRoot(outputDir); err != nil {
		logger.Error("export directory unavailable", slog.String("component", "api"), slog.Any("error", err))
		return nil
	}
	logger.Info("export API enabled", slog.String("component", "api"), slog.String("operation", "export_api_start"))
	return &apiExportRunner{
		service: service, store: store, outputDir: outputDir,
		logger: logger.With("component", "api"), clock: time.Now,
	}
}

func (runner *apiExportRunner) Export(
	ctx context.Context,
	request httptransport.ExportRequest,
) (httptransport.ExportArtifact, error) {
	if err := ensureExportRoot(runner.outputDir); err != nil {
		return httptransport.ExportArtifact{}, fmt.Errorf("%w: export directory", httptransport.ErrExportUnavailable)
	}
	var lastCollision error
	for attempt := 0; attempt < exportCreateAttempts; attempt++ {
		id := runner.nextID()
		dir := filepath.Join(runner.outputDir, id)
		if err := os.Mkdir(dir, 0o700); err != nil {
			if errors.Is(err, os.ErrExist) {
				lastCollision = err
				continue
			}
			return httptransport.ExportArtifact{}, fmt.Errorf("%w: create export artifact: %v", httptransport.ErrExportUnavailable, err)
		}
		artifact, err := runner.writeExport(ctx, id, dir, request)
		if err != nil {
			_ = os.RemoveAll(dir)
			return httptransport.ExportArtifact{}, err
		}
		return artifact, nil
	}
	return httptransport.ExportArtifact{}, fmt.Errorf("export destination collision: %w", lastCollision)
}

func (runner *apiExportRunner) GetExport(ctx context.Context, id string) (httptransport.ExportArtifact, error) {
	if err := ensureExportRoot(runner.outputDir); err != nil {
		return httptransport.ExportArtifact{}, fmt.Errorf("%w: export directory", httptransport.ErrExportUnavailable)
	}
	id, err := cleanExportID(id)
	if err != nil {
		return httptransport.ExportArtifact{}, httptransport.ErrExportNotFound
	}
	artifact, err := runner.artifactFromDirectory(filepath.Join(runner.outputDir, id), id)
	if err != nil {
		return httptransport.ExportArtifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return httptransport.ExportArtifact{}, err
	}
	return artifact, nil
}

func (runner *apiExportRunner) writeExport(
	ctx context.Context,
	id string,
	dir string,
	request httptransport.ExportRequest,
) (httptransport.ExportArtifact, error) {
	schemaVersion, err := runner.schemaVersion(ctx)
	if err != nil {
		return httptransport.ExportArtifact{}, err
	}
	result, err := runner.service.ExportMarkdown(ctx, app.MarkdownExportInput{
		ExportID: id, CreatedAt: runner.clock().UTC().UnixMilli(),
		SchemaVersion: schemaVersion, DocumentIDs: request.DocumentIDs,
	}, exporter.NewDirectoryWriter(dir))
	if err != nil {
		return httptransport.ExportArtifact{}, err
	}
	artifact, err := runner.artifactFromManifest(dir, result.Manifest)
	if err != nil {
		return httptransport.ExportArtifact{}, err
	}
	runner.logger.InfoContext(ctx, "export API completed",
		slog.String("operation", "export_api"),
		slog.String("export_id", artifact.ID),
		slog.Int("document_count", artifact.DocumentCount),
		slog.Int("blob_count", artifact.BlobCount),
		slog.Int64("size_bytes", artifact.SizeBytes),
	)
	return artifact, nil
}

func (runner *apiExportRunner) schemaVersion(ctx context.Context) (*int64, error) {
	if runner.store == nil {
		return nil, nil
	}
	version, err := runner.store.Queries().SchemaVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("read export schema version: %w", err)
	}
	return &version, nil
}

func (runner *apiExportRunner) nextID() string {
	seq := runner.sequence.Add(1)
	return fmt.Sprintf("export_%s_%d", runner.clock().UTC().Format("20060102T150405000Z"), seq)
}

func (runner *apiExportRunner) artifactFromDirectory(dir string, expectedID string) (httptransport.ExportArtifact, error) {
	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return httptransport.ExportArtifact{}, httptransport.ErrExportNotFound
	}
	if err != nil {
		return httptransport.ExportArtifact{}, fmt.Errorf("inspect export artifact: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return httptransport.ExportArtifact{}, fmt.Errorf("export artifact is invalid")
	}
	body, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return httptransport.ExportArtifact{}, fmt.Errorf("read export manifest: %w", err)
	}
	var manifest app.ExportManifest
	if err = json.Unmarshal(body, &manifest); err != nil {
		return httptransport.ExportArtifact{}, fmt.Errorf("decode export manifest: %w", err)
	}
	if manifest.ExportID != expectedID {
		return httptransport.ExportArtifact{}, fmt.Errorf("export manifest id mismatch")
	}
	return runner.artifactFromManifest(dir, manifest)
}

func (runner *apiExportRunner) artifactFromManifest(
	dir string,
	manifest app.ExportManifest,
) (httptransport.ExportArtifact, error) {
	if _, err := cleanExportID(manifest.ExportID); err != nil {
		return httptransport.ExportArtifact{}, fmt.Errorf("export manifest id is invalid")
	}
	size, checksum, err := exportDirectoryStats(dir)
	if err != nil {
		return httptransport.ExportArtifact{}, err
	}
	return httptransport.ExportArtifact{
		ID: manifest.ExportID, CreatedAt: manifest.CreatedAt, ManifestVersion: manifest.ManifestVersion,
		DocumentCount: manifest.Counts.Documents, BlobCount: manifest.Counts.Blobs,
		SizeBytes: size, SHA256: checksum,
	}, nil
}

func cleanExportID(id string) (string, error) {
	if id == "" || strings.TrimSpace(id) != id || strings.HasPrefix(id, ".") {
		return "", fmt.Errorf("export id is invalid")
	}
	for _, char := range id {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' || char == '_' || char == '-' {
			continue
		}
		return "", fmt.Errorf("export id is invalid")
	}
	return id, nil
}
