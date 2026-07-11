package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

type workflowStore struct {
	fakeStore
	events []LedgerEvent
}

func (s *workflowStore) AppendLedgerEvent(_ context.Context, event LedgerEvent) (LedgerEvent, error) {
	event.Sequence = int64(len(s.events) + 1)
	s.events = append(s.events, event)
	return event, nil
}

func (s *workflowStore) ListLedgerEvents(_ context.Context, missionID string) ([]LedgerEvent, error) {
	var events []LedgerEvent
	for _, event := range s.events {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

func TestRequestWorkflowRunAppendsRequestedEventAndProjectsQueuedRun(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)

	view, err := svc.RequestWorkflowRun(context.Background(), RequestWorkflowRunRequest{
		WorkflowRunID:       "wfr_test",
		MissionID:           "mis_1",
		RequestedBySurface:  WorkflowSurfaceWeb,
		AgentExecutor:       "codex",
		MCPMode:             "auto",
		StepInstructionMode: WorkflowStepInstructionModeLayered,
		UserInstructionRaw:  "다각도로 조사해줘",
		RunGoal:             "가능성을 열어두고 조사한다.",
		Instruction:         "Compare the pinned sources and summarize the next useful step.",
		MaxSteps:            2,
		MaxDurationMS:       60000,
		StopCondition:       "stop after two steps",
	})
	if err != nil {
		t.Fatalf("RequestWorkflowRun returned error: %v", err)
	}
	if view.WorkflowRunID != "wfr_test" || view.Status != WorkflowStatusQueued {
		t.Fatalf("unexpected workflow view: %#v", view)
	}
	if view.UserInstructionRaw != "다각도로 조사해줘" || view.RunGoal != "가능성을 열어두고 조사한다." {
		t.Fatalf("expected layered workflow intent in projection, got %#v", view)
	}
	if view.StepInstructionMode != WorkflowStepInstructionModeLayered {
		t.Fatalf("expected layered step instruction mode, got %#v", view)
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one event, got %d", len(store.events))
	}
	event := store.events[0]
	if event.EventType != WorkflowRunRequestedEvent {
		t.Fatalf("expected requested event, got %q", event.EventType)
	}
	var payload WorkflowRunRequestedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("payload did not unmarshal: %v", err)
	}
	if payload.WorkflowRunID != "wfr_test" || payload.MissionID != "mis_1" || payload.MaxSteps != 2 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if payload.UserInstructionRaw != "다각도로 조사해줘" || payload.RunGoal != "가능성을 열어두고 조사한다." {
		t.Fatalf("expected layered workflow intent in payload, got %#v", payload)
	}
	if payload.StepInstructionMode != WorkflowStepInstructionModeLayered {
		t.Fatalf("expected layered step instruction mode in payload, got %#v", payload)
	}
}

func TestRequestWorkflowRunDefaultsBudgetAndUsesLayeredInstructionMode(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)

	view, err := svc.RequestWorkflowRun(context.Background(), RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_default_budget",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceWeb,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "Make bounded progress.",
	})
	if err != nil {
		t.Fatalf("RequestWorkflowRun returned error: %v", err)
	}
	if view.MaxSteps != 10 || view.MaxDurationMS != int64((25*time.Minute)/time.Millisecond) {
		t.Fatalf("unexpected default workflow budget: %#v", view)
	}
	if view.StepInstructionMode != WorkflowStepInstructionModeLayered {
		t.Fatalf("expected layered step instruction mode by default, got %#v", view)
	}
	if view.UserInstructionRaw != "Make bounded progress." || view.RunGoal != "Make bounded progress." {
		t.Fatalf("layered mode should project original request and run goal fallbacks: %#v", view)
	}
	var payload WorkflowRunRequestedPayload
	if err := json.Unmarshal(store.events[0].Payload, &payload); err != nil {
		t.Fatalf("payload did not unmarshal: %v", err)
	}
	if payload.StepInstructionMode != WorkflowStepInstructionModeLayered {
		t.Fatalf("expected layered step instruction mode in payload, got %#v", payload)
	}
	if payload.UserInstructionRaw != "Make bounded progress." || payload.RunGoal != "Make bounded progress." {
		t.Fatalf("layered mode should store original request and run goal fallbacks: %#v", payload)
	}
}

