package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
)

type fakeExportRunner struct {
	artifact ExportArtifact
	request  ExportRequest
	err      error
	called   bool
}

func (runner *fakeExportRunner) Export(_ context.Context, request ExportRequest) (ExportArtifact, error) {
	runner.called = true
	runner.request = request
	return runner.artifact, runner.err
}

func (runner *fakeExportRunner) GetExport(_ context.Context, _ string) (ExportArtifact, error) {
	runner.called = true
	return runner.artifact, runner.err
}

func TestExportRouteCreatesArtifact(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	runner := &fakeExportRunner{artifact: exportTestArtifact()}
	router := NewRouter(service, WithExportRunner(runner))

	response := serveJSON(router, http.MethodPost, "/api/v1/export", `{"documentIds":["doc_2","doc_1"],"includeBlobs":true}`)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !runner.called || strings.Join(runner.request.DocumentIDs, ",") != "doc_2,doc_1" {
		t.Fatalf("unexpected export request called=%v request=%#v", runner.called, runner.request)
	}
	if !strings.Contains(response.Body.String(), `"export":{"id":"export_1"`) {
		t.Fatalf("unexpected export response: %s", response.Body.String())
	}
}

func TestExportRouteAcceptsNullDocumentIDs(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	runner := &fakeExportRunner{artifact: exportTestArtifact()}
	router := NewRouter(service, WithExportRunner(runner))

	response := serveJSON(router, http.MethodPost, "/api/v1/export", `{"documentIds":null}`)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if runner.request.DocumentIDs != nil {
		t.Fatalf("expected nil document IDs for full export, got %#v", runner.request.DocumentIDs)
	}
}

func TestExportRouteUnavailable(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service)

	response := serveJSON(router, http.MethodPost, "/api/v1/export", `{}`)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", response.Code, response.Body.String())
	}
}

func TestExportRouteRejectsOptions(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service, WithExportRunner(&fakeExportRunner{}))

	response := serveJSON(router, http.MethodPost, "/api/v1/export", `{"destinationPath":"/tmp/export"}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", response.Code, response.Body.String())
	}

	response = serveJSON(router, http.MethodPost, "/api/v1/export", `{"documentIds":[]}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected empty document IDs status 422, got %d: %s", response.Code, response.Body.String())
	}
}

func TestExportRouteMapsErrors(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{name: "document missing", err: app.ErrNotFound, status: http.StatusNotFound},
		{name: "artifact missing", err: ErrExportNotFound, status: http.StatusNotFound},
		{name: "storage unavailable", err: ErrExportUnavailable, status: http.StatusServiceUnavailable},
		{name: "unexpected", err: errors.New("boom"), status: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter(service, WithExportRunner(&fakeExportRunner{err: tc.err}))
			response := serveJSON(router, http.MethodPost, "/api/v1/export", `{}`)
			if response.Code != tc.status {
				t.Fatalf("expected status %d, got %d: %s", tc.status, response.Code, response.Body.String())
			}
		})
	}
}

func TestExportRouteGetArtifact(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	runner := &fakeExportRunner{artifact: exportTestArtifact()}
	router := NewRouter(service, WithExportRunner(runner))

	response := serveJSON(router, http.MethodGet, "/api/v1/exports/export_1", "")
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !runner.called || !strings.Contains(response.Body.String(), `"id":"export_1"`) {
		t.Fatalf("unexpected export response called=%v body=%s", runner.called, response.Body.String())
	}
}

func TestExportRouteResponseDoesNotExposePath(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service, WithExportRunner(&fakeExportRunner{artifact: exportTestArtifact()}))

	response := serveJSON(router, http.MethodPost, "/api/v1/export", `{}`)
	body := response.Body.String()
	for _, forbidden := range []string{"fileName", "filename", "path", "/tmp"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("export response exposed %q: %s", forbidden, body)
		}
	}
	if !strings.Contains(body, `"downloadUrl":null`) {
		t.Fatalf("expected null downloadUrl, got %s", body)
	}
}

func TestExportRouteOpenAPIShape(t *testing.T) {
	spec, err := OpenAPISpec(app.NewService()).Downgrade()
	if err != nil {
		t.Fatalf("downgrade spec: %v", err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
		Paths map[string]map[string]any `json:"paths"`
	}
	if err = json.Unmarshal(spec, &doc); err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	if doc.Paths["/api/v1/export"]["post"].(map[string]any)["operationId"] != "create-export" {
		t.Fatalf("unexpected create export operation: %#v", doc.Paths["/api/v1/export"]["post"])
	}
	if doc.Paths["/api/v1/exports/{id}"]["get"].(map[string]any)["operationId"] != "get-export" {
		t.Fatalf("unexpected get export operation: %#v", doc.Paths["/api/v1/exports/{id}"]["get"])
	}
	data, err := json.Marshal(doc.Components.Schemas["ExportArtifact"])
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	for _, forbidden := range []string{"fileName", "filename", "path"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("export schema exposed %q: %s", forbidden, string(data))
		}
	}
	bodySchema, err := json.Marshal(doc.Components.Schemas["CreateExportInputBody"])
	if err != nil {
		t.Fatalf("marshal create schema: %v", err)
	}
	if !strings.Contains(string(bodySchema), `"nullable":true`) {
		t.Fatalf("expected nullable documentIds schema: %s", string(bodySchema))
	}
}

func exportTestArtifact() ExportArtifact {
	return ExportArtifact{
		ID: "export_1", CreatedAt: 1760000000000, ManifestVersion: 1,
		DocumentCount: 2, BlobCount: 1, SizeBytes: 2048, SHA256: strings.Repeat("b", 64),
	}
}
