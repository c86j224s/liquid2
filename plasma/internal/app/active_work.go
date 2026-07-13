package app

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
)

const (
	ActiveWorkTurn     = "agent_turn"
	ActiveWorkReport   = "report_generation"
	ActiveWorkWorkflow = "workflow_run"

	BlockingReasonAgentTurn = "agent_turn_running"
	BlockingReasonReport    = "report_generation_running"
	BlockingReasonWorkflow  = "workflow_running"
)

type ActiveWorkView struct {
	Kind           string `json:"kind"`
	Status         string `json:"status"`
	ReasonCode     string `json:"reason_code"`
	Action         string `json:"action"`
	Target         string `json:"target"`
	PendingEventID string `json:"pending_event_id,omitempty"`
	WorkflowRunID  string `json:"workflow_run_id,omitempty"`
}

type ActiveWorkControl struct {
	Control     string   `json:"control"`
	ReasonCodes []string `json:"reason_codes"`
}

type ActiveWorkState struct {
	Items           []ActiveWorkView    `json:"items"`
	Blocks          []ActiveWorkView    `json:"blocks"`
	BlockedControls []ActiveWorkControl `json:"blocked_controls"`
}

// ActiveWorkFromMissionState projects only durable mission ledger and workflow state.
func ActiveWorkFromMissionState(events []LedgerEvent, runs []WorkflowRunView) ActiveWorkState {
	stateEvents := ledgerStateEventsFromApp(events)
	items := make([]ActiveWorkView, 0, 3)
	if pending, ok := ledgerstate.OpenAgentPendingEvent(stateEvents); ok {
		items = append(items, ActiveWorkView{Kind: ActiveWorkTurn, Status: "running", ReasonCode: BlockingReasonAgentTurn, Action: "cancel_turn", Target: "conversation", PendingEventID: pending.EventID})
	}
	if pending, ok := ledgerstate.OpenReportPendingEvent(stateEvents); ok {
		items = append(items, ActiveWorkView{Kind: ActiveWorkReport, Status: "running", ReasonCode: BlockingReasonReport, Action: "cancel_report", Target: "reports", PendingEventID: pending.EventID})
	}
	for _, run := range runs {
		if !isActiveWorkflowStatus(run.Status) {
			continue
		}
		items = append(items, ActiveWorkView{Kind: ActiveWorkWorkflow, Status: run.Status, ReasonCode: BlockingReasonWorkflow, Action: "view_workflow", Target: "workflow", WorkflowRunID: run.WorkflowRunID})
	}
	return ActiveWorkState{Items: items, Blocks: items, BlockedControls: activeWorkControls(items)}
}

func isActiveWorkflowStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case WorkflowStatusQueued, WorkflowStatusRunning, WorkflowStatusStopping:
		return true
	default:
		return false
	}
}

func activeWorkControls(items []ActiveWorkView) []ActiveWorkControl {
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