func TestRequestWorkflowRunTreatsCurrentModeAsLayeredCompatibilityInput(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)

	view, err := svc.RequestWorkflowRun(context.Background(), RequestWorkflowRunRequest{
		WorkflowRunID:       "wfr_legacy_current",
		MissionID:           "mis_1",
		RequestedBySurface:  WorkflowSurfaceWeb,
		AgentExecutor:       "codex",
		MCPMode:             "auto",
		StepInstructionMode: WorkflowStepInstructionModeCurrent,
		UserInstructionRaw:  "원문 요청",
		RunGoal:             "도출 목표",
		Instruction:         "이번 스텝",
	})
	if err != nil {
		t.Fatalf("RequestWorkflowRun returned error: %v", err)
	}
	if view.StepInstructionMode != WorkflowStepInstructionModeLayered {
		t.Fatalf("expected current compatibility input to be stored as layered, got %#v", view)
	}
	if view.UserInstructionRaw != "원문 요청" || view.RunGoal != "도출 목표" {
		t.Fatalf("expected layered intent fields to survive current compatibility input: %#v", view)
	}
}

func TestRequestWorkflowRunRejectsExistingNonTerminalRun(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.RequestWorkflowRun(ctx, RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_first",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceWeb,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "first",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	}); err != nil {
		t.Fatalf("first RequestWorkflowRun returned error: %v", err)
	}
	_, err := svc.RequestWorkflowRun(ctx, RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_second",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceCLI,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "second",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input for duplicate active workflow, got %v", err)
	}
	if len(store.events) != 1 {
		t.Fatalf("second request should not append an event, got %#v", store.events)
	}
}

func TestRequestWorkflowRunAllowsNewRunAfterTerminalEvent(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.RequestWorkflowRun(ctx, RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_first",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceWeb,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "first",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	}); err != nil {
		t.Fatalf("first RequestWorkflowRun returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_first_completed",
		MissionID: "mis_1",
		EventType: WorkflowRunCompletedEvent,
		Producer:  Producer{Type: "workflow", ID: "wfr_first"},
		Payload: mustJSONRaw(WorkflowRunTerminalPayload{
			WorkflowRunID: "wfr_first",
			MissionID:     "mis_1",
			Reason:        "done",
		}),
	}); err != nil {
		t.Fatalf("append terminal event returned error: %v", err)
	}
	view, err := svc.RequestWorkflowRun(ctx, RequestWorkflowRunRequest{
		WorkflowRunID:             "wfr_second",
		MissionID:                 "mis_1",
		RequestedBySurface:        WorkflowSurfaceCLI,
		AgentExecutor:             "codex",
		MCPMode:                   "auto",
		Instruction:               "second",
		MaxSteps:                  1,
		MaxDurationMS:             60000,
		ContinueFromWorkflowRunID: "wfr_first",
	})
	if err != nil {
		t.Fatalf("second RequestWorkflowRun after terminal should succeed, got %v", err)
	}
	if view.ContinueFromWorkflowRunID != "wfr_first" {
		t.Fatalf("expected continuation source in projection, got %#v", view)
	}
	var payload WorkflowRunRequestedPayload
	if err := json.Unmarshal(store.events[len(store.events)-1].Payload, &payload); err != nil {
		t.Fatalf("payload did not unmarshal: %v", err)
	}
	if payload.ContinueFromWorkflowRunID != "wfr_first" {
		t.Fatalf("expected continuation source in payload, got %#v", payload)
	}
}

func TestRequestWorkflowRunRejectsInvalidStartAfterEvent(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	_, err := svc.RequestWorkflowRun(context.Background(), RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_invalid_start_after",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceMCP,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "run later",
		MaxSteps:           1,
		MaxDurationMS:      60000,
		StartAfterEventID:  "evt_missing",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input for missing start_after_event_id, got %v", err)
	}
}

func TestRequestWorkflowRunAcceptsOpenTurnStartAfterEvent(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_user_open",
		MissionID: "mis_1",
		EventType: "turn.user",
		Producer:  Producer{Type: "user", ID: "test"},
		Payload:   mustJSONRaw(map[string]any{"kind": "user_turn", "text": "start"}),
	}); err != nil {
		t.Fatalf("append turn.user returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_pending_open",
		MissionID: "mis_1",
		EventType: "turn.agent.pending",
		Producer:  Producer{Type: "agent", ID: "codex"},
		Payload:   mustJSONRaw(map[string]any{"user_event_id": "evt_user_open", "agent_executor": "codex"}),
	}); err != nil {
		t.Fatalf("append pending returned error: %v", err)
	}
	view, err := svc.RequestWorkflowRun(ctx, RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_open_start_after",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceMCP,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "run later",
		MaxSteps:           1,
		MaxDurationMS:      60000,
		StartAfterEventID:  "evt_user_open",
	})
	if err != nil {
		t.Fatalf("RequestWorkflowRun with open turn start_after_event_id returned error: %v", err)
	}
	if view.Status != WorkflowStatusQueued || view.StartAfterEventID != "evt_user_open" {
		t.Fatalf("unexpected workflow view: %#v", view)
	}
}

