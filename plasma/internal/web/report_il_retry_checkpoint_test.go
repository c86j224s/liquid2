package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportilphase0"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestReportILLongFormRetryResumesFromFinalCheckpoint(t *testing.T) {
	testReportILLongFormRetry(t, true)
}

func TestReportILLongFormRetryRecoversLegacyFinalArtifact(t *testing.T) {
	testReportILLongFormRetry(t, false)
}

func testReportILLongFormRetry(t *testing.T, durableCheckpoint bool) {
	chromePath := testChromePath()
	if chromePath == "" {
		t.Skip("Chrome or Chromium is required")
	}
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := app.NewService(store)
	synthetic := &reportILSyntheticExecutor{service: service}
	observer := &reportILAcceptanceExecutor{delegate: synthetic}
	server := httptest.NewServer(NewServer(service, Options{AgentExecutor: observer, ReportILChromePath: chromePath}))
	defer server.Close()

	missionID, originalPendingID, err := startReportILAcceptanceHTTP(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	terminal, events, err := waitReportILAcceptanceTerminal(ctx, service, missionID, originalPendingID, 20*time.Second)
	if err != nil || terminal.EventType != "report.artifact.created" {
		t.Fatalf("initial IL run: terminal=%#v err=%v", terminal, err)
	}
	failedPendingID := originalPendingID
	if durableCheckpoint {
		checkpointEvent := lastReportILCheckpointEvent(t, events, originalPendingID, "il_long_form_final")
		failedPendingID = "evt_retry_checkpoint_failed"
		failedTerminalID := "evt_retry_checkpoint_terminal"
		checkpointPayload := map[string]any{}
		if err := json.Unmarshal(checkpointEvent.Payload, &checkpointPayload); err != nil {
			t.Fatal(err)
		}
		checkpointPayload["pending_event_id"] = failedPendingID
		checkpoint := checkpointPayload["checkpoint"].(map[string]any)
		checkpoint["pending_event_id"] = failedPendingID
		if _, err := service.AppendEvents(ctx, missionID, []ledger.AppendRequest{
			{EventID: failedPendingID, MissionID: missionID, EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "test"}, Payload: checkpointTestJSON(map[string]any{
				"title": "Experimental IL product acceptance", "report_mode": "long_form", "pipeline_family": reportilcontract.PipelineFamily,
				"pipeline_graph": "report_il_validation_profiles_v7", "rigor_level": "strict", "rigor_label": "검증형",
				"origin_pending_event_id": failedPendingID, "attempt_number": 1, "retry_strategy": "initial",
			})},
			{EventID: "evt_retry_checkpoint", MissionID: missionID, EventType: "report.il.checkpoint.created", CausationEventID: failedPendingID, CorrelationID: failedPendingID, Producer: ledger.Producer{Type: "system", ID: "report-il"}, Payload: checkpointTestJSON(checkpointPayload)},
			{EventID: failedTerminalID, MissionID: missionID, EventType: "report.draft.failed", Producer: ledger.Producer{Type: "agent", ID: "codex"}, Payload: checkpointTestJSON(map[string]any{
				"pending_event_id": failedPendingID, "kind": "report_draft_failed", "failed_stage_kind": "il_reader", "failed_stage_id": "il_reader",
			})},
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := service.LoadReportILResumeCheckpoint(ctx, missionID, failedPendingID); err != nil {
			t.Fatalf("checkpoint preflight: %v", err)
		}
	} else {
		failedPendingID = "evt_retry_legacy_failed"
		if _, err := service.AppendEvents(ctx, missionID, []ledger.AppendRequest{
			{EventID: failedPendingID, MissionID: missionID, EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "test"}, Payload: checkpointTestJSON(map[string]any{
				"title": "Experimental IL product acceptance", "report_mode": "long_form", "pipeline_family": reportilcontract.PipelineFamily,
				"pipeline_graph": "report_il_validation_profiles_v7", "rigor_level": "strict", "rigor_label": "검증형",
				"origin_pending_event_id": failedPendingID, "attempt_number": 1, "retry_strategy": "initial",
			})},
			{EventID: "evt_legacy_final_completed", MissionID: missionID, EventType: "report.il_long_form_final.completed", Producer: ledger.Producer{Type: "system", ID: "report-il"}, Payload: checkpointTestJSON(map[string]any{"pending_event_id": failedPendingID})},
			{EventID: "evt_legacy_reader_failed", MissionID: missionID, EventType: "report.draft.failed", Producer: ledger.Producer{Type: "agent", ID: "codex"}, Payload: checkpointTestJSON(map[string]any{
				"pending_event_id": failedPendingID, "kind": "report_draft_failed", "failed_stage_kind": "il_reader", "failed_stage_id": "il_reader",
			})},
		}); err != nil {
			t.Fatal(err)
		}
		legacyEvents, err := service.ListEvents(ctx, missionID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := reportilphase0.RecoverLegacyCheckpoint(ctx, missionID, failedPendingID, legacyEvents, service.ReportILSourceReader(), service); err != nil {
			t.Fatalf("legacy checkpoint preflight: %v", err)
		}
	}
	before := observer.snapshot()
	observer.delegate = &reportILRetrySyntheticExecutor{delegate: synthetic}
	retry, err := reportILAcceptancePostJSON(server.URL+"/api/missions/"+missionID+"/reports/retry", map[string]any{
		"failed_pending_event_id": failedPendingID, "strategy": "resume_failed", "retry_request_id": "retry-checkpoint-e2e",
	})
	if err != nil {
		t.Fatal(err)
	}
	retryPendingID, err := acceptanceNestedString(retry, "pending_event", "EventID")
	if err != nil {
		t.Fatal(err)
	}
	retryTerminal, retryEventsLedger, err := waitReportILRetryTerminal(ctx, service, missionID, retryPendingID, 20*time.Second)
	if err != nil || retryTerminal.EventType != "report.artifact.created" {
		debugEvents := retryEventsLedger
		if failure := latestReportILFailureCause(debugEvents, retryPendingID); failure != "" {
			t.Logf("retry failure cause: %s", failure)
		}
		failures := make([]string, 0)
		retryEvents := make([]string, 0)
		for _, event := range debugEvents {
			var eventPayload map[string]any
			_ = json.Unmarshal(event.Payload, &eventPayload)
			if event.EventID == retryPendingID || eventPayload["pending_event_id"] == retryPendingID {
				retryEvents = append(retryEvents, event.EventType)
			}
			if event.EventType == "report.draft.failed" || strings.HasSuffix(event.EventType, ".failed") {
				var payload map[string]any
				_ = json.Unmarshal(event.Payload, &payload)
				if payload["pending_event_id"] == retryPendingID {
					failures = append(failures, fmt.Sprintf("%s:%v", event.EventType, payload))
				}
			}
		}
		t.Fatalf("resumed IL run: terminal=%#v err=%v failures=%#v calls=%#v events=%#v", retryTerminal, err, failures, observer.snapshot()[len(before):], retryEvents)
	}
	after := observer.snapshot()
	newCalls := after[len(before):]
	if len(newCalls) != 2 || newCalls[0].Stage != "il_reader" || newCalls[1].Stage != "il_continuity" {
		t.Fatalf("resume regenerated completed authoring stages: %#v", newCalls)
	}
}

type reportILRetrySyntheticExecutor struct {
	delegate *reportILSyntheticExecutor
}

func (executor *reportILRetrySyntheticExecutor) Run(ctx context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	if strings.TrimPrefix(strings.TrimSpace(req.UserText), "report IL ") == "il_reader" {
		server, err := executor.delegate.syntheticServer(req)
		if err != nil {
			return agentexec.AgentResult{}, err
		}
		if err := readSyntheticEditorialMemory(ctx, server, ""); err != nil {
			return agentexec.AgentResult{}, err
		}
		if err := executor.delegate.reviseSyntheticPublicationDocument(ctx, server); err != nil {
			return agentexec.AgentResult{}, err
		}
		usage := agentusage.New("codex", "codex", req.Model, req.ReasoningEffort, req.Prompt).
			WithProviderUsage(agentusage.ProviderUsage{InputTokens: 10, OutputTokens: 10}, "synthetic_retry")
		return agentexec.AgentResult{Text: "workspace finalized", Usage: usage}, nil
	}
	return executor.delegate.Run(ctx, req)
}

func waitReportILRetryTerminal(ctx context.Context, service *app.Service, missionID, pendingID string, timeout time.Duration) (ledger.Event, []ledger.Event, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		events, err := service.ListEvents(ctx, missionID)
		if err != nil {
			return ledger.Event{}, nil, err
		}
		for _, event := range events {
			if event.EventType != "report.artifact.created" && event.EventType != "report.draft.failed" {
				continue
			}
			var payload struct {
				PendingID string `json:"pending_event_id"`
			}
			if json.Unmarshal(event.Payload, &payload) == nil && payload.PendingID == pendingID {
				return event, events, nil
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return ledger.Event{}, nil, fmt.Errorf("timed out waiting for retry terminal")
}

func latestReportILFailureCause(events []ledger.Event, pendingID string) string {
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		if event.EventType != "report.draft.failed" {
			continue
		}
		var payload struct {
			PendingID string `json:"pending_event_id"`
			Error     string `json:"error"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && payload.PendingID == pendingID {
			return payload.Error
		}
	}
	return ""
}

func checkpointTestJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}

func lastReportILCheckpointEvent(t *testing.T, events []ledger.Event, pendingID, stage string) ledger.Event {
	t.Helper()
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		if event.EventType != "report.il.checkpoint.created" {
			continue
		}
		var payload struct {
			PendingID  string `json:"pending_event_id"`
			Checkpoint struct {
				Stage string `json:"stage"`
			} `json:"checkpoint"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && payload.PendingID == pendingID && payload.Checkpoint.Stage == stage {
			return event
		}
	}
	t.Fatal(fmt.Sprintf("checkpoint %s not found", stage))
	return ledger.Event{}
}
