package httptransport

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/logging"
)

func TestHealthRoute(t *testing.T) {
	router := NewRouter(app.NewService())
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if body := response.Body.String(); body != "{\"ok\":true}\n" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestRequestLoggerOmitsQueryString(t *testing.T) {
	var output bytes.Buffer
	logger, err := logging.New(&output, logging.Config{Level: "debug"})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	service := app.NewService(app.WithLogger(logger))
	router := NewRouter(service, WithLogger(logger))
	request := httptest.NewRequest(http.MethodGet, "/healthz?token=secret", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	logs := output.String()
	if !strings.Contains(logs, `"operation":"http_request"`) {
		t.Fatalf("expected request log, got %q", logs)
	}
	if !strings.Contains(logs, `"path":"/healthz"`) {
		t.Fatalf("expected path without query string, got %q", logs)
	}
	if strings.Contains(logs, "secret") {
		t.Fatalf("expected query string to be omitted, got %q", logs)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	router := NewRouter(app.NewService(), WithCORSOrigins([]string{"http://localhost:3000"}))
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected configured origin header, got %q", got)
	}
}

func TestCORSAllowsWildcardOrigin(t *testing.T) {
	router := NewRouter(app.NewService(), WithCORSOrigins([]string{"*"}))
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Origin", "http://phone.local:3000")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard origin header, got %q", got)
	}
}

func TestCORSHandlesPreflightForConfiguredOrigin(t *testing.T) {
	router := NewRouter(app.NewService(), WithCORSOrigins([]string{"http://localhost:3000"}))
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/documents", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected configured origin header, got %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Headers"); got != corsAllowedHeaders {
		t.Fatalf("expected allowed headers, got %q", got)
	}
}

func TestCORSPreflightDoesNotReflectRequestedHeaders(t *testing.T) {
	router := NewRouter(app.NewService(), WithCORSOrigins([]string{"http://localhost:3000"}))
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/documents", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "x-injected")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Headers"); got != corsAllowedHeaders {
		t.Fatalf("expected fixed allowed headers, got %q", got)
	}
}