func TestRequestWorkflowStopClosesQueuedRunImmediately(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.RequestWorkflowRun(ctx, RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_queued_stop",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceWeb,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "queued run",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	}); err != nil {
		t.Fatalf("RequestWorkflowRun returned error: %v", err)
	}
	view, err := svc.RequestWorkflowStop(ctx, RequestWorkflowStopRequest{
		WorkflowRunID:      "wfr_queued_stop",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceWeb,
		Reason:             "user stop",
	})
	if err != nil {
		t.Fatalf("RequestWorkflowStop returned error: %v", err)
	}
	if view.Status != WorkflowStatusStopped || view.TerminalEventID == "" {
		t.Fatalf("expected queued run to become stopped, got %#v", view)
	}
	if got := workflowEventTypes(store.events); !equalStrings(got, []string{WorkflowRunRequestedEvent, WorkflowRunStopRequestedEvent, WorkflowRunStoppedEvent}) {
		t.Fatalf("unexpected workflow events: %#v", got)
	}
}

func TestBuildWorkflowRunTerminalAppendRequestBuildsStoppedEvent(t *testing.T) {
	now := time.Now().UTC()
	events := []LedgerEvent{
		workflowEvent("evt_req", WorkflowRunRequestedEvent, now, WorkflowRunRequestedPayload{
			WorkflowRunID:      "wfr_1",
			MissionID:          "mis_1",
			RequestedBySurface: WorkflowSurfaceWeb,
			AgentExecutor:      "codex",
			MCPMode:            "auto",
			Instruction:        "run",
			MaxSteps:           3,
			MaxDurationMS:      60000,
		}),
		workflowEvent("evt_step_done", WorkflowStepCompletedEvent, now.Add(time.Second), WorkflowStepCompletedPayload{
			WorkflowRunID:  "wfr_1",
			MissionID:      "mis_1",
			WorkflowStepID: "wfs_1",
			Decision:       "continue",
		}),
	}
	req, ok, err := BuildWorkflowRunTerminalAppendRequest(events, WorkflowRunTerminalEventRequest{
		WorkflowRunID: "wfr_1",
		MissionID:     "mis_1",
		EventType:     WorkflowRunStoppedEvent,
		Reason:        "user stop",
	})
	if err != nil || !ok {
		t.Fatalf("BuildWorkflowRunTerminalAppendRequest returned ok=%v err=%v", ok, err)
	}
	if req.EventType != WorkflowRunStoppedEvent || req.MissionID != "mis_1" ||
		req.Producer.Type != "workflow" || req.Producer.ID != "wfr_1" {
		t.Fatalf("unexpected terminal request shell: %#v", req)
	}
	var payload WorkflowRunTerminalPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		t.Fatalf("payload is not terminal payload: %v", err)
	}
	if payload.WorkflowRunID != "wfr_1" || payload.MissionID != "mis_1" ||
		payload.Reason != "user stop" || payload.StopReason != "user stop" ||
		payload.CompletedStepCount != 1 || payload.TerminalAt == "" {
		t.Fatalf("unexpected terminal payload: %#v", payload)
	}
}

func TestBuildWorkflowRunTerminalAppendRequestSkipsExistingTerminalEvent(t *testing.T) {
	now := time.Now().UTC()
	events := []LedgerEvent{
		workflowEvent("evt_req", WorkflowRunRequestedEvent, now, WorkflowRunRequestedPayload{
			WorkflowRunID:      "wfr_1",
			MissionID:          "mis_1",
			RequestedBySurface: WorkflowSurfaceWeb,
			AgentExecutor:      "codex",
			MCPMode:            "auto",
			Instruction:        "run",
			MaxSteps:           3,
			MaxDurationMS:      60000,
		}),
		workflowEvent("evt_terminal", WorkflowRunInterruptedEvent, now.Add(time.Second), WorkflowRunTerminalPayload{
			WorkflowRunID: "wfr_1",
			MissionID:     "mis_1",
			Reason:        "already done",
		}),
	}
	req, ok, err := BuildWorkflowRunTerminalAppendRequest(events, WorkflowRunTerminalEventRequest{
		WorkflowRunID: "wfr_1",
		MissionID:     "mis_1",
		EventType:     WorkflowRunStoppedEvent,
		Reason:        "user stop",
	})
	if err != nil || ok || req.EventID != "" {
		t.Fatalf("expected existing terminal event to be skipped, req=%#v ok=%v err=%v", req, ok, err)
	}
}

