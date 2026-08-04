package agentpolicy

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestValidateAppendPreservesExecutorLock(t *testing.T) {
	existing := []ledger.Event{lockingEvent(t, "turn.agent.response", "codex")}

	if err := ValidateAppend(existing, []ledger.Event{lockingEvent(t, "report.part_plan.created", "codex")}); err != nil {
		t.Fatalf("same executor append returned error: %v", err)
	}
	err := ValidateAppend(existing, []ledger.Event{lockingEvent(t, "report.part_plan.created", "claude")})
	if !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("expected executor conflict, got %v", err)
	}
}

func TestValidateAppendRejectsMixedExecutors(t *testing.T) {
	err := ValidateAppend(nil, []ledger.Event{
		lockingEvent(t, "turn.user", "codex"),
		lockingEvent(t, "report.plan.created", "claude"),
	})
	if !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("expected mixed executor error, got %v", err)
	}
}

func TestExplicitLockingExecutorIgnoresMalformedPayload(t *testing.T) {
	event := ledger.Event{EventType: "turn.user", Payload: json.RawMessage(`{"agent_executor":`)}
	if executor, ok := ExplicitLockingExecutor(event); ok || executor != "" {
		t.Fatalf("malformed payload must not lock an executor, got %q, %v", executor, ok)
	}
}

func lockingEvent(t *testing.T, eventType, executor string) ledger.Event {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"agent_executor": executor})
	if err != nil {
		t.Fatal(err)
	}
	return ledger.Event{EventType: eventType, Payload: payload}
}
