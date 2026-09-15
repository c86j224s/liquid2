package ledgerstate

import (
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
)

func TestProjectReportProgressMarksReusedLongFormStagesCompleted(t *testing.T) {
	rootID := "evt_reused_root"
	retryID := "evt_reused_retry"
	events := []Event{
		reportEvent(rootID, "report.draft.pending", map[string]any{
			"origin_pending_event_id": rootID, "report_mode": "long_form",
			"pipeline_family": reportpipeline.ExperimentalIL, "pipeline_graph": reportpipeline.ExperimentalILValidationProfilesGraph,
		}),
		reportEvent("evt_root_failed", "report.draft.failed", map[string]any{"pending_event_id": rootID, "failed_stage_kind": "il_reader"}),
		reportEvent(retryID, "report.draft.pending", map[string]any{
			"origin_pending_event_id": rootID, "retry_of_pending_event_id": rootID, "retry_strategy": "resume_failed",
			"report_mode": "long_form", "pipeline_family": reportpipeline.ExperimentalIL,
			"pipeline_graph": reportpipeline.ExperimentalILValidationProfilesGraph, "rigor_level": "strict",
		}),
	}
	events = append(events,
		reportEvent("evt_source_started", "report.source_packet.started", map[string]any{"pending_event_id": retryID}),
		reportEvent("evt_source_completed", "report.source_packet.completed", map[string]any{"pending_event_id": retryID}),
	)
	for index, stage := range []string{"il_source_selection", "il_editorial_memory", "il_narrative", "il_long_form_plan", "il_long_form_sections", "il_long_form_parts", "il_long_form_final"} {
		events = append(events, reportEvent("evt_reused_"+itoa(index+1), "report."+stage+".reused", map[string]any{"pending_event_id": retryID}))
	}
	events = append(events, reportEvent("evt_root_plan", "report.il_long_form_plan.created", map[string]any{
		"pending_event_id": rootID,
		"plan": map[string]any{"parts": []any{
			map[string]any{"sections": []any{map[string]any{"title": "One"}, map[string]any{"title": "Two"}, map[string]any{"title": "Three"}}},
			map[string]any{"sections": []any{map[string]any{"title": "Four"}, map[string]any{"title": "Five"}, map[string]any{"title": "Six"}}},
		}},
	}))
	progress := ProjectReportProgress(events)
	for _, stage := range []string{"il_source_selection", "il_editorial_memory", "il_narrative", "il_long_form_plan", "il_long_form_sections", "il_long_form_parts", "il_long_form_final"} {
		if state := reportNodeState(progress.Nodes, stage); state != "completed" {
			t.Fatalf("reused stage %s state=%q: %#v", stage, state, progress.Nodes)
		}
	}
	if state := reportNodeState(progress.Nodes, "il_reader"); state != "running" {
		t.Fatalf("reader state=%q, want running: %#v", state, progress.Nodes)
	}
}

func TestProjectReportProgressCurrentRetryOverridesAncestorFailure(t *testing.T) {
	rootID := "evt_reused_failure_root"
	retryID := "evt_reused_failure_retry"
	events := []Event{
		reportEvent(rootID, "report.draft.pending", map[string]any{
			"origin_pending_event_id": rootID, "report_mode": "long_form",
			"pipeline_family": reportpipeline.ExperimentalIL, "pipeline_graph": reportpipeline.ExperimentalILValidationProfilesGraph,
		}),
		reportEvent("evt_memory_started", "report.il_editorial_memory.started", map[string]any{"pending_event_id": rootID}),
		reportEvent("evt_memory_failed", "report.il_editorial_memory.failed", map[string]any{
			"pending_event_id": rootID, "safe_error_message": "old failure",
		}),
		reportEvent("evt_root_failed", "report.draft.failed", map[string]any{
			"pending_event_id": rootID, "failed_stage_kind": "il_editorial_memory", "safe_error_message": "old failure",
		}),
		reportEvent(retryID, "report.draft.pending", map[string]any{
			"origin_pending_event_id": rootID, "retry_of_pending_event_id": rootID, "retry_strategy": "resume_failed",
			"report_mode": "long_form", "pipeline_family": reportpipeline.ExperimentalIL,
			"pipeline_graph": reportpipeline.ExperimentalILValidationProfilesGraph,
		}),
		reportEvent("evt_source_started", "report.source_packet.started", map[string]any{"pending_event_id": retryID}),
		reportEvent("evt_source_completed", "report.source_packet.completed", map[string]any{"pending_event_id": retryID}),
		reportEvent("evt_selection_reused", "report.il_source_selection.reused", map[string]any{"pending_event_id": retryID}),
		reportEvent("evt_retry_memory_started", "report.il_editorial_memory.started", map[string]any{"pending_event_id": retryID}),
	}
	progress := ProjectReportProgress(events)
	memory := reportProgressNode(progress.Nodes, "il_editorial_memory")
	if progress.State != "running" || memory.State != "running" || memory.AttemptID != retryID || memory.Error != "" {
		t.Fatalf("current attempt did not override ancestor failure: %#v %#v", progress, memory)
	}
}

func reportProgressNode(nodes []ReportProgressNode, id string) ReportProgressNode {
	for _, node := range nodes {
		if node.ID == id {
			return node
		}
	}
	return ReportProgressNode{}
}
