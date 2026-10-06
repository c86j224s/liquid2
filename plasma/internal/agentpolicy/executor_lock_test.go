package agentpolicy

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestValidateAppendPreservesExecutorLock(t *testing.T) {
	existing := []ledger.Event{reportPendingEvent(t, "evt_pending", "codex")}

	if err := ValidateAppend(existing, []ledger.Event{lockingEvent(t, "report.part_plan.created", "codex")}); err != nil {
		t.Fatalf("same executor append returned error: %v", err)
	}
	if err := ValidateAppend(existing, []ledger.Event{lockingEvent(t, "report.section.evidence_gap", "codex")}); err != nil {
		t.Fatalf("evidence gap append should preserve executor lock: %v", err)
	}
	if err := ValidateAppend(existing, []ledger.Event{lockingEvent(t, "report.plan.section_repair.completed", "codex")}); err != nil {
		t.Fatalf("Section plan repair append should preserve executor lock: %v", err)
	}
	err := ValidateAppend(existing, []ledger.Event{lockingEvent(t, "report.section.evidence_gap", "claude")})
	if !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("expected executor conflict, got %v", err)
	}
	err = ValidateAppend(existing, []ledger.Event{lockingEvent(t, "report.plan.section_repair.completed", "claude")})
	if !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("expected Section plan repair executor conflict, got %v", err)
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

func TestIndependentReportPendingActivelyLocksItsExecutor(t *testing.T) {
	for _, family := range []string{"report_il_experimental", "report_unverified"} {
		t.Run(family, func(t *testing.T) {
			pending := ledger.Event{EventID: "evt_independent_pending", EventType: "report.draft.pending", Payload: json.RawMessage(`{"agent_executor":"codex","pipeline_family":"` + family + `"}`)}
			if active := ActiveExecutorFromEvents([]ledger.Event{pending}); active != "codex" {
				t.Fatalf("in-flight independent report should actively lock its own executor, got %q", active)
			}
		})
	}
}

func TestIndependentReportCompletionReleasesActiveLock(t *testing.T) {
	for _, family := range []string{"report_il_experimental", "report_unverified"} {
		t.Run(family, func(t *testing.T) {
			pending := ledger.Event{EventID: "evt_independent_pending", EventType: "report.draft.pending", Payload: json.RawMessage(`{"agent_executor":"codex","pipeline_family":"` + family + `"}`)}
			terminal := ledger.Event{EventID: "evt_independent_terminal", EventType: "report.artifact.created", Payload: json.RawMessage(`{"agent_executor":"codex","pipeline_family":"` + family + `","pending_event_id":"evt_independent_pending"}`)}
			events := []ledger.Event{pending, terminal}
			if active := ActiveExecutorFromEvents(events); active != "" {
				t.Fatalf("completed independent report should release its active lock, got %q", active)
			}
			if err := ValidateMissionExecutor(events, "claude"); err != nil {
				t.Fatalf("mission should be free to switch executor once the independent report finished: %v", err)
			}
		})
	}
}

func TestValidateAppendExemptsIndependentReportBatchFromMissionLock(t *testing.T) {
	for _, family := range []string{"report_il_experimental", "report_unverified"} {
		t.Run(family, func(t *testing.T) {
			existing := []ledger.Event{
				{EventID: "evt_open_pending", EventType: "turn.agent.pending", Payload: json.RawMessage(`{"agent_executor":"claude","user_event_id":"evt_user"}`)},
			}
			pending := ledger.Event{EventID: "evt_independent_pending", EventType: "report.draft.pending", Payload: json.RawMessage(`{"agent_executor":"codex","pipeline_family":"` + family + `"}`)}
			terminal := ledger.Event{EventID: "evt_independent_terminal", EventType: "report.artifact.created", Payload: json.RawMessage(`{"agent_executor":"codex","pipeline_family":"` + family + `","pending_event_id":"evt_independent_pending"}`)}
			if err := ValidateAppend(existing, []ledger.Event{pending, terminal}); err != nil {
				t.Fatalf("independent report batch should bypass the active mission lock: %v", err)
			}
		})
	}
}

