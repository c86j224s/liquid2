package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRetiredHumanizeLaunchReturnsGoneWithoutService(t *testing.T) {
	server := &Server{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/missions/mis_one/artifacts/art_one/humanized_markdown_export", strings.NewReader(`{}`))
	server.handleReportArtifactHumanizedMarkdownExport(response, request, "mis_one", "art_one")
	if response.Code != http.StatusGone {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "removed") {
		t.Fatalf("missing retirement explanation: %s", response.Body.String())
	}
}
