package agentpolicy

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func completedTurnEvents(userEventID, userText, executor, responseText string) []ledger.Event {
	userPayload, _ := json.Marshal(map[string]any{"text": userText})
	responsePayload, _ := json.Marshal(map[string]any{
		"kind":           "agent_response",
		"text":           responseText,
		"user_event_id":  userEventID,
		"agent_executor": executor,
	})
	return []ledger.Event{
		{EventID: userEventID, EventType: "turn.user", Payload: userPayload},
		{EventID: userEventID + "_response", EventType: "turn.agent.response", Payload: responsePayload},
	}
}

func TestSwitchRecapEmptyForFreshMission(t *testing.T) {
	events := completedTurnEvents("evt_user_1", "안녕", "codex", "안녕하세요")
	if got := SwitchRecap(events, "codex"); got != "" {
		t.Fatalf("expected no recap when the mission never used another executor, got %q", got)
	}
}

func TestSwitchRecapIncludesPriorExecutorTurns(t *testing.T) {
	events := completedTurnEvents("evt_user_1", "지난 조사 요약해줘", "codex", "요약 결과입니다")
	recap := SwitchRecap(events, "claude")
	if recap == "" {
		t.Fatal("expected a recap when switching away from the executor that ran prior turns")
	}
	for _, expected := range []string{"지난 조사 요약해줘", "요약 결과입니다", "codex"} {
		if !strings.Contains(recap, expected) {
			t.Fatalf("expected recap to contain %q, got %s", expected, recap)
		}
	}
}

func TestSwitchRecapIgnoresIncompleteTurns(t *testing.T) {
	userPayload, _ := json.Marshal(map[string]any{"text": "still running"})
	pendingPayload, _ := json.Marshal(map[string]any{"agent_executor": "codex", "user_event_id": "evt_user_1"})
	events := []ledger.Event{
		{EventID: "evt_user_1", EventType: "turn.user", Payload: userPayload},
		{EventID: "evt_pending_1", EventType: "turn.agent.pending", Payload: pendingPayload},
	}
	if got := SwitchRecap(events, "claude"); got != "" {
		t.Fatalf("expected no recap from an unresolved turn, got %q", got)
	}
}

func TestPriorExecutorReturnsEmptyForFreshMission(t *testing.T) {
	events := completedTurnEvents("evt_user_1", "안녕", "codex", "안녕하세요")
	if got := PriorExecutor(events, "codex"); got != "" {
		t.Fatalf("expected no prior executor when the mission never used another executor, got %q", got)
	}
}

func TestPriorExecutorReturnsLastDifferentExecutor(t *testing.T) {
	events := completedTurnEvents("evt_user_1", "지난 조사 요약해줘", "codex", "요약 결과입니다")
	if got := PriorExecutor(events, "claude"); got != "codex" {
		t.Fatalf("expected prior executor codex, got %q", got)
	}
}

func TestBuildExecutorSwitchedAppendRequestRecordsFromAndTo(t *testing.T) {
	req := BuildExecutorSwitchedAppendRequest("evt_switch", "mis_1", "codex", "claude", ledger.Producer{Type: "agent", ID: "claude"})
	if req.EventType != ExecutorSwitchedEvent {
		t.Fatalf("expected event type %q, got %q", ExecutorSwitchedEvent, req.EventType)
	}
	var payload struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if payload.From != "codex" || payload.To != "claude" {
		t.Fatalf("expected from=codex to=claude, got %+v", payload)
	}
}

func TestSwitchRecapCapsTurnCountAndFieldLength(t *testing.T) {
	var events []ledger.Event
	for i := 0; i < switchRecapMaxTurns+5; i++ {
		id := "evt_user_" + string(rune('a'+i))
		events = append(events, completedTurnEvents(id, strings.Repeat("x", switchRecapMaxFieldLen+50), "codex", "ok")...)
	}
	recap := SwitchRecap(events, "claude")
	if strings.Count(recap, "- user:") > switchRecapMaxTurns {
		t.Fatalf("expected at most %d recapped turns, got recap with %d", switchRecapMaxTurns, strings.Count(recap, "- user:"))
	}
	if strings.Contains(recap, strings.Repeat("x", switchRecapMaxFieldLen+1)) {
		t.Fatal("expected long user text to be truncated")
	}
}
