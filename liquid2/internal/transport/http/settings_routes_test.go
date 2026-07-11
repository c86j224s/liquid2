package httptransport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
)

func TestSettingsRoutes(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service)

	response := serveJSON(router, http.MethodGet, "/api/v1/settings", "")
	assertBodyContains(t, response, http.StatusOK, `"feedSchedulerEnabled":false`)
	assertBodyContains(t, response, http.StatusOK, `"feedPollIntervalSeconds":7200`)
	assertBodyContains(t, response, http.StatusOK, `"feedNextPollAt":null`)

	response = serveJSON(router, http.MethodPatch, "/api/v1/settings", `{
		"feedSchedulerEnabled": true,
		"feedPollIntervalSeconds": 300
	}`)
	assertBodyContains(t, response, http.StatusOK, `"feedSchedulerEnabled":true`)
	assertBodyContains(t, response, http.StatusOK, `"feedPollIntervalSeconds":300`)
	assertBodyContains(t, response, http.StatusOK, `"feedNextPollAt":`)
}

func TestSettingsRouteValidation(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	router := NewRouter(service)

	response := serveJSON(router, http.MethodPatch, "/api/v1/settings", `{
		"feedPollIntervalSeconds": 30
	}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
}

func assertBodyContains(t *testing.T, response *httptest.ResponseRecorder, status int, fragment string) {
	t.Helper()
	if response.Code != status || !strings.Contains(response.Body.String(), fragment) {
		t.Fatalf("expected status %d and fragment %q, got %d: %s",
			status, fragment, response.Code, response.Body.String())
	}
}