func TestClaimWorkflowRunStartOnlyClaimsOnce(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.RequestWorkflowRun(ctx, RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_claim",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceWeb,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "claim once",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	}); err != nil {
		t.Fatalf("RequestWorkflowRun returned error: %v", err)
	}
	view, claimed, err := svc.ClaimWorkflowRunStart(ctx, "mis_1", "wfr_claim", time.Now())
	if err != nil {
		t.Fatalf("ClaimWorkflowRunStart returned error: %v", err)
	}
	if !claimed || view.Status != WorkflowStatusRunning || view.StartedEventID == "" {
		t.Fatalf("expected first claim to start run, claimed=%v view=%#v", claimed, view)
	}
	view, claimed, err = svc.ClaimWorkflowRunStart(ctx, "mis_1", "wfr_claim", time.Now())
	if err != nil {
		t.Fatalf("second ClaimWorkflowRunStart returned error: %v", err)
	}
	if claimed || view.StartedEventID == "" {
		t.Fatalf("expected second claim to be ignored, claimed=%v view=%#v", claimed, view)
	}
	if got := workflowEventTypes(store.events); !equalStrings(got, []string{WorkflowRunRequestedEvent, WorkflowRunStartedEvent}) {
		t.Fatalf("unexpected workflow events: %#v", got)
	}
}

func TestAppendEventsIfNoActiveAgentWorkRejectsOpenAgentTurn(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_user_open",
		MissionID: "mis_1",
		EventType: "turn.user",
		Producer:  Producer{Type: "user", ID: "test"},
		Payload:   mustJSONRaw(map[string]any{"kind": "user_turn", "text": "open"}),
	}); err != nil {
		t.Fatalf("append turn.user returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_pending_open",
		MissionID: "mis_1",
		EventType: "turn.agent.pending",
		Producer:  Producer{Type: "agent", ID: "codex"},
		Payload:   mustJSONRaw(map[string]any{"user_event_id": "evt_user_open", "agent_executor": "codex"}),
	}); err != nil {
		t.Fatalf("append turn.agent.pending returned error: %v", err)
	}
	_, err := svc.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []AppendEventRequest{testTurnUserEventRequest("evt_user_next")})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected open agent turn rejection, got %v", err)
	}
	if len(store.events) != 2 {
		t.Fatalf("conditional append should not add events, got %#v", store.events)
	}
}

func TestAppendEventsIfNoActiveAgentWorkRejectsOpenReportDraft(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_report_pending",
		MissionID: "mis_1",
		EventType: "report.draft.pending",
		Producer:  Producer{Type: "user", ID: "test"},
		Payload:   mustJSONRaw(map[string]any{"kind": "markdown_report_artifact_pending"}),
	}); err != nil {
		t.Fatalf("append report.draft.pending returned error: %v", err)
	}
	_, err := svc.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []AppendEventRequest{testTurnUserEventRequest("evt_user_next")})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected open report draft rejection, got %v", err)
	}
	if len(store.events) != 1 {
		t.Fatalf("conditional append should not add events, got %#v", store.events)
	}
}

func TestAppendEventsIfNoActiveAgentWorkRejectsOpenReportDesign(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_report_design_pending",
		MissionID: "mis_1",
		EventType: "report.design.pending",
		Producer:  Producer{Type: "user", ID: "test"},
		Payload:   mustJSONRaw(map[string]any{"kind": "designed_html_report_pending", "source_artifact_id": "art_report"}),
	}); err != nil {
		t.Fatalf("append report.design.pending returned error: %v", err)
	}
	_, err := svc.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []AppendEventRequest{testTurnUserEventRequest("evt_user_next")})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected open report design rejection, got %v", err)
	}
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_report_design_done",
		MissionID: "mis_1",
		EventType: "report.artifact.exported",
		Producer:  Producer{Type: "agent", ID: "codex"},
		Payload: mustJSONRaw(map[string]any{
			"kind":             "designed_html_report_artifact",
			"pending_event_id": "evt_report_design_pending",
			"artifact_id":      "art_html",
			"target":           "designed_html",
		}),
	}); err != nil {
		t.Fatalf("append report.artifact.exported returned error: %v", err)
	}
	appended, err := svc.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []AppendEventRequest{testTurnUserEventRequest("evt_user_after_design")})
	if err != nil {
		t.Fatalf("expected completed report design not to block active work, got %v", err)
	}
	if len(appended) != 1 || appended[0].EventID != "evt_user_after_design" {
		t.Fatalf("unexpected appended events: %#v", appended)
	}
}

