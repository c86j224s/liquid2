package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

const largePollingFixtureEvents = 240

// TestMissionPollingLargeFixtureMetrics is a reproducible, representative
// payload harness. It compares response bytes rather than timing; the exact
// byte log can vary slightly with generated timestamps, while the ratio check
// remains stable across developer machines.
func TestMissionPollingLargeFixtureMetrics(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	handler := NewServer(service, Options{}).(*Server)
	server := httptest.NewServer(handler)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Polling fixture"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	largePayload := `{"note":"` + strings.Repeat("evidence ", 160) + `"}`
	for index := 0; index < largePollingFixtureEvents; index++ {
		if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
			EventID:   fmt.Sprintf("evt_polling_fixture_%03d", index),
			MissionID: missionID,
			EventType: "mission.note",
			Producer:  app.Producer{Type: "test", ID: "polling-fixture"},
			Payload:   []byte(largePayload),
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_polling_fixture_pending", MissionID: missionID, EventType: "turn.agent.pending",
		Producer: app.Producer{Type: "test", ID: "polling-fixture"}, Payload: []byte(`{"user_event_id":"evt_polling_fixture_user"}`),
	}); err != nil {
		t.Fatal(err)
	}
	_, cancelTurn := context.WithCancel(context.Background())
	defer cancelTurn()
	runID := handler.runningTurns.start(missionID, "test", cancelTurn)
	defer handler.runningTurns.finish(missionID, runID)

	activityBytes := responseBytes(t, server.URL+"/api/missions/"+missionID+"/activity")
	detailBytes := responseBytes(t, server.URL+"/api/missions/"+missionID)
	if activityBytes >= detailBytes/20 {
		t.Fatalf("activity response = %d bytes, detail = %d bytes; expected activity to stay below 5%% of detail", activityBytes, detailBytes)
	}
	t.Logf("large fixture: evidence_events=%d detail_bytes=%d activity_bytes=%d", largePollingFixtureEvents, detailBytes, activityBytes)
}

func responseBytes(t *testing.T, url string) int {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d", url, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return len(body)
}
