package app

import (
	"encoding/json"
	"strings"
)

const (
	turnAgentPendingEvent  = "turn.agent.pending"
	turnAgentResponseEvent = "turn.agent.response"
)

// MissionActivityEventTypes returns the only ledger event types a mission-list
// activity projection needs. It deliberately excludes source, evidence, and
// other detail-only activity so list refreshes do not load full ledgers.
func MissionActivityEventTypes() []string {
	return []string{
		turnAgentPendingEvent,
		turnAgentResponseEvent,
		"report.draft.pending",
		"report.design.pending",
		"report.humanize.pending",
		"report.patch.pending",
		"report.drafted",
		"report.artifact.created",
		"report.artifact.exported",
		"report.humanize.skipped",
		"report.draft.failed",
		"report.design.failed",
		"report.humanize.failed",
		"report.patch.failed",
		WorkflowRunRequestedEvent,
		WorkflowRunStartedEvent,
		WorkflowRunStopRequestedEvent,
		WorkflowSourceSkippedEvent,
		WorkflowStepStartedEvent,
		WorkflowStepCompletedEvent,
		WorkflowRunCompletedEvent,
		WorkflowRunPausedEvent,
		WorkflowRunStoppedEvent,
		WorkflowRunFailedEvent,
		WorkflowRunInterruptedEvent,
	}
}

// MissionActivityFromEvents builds the list-specific activity view from
// existing ledger and workflow projections. Read state remains a browser concern.
func MissionActivityFromEvents(events []LedgerEvent) MissionActivitySummary {
	summary := MissionActivitySummary{
		ActiveWork: ActiveWorkFromMissionState(events, projectWorkflowRuns(events)),
	}
	if len(events) == 0 {
		return summary
	}
	summary.LastSequence = events[len(events)-1].Sequence
	for index := len(events) - 1; index >= 0; index-- {
		if terminal, ok := terminalActivityFromEvent(events[index]); ok {
			summary.LatestTerminalActivity = &terminal
			break
		}
	}
	return summary
}

// MissionActivityFromInput derives a list summary from a storage-efficient
// subset of the ledger while retaining the sequence of the complete ledger.
func MissionActivityFromInput(input MissionActivityInput) MissionActivitySummary {
	summary := MissionActivityFromEvents(input.Events)
	summary.LastSequence = input.LastSequence
	return summary
}

func terminalActivityFromEvent(event LedgerEvent) (TerminalActivityView, bool) {
	activity := TerminalActivityView{EventID: event.EventID, Sequence: event.Sequence}
	switch event.EventType {
	case turnAgentResponseEvent:
		outcome, ok := terminalTurnOutcome(event.Payload)
		if !ok {
			return TerminalActivityView{}, false
		}
		activity.Kind, activity.Outcome = TerminalActivityTurn, outcome
	case "report.drafted", "report.artifact.created", "report.artifact.exported", "report.humanize.skipped":
		activity.Kind, activity.Outcome = TerminalActivityReport, TerminalActivityCompleted
	case "report.draft.failed", "report.design.failed", "report.humanize.failed", "report.patch.failed":
		activity.Kind, activity.Outcome = TerminalActivityReport, TerminalActivityFailed
	case WorkflowRunCompletedEvent:
		activity.Kind, activity.Outcome = TerminalActivityWorkflow, TerminalActivityCompleted
	case WorkflowRunPausedEvent:
		activity.Kind, activity.Outcome = TerminalActivityWorkflow, TerminalActivityPaused
	case WorkflowRunStoppedEvent:
		activity.Kind, activity.Outcome = TerminalActivityWorkflow, TerminalActivityStopped
	case WorkflowRunFailedEvent, WorkflowRunInterruptedEvent:
		activity.Kind, activity.Outcome = TerminalActivityWorkflow, TerminalActivityFailed
	default:
		return TerminalActivityView{}, false
	}
	return activity, true
}

func terminalTurnOutcome(payload []byte) (TerminalActivityOutcome, bool) {
	var turn struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(payload, &turn); err != nil {
		return "", false
	}
	switch strings.TrimSpace(turn.Kind) {
	case "", "agent_response", "agent_compacted", "agent_compaction_skipped":
		return TerminalActivityCompleted, true
	case "agent_error", "placeholder":
		return TerminalActivityFailed, true
	case "agent_canceled":
		return TerminalActivityCanceled, true
	default:
		return "", false
	}
}
