package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
)

type fakeBackupRunner struct {
	artifact BackupArtifact
	called   bool
}

func (runner *fakeBackupRunner) Backup(_ context.Context) (BackupArtifact, error) {
	runner.called = true
	return runner.artifact, nil
}

func TestBackupRouteCreatesArtifact(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	runner := &fakeBackupRunner{artifact: BackupArtifact{
		ID: "backup_1", CreatedAt: 1760000000000, SourceType: "sqlite",
		SchemaVersion: 6, SizeBytes: 1024, SHA256: strings.Repeat("a", 64),
	}}
	router := NewRouter(service, WithBackupRunner(runner))

	response := serveJSON(router, http.MethodPost, "/api/v1/backup", `{}`)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !runner.called || !strings.Contains(response.Body.String(), `"backup":{"id":"backup_1"`) {
		t.Fatalf("unexpected backup response called=%v body=%s", runner.called, response.Body.String())
	}
}

func TestBackupRouteUnavailable(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service)

	response := serveJSON(router, http.MethodPost, "/api/v1/backup", `{}`)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", response.Code, response.Body.String())
	}
}

func TestBackupRouteRejectsOptions(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service, WithBackupRunner(&fakeBackupRunner{}))

	response := serveJSON(router, http.MethodPost, "/api/v1/backup", `{"destinationPath":"/tmp/backup.sqlite3"}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", response.Code, response.Body.String())
	}
}

func TestBackupRouteResponseDoesNotExposePath(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service, WithBackupRunner(&fakeBackupRunner{artifact: BackupArtifact{
		ID: "backup_1", CreatedAt: 1760000000000, SourceType: "sqlite",
		SchemaVersion: 6, SizeBytes: 1024, SHA256: strings.Repeat("a", 64),
	}}))

	response := serveJSON(router, http.MethodPost, "/api/v1/backup", `{}`)
	body := response.Body.String()
	for _, forbidden := range []string{"fileName", "filename", "path", "/tmp"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("backup response exposed %q: %s", forbidden, body)
		}
	}
	if !strings.Contains(body, `"downloadUrl":null`) {
		t.Fatalf("expected null downloadUrl, got %s", body)
	}
}

func TestBackupRouteOpenAPIShape(t *testing.T) {
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
	if _, ok := doc.Paths["/api/v1/backup"]; !ok {
		t.Fatal("expected backup OpenAPI path")
	}
	if doc.Paths["/api/v1/backup"]["post"].(map[string]any)["operationId"] != "create-backup" {
		t.Fatalf("unexpected backup operation: %#v", doc.Paths["/api/v1/backup"]["post"])
	}
	schema := doc.Components.Schemas["BackupArtifact"]
	if schema == nil {
		t.Fatal("expected BackupArtifact schema")
	}
	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	for _, forbidden := range []string{"fileName", "filename", "path"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("backup schema exposed %q: %s", forbidden, string(data))
		}
	}
}
