package missionusage

import (
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func TestProjectCorrectsCumulativeUsageAndGroupsSurfaces(t *testing.T) {
	events := []ledger.Event{
		usageEvent(t, "evt_second", 2, "turn.agent.response", "turn", "session-1", "session_cumulative", 160, 100, 18),
		usageEvent(t, "evt_first", 1, "turn.agent.response", "turn", "session-1", "session_cumulative", 100, 60, 10),
		usageEvent(t, "evt_report", 3, "report.section.created", "report_section", "session-2", "call", 40, 0, 8),
	}

	got := Project(events)
	if got.UsageRecordCount != 3 || got.UsageAvailableCount != 3 || got.UsageUnavailableCount != 0 || got.UsagePartial {
		t.Fatalf("unexpected usage metadata: %#v", got)
	}
	if got.SessionCount != 2 || got.InputTokens != 200 || got.CachedInputTokens != 100 || got.UncachedInputTokens != 60 ||
		got.OutputTokens != 26 || got.TotalTokens != 226 {
		t.Fatalf("unexpected corrected totals: %#v", got)
	}
	if got.PerCall.P50 != 68 || got.PerCall.P90 != 110 || got.PerCall.Max != 110 {
		t.Fatalf("unexpected percentiles: %#v", got.PerCall)
	}
	if len(got.Categories) != 2 || got.Categories[0].Key != "conversation" || got.Categories[0].TotalTokens != 178 ||
		got.Categories[1].Key != "report_writing" || got.Categories[1].TotalTokens != 48 {
		t.Fatalf("unexpected categories: %#v", got.Categories)
	}
}

func TestProjectMarksUnavailableAndCounterResetPartial(t *testing.T) {
	events := []ledger.Event{
		usageEvent(t, "evt_initial", 1, "workflow.step.completed", "workflow_step", "session-1", "session_cumulative", 100, 20, 10),
		usageEvent(t, "evt_reset", 2, "workflow.step.failed", "workflow_step", "session-1", "session_cumulative", 20, 0, 3),
		{EventID: "evt_unavailable", Sequence: 3, Payload: jsonValue(t, map[string]any{
			"agent_usage": map[string]any{"schema_version": 2, "surface": "report_plan", "usage_unavailable": true},
		})},
	}

	got := Project(events)
	if !got.UsagePartial || got.CounterResetCount != 1 || got.UsageRecordCount != 3 || got.UsageAvailableCount != 2 || got.UsageUnavailableCount != 1 {
		t.Fatalf("unexpected partial metadata: %#v", got)
	}
	if got.TotalTokens != 133 || got.FailedCallCount != 1 || got.FailedTotalTokens != 23 {
		t.Fatalf("unexpected reset or failure totals: %#v", got)
	}
}

func TestProjectKeepsUnknownSurfacesVisibleAndDeduplicatesEvents(t *testing.T) {
	event := usageEvent(t, "evt_unknown", 1, "agent.completed", "future_surface", "", "call", 7, 0, 2)
	got := Project([]ledger.Event{event, event})
	if got.UsageRecordCount != 1 || got.UsageAvailableCount != 1 || len(got.Categories) != 1 || got.Categories[0].Key != "other" {
		t.Fatalf("unexpected unknown surface projection: %#v", got)
	}
}

func TestProjectGroupsCorrectedWorkflowUsageByRun(t *testing.T) {
	events := []ledger.Event{
		workflowUsageEvent(t, "evt_run_1_first", 1, "wfr_1", "session-1", false, "gpt-test", "high", "session_cumulative", 100, 60, 10),
		workflowUsageEvent(t, "evt_run_1_second", 2, "wfr_1", "session-1", true, "gpt-test", "high", "session_cumulative", 160, 100, 18),
		workflowUsageEvent(t, "evt_run_2", 3, "wfr_2", "session-2", false, "gpt-next", "medium", "call", 40, 0, 8),
	}

	got := Project(events)
	if len(got.WorkflowRuns) != 2 {
		t.Fatalf("expected two workflow runs, got %#v", got.WorkflowRuns)
	}
	first := got.WorkflowRuns[0]
	if first.WorkflowRunID != "wfr_1" || first.CallCount != 2 || first.SessionCount != 1 || first.ResumedCallCount != 1 ||
		first.InputTokens != 160 || first.CachedInputTokens != 100 || first.UncachedInputTokens != 60 ||
		first.OutputTokens != 18 || first.TotalTokens != 178 || first.AgentModel != "gpt-test" || first.ReasoningEffort != "high" ||
		first.PerCall.P50 != 68 || first.PerCall.P90 != 110 || first.PerCall.Max != 110 {
		t.Fatalf("unexpected first workflow run: %#v", first)
	}
	second := got.WorkflowRuns[1]
	if second.WorkflowRunID != "wfr_2" || second.CallCount != 1 || second.TotalTokens != 48 || second.AgentModel != "gpt-next" || second.ReasoningEffort != "medium" {
		t.Fatalf("unexpected second workflow run: %#v", second)
	}
	if len(got.Categories) != 1 || got.Categories[0].Label != "자율 조사" || got.Categories[0].TotalTokens != 226 {
		t.Fatalf("unexpected workflow category: %#v", got.Categories)
	}
}

func TestProjectSubtractsLatestForkSourceBaseline(t *testing.T) {
	events := []ledger.Event{
		usageEvent(t, "evt_plan", 1, "report.plan.created", "report_plan", "session-plan", "session_cumulative", 100, 60, 10),
		requiredUsageEvent(t, "evt_requirements", 2, "report.requirements.mapped", "previous_provider_session_id", "session-plan"),
		correlatedUsageEvent(t, "evt_requirements_usage", 3, "evt_requirements", "report_requirements", "session-plan", "", 160, 100, 18),
		correlatedUsageEvent(t, "evt_section_usage", 4, "evt_section", "report_section", "session-section", "session-plan", 250, 160, 30),
	}

	got := Project(events)
	if got.UsageRecordCount != 3 || got.UsageAvailableCount != 3 || got.UsageUnavailableCount != 0 || got.UsagePartial {
		t.Fatalf("unexpected fork usage metadata: %#v", got)
	}
	if got.SessionCount != 2 || got.InputTokens != 250 || got.CachedInputTokens != 160 || got.OutputTokens != 30 || got.TotalTokens != 280 {
		t.Fatalf("fork source baseline was not subtracted exactly once: %#v", got)
	}
	if len(got.Categories) != 2 || got.Categories[0].Key != "report_planning" || got.Categories[0].TotalTokens != 178 ||
		got.Categories[1].Key != "report_writing" || got.Categories[1].TotalTokens != 102 {
		t.Fatalf("unexpected fork categories: %#v", got.Categories)
	}
}

func TestProjectDoesNotGuessAcrossHistoricalUsageGap(t *testing.T) {
	events := []ledger.Event{
		usageEvent(t, "evt_plan", 1, "report.plan.created", "report_plan", "session-plan", "session_cumulative", 100, 60, 10),
		requiredUsageEvent(t, "evt_requirements", 2, "report.requirements.mapped", "previous_provider_session_id", "session-plan"),
		correlatedUsageEvent(t, "evt_section_first", 3, "evt_section_first_result", "report_section", "session-section", "session-plan", 250, 160, 30),
		correlatedUsageEvent(t, "evt_section_second", 4, "evt_section_second_result", "report_section", "session-section", "session-plan", 280, 180, 35),
	}

	got := Project(events)
	if !got.UsagePartial || got.UsageRecordCount != 3 || got.UsageAvailableCount != 2 || got.UsageUnavailableCount != 2 {
		t.Fatalf("historical usage gap was not exposed: %#v", got)
	}
	if got.InputTokens != 130 || got.CachedInputTokens != 80 || got.OutputTokens != 15 || got.TotalTokens != 145 {
		t.Fatalf("untrusted fork snapshot was counted: %#v", got)
	}
}

func usageEvent(t *testing.T, eventID string, sequence int64, eventType, surface, sessionID, scope string, input, cached, output int) ledger.Event {
	t.Helper()
	return ledger.Event{
		EventID: eventID, Sequence: sequence, EventType: eventType,
		Payload: jsonValue(t, map[string]any{"agent_usage": map[string]any{
			"schema_version": 2,
			"surface":        surface,
			"session":        map[string]any{"agent_session_id": sessionID},
			"provider_usage": map[string]any{
				"scope": scope, "input_tokens": input, "cached_input_tokens": cached,
				"output_tokens": output, "total_tokens": input + output,
			},
		}}),
	}
}

func workflowUsageEvent(t *testing.T, eventID string, sequence int64, workflowRunID string, sessionID string, resumed bool, model string, effort string, scope string, input int, cached int, output int) ledger.Event {
	t.Helper()
	return ledger.Event{
		EventID: eventID, Sequence: sequence, EventType: "turn.agent.response",
		Payload: jsonValue(t, map[string]any{
			"workflow_run_id": workflowRunID,
			"agent_usage": map[string]any{
				"schema_version":   2,
				"surface":          "workflow_step",
				"model":            model,
				"reasoning_effort": effort,
				"session": map[string]any{
					"agent_session_id": sessionID,
					"resumed":          resumed,
				},
				"provider_usage": map[string]any{
					"scope": scope, "input_tokens": input, "cached_input_tokens": cached,
					"output_tokens": output, "total_tokens": input + output,
				},
			},
		}),
	}
}

func requiredUsageEvent(t *testing.T, eventID string, sequence int64, eventType string, sessionKey string, sessionID string) ledger.Event {
	t.Helper()
	return ledger.Event{
		EventID: eventID, Sequence: sequence, EventType: eventType,
		Payload: jsonValue(t, map[string]any{sessionKey: sessionID}),
	}
}

func correlatedUsageEvent(t *testing.T, eventID string, sequence int64, correlationEventID string, surface string, sessionID string, forkSourceSessionID string, input int, cached int, output int) ledger.Event {
	t.Helper()
	payload := map[string]any{
		"correlation_event_id": correlationEventID,
		"agent_usage": map[string]any{
			"schema_version": 2,
			"surface":        surface,
			"session":        map[string]any{"agent_session_id": sessionID},
			"provider_usage": map[string]any{
				"scope": "session_cumulative", "input_tokens": input, "cached_input_tokens": cached,
				"output_tokens": output, "total_tokens": input + output,
			},
		},
	}
	if forkSourceSessionID != "" {
		payload["fork_source_agent_session_id"] = forkSourceSessionID
	}
	return ledger.Event{EventID: eventID, Sequence: sequence, EventType: "agent.usage.recorded", Payload: jsonValue(t, payload)}
}

func jsonValue(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
