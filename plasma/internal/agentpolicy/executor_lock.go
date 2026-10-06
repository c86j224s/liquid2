package agentpolicy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// NormalizeExecutorName validates the stable provider names accepted by the product.
func NormalizeExecutorName(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "codex", nil
	}
	switch value {
	case "codex", "claude":
		return value, nil
	default:
		return "", fmt.Errorf("%w: unsupported agent executor %q", producterror.ErrInvalidInput, value)
	}
}

// ActiveExecutorFromEvents returns the executor currently bound to a mission
// because it owns an in-flight, not-yet-terminal provider-backed operation
// (an open agent turn, an open report pipeline stage, or a non-terminal
// workflow run). Once every such operation reaches a terminal state, the
// mission has no active executor and a request may use a different one.
//
// This intentionally replaces the previous "first use permanently locks the
// mission" behavior: the lock exists only to prevent two executors from
// racing on the same in-flight operation, not to fix a mission's identity.
func ActiveExecutorFromEvents(events []ledger.Event) string {
	if pending, ok := ledgerstate.OpenAgentPendingEvent(toLedgerStateEvents(events)); ok {
		if executor := rawEventExecutor(pending.Payload); executor != "" {
			return executor
		}
	}
	if pending, ok := ledgerstate.OpenReportPendingEvent(toLedgerStateEvents(events)); ok {
		if executor := rawEventExecutor(pending.Payload); executor != "" {
			return executor
		}
	}
	for _, run := range workflowstate.ProjectRuns(toWorkflowStateEvents(events)) {
		if workflowstate.TerminalStatus(run.Status) {
			continue
		}
		if strings.TrimSpace(run.AgentExecutor) == "" {
			continue
		}
		if executor, err := NormalizeExecutorName(run.AgentExecutor); err == nil {
			return executor
		}
	}
	return ""
}

// rawEventExecutor reads the agent_executor field directly off a pending
// event's payload, without the "independent report" exemption that
// explicitLockingExecutor applies. An in-flight independent report still
// actively occupies its executor even though it never permanently locks the
// mission's identity.
func rawEventExecutor(payload json.RawMessage) string {
	var typed struct {
		AgentExecutor string `json:"agent_executor"`
	}
	if json.Unmarshal(payload, &typed) != nil || strings.TrimSpace(typed.AgentExecutor) == "" {
		return ""
	}
	executor, err := NormalizeExecutorName(typed.AgentExecutor)
	if err != nil {
		return ""
	}
	return executor
}

// ValidateMissionExecutor rejects an executor that conflicts with an
// operation the mission is currently, actively running.
func ValidateMissionExecutor(events []ledger.Event, requested string) error {
	requested, err := NormalizeExecutorName(requested)
	if err != nil {
		return err
	}
	active := ActiveExecutorFromEvents(events)
	if active == "" || active == requested {
		return nil
	}
	return fmt.Errorf("%w: this mission has an operation still in progress with %s; wait for it to finish before switching to %s", producterror.ErrInvalidInput, active, requested)
}

func toLedgerStateEvents(events []ledger.Event) []ledgerstate.Event {
	converted := make([]ledgerstate.Event, 0, len(events))
	for _, event := range events {
		converted = append(converted, ledgerstate.Event{
			EventID:   event.EventID,
			Sequence:  event.Sequence,
			EventType: event.EventType,
			Payload:   event.Payload,
			CreatedAt: event.CreatedAt,
		})
	}
	return converted
}

func toWorkflowStateEvents(events []ledger.Event) []workflowstate.Event {
	converted := make([]workflowstate.Event, 0, len(events))
	for _, event := range events {
		converted = append(converted, workflowstate.Event{
			EventID:   event.EventID,
			MissionID: event.MissionID,
			Sequence:  event.Sequence,
			EventType: event.EventType,
			Payload:   event.Payload,
			CreatedAt: event.CreatedAt,
		})
	}
	return converted
}