func TestAppendEventsIfNoActiveAgentWorkRejectsOpenReportHumanize(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_report_humanize_pending",
		MissionID: "mis_1",
		EventType: "report.humanize.pending",
		Producer:  Producer{Type: "agent", ID: "codex"},
		Payload: mustJSONRaw(map[string]any{
			"kind":                    "humanized_markdown_report_pending",
			"pending_event_id":        "evt_report_humanize_pending",
			"report_pending_event_id": "evt_report_pending",
			"source_artifact_id":      "art_report",
			"target":                  "humanized_markdown",
		}),
	}); err != nil {
		t.Fatalf("append report.humanize.pending returned error: %v", err)
	}
	_, err := svc.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []AppendEventRequest{testTurnUserEventRequest("evt_user_next")})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected open report humanize rejection, got %v", err)
	}
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_report_humanize_done",
		MissionID: "mis_1",
		EventType: "report.humanize.skipped",
		Producer:  Producer{Type: "agent", ID: "codex"},
		Payload: mustJSONRaw(map[string]any{
			"kind":               "humanized_markdown_report_skipped",
			"pending_event_id":   "evt_report_humanize_pending",
			"source_artifact_id": "art_report",
			"target":             "humanized_markdown",
		}),
	}); err != nil {
		t.Fatalf("append report.humanize.skipped returned error: %v", err)
	}
	appended, err := svc.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []AppendEventRequest{testTurnUserEventRequest("evt_user_after_humanize")})
	if err != nil {
		t.Fatalf("expected completed report humanize not to block active work, got %v", err)
	}
	if len(appended) != 1 || appended[0].EventID != "evt_user_after_humanize" {
		t.Fatalf("unexpected appended events: %#v", appended)
	}
}

func TestAppendEventsIfNoActiveAgentWorkRejectsOpenReportPatch(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_report_patch_pending",
		MissionID: "mis_1",
		EventType: "report.patch.pending",
		Producer:  Producer{Type: "user", ID: "test"},
		Payload: mustJSONRaw(map[string]any{
			"kind":             "markdown_report_patch_pending",
			"pending_event_id": "evt_report_patch_pending",
			"base_artifact_id": "art_report",
			"agent_executor":   "codex",
		}),
	}); err != nil {
		t.Fatalf("append report.patch.pending returned error: %v", err)
	}
	_, err := svc.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []AppendEventRequest{testTurnUserEventRequest("evt_user_next")})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected open report patch rejection, got %v", err)
	}
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_report_patch_failed",
		MissionID: "mis_1",
		EventType: "report.patch.failed",
		Producer:  Producer{Type: "agent", ID: "codex"},
		Payload: mustJSONRaw(map[string]any{
			"kind":             "report_patch_failed",
			"pending_event_id": "evt_report_patch_pending",
			"base_artifact_id": "art_report",
			"agent_executor":   "codex",
			"error":            "test failure",
		}),
	}); err != nil {
		t.Fatalf("append report.patch.failed returned error: %v", err)
	}
	appended, err := svc.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []AppendEventRequest{testTurnUserEventRequest("evt_user_after_patch")})
	if err != nil {
		t.Fatalf("expected completed report patch not to block active work, got %v", err)
	}
	if len(appended) != 1 || appended[0].EventID != "evt_user_after_patch" {
		t.Fatalf("unexpected appended events: %#v", appended)
	}
}

func TestAppendEventsIfNoActiveAgentWorkAllowsCompletedLegacyReportDraft(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_report_pending",
		MissionID: "mis_1",
		EventType: "report.draft.pending",
		Producer:  Producer{Type: "user", ID: "test"},
		Payload:   mustJSONRaw(map[string]any{"kind": "report_draft_pending"}),
	}); err != nil {
		t.Fatalf("append report.draft.pending returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_report_drafted",
		MissionID: "mis_1",
		EventType: "report.drafted",
		Producer:  Producer{Type: "agent", ID: "codex"},
		Payload: mustJSONRaw(map[string]any{
			"report_version_id": "rvn_1",
			"generation":        map[string]any{"pending_event_id": "evt_report_pending"},
		}),
	}); err != nil {
		t.Fatalf("append report.drafted returned error: %v", err)
	}
	appended, err := svc.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []AppendEventRequest{testTurnUserEventRequest("evt_user_next")})
	if err != nil {
		t.Fatalf("expected completed report draft not to block active work, got %v", err)
	}
	if len(appended) != 1 || appended[0].EventID != "evt_user_next" {
		t.Fatalf("unexpected appended events: %#v", appended)
	}
}

