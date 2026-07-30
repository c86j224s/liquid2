package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestReportRedpenRouteCreatesUpdatesDownloadsAndProtectsSource(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	missionID := "mis_redpen"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "Redpen mission"}); err != nil {
		t.Fatal(err)
	}
	source := createReportRedpenSource(t, ctx, svc, missionID)
	server := httptest.NewServer(NewServer(svc, Options{}))
	defer server.Close()
	endpoint := server.URL + "/api/missions/" + missionID + "/artifacts/" + source.ArtifactID + "/redpen"

	emptyState := getJSON(t, endpoint)
	if exists, _ := emptyState["exists"].(bool); exists {
		t.Fatalf("new report should not have a redpen workcopy: %#v", emptyState)
	}

	first := postJSON(t, endpoint, map[string]any{
		"content":                      "# 보고서\n\n첫 번째 교정 문장입니다.\n",
		"expected_current_artifact_id": "",
	})
	if exists, _ := first["exists"].(bool); !exists {
		t.Fatalf("first save should create a redpen workcopy: %#v", first)
	}
	if changed, _ := first["changed"].(bool); !changed {
		t.Fatalf("first save should report a change: %#v", first)
	}
	firstArtifactID := nestedString(t, first, "workcopy", "artifact", "artifact_id")
	if firstArtifactID == "" || nestedNumber(t, first, "workcopy", "revision") != 1 {
		t.Fatalf("unexpected first redpen response: %#v", first)
	}
	storedSource, err := svc.GetRawArtifact(ctx, source.ArtifactID)
	if err != nil || string(storedSource.Content) != "# 보고서\n\n원문 문장입니다.\n" {
		t.Fatalf("source report changed after redpen save: artifact=%#v err=%v", storedSource, err)
	}
	redpenRead := getJSON(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+firstArtifactID)
	if content, _ := redpenRead["content"].(string); !strings.Contains(content, "첫 번째 교정") {
		t.Fatalf("redpen artifact should be readable as a report: %#v", redpenRead)
	}

	status, stale := postJSONFailure(t, endpoint, map[string]any{
		"content":                      "stale update",
		"expected_current_artifact_id": source.ArtifactID,
	})
	if status != http.StatusConflict || !strings.Contains(nestedString(t, stale, "error", "message"), "another session") {
		t.Fatalf("expected stale redpen conflict, got %d %#v", status, stale)
	}

	second := postJSON(t, endpoint, map[string]any{
		"content":                      "# 보고서\n\n두 번째 교정 문장입니다.\n",
		"expected_current_artifact_id": firstArtifactID,
	})
	secondArtifactID := nestedString(t, second, "workcopy", "artifact", "artifact_id")
	if secondArtifactID == firstArtifactID || nestedNumber(t, second, "workcopy", "revision") != 2 {
		t.Fatalf("unexpected second redpen response: %#v", second)
	}
	if nestedString(t, second, "workcopy", "workcopy_id") != nestedString(t, first, "workcopy", "workcopy_id") {
		t.Fatalf("updates must preserve one logical workcopy: first=%#v second=%#v", first, second)
	}

	noOp := postJSON(t, endpoint, map[string]any{
		"content":                      "# 보고서\n\n두 번째 교정 문장입니다.\n",
		"expected_current_artifact_id": secondArtifactID,
	})
	if changed, _ := noOp["changed"].(bool); changed || nestedNumber(t, noOp, "workcopy", "revision") != 2 {
		t.Fatalf("same-content save should not create a revision: %#v", noOp)
	}
	reverted := postJSON(t, endpoint, map[string]any{
		"content":                      "# 보고서\n\n원문 문장입니다.\n",
		"expected_current_artifact_id": secondArtifactID,
	})
	if nestedString(t, reverted, "workcopy", "artifact", "artifact_id") != source.ArtifactID || nestedNumber(t, reverted, "workcopy", "revision") != 3 {
		t.Fatalf("revert should reuse source bytes as a new workcopy revision: %#v", reverted)
	}

	latest := getJSON(t, endpoint)
	if content, _ := latest["content"].(string); !strings.Contains(content, "원문 문장") {
		t.Fatalf("redpen read did not return the latest revision: %#v", latest)
	}
	download, err := http.Get(endpoint + "/download")
	if err != nil {
		t.Fatal(err)
	}
	defer download.Body.Close()
	body, err := io.ReadAll(download.Body)
	if err != nil {
		t.Fatal(err)
	}
	if download.StatusCode != http.StatusOK || !strings.Contains(string(body), "원문 문장") || !strings.Contains(download.Header.Get("Content-Disposition"), "redpen.md") {
		t.Fatalf("unexpected redpen download: status=%d disposition=%q body=%q", download.StatusCode, download.Header.Get("Content-Disposition"), body)
	}

	status, _ = getJSONFailure(t, server.URL+"/api/missions/"+missionID+"/artifacts/"+secondArtifactID+"/redpen")
	if status != http.StatusNotFound {
		t.Fatalf("redpen artifact must not become a new redpen source, got %d", status)
	}
	otherMissionID := "mis_redpen_other"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: otherMissionID, Title: "Other"}); err != nil {
		t.Fatal(err)
	}
	status, _ = getJSONFailure(t, server.URL+"/api/missions/"+otherMissionID+"/artifacts/"+source.ArtifactID+"/redpen")
	if status != http.StatusNotFound {
		t.Fatalf("cross-mission redpen lookup must be hidden, got %d", status)
	}

	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	redpenEvents := 0
	for _, event := range events {
		if event.EventType != app.ReportRedpenSavedEvent {
			continue
		}
		redpenEvents++
		if strings.Contains(string(event.Payload), "교정 문장") || strings.Contains(string(event.Payload), `"content"`) {
			t.Fatalf("redpen event leaked edited content: %s", event.Payload)
		}
	}
	if redpenEvents != 3 {
		t.Fatalf("expected three changed redpen revisions, got %d", redpenEvents)
	}
}

func createReportRedpenSource(t *testing.T, ctx context.Context, svc *app.Service, missionID string) app.RawArtifact {
	t.Helper()
	artifact, event, err := svc.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_redpen_source", MissionID: missionID,
		MediaType: "text/markdown; charset=utf-8", Filename: "report.md",
		Producer: app.Producer{Type: "agent", ID: "reporter"}, Content: []byte("# 보고서\n\n원문 문장입니다.\n"),
	}, func(artifact app.RawArtifact) app.AppendEventRequest {
		payload, _ := json.Marshal(map[string]any{
			"kind": "markdown_report_artifact", "artifact_id": artifact.ArtifactID, "title": "보고서",
		})
		return app.AppendEventRequest{
			EventID: "evt_redpen_source", MissionID: missionID, EventType: "report.artifact.created",
			Producer: app.Producer{Type: "agent", ID: "reporter"}, Payload: payload,
		}
	})
	if err != nil {
		t.Fatalf("create report source: %v", err)
	}
	if event.EventType != "report.artifact.created" {
		t.Fatalf("unexpected source event: %#v", event)
	}
	return artifact
}

func nestedNumber(t *testing.T, value map[string]any, keys ...string) float64 {
	t.Helper()
	current := any(value)
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("expected object at %q in %#v", key, value)
		}
		current = object[key]
	}
	number, ok := current.(float64)
	if !ok {
		t.Fatalf("expected number at %v in %#v", keys, value)
	}
	return number
}