// ValidateAppend ensures one append cannot introduce mixed or conflicting executors.
func ValidateAppend(events, appended []ledger.Event) error {
	requested := ""
	for index, event := range appended {
		prior := append(append([]ledger.Event(nil), events...), appended[:index]...)
		if isIndependentReportEvent(event) || independentReportTerminal(event, prior) {
			continue
		}
		executor, ok, err := explicitLockingExecutor(event)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if requested == "" {
			requested = executor
			continue
		}
		if requested != executor {
			return fmt.Errorf("%w: mixed agent executors in one append are not supported", producterror.ErrInvalidInput)
		}
	}
	if requested == "" {
		return nil
	}
	return ValidateMissionExecutor(events, requested)
}

// ExplicitLockingExecutor reads a valid explicit executor from a locking event.
func ExplicitLockingExecutor(event ledger.Event) (string, bool) {
	name, ok, err := explicitLockingExecutor(event)
	if err != nil {
		return "", false
	}
	return name, ok
}

func explicitLockingExecutor(event ledger.Event) (string, bool, error) {
	if !EventLocksExecutor(event.EventType) || isIndependentReportEvent(event) {
		return "", false, nil
	}
	var payload struct {
		AgentExecutor string `json:"agent_executor"`
	}
	if json.Unmarshal(event.Payload, &payload) != nil || strings.TrimSpace(payload.AgentExecutor) == "" {
		return "", false, nil
	}
	name, err := NormalizeExecutorName(payload.AgentExecutor)
	if err != nil {
		return "", true, err
	}
	return name, true, nil
}

func isIndependentReportEvent(event ledger.Event) bool {
	return event.EventType == "report.draft.pending" && independentReportFamily(event) != ""
}

func independentReportFamily(event ledger.Event) string {
	if event.EventType != "report.draft.pending" {
		return ""
	}
	var payload struct {
		PipelineFamily string `json:"pipeline_family"`
	}
	if json.Unmarshal(event.Payload, &payload) != nil {
		return ""
	}
	family := strings.TrimSpace(payload.PipelineFamily)
	if !reportpipeline.Independent(family) {
		return ""
	}
	return family
}

func independentReportTerminal(event ledger.Event, prior []ledger.Event) bool {
	if event.EventType != "report.artifact.created" {
		return false
	}
	var payload struct {
		PipelineFamily string `json:"pipeline_family"`
		PendingID      string `json:"pending_event_id"`
	}
	if json.Unmarshal(event.Payload, &payload) != nil || !reportpipeline.Independent(strings.TrimSpace(payload.PipelineFamily)) || strings.TrimSpace(payload.PendingID) == "" {
		return false
	}
	for _, candidate := range prior {
		if candidate.EventID == payload.PendingID && independentReportFamily(candidate) == strings.TrimSpace(payload.PipelineFamily) {
			return true
		}
	}
	return false
}

// EventLocksExecutor reports whether an event commits the mission to a provider.
func EventLocksExecutor(eventType string) bool {
	switch eventType {
	case "turn.user", "turn.agent.pending", "turn.agent.response", "turn.agent.compacted",
		workflowstate.WorkflowRunRequestedEvent, workflowstate.WorkflowRunStartedEvent,
		workflowstate.WorkflowStepStartedEvent, workflowstate.WorkflowStepCompletedEvent,
		workflowstate.WorkflowRunPausedEvent, workflowstate.WorkflowRunCompletedEvent,
		workflowstate.WorkflowRunStoppedEvent, workflowstate.WorkflowRunFailedEvent,
		workflowstate.WorkflowRunInterruptedEvent,
		"report.draft.pending", "report.plan.created", "report.requirements.started",
		"report.requirements.mapped", "report.section.started", "report.part_plan.created",
		"report.plan.section_repair.completed",
		"report.section.evidence_gap", "report.section.created", "report.part.created", "report.part_edit.started",
		"report.part.edited", "report.final_edit.reader.started",
		"report.final_edit.reader.submitted", "report.final_edit.style.started",
		"report.final_edit.style.submitted", "report.final_edit.gate.started",
		"report.final_edit.gate.submitted", "report.artifact.created",
		"report.design.pending", "report.patch.pending", "report.patch.failed",
		"report.artifact.exported":
		return true
	default:
		return false
	}
}