func TestAppendEventsIfNoActiveAgentWorkRejectsActiveWorkflow(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.RequestWorkflowRun(ctx, RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_active",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceCLI,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "active",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	}); err != nil {
		t.Fatalf("RequestWorkflowRun returned error: %v", err)
	}
	_, err := svc.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []AppendEventRequest{testTurnUserEventRequest("evt_user_next")})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected active workflow rejection, got %v", err)
	}
	if len(store.events) != 1 {
		t.Fatalf("conditional append should not add events, got %#v", store.events)
	}
}

func TestRequestWorkflowRunRejectsOpenReportDraft(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_report_pending",
		MissionID: "mis_1",
		EventType: "report.draft.pending",
		Producer:  Producer{Type: "user", ID: "test"},
		Payload:   mustJSONRaw(map[string]any{"kind": "markdown_report_artifact_pending"}),
	}); err != nil {
		t.Fatalf("append report.draft.pending returned error: %v", err)
	}
	_, err := svc.RequestWorkflowRun(ctx, RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_after_report",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceCLI,
		AgentExecutor:      "codex",
		MCPMode:            "auto",
		Instruction:        "should not start",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected report draft conflict, got %v", err)
	}
}

func TestAgentProviderLockRejectsMixedProviderTurnAppend(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	appendCompletedAgentTurn(t, svc, ctx, "mis_1", "codex")

	_, err := svc.AppendEventsIfNoActiveAgentWork(ctx, "mis_1", []AppendEventRequest{
		agentTurnUserEventRequest("evt_user_claude", "claude"),
		{
			EventID:   "evt_pending_claude",
			MissionID: "mis_1",
			EventType: "turn.agent.pending",
			Producer:  Producer{Type: "agent", ID: "claude"},
			Payload: mustJSONRaw(map[string]any{
				"kind":           "agent_pending",
				"user_event_id":  "evt_user_claude",
				"agent_executor": "claude",
			}),
		},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected provider lock rejection, got %v", err)
	}
	if !strings.Contains(err.Error(), "already using codex") {
		t.Fatalf("expected locked provider message, got %v", err)
	}
	if len(store.events) != 2 {
		t.Fatalf("mixed-provider append should not add events, got %#v", store.events)
	}
}

func TestAgentProviderLockRejectsMixedProviderWorkflowRequest(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	appendCompletedAgentTurn(t, svc, ctx, "mis_1", "codex")

	_, err := svc.RequestWorkflowRun(ctx, RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_claude",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceCLI,
		AgentExecutor:      "claude",
		MCPMode:            "auto",
		Instruction:        "run",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected provider lock rejection, got %v", err)
	}
	if !strings.Contains(err.Error(), "already using codex") {
		t.Fatalf("expected locked provider message, got %v", err)
	}
	if len(store.events) != 2 {
		t.Fatalf("mixed-provider workflow should not add events, got %#v", store.events)
	}
}

func TestAgentProviderLockRejectsMixedProviderDirectEvent(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	appendCompletedAgentTurn(t, svc, ctx, "mis_1", "codex")

	_, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_claude_response",
		MissionID: "mis_1",
		EventType: "turn.agent.response",
		Producer:  Producer{Type: "agent", ID: "claude"},
		Payload:   mustJSONRaw(map[string]any{"kind": "agent_response", "agent_executor": "claude", "text": "no"}),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected provider lock rejection, got %v", err)
	}
	if len(store.events) != 2 {
		t.Fatalf("mixed-provider direct append should not add events, got %#v", store.events)
	}
}

func TestAgentProviderLockRejectsInvalidExplicitProviderAppend(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)

	_, err := svc.AppendEvent(context.Background(), AppendEventRequest{
		EventID:   "evt_unknown_response",
		MissionID: "mis_1",
		EventType: "turn.agent.response",
		Producer:  Producer{Type: "agent", ID: "unknown"},
		Payload:   mustJSONRaw(map[string]any{"kind": "agent_response", "agent_executor": "unknown", "text": "no"}),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid provider rejection, got %v", err)
	}
	if len(store.events) != 0 {
		t.Fatalf("invalid provider append should not add events, got %#v", store.events)
	}
}

func TestRequestWorkflowRunRejectsInvalidProvider(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)

	_, err := svc.RequestWorkflowRun(context.Background(), RequestWorkflowRunRequest{
		WorkflowRunID:      "wfr_unknown",
		MissionID:          "mis_1",
		RequestedBySurface: WorkflowSurfaceCLI,
		AgentExecutor:      "unknown",
		MCPMode:            "auto",
		Instruction:        "run",
		MaxSteps:           1,
		MaxDurationMS:      60000,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid provider rejection, got %v", err)
	}
	if len(store.events) != 0 {
		t.Fatalf("invalid provider workflow should not add events, got %#v", store.events)
	}
}

