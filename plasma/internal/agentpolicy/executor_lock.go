package agentpolicy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
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

// LockedExecutorFromEvents returns the first durable event that explicitly
// fixes a mission to one executor.
func LockedExecutorFromEvents(events []ledger.Event) string {
	for _, event := range events {
		if executor, ok := ExplicitLockingExecutor(event); ok {
			return executor
		}
	}
	return ""
}

// ValidateMissionExecutor rejects an executor that conflicts with durable state.
func ValidateMissionExecutor(events []ledger.Event, requested string) error {
	requested, err := NormalizeExecutorName(requested)
	if err != nil {
		return err
	}
	locked := LockedExecutorFromEvents(events)
	if locked == "" || locked == requested {
		return nil
	}
	return fmt.Errorf("%w: this mission is already using %s; create a new mission to use %s", producterror.ErrInvalidInput, locked, requested)
}

// ValidateAppend ensures one append cannot introduce mixed or conflicting executors.
func ValidateAppend(events, appended []ledger.Event) error {
	requested := ""
	for _, event := range appended {
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
	if !EventLocksExecutor(event.EventType) {
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
