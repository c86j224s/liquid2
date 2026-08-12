package web

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestMissionUsageRouteIsMissionScoped(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	server := httptest.NewServer(NewServer(service, Options{}))
	defer server.Close()

	first := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Measured"})
	firstID := nestedString(t, first, "projection", "mission_id")
	second := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Empty"})
	secondID := nestedString(t, second, "projection", "mission_id")
	if _, err := service.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_usage", MissionID: firstID, EventType: "turn.agent.response",
		Producer: app.Producer{Type: "agent", ID: "test"},
		Payload: mustJSON(map[string]any{"workflow_run_id": "wfr_test", "agent_usage": map[string]any{
			"schema_version": 2, "surface": "workflow_step", "model": "gpt-test", "reasoning_effort": "high",
			"session":        map[string]any{"agent_session_id": "session-test", "resumed": true},
			"provider_usage": map[string]any{"scope": "call", "input_tokens": 10, "output_tokens": 2, "total_tokens": 12},
		}}),
	}); err != nil {
		t.Fatal(err)
	}

	measured := nestedMap(t, getJSON(t, server.URL+"/api/missions/"+firstID+"/usage"), "usage")
	if measured["total_tokens"] != float64(12) || measured["usage_available_count"] != float64(1) {
		t.Fatalf("unexpected measured usage: %#v", measured)
	}
	runs, ok := measured["workflow_runs"].([]any)
	if !ok || len(runs) != 1 {
		t.Fatalf("expected workflow run usage: %#v", measured)
	}
	run, ok := runs[0].(map[string]any)
	if !ok || run["workflow_run_id"] != "wfr_test" || run["resumed_call_count"] != float64(1) || run["agent_model"] != "gpt-test" {
		t.Fatalf("unexpected workflow run usage: %#v", runs[0])
	}
	empty := nestedMap(t, getJSON(t, server.URL+"/api/missions/"+secondID+"/usage"), "usage")
	if empty["total_tokens"] != float64(0) || empty["usage_record_count"] != float64(0) {
		t.Fatalf("empty mission inherited usage: %#v", empty)
	}
}
