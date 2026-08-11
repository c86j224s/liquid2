package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestReportArtifactDeletePreviewAndDeleteRoutes(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report delete"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	artifactID := seedHTTPCompletedReportRun(t, ctx, service, missionID)

	preview := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report_delete_preview")
	if preview["eligible"] != true || preview["run_id"] != "evt_http_report_pending" ||
		preview["deletable_event_count"] != float64(2) || preview["deletable_artifact_count"] != float64(1) ||
		strings.TrimSpace(preview["delete_facts_hash"].(string)) == "" {
		t.Fatalf("unexpected delete preview: %#v", preview)
	}
	status, failure := deleteJSONBodyFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report", map[string]any{
		"confirm_artifact_id": "art_other",
		"expected_revision":   int(preview["revision"].(float64)),
		"delete_facts_hash":   preview["delete_facts_hash"],
	})
	if status != http.StatusBadRequest || !strings.Contains(nestedString(t, failure, "error", "message"), "confirmation") {
		t.Fatalf("expected confirmation mismatch, got %d %#v", status, failure)
	}
	status, failure = deleteJSONBodyFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report", map[string]any{
		"confirm_artifact_id": artifactID,
		"expected_revision":   int(preview["revision"].(float64)) + 1,
		"delete_facts_hash":   preview["delete_facts_hash"],
	})
	if status != http.StatusConflict || !strings.Contains(nestedString(t, failure, "error", "message"), "revision") {
		t.Fatalf("expected revision conflict, got %d %#v", status, failure)
	}
	deleted := deleteJSONBody(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report", map[string]any{
		"confirm_artifact_id": artifactID,
		"expected_revision":   int(preview["revision"].(float64)),
		"delete_facts_hash":   preview["delete_facts_hash"],
	})
	if deleted["deleted"] != true || deleted["run_id"] != "evt_http_report_pending" {
		t.Fatalf("unexpected delete result: %#v", deleted)
	}
	status, _ = getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID)
	if status != http.StatusNotFound {
		t.Fatalf("expected purged artifact read 404, got %d", status)
	}
}

func TestReportDeleteRoutesRejectDerivativeTargetAndDeleteItFromFinal(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report derivative delete"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	artifactID := seedHTTPCompletedReportRun(t, ctx, service, missionID)
	before := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report_delete_preview")
	htmlExport := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/html_export", map[string]any{})
	htmlArtifactID := nestedString(t, htmlExport, "artifact", "artifact_id")
	if htmlArtifactID == "" {
		t.Fatalf("HTML export did not return artifact id: %#v", htmlExport)
	}
	if status, _ := getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+htmlArtifactID+"/report_delete_preview"); status != http.StatusNotFound {
		t.Fatalf("derivative report delete preview should be 404, got %d", status)
	}
	after := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report_delete_preview")
	if after["eligible"] != true || after["deletable_artifact_count"] != float64(2) ||
		after["revision"].(float64) <= before["revision"].(float64) {
		t.Fatalf("HTML export should advance revision and be deleted from final target: before=%#v after=%#v", before, after)
	}
	if status, _ := deleteJSONBodyFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report", map[string]any{
		"confirm_artifact_id": artifactID,
		"expected_revision":   int(before["revision"].(float64)),
		"delete_facts_hash":   before["delete_facts_hash"],
	}); status != http.StatusConflict {
		t.Fatalf("stale delete after HTML export should conflict, got %d", status)
	}
	deleteJSONBody(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report", map[string]any{
		"confirm_artifact_id": artifactID,
		"expected_revision":   int(after["revision"].(float64)),
		"delete_facts_hash":   after["delete_facts_hash"],
	})
	if status, _ := getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+htmlArtifactID); status != http.StatusNotFound {
		t.Fatalf("HTML derivative artifact should be deleted with final report, got %d", status)
	}
}

func TestReportDeleteRoutesRedpenCommitAdvancesRevisionAndDeletesWorkcopy(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report redpen delete"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	artifactID := seedHTTPCompletedReportRun(t, ctx, service, missionID)
	before := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report_delete_preview")
	redpen := postJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/redpen", map[string]any{
		"content":                      "# Report\n\nEdited.",
		"expected_current_artifact_id": "",
	})
	redpenArtifactID := nestedString(t, redpen, "workcopy", "artifact", "artifact_id")
	if redpenArtifactID == "" {
		t.Fatalf("redpen save did not return artifact id: %#v", redpen)
	}
	after := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report_delete_preview")
	if after["eligible"] != true || after["deletable_artifact_count"] != float64(2) ||
		after["revision"].(float64) <= before["revision"].(float64) {
		t.Fatalf("redpen save should advance revision and be deleted from final target: before=%#v after=%#v", before, after)
	}
	if status, _ := deleteJSONBodyFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report", map[string]any{
		"confirm_artifact_id": artifactID,
		"expected_revision":   int(before["revision"].(float64)),
		"delete_facts_hash":   before["delete_facts_hash"],
	}); status != http.StatusConflict {
		t.Fatalf("stale delete after redpen should conflict, got %d", status)
	}
	deleteJSONBody(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report", map[string]any{
		"confirm_artifact_id": artifactID,
		"expected_revision":   int(after["revision"].(float64)),
		"delete_facts_hash":   after["delete_facts_hash"],
	})
	if status, _ := getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+redpenArtifactID); status != http.StatusNotFound {
		t.Fatalf("redpen artifact should be deleted with final report, got %d", status)
	}
}

func TestReportDeleteRouteConflictsWhenFactsHashChanges(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Report hash conflict"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	artifactID := seedHTTPCompletedReportRun(t, ctx, service, missionID)
	preview := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report_delete_preview")
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_http_report_hash_reference",
		MissionID: missionID,
		EventType: "mission.note",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   []byte(`{"artifact_id":"` + artifactID + `"}`),
	}); err != nil {
		t.Fatalf("AppendEvent reference returned error: %v", err)
	}
	status, failure := deleteJSONBodyFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+artifactID+"/report", map[string]any{
		"confirm_artifact_id": artifactID,
		"expected_revision":   int(preview["revision"].(float64)),
		"delete_facts_hash":   preview["delete_facts_hash"],
	})
	if status != http.StatusConflict || !strings.Contains(nestedString(t, failure, "error", "message"), "facts") {
		t.Fatalf("expected facts hash conflict, got %d %#v", status, failure)
	}
}

func seedHTTPCompletedReportRun(t *testing.T, ctx context.Context, service *app.Service, missionID string) string {
	t.Helper()
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_http_report_pending",
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   mustJSON(map[string]any{"title": "HTTP report"}),
	}); err != nil {
		t.Fatalf("AppendEvent returned error: %v", err)
	}
	artifact, _, err := service.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_http_report_final",
		MissionID:  missionID,
		MediaType:  "text/markdown",
		Filename:   "report.md",
		Producer:   app.Producer{Type: "agent", ID: "test"},
		Content:    []byte("# Report"),
	}, func(artifact app.RawArtifact) app.AppendEventRequest {
		return app.AppendEventRequest{
			EventID:   "evt_http_report_final",
			MissionID: missionID,
			EventType: "report.artifact.created",
			Producer:  app.Producer{Type: "agent", ID: "test"},
			Payload: mustJSON(map[string]any{
				"kind":             "markdown_report_artifact",
				"pending_event_id": "evt_http_report_pending",
				"artifact_id":      artifact.ArtifactID,
			}),
		}
	})
	if err != nil {
		t.Fatalf("CreateRawArtifactWithEvent returned error: %v", err)
	}
	return artifact.ArtifactID
}