func TestForgedExperimentalTerminalStillLocksClassicExecutor(t *testing.T) {
	existing := []ledger.Event{lockingEvent(t, "turn.agent.response", "claude")}
	classicPending := ledger.Event{EventID: "evt_classic_pending", EventType: "report.draft.pending", Payload: json.RawMessage(`{"agent_executor":"claude","report_mode":"long_form"}`)}
	for _, terminal := range []ledger.Event{
		{EventID: "evt_forged", EventType: "report.artifact.created", Payload: json.RawMessage(`{"agent_executor":"codex","pipeline_family":"report_il_experimental","pending_event_id":"evt_classic_pending"}`)},
		{EventID: "evt_standalone", EventType: "report.artifact.created", Payload: json.RawMessage(`{"agent_executor":"codex","pipeline_family":"report_il_experimental","pending_event_id":"evt_missing"}`)},
	} {
		if err := ValidateAppend(append(existing, classicPending), []ledger.Event{terminal}); !errors.Is(err, producterror.ErrInvalidInput) {
			t.Fatalf("forged IL terminal must conflict with classic lock, got %v", err)
		}
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

func reportPendingEvent(t *testing.T, eventID, executor string) ledger.Event {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"agent_executor": executor})
	if err != nil {
		t.Fatal(err)
	}
	return ledger.Event{EventID: eventID, EventType: "report.draft.pending", Payload: payload}
}

func TestActiveExecutorFromEventsIgnoresCompletedTurn(t *testing.T) {
	events := []ledger.Event{
		{EventID: "evt_user", EventType: "turn.user", Payload: json.RawMessage(`{"agent_executor":"codex"}`)},
		{EventID: "evt_pending", EventType: "turn.agent.pending", Payload: json.RawMessage(`{"agent_executor":"codex","user_event_id":"evt_user"}`)},
		{EventID: "evt_response", EventType: "turn.agent.response", Payload: json.RawMessage(`{"agent_executor":"codex","user_event_id":"evt_user"}`)},
	}
	if active := ActiveExecutorFromEvents(events); active != "" {
		t.Fatalf("a completed turn should not keep the mission actively locked, got %q", active)
	}
	if err := ValidateMissionExecutor(events, "claude"); err != nil {
		t.Fatalf("mission should be free to switch executor once the prior turn completed: %v", err)
	}
}

func TestActiveExecutorFromEventsBlocksSwitchDuringOpenTurn(t *testing.T) {
	events := []ledger.Event{
		{EventID: "evt_user", EventType: "turn.user", Payload: json.RawMessage(`{"agent_executor":"codex"}`)},
		{EventID: "evt_pending", EventType: "turn.agent.pending", Payload: json.RawMessage(`{"agent_executor":"codex","user_event_id":"evt_user"}`)},
	}
	if active := ActiveExecutorFromEvents(events); active != "codex" {
		t.Fatalf("an open turn should actively lock the mission, got %q", active)
	}
	err := ValidateMissionExecutor(events, "claude")
	if !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("expected switch to be rejected while a turn is in flight, got %v", err)
	}
}

func TestActiveExecutorFromEventsBlocksSwitchDuringOpenWorkflowRun(t *testing.T) {
	events := []ledger.Event{
		{
			EventID:   "evt_wf_requested",
			MissionID: "mission-1",
			EventType: "workflow.run.requested",
			Payload:   json.RawMessage(`{"workflow_run_id":"run-1","mission_id":"mission-1","agent_executor":"codex"}`),
		},
	}
	if active := ActiveExecutorFromEvents(events); active != "codex" {
		t.Fatalf("a non-terminal workflow run should actively lock the mission, got %q", active)
	}
	if err := ValidateMissionExecutor(events, "claude"); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("expected switch to be rejected while a workflow run is in flight, got %v", err)
	}
}