func appendCompletedAgentTurn(t *testing.T, svc *Service, ctx context.Context, missionID string, executor string) {
	t.Helper()
	userEventID := "evt_user_" + executor
	if _, err := svc.AppendEvent(ctx, agentTurnUserEventRequest(userEventID, executor)); err != nil {
		t.Fatalf("append %s turn.user returned error: %v", executor, err)
	}
	if _, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_response_" + executor,
		MissionID: missionID,
		EventType: "turn.agent.response",
		Producer:  Producer{Type: "agent", ID: executor},
		Payload: mustJSONRaw(map[string]any{
			"kind":           "agent_response",
			"user_event_id":  userEventID,
			"agent_executor": executor,
			"text":           "done",
		}),
	}); err != nil {
		t.Fatalf("append %s turn.agent.response returned error: %v", executor, err)
	}
}

func agentTurnUserEventRequest(eventID string, executor string) AppendEventRequest {
	return AppendEventRequest{
		EventID:   eventID,
		MissionID: "mis_1",
		EventType: "turn.user",
		Producer:  Producer{Type: "user", ID: "test"},
		Payload:   mustJSONRaw(map[string]any{"kind": "user_turn", "text": "next", "agent_executor": executor}),
	}
}

func testTurnUserEventRequest(eventID string) AppendEventRequest {
	return AppendEventRequest{
		EventID:   eventID,
		MissionID: "mis_1",
		EventType: "turn.user",
		Producer:  Producer{Type: "user", ID: "test"},
		Payload:   mustJSONRaw(map[string]any{"kind": "user_turn", "text": "next"}),
	}
}

