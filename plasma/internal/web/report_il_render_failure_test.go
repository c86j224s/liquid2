package web

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestExperimentalRenderFailureClosesTypedLineageWithoutProviderCall(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	executor := &fakeAgentExecutor{}
	svc := app.NewService(store)
	server := NewServer(svc, Options{AgentExecutor: executor, ReportILChromePath: filepath.Join(t.TempDir(), "missing-chrome")}).(*Server)
	missionID := createMissionForTest(t, ctx, svc)
	result, err := server.startReportDraft(ctx, missionID, reportDraftRequest{Title: "invalid renderer", PipelineFamily: reportilcontract.PipelineFamily})
	if err != nil {
		t.Fatal(err)
	}
	pending := result["pending_event"].(app.LedgerEvent)
	deadline := time.Now().Add(3 * time.Second)
	var events []app.LedgerEvent
	for time.Now().Before(deadline) {
		events, err = svc.ListEvents(ctx, missionID)
		if err != nil {
			t.Fatal(err)
		}
		if countLedgerEvents(events, "report.draft.failed") == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(executor.requests) != 0 {
		t.Fatalf("render preflight made provider calls: %#v", executor.requests)
	}
	var companion, terminal app.LedgerEvent
	for _, event := range events {
		switch event.EventType {
		case "report.il_render.failed":
			companion = event
		case "report.draft.failed":
			terminal = event
		}
	}
	if companion.EventID == "" || terminal.EventID == "" {
		t.Fatalf("expected render companion and draft terminal for pending %s, events=%#v", pending.EventID, events)
	}
	var companionPayload, terminalPayload map[string]any
	if err := json.Unmarshal(companion.Payload, &companionPayload); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(terminal.Payload, &terminalPayload); err != nil {
		t.Fatal(err)
	}
	if companionPayload["pending_event_id"] != pending.EventID || companionPayload["stage_id"] != "il_render" || companionPayload["terminal_event_id"] != terminal.EventID || companion.CorrelationID != terminal.EventID || terminalPayload["pending_event_id"] != pending.EventID || terminalPayload["failed_stage_kind"] != "il_render" || terminalPayload["stage_failure_event_id"] != companion.EventID {
		t.Fatalf("typed render failure lineage is incomplete: companion=%#v terminal=%#v", companionPayload, terminalPayload)
	}
	privatePath := server.reportILChromePath
	if strings.Contains(string(companion.Payload), privatePath) || strings.Contains(string(terminal.Payload), privatePath) {
		t.Fatal("typed render failure exposed configured Chrome path")
	}
}
