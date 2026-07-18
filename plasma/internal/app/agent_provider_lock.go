package app

import (
	"encoding/json"
	"fmt"
	"strings"
)

func NormalizeAgentExecutorName(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "codex", nil
	}
	switch value {
	case "codex", "claude":
		return value, nil
	default:
		return "", fmt.Errorf("%w: unsupported agent executor %q", ErrInvalidInput, value)
	}
}

func LockedAgentExecutorFromEvents(events []LedgerEvent) string {
	for _, event := range events {
		executor, ok := ExplicitLockingAgentExecutor(event)
		if !ok {
			continue
		}
		return executor
	}
	return ""
}

func ValidateMissionAgentExecutorForEvents(events []LedgerEvent, requested string) error {
	requested, err := NormalizeAgentExecutorName(requested)
	if err != nil {
		return err
	}
	locked := LockedAgentExecutorFromEvents(events)
	if locked == "" || locked == requested {
		return nil
	}
	return fmt.Errorf("%w: this mission is already using %s; create a new mission to use %s", ErrInvalidInput, locked, requested)
}

func ValidateAgentExecutorAppend(events []LedgerEvent, appended []LedgerEvent) error {
	requested := ""
	for _, event := range appended {
		executor, ok, err := explicitLockingAgentExecutor(event)
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
			return fmt.Errorf("%w: mixed agent executors in one append are not supported", ErrInvalidInput)
		}
	}
	if requested == "" {
		return nil
	}
	return ValidateMissionAgentExecutorForEvents(events, requested)
}

func ExplicitLockingAgentExecutor(event LedgerEvent) (string, bool) {
	name, ok, err := explicitLockingAgentExecutor(event)
	if err != nil {
		return "", false
	}
	return name, ok
}

func explicitLockingAgentExecutor(event LedgerEvent) (string, bool, error) {
	if !EventLocksAgentExecutor(event.EventType) {
		return "", false, nil
	}
	var payload struct {
		AgentExecutor string `json:"agent_executor"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return "", false, nil
	}
	if strings.TrimSpace(payload.AgentExecutor) == "" {
		return "", false, nil
	}
	name, err := NormalizeAgentExecutorName(payload.AgentExecutor)
	if err != nil {
		return "", true, err
	}
	return name, true, nil
}

func EventLocksAgentExecutor(eventType string) bool {
	switch eventType {
	case "turn.user",
		"turn.agent.pending",
		"turn.agent.response",
		"turn.agent.compacted",
		WorkflowRunRequestedEvent,
		WorkflowRunStartedEvent,
		WorkflowStepStartedEvent,
		WorkflowStepCompletedEvent,
		WorkflowRunPausedEvent,
		WorkflowRunCompletedEvent,
		WorkflowRunStoppedEvent,
		WorkflowRunFailedEvent,
		WorkflowRunInterruptedEvent,
		"report.draft.pending",
		"report.plan.created",
		"report.section.started",
		"report.section.created",
		"report.part.created",
		"report.artifact.created",
		"report.design.pending",
		"report.patch.pending",
		"report.patch.failed",
		"report.artifact.exported":
		return true
	default:
		return false
	}
}