func TestAppendWorkflowEventValidatesPayloadContract(t *testing.T) {
	svc := NewService(&workflowStore{})
	_, err := svc.AppendEvent(context.Background(), AppendEventRequest{
		EventID:   "evt_bad_workflow",
		MissionID: "mis_1",
		EventType: WorkflowRunRequestedEvent,
		Producer:  Producer{Type: "workflow", ID: "web"},
		Payload: mustJSONRaw(map[string]any{
			"workflow_run_id":      "bad",
			"mission_id":           "mis_1",
			"requested_by_surface": WorkflowSurfaceWeb,
			"agent_executor":       "codex",
			"mcp_mode":             "auto",
			"instruction":          "run",
			"max_steps":            1,
			"max_duration_ms":      1000,
		}),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func workflowEventTypes(events []LedgerEvent) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType)
	}
	return types
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func TestWorkflowProjectionStatusTransitionsAndStopRequest(t *testing.T) {
	now := time.Now().UTC()
	events := []LedgerEvent{
		workflowEvent("evt_req", WorkflowRunRequestedEvent, now, WorkflowRunRequestedPayload{
			WorkflowRunID:      "wfr_1",
			MissionID:          "mis_1",
			RequestedBySurface: WorkflowSurfaceCLI,
			AgentExecutor:      "codex",
			MCPMode:            "auto",
			Instruction:        "run",
			MaxSteps:           3,
			MaxDurationMS:      60000,
			StopCondition:      "bounded",
			ArgumentSummary:    "run",
		}),
		workflowEvent("evt_started", WorkflowRunStartedEvent, now.Add(time.Second), WorkflowRunStartedPayload{
			WorkflowRunID: "wfr_1",
			MissionID:     "mis_1",
		}),
		workflowEvent("evt_step_started", WorkflowStepStartedEvent, now.Add(2*time.Second), WorkflowStepStartedPayload{
			WorkflowRunID:  "wfr_1",
			MissionID:      "mis_1",
			WorkflowStepID: "wfs_1",
			StepIndex:      1,
			Instruction:    "run one step",
		}),
		workflowEvent("evt_step_done", WorkflowStepCompletedEvent, now.Add(3*time.Second), WorkflowStepCompletedPayload{
			WorkflowRunID:  "wfr_1",
			MissionID:      "mis_1",
			WorkflowStepID: "wfs_1",
			Decision:       "continue",
			ResultEventID:  "evt_result",
		}),
		workflowEvent("evt_stop", WorkflowRunStopRequestedEvent, now.Add(4*time.Second), WorkflowRunStopRequestedPayload{
			WorkflowRunID:      "wfr_1",
			MissionID:          "mis_1",
			RequestedBySurface: WorkflowSurfaceWeb,
			Reason:             "user stopped",
		}),
	}
	runs := projectWorkflowRuns(events)
	if len(runs) != 1 {
		t.Fatalf("expected one run, got %#v", runs)
	}
	run := runs[0]
	if run.Status != WorkflowStatusStopping || run.CompletedStepCount != 1 || run.CurrentStep != nil {
		t.Fatalf("unexpected stopping projection: %#v", run)
	}
	if run.StopRequestedEventID != "evt_stop" || run.StopReason != "user stopped" {
		t.Fatalf("stop request was not projected: %#v", run)
	}
}

func TestWorkflowTerminalEventTakesPrecedenceOverStopRequest(t *testing.T) {
	now := time.Now().UTC()
	events := []LedgerEvent{
		workflowEvent("evt_req", WorkflowRunRequestedEvent, now, WorkflowRunRequestedPayload{
			WorkflowRunID:      "wfr_1",
			MissionID:          "mis_1",
			RequestedBySurface: WorkflowSurfaceWeb,
			AgentExecutor:      "codex",
			MCPMode:            "auto",
			Instruction:        "run",
			MaxSteps:           1,
			MaxDurationMS:      60000,
		}),
		workflowEvent("evt_stop", WorkflowRunStopRequestedEvent, now.Add(time.Second), WorkflowRunStopRequestedPayload{
			WorkflowRunID:      "wfr_1",
			MissionID:          "mis_1",
			RequestedBySurface: WorkflowSurfaceWeb,
			Reason:             "user stopped",
		}),
		workflowEvent("evt_completed", WorkflowRunCompletedEvent, now.Add(2*time.Second), WorkflowRunTerminalPayload{
			WorkflowRunID:      "wfr_1",
			MissionID:          "mis_1",
			Reason:             "agent declared complete",
			CompletedStepCount: 1,
		}),
	}
	run := projectWorkflowRuns(events)[0]
	if run.Status != WorkflowStatusCompleted || run.TerminalEventID != "evt_completed" {
		t.Fatalf("terminal event did not win: %#v", run)
	}
}

func TestWorkflowProjectionPausedRunCarriesContinuationInstruction(t *testing.T) {
	now := time.Now().UTC()
	events := []LedgerEvent{
		workflowEvent("evt_req", WorkflowRunRequestedEvent, now, WorkflowRunRequestedPayload{
			WorkflowRunID:      "wfr_1",
			MissionID:          "mis_1",
			RequestedBySurface: WorkflowSurfaceWeb,
			AgentExecutor:      "codex",
			MCPMode:            "auto",
			Instruction:        "run",
			MaxSteps:           1,
			MaxDurationMS:      60000,
		}),
		workflowEvent("evt_paused", WorkflowRunPausedEvent, now.Add(time.Second), WorkflowRunTerminalPayload{
			WorkflowRunID:      "wfr_1",
			MissionID:          "mis_1",
			Reason:             "max_steps reached",
			NextInstruction:    "read primary source",
			CompletedStepCount: 1,
		}),
	}
	run := projectWorkflowRuns(events)[0]
	if run.Status != WorkflowStatusPaused || run.TerminalEventID != "evt_paused" {
		t.Fatalf("paused terminal event was not projected: %#v", run)
	}
	if run.StopReason != "max_steps reached" || run.ContinuationInstruction != "read primary source" {
		t.Fatalf("paused run did not carry reason and continuation: %#v", run)
	}
	if !workflowstate.TerminalStatus(run.Status) {
		t.Fatalf("paused run must be terminal so the next workflow can start: %#v", run)
	}
}

func TestWorkflowProjectionMarksStaleRunningRunInterrupted(t *testing.T) {
	old := time.Now().UTC().Add(-2 * workflowStaleAfter)
	events := []LedgerEvent{
		workflowEvent("evt_req", WorkflowRunRequestedEvent, old, WorkflowRunRequestedPayload{
			WorkflowRunID:      "wfr_stale",
			MissionID:          "mis_1",
			RequestedBySurface: WorkflowSurfaceWeb,
			AgentExecutor:      "codex",
			MCPMode:            "auto",
			Instruction:        "run",
			MaxSteps:           2,
			MaxDurationMS:      60000,
		}),
		workflowEvent("evt_started", WorkflowRunStartedEvent, old.Add(time.Second), WorkflowRunStartedPayload{
			WorkflowRunID: "wfr_stale",
			MissionID:     "mis_1",
		}),
	}
	run := projectWorkflowRuns(events)[0]
	if run.Status != WorkflowStatusInterrupted {
		t.Fatalf("expected stale run to be interrupted, got %#v", run)
	}
}

func workflowEvent(eventID string, eventType string, createdAt time.Time, payload any) LedgerEvent {
	return LedgerEvent{
		EventID:   eventID,
		MissionID: "mis_1",
		Sequence:  int64(len(eventID)),
		EventType: eventType,
		Producer:  Producer{Type: "workflow", ID: "test"},
		Payload:   mustJSONRaw(payload),
		CreatedAt: createdAt,
	}
}
