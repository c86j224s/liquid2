package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/internal/app"
	httptransport "github.com/c86j224s/liquid2/internal/transport/http"
)

func TestNewExportRunnerDisabledWithoutDir(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	t.Setenv("LIQUID2_EXPORT_DIR", "")
	if runner := newExportRunner(testLogger(), service, nil); runner != nil {
		t.Fatalf("expected nil runner without export dir, got %v", runner)
	}
	if runner := newExportRunner(testLogger(), nil, nil); runner != nil {
		t.Fatalf("expected nil runner without service, got %v", runner)
	}

	parentFile := filepath.Join(t.TempDir(), "not-dir")
	if err := os.WriteFile(parentFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write parent file: %v", err)
	}
	t.Setenv("LIQUID2_EXPORT_DIR", filepath.Join(parentFile, "exports"))
	if runner := newExportRunner(testLogger(), service, nil); runner != nil {
		t.Fatalf("expected nil runner for unavailable export dir, got %v", runner)
	}
}

func TestAPIExportRunnerCreatesAndReadsArtifact(t *testing.T) {
	ctx := context.Background()
	service := app.NewService(app.WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	detail, err := service.CreateUploadedDocument(ctx, app.UploadedDocumentInput{
		Filename: "Report.pdf", MimeType: "application/pdf", Data: []byte("PDF bytes"),
		Content: "Extracted text", Format: app.ContentFormatText,
	})
	if err != nil {
		t.Fatalf("create uploaded document: %v", err)
	}
	outDir := t.TempDir()
	t.Setenv("LIQUID2_EXPORT_DIR", outDir)
	runner := newExportRunner(testLogger(), service, nil).(*apiExportRunner)
	runner.clock = func() time.Time { return time.Date(2026, 6, 12, 1, 2, 3, 0, time.UTC) }

	artifact, err := runner.Export(ctx, httptransport.ExportRequest{})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if artifact.ID != "export_20260612T010203000Z_1" || artifact.CreatedAt != 1781226123000 {
		t.Fatalf("unexpected artifact identity: %#v", artifact)
	}
	if artifact.ManifestVersion != 1 || artifact.DocumentCount != 1 || artifact.BlobCount != 1 {
		t.Fatalf("unexpected artifact counts: %#v", artifact)
	}
	if artifact.SizeBytes <= 0 || len(artifact.SHA256) != 64 || artifact.DownloadURL != nil {
		t.Fatalf("unexpected artifact metadata: %#v", artifact)
	}
	body, err := os.ReadFile(filepath.Join(outDir, artifact.ID, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(body), detail.Document.ID) || strings.Contains(string(body), outDir) {
		t.Fatalf("unexpected manifest body: %s", string(body))
	}
	got, err := runner.GetExport(ctx, artifact.ID)
	if err != nil {
		t.Fatalf("get export: %v", err)
	}
	if got != artifact {
		t.Fatalf("expected GET metadata to match created artifact\ncreated=%#v\ngot=%#v", artifact, got)
	}
}

func TestNewExportRunnerSecuresExistingRoot(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	root := t.TempDir()
	if err := os.Chmod(root, 0o777); err != nil {
		t.Fatalf("make export root permissive: %v", err)
	}
	t.Setenv("LIQUID2_EXPORT_DIR", root)

	runner := newExportRunner(testLogger(), service, nil)
	if runner == nil {
		t.Fatal("expected export runner")
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("stat export root: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o700 {
		t.Fatalf("expected export root mode 0700, got %o", mode)
	}
}

func TestAPIExportRunnerIncludesSQLiteSchemaVersion(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t, ctx)
	t.Cleanup(func() { _ = store.Close() })
	service := app.NewService(app.WithRepository(app.NewSQLiteRepository(store)))
	t.Cleanup(func() { _ = service.Close() })
	if _, err := service.CreateDocument(ctx, app.CreateDocumentInput{Title: "SQLite Doc"}); err != nil {
		t.Fatalf("create document: %v", err)
	}
	outDir := t.TempDir()
	t.Setenv("LIQUID2_EXPORT_DIR", outDir)
	runner := newExportRunner(testLogger(), service, store).(*apiExportRunner)

	artifact, err := runner.Export(ctx, httptransport.ExportRequest{})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(outDir, artifact.ID, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest app.ExportManifest
	if err = json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.Source.SchemaVersion == nil || *manifest.Source.SchemaVersion != 13 {
		t.Fatalf("expected schema version 13 in manifest, got %#v", manifest.Source.SchemaVersion)
	}
}

func TestAPIExportRunnerMapsMissingDocument(t *testing.T) {
	ctx := context.Background()
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	t.Setenv("LIQUID2_EXPORT_DIR", t.TempDir())
	runner := newExportRunner(testLogger(), service, nil).(*apiExportRunner)

	_, err := runner.Export(ctx, httptransport.ExportRequest{DocumentIDs: []string{"missing"}})
	if !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("expected app not found, got %v", err)
	}
}

func TestAPIExportRunnerGetMissingOrInvalidArtifact(t *testing.T) {
	ctx := context.Background()
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	t.Setenv("LIQUID2_EXPORT_DIR", t.TempDir())
	runner := newExportRunner(testLogger(), service, nil).(*apiExportRunner)

	for _, id := range []string{"missing", "../escape", ".hidden"} {
		_, err := runner.GetExport(ctx, id)
		if !errors.Is(err, httptransport.ErrExportNotFound) {
			t.Fatalf("expected export not found for %q, got %v", id, err)
		}
	}
}
