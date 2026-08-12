package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestRunnerProactivelyCompactsBeforeOpeningNextStep(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	appendContextUsageResponse(t, svc, mission.MissionID, "evt_trigger", 142120, 258400)
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_proactive", MaxSteps: 1})

	agent := &fakeAgent{responses: []AgentResult{
		{Text: "compact summary", SessionID: "agent-session-1"},
		{Text: "step result\n" + controlMarker + ` {"decision":"stop","reason":"done"}`, SessionID: "agent-session-1"},
	}}
	view, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_proactive")
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != app.WorkflowStatusCompleted || len(agent.requests) != 2 {
		t.Fatalf("unexpected proactive run: view=%#v requests=%#v", view, agent.requests)
	}
	if !agent.requests[0].Compaction || agent.requests[1].Compaction {
		t.Fatalf("expected compaction before workflow step, got %#v", agent.requests)
	}
	events, err := svc.ListEvents(ctx, mission.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	compactSequence := int64(0)
	stepSequence := int64(0)
	for _, event := range events {
		switch event.EventType {
		case "turn.agent.compacted":
			compactSequence = event.Sequence
			var payload map[string]any
			if json.Unmarshal(event.Payload, &payload) != nil ||
				payload["context_trigger_event_id"] != "evt_trigger" ||
				payload["context_threshold_percent"] != float64(55) {
				t.Fatalf("unexpected compact payload: %#v", payload)
			}
		case app.WorkflowStepStartedEvent:
			stepSequence = event.Sequence
		}
	}
	if compactSequence == 0 || stepSequence == 0 || compactSequence >= stepSequence {
		t.Fatalf("compaction must precede step start: compact=%d step=%d", compactSequence, stepSequence)
	}
}

func TestRunnerSkipsProactiveCompactionBelowThreshold(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	appendContextUsageResponse(t, svc, mission.MissionID, "evt_trigger", 142119, 258400)
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_below", MaxSteps: 1})

	agent := &fakeAgent{responses: []AgentResult{{Text: "done\n" + controlMarker + ` {"decision":"stop","reason":"done"}`, SessionID: "agent-session-1"}}}
	if _, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_below"); err != nil {
		t.Fatal(err)
	}
	if len(agent.requests) != 1 || agent.requests[0].Compaction {
		t.Fatalf("unexpected below-threshold requests: %#v", agent.requests)
	}
}

func TestRunnerDoesNotRepeatRecordedProactiveCompaction(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	appendContextUsageResponse(t, svc, mission.MissionID, "evt_trigger", 143000, 258400)
	appendRawEvent(t, svc, mission.MissionID, "evt_compact", "turn.agent.compacted", map[string]any{
		"reason": "context_window_threshold", "context_trigger_event_id": "evt_trigger",
	})
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_dedupe", MaxSteps: 1})

	agent := &fakeAgent{responses: []AgentResult{{Text: "done\n" + controlMarker + ` {"decision":"stop","reason":"done"}`, SessionID: "agent-session-1"}}}
	if _, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_dedupe"); err != nil {
		t.Fatal(err)
	}
	if len(agent.requests) != 1 || agent.requests[0].Compaction {
		t.Fatalf("unexpected duplicate compaction: %#v", agent.requests)
	}
}

func TestRunnerFailsBeforeStepWhenProactiveCompactionFails(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	appendContextUsageResponse(t, svc, mission.MissionID, "evt_trigger", 143000, 258400)
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_failure", MaxSteps: 1})

	agent := &fakeAgent{err: errors.New("compact unavailable")}
	view, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_failure")
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != app.WorkflowStatusFailed || len(agent.requests) != 1 || !agent.requests[0].Compaction {
		t.Fatalf("unexpected failure result: view=%#v requests=%#v", view, agent.requests)
	}
	events, err := svc.ListEvents(ctx, mission.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if countEvents(events, app.WorkflowStepStartedEvent) != 0 {
		t.Fatalf("failed preflight must not open a workflow step: %#v", events)
	}
}

func appendContextUsageResponse(t *testing.T, svc *app.Service, missionID string, eventID string, used int, window int) {
	t.Helper()
	appendRawEvent(t, svc, missionID, "evt_user_"+eventID, "turn.user", map[string]any{"kind": "user_turn", "text": "research"})
	appendRawEvent(t, svc, missionID, eventID, "turn.agent.response", map[string]any{
		"kind": "agent_response", "user_event_id": "evt_user_" + eventID,
		"agent_executor": "codex", "agent_session_id": "agent-session-1",
		"agent_usage": map[string]any{
			"schema_version": 2,
			"context_window": map[string]any{
				"used_tokens": used, "window_tokens": window, "source": "codex_session_token_count",
			},
		},
	})
}
