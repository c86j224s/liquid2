package app_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestAppendReportTerminalIfOpenClosesPendingOnceConcurrently(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	const missionID = "mis_terminal_race"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "terminal race"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{{
		EventID:   "evt_pending",
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "agent", ID: "codex"},
		Payload:   jsonPayload(map[string]any{"report_mode": "long_form"}),
	}}); err != nil {
		t.Fatal(err)
	}

	terminal := func(id, kind string) []app.AppendEventRequest {
		return []app.AppendEventRequest{{
			EventID:   id,
			MissionID: missionID,
			EventType: "report.draft.failed",
			Producer:  app.Producer{Type: "agent", ID: kind},
			Payload:   jsonPayload(map[string]any{"kind": kind, "pending_event_id": "evt_pending"}),
		}}
	}
	type result struct {
		ok  bool
		err error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i, kind := range []string{"worker_failed", "user_canceled"} {
		wg.Add(1)
		go func(i int, kind string) {
			defer wg.Done()
			_, ok, err := svc.AppendReportTerminalIfOpen(ctx, missionID, "evt_pending", terminal("evt_terminal_"+string(rune('a'+i)), kind))
			results <- result{ok: ok, err: err}
		}(i, kind)
	}
	wg.Wait()
	close(results)

	var winners int
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.ok {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly one terminal winner, got %d", winners)
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected pending plus one terminal event, got %d events", len(events))
	}
}

func TestAppendReportTerminalIfOpenRejectsWrongPendingTypeAndCorrelation(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	const missionID = "mis_terminal_validation"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "terminal validation"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{EventID: "evt_design_pending", MissionID: missionID, EventType: "report.design.pending", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: jsonPayload(map[string]any{"source_artifact_id": "art_1"})}); err != nil {
		t.Fatal(err)
	}
	for _, req := range []app.AppendEventRequest{
		{EventID: "evt_wrong_type", MissionID: missionID, EventType: "report.patch.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: jsonPayload(map[string]any{"pending_event_id": "evt_design_pending"})},
		{EventID: "evt_wrong_correlation", MissionID: missionID, EventType: "report.design.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: jsonPayload(map[string]any{"pending_event_id": "evt_other"})},
	} {
		if _, _, err := svc.AppendReportTerminalIfOpen(ctx, missionID, "evt_design_pending", []app.AppendEventRequest{req}); err == nil {
			t.Fatalf("expected conditional terminal validation error for %s", req.EventID)
		}
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("invalid closures must not append, got %#v", events)
	}
}

func jsonPayload(value map[string]any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}
