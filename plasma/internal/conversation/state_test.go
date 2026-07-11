package conversation

import (
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestLatestAgentSessionIDKeepsResearchSessionForIsolatedReport(t *testing.T) {
	events := []app.LedgerEvent{
		ledgerEvent(t, "report.artifact.created", map[string]any{
			"agent_executor":                 "codex",
			"agent_session_id":               "report-session",
			"report_session_policy":          "isolated_fork",
			"pre_report_research_session_id": "research-session",
		}),
	}

	if got := LatestAgentSessionID(events, "codex"); got != "research-session" {
		t.Fatalf("expected pre-report research session, got %q", got)
	}
}

func TestLatestAgentSessionIDIgnoresIsolatedReportWithoutPreReportSession(t *testing.T) {
	events := []app.LedgerEvent{
		ledgerEvent(t, "report.artifact.created", map[string]any{
			"agent_executor":        "codex",
			"agent_session_id":      "report-session",
			"report_session_policy": "isolated_fork",
		}),
		ledgerEvent(t, "turn.agent.response", map[string]any{
			"kind":             "agent_response",
			"agent_executor":   "codex",
			"agent_session_id": "research-session",
		}),
	}
	events[0].Sequence = 1
	events[1].Sequence = 2

	if got := LatestAgentSessionID(events, "codex"); got != "research-session" {
		t.Fatalf("expected isolated report without pre-report session to be ignored, got %q", got)
	}
}

func TestLatestAgentSessionIDIgnoresAgentErrorSessionID(t *testing.T) {
	events := []app.LedgerEvent{
		ledgerEvent(t, "turn.agent.response", map[string]any{
			"kind":             "agent_response",
			"agent_executor":   "codex",
			"agent_session_id": "successful-session",
		}),
		ledgerEvent(t, "turn.agent.response", map[string]any{
			"kind":             "agent_error",
			"agent_executor":   "codex",
			"agent_session_id": "failed-session",
		}),
	}
	events[0].Sequence = 1
	events[1].Sequence = 2

	if got := LatestAgentSessionID(events, "codex"); got != "successful-session" {
		t.Fatalf("expected latest successful agent session, got %q", got)
	}
}

func TestLatestOpenAgentPendingIgnoresCompletedUserTurns(t *testing.T) {
	events := []app.LedgerEvent{
		ledgerEvent(t, "turn.agent.pending", map[string]any{"user_event_id": "evt_user_1", "agent_executor": "codex"}),
		ledgerEvent(t, "turn.agent.response", map[string]any{"user_event_id": "evt_user_1", "kind": "agent_response", "agent_executor": "codex"}),
		ledgerEvent(t, "turn.agent.pending", map[string]any{"user_event_id": "evt_user_2", "agent_executor": "claude", "workflow_run_id": "wfr_1"}),
	}

	pending, ok := LatestOpenAgentPending(events, "wfr_1")
	if !ok || pending.UserEventID != "evt_user_2" || pending.AgentExecutor != "claude" {
		t.Fatalf("unexpected pending turn: ok=%v pending=%#v", ok, pending)
	}
}

func ledgerEvent(t *testing.T, eventType string, payloadValue any) app.LedgerEvent {
	t.Helper()
	payload, err := json.Marshal(payloadValue)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return app.LedgerEvent{EventType: eventType, Payload: payload}
}
