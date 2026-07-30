package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestAgentProviderLockIncludesReportPartPlanCreated(t *testing.T) {
	if !EventLocksAgentExecutor("report.part_plan.created") {
		t.Fatal("report.part_plan.created must lock the mission agent executor")
	}
}

func TestAgentProviderLockIncludesFinalEditEvents(t *testing.T) {
	for _, eventType := range []string{
		"report.final_edit.reader.started",
		"report.final_edit.reader.submitted",
		"report.final_edit.style.started",
		"report.final_edit.style.submitted",
		"report.final_edit.gate.started",
		"report.final_edit.gate.submitted",
	} {
		if !EventLocksAgentExecutor(eventType) {
			t.Errorf("%s must lock the mission agent executor", eventType)
		}
	}
}

func TestAgentProviderLockRejectsMixedProviderPartPlanAppend(t *testing.T) {
	store := &workflowStore{}
	svc := NewService(store)
	ctx := context.Background()
	appendCompletedAgentTurn(t, svc, ctx, "mis_1", "codex")

	_, err := svc.AppendEvent(ctx, AppendEventRequest{
		EventID:   "evt_part_plan_claude",
		MissionID: "mis_1",
		EventType: "report.part_plan.created",
		Producer:  Producer{Type: "agent_session", ID: "claude-part-owner"},
		Payload: mustJSONRaw(map[string]any{
			"kind":             "sectional_markdown_report_part_plan",
			"pending_event_id": "evt_pending",
			"plan_event_id":    "evt_plan",
			"part_index":       1,
			"agent_executor":   "claude",
			"brief":            "brief",
		}),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected provider lock rejection, got %v", err)
	}
	if !strings.Contains(err.Error(), "already using codex") {
		t.Fatalf("expected locked provider message, got %v", err)
	}
	if len(store.events) != 2 {
		t.Fatalf("mixed-provider Part plan append should not add events, got %#v", store.events)
	}
}
