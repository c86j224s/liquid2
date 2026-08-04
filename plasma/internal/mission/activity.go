package mission

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

const (
	turnAgentPendingEvent  = "turn.agent.pending"
	turnAgentResponseEvent = "turn.agent.response"
)

// ActivityEventTypes returns only event types needed for mission-list activity.
func ActivityEventTypes() []string {
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
		workflowstate.WorkflowRunRequestedEvent,
		workflowstate.WorkflowRunStartedEvent,
		workflowstate.WorkflowRunStopRequestedEvent,
		workflowstate.WorkflowSourceSkippedEvent,
		workflowstate.WorkflowStepStartedEvent,
		workflowstate.WorkflowStepCompletedEvent,
		workflowstate.WorkflowRunCompletedEvent,
		workflowstate.WorkflowRunPausedEvent,
		workflowstate.WorkflowRunStoppedEvent,
		workflowstate.WorkflowRunFailedEvent,
		workflowstate.WorkflowRunInterruptedEvent,
	}
}

// ActivityFromEvents derives a mission-list summary from durable events.
func ActivityFromEvents(events []ledger.Event) ActivitySummary {
	summary := ActivitySummary{
		ActiveWork: ActiveWorkFromState(events, workflowstate.ProjectRuns(workflowEvents(events))),
	}
	if len(events) == 0 {
		return summary
	}
	summary.LastSequence = events[len(events)-1].Sequence
	for index := len(events) - 1; index >= 0; index-- {
		if terminal, ok := TerminalActivityFromEvent(events[index]); ok {
			summary.LatestTerminalActivity = &terminal
			break
		}
	}
	return summary
}

// ActivityFromInput preserves the ledger's full last sequence while using the
// reduced event subset loaded for list views.
func ActivityFromInput(input ActivityInput) ActivitySummary {
	summary := ActivityFromEvents(input.Events)
	summary.LastSequence = input.LastSequence
	return summary
}

// ActiveWorkFromState derives blocking activity only from durable mission and
// workflow state. The result must not depend on process-local locks or UI state.
func ActiveWorkFromState(events []ledger.Event, runs []workflowstate.WorkflowRunView) ActiveWorkState {
	items := make([]ActiveWorkView, 0, 3)
	stateEvents := ledgerStateEvents(events)
	if pending, ok := ledgerstate.OpenAgentPendingEvent(stateEvents); ok {
		items = append(items, ActiveWorkView{Kind: ActiveWorkTurn, Status: "running", ReasonCode: BlockingReasonAgentTurn, Action: "cancel_turn", Target: "conversation", PendingEventID: pending.EventID})
	}
	if pending, ok := ledgerstate.OpenReportPendingEvent(stateEvents); ok {
		items = append(items, ActiveWorkView{Kind: ActiveWorkReport, Status: "running", ReasonCode: BlockingReasonReport, Action: "cancel_report", Target: "reports", PendingEventID: pending.EventID})
	}
	for _, run := range runs {
		if !activeWorkflowStatus(run.Status) {
			continue
		}
		items = append(items, ActiveWorkView{Kind: ActiveWorkWorkflow, Status: run.Status, ReasonCode: BlockingReasonWorkflow, Action: "view_workflow", Target: "workflow", WorkflowRunID: run.WorkflowRunID})
	}
	return ActiveWorkState{Items: items, Blocks: items, BlockedControls: blockedControls(items)}
}

// TerminalActivityFromEvent recognizes user-visible terminal activity events.
func TerminalActivityFromEvent(event ledger.Event) (TerminalActivityView, bool) {
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
	case workflowstate.WorkflowRunCompletedEvent:
		activity.Kind, activity.Outcome = TerminalActivityWorkflow, TerminalActivityCompleted
	case workflowstate.WorkflowRunPausedEvent:
		activity.Kind, activity.Outcome = TerminalActivityWorkflow, TerminalActivityPaused
	case workflowstate.WorkflowRunStoppedEvent:
		activity.Kind, activity.Outcome = TerminalActivityWorkflow, TerminalActivityStopped
	case workflowstate.WorkflowRunFailedEvent, workflowstate.WorkflowRunInterruptedEvent:
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
	if json.Unmarshal(payload, &turn) != nil {
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

func activeWorkflowStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case workflowstate.WorkflowStatusQueued, workflowstate.WorkflowStatusRunning, workflowstate.WorkflowStatusStopping:
		return true
	default:
		return false
	}
}

func blockedControls(items []ActiveWorkView) []ActiveWorkControl {
	if len(items) == 0 {
		return []ActiveWorkControl{}
	}
	reasons := make([]string, 0, len(items))
	for _, item := range items {
		reasons = append(reasons, item.ReasonCode)
	}
	return []ActiveWorkControl{
		{Control: "turn_submit", ReasonCodes: reasons},
		{Control: "workflow_start", ReasonCodes: reasons},
		{Control: "report_start", ReasonCodes: reasons},
	}
}

func ledgerStateEvents(events []ledger.Event) []ledgerstate.Event {
	converted := make([]ledgerstate.Event, 0, len(events))
	for _, event := range events {
		converted = append(converted, ledgerstate.Event{
			EventID: event.EventID, Sequence: event.Sequence, EventType: event.EventType,
			Payload: event.Payload, CreatedAt: event.CreatedAt,
		})
	}
	return converted
}

func workflowEvents(events []ledger.Event) []workflowstate.Event {
	converted := make([]workflowstate.Event, 0, len(events))
	for _, event := range events {
		converted = append(converted, workflowstate.Event{
			EventID: event.EventID, MissionID: event.MissionID, Sequence: event.Sequence,
			EventType: event.EventType, Payload: event.Payload, CreatedAt: event.CreatedAt,
		})
	}
	return converted
}
