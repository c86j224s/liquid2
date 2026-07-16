package ledgerstate

import (
	"encoding/json"
	"testing"
	"time"
)

func reportEvent(id, kind string, payload map[string]any) Event {
	b, _ := json.Marshal(payload)
	return Event{EventID: id, EventType: kind, Payload: b}
}

func TestProjectReportProgressLongFormFailure(t *testing.T) {
	events := []Event{
		reportEvent("evt_pending", "report.draft.pending", map[string]any{"report_mode": "long_form"}),
		reportEvent("evt_plan", "report.plan.created", map[string]any{"pending_event_id": "evt_pending", "plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one", "two"}}}}}),
		reportEvent("evt_section", "report.section.created", map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}),
		reportEvent("evt_failed", "report.section.failed", map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 2, "safe_error_message": "agent unavailable"}),
		reportEvent("evt_terminal", "report.draft.failed", map[string]any{"pending_event_id": "evt_pending", "failed_stage_kind": "section", "failed_stage_id": "section-1-2"}),
	}
	progress := ProjectReportProgress(events)
	if progress.State != "failed" || !progress.Retry.ResumeFailed || !progress.Retry.Restart {
		t.Fatalf("unexpected retry state: %#v", progress)
	}
	states := map[string]string{}
	for _, node := range progress.Nodes {
		states[node.ID] = node.State
	}
	if states["plan"] != "completed" || states["section-1-1"] != "completed" || states["section-1-2"] != "failed" {
		t.Fatalf("unexpected nodes: %#v", states)
	}
}

func TestProjectReportProgressSupportsLegacyTerminalStageIDInKind(t *testing.T) {
	progress := ProjectReportProgress([]Event{reportEvent("evt_pending", "report.draft.pending", map[string]any{"report_mode": "long_form"}), reportEvent("evt_terminal", "report.draft.failed", map[string]any{"pending_event_id": "evt_pending", "failed_stage_kind": "final"})})
	if progress.State != "failed" {
		t.Fatalf("legacy terminal should fail: %#v", progress)
	}
}

func TestProjectReportProgressLegacyPendingIsConservative(t *testing.T) {
	progress := ProjectReportProgress([]Event{reportEvent("evt_legacy", "report.draft.pending", map[string]any{"report_mode": "long_form"})})
	if progress.Attempt != 1 || progress.OriginID != "evt_legacy" || progress.State != "running" {
		t.Fatalf("legacy normalization failed: %#v", progress)
	}
	if len(progress.Nodes) == 0 || progress.Nodes[0].State != "running" {
		t.Fatalf("must not fabricate completion: %#v", progress.Nodes)
	}
	if progress.Nodes[0].StartedAt != nil || progress.Nodes[0].DurationMS != nil {
		t.Fatalf("legacy event must omit timing: %#v", progress.Nodes[0])
	}
}

func TestProjectReportProgressProjectsNodeTimingFromLedgerBoundaries(t *testing.T) {
	base := time.Date(2026, 7, 13, 1, 0, 0, 0, time.UTC)
	events := []Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one", "two"}}}}}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(25 * time.Second)},
		{EventID: "evt_terminal", EventType: "report.draft.failed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "failed_stage_id": "section-1-2"}), CreatedAt: base.Add(40 * time.Second)},
	}
	progress := ProjectReportProgress(events)
	nodes := map[string]ReportProgressNode{}
	for _, node := range progress.Nodes {
		nodes[node.ID] = node
	}
	assertNodeTiming(t, nodes["plan"], base, 10_000)
	assertNodeTiming(t, nodes["section-1-1"], base.Add(10*time.Second), 15_000)
	assertNodeTiming(t, nodes["section-1-2"], base.Add(25*time.Second), 15_000)
	if nodes["part-1"].StartedAt != nil || nodes["part-1"].DurationMS != nil {
		t.Fatalf("unreached node must omit timing: %#v", nodes["part-1"])
	}
}

func TestProjectReportProgressRunsEverySectionBeforePartAssembly(t *testing.T) {
	events := []Event{
		reportEvent("evt_pending", "report.draft.pending", map[string]any{"report_mode": "long_form"}),
		reportEvent("evt_plan", "report.plan.created", map[string]any{"pending_event_id": "evt_pending", "plan": map[string]any{"parts": []any{
			map[string]any{"sections": []any{"part one"}},
			map[string]any{"sections": []any{"part two"}},
		}}}),
		reportEvent("evt_section_1", "report.section.created", map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}),
	}

	progress := ProjectReportProgress(events)
	if got := reportNodeState(progress.Nodes, "section-2-1"); got != "running" {
		t.Fatalf("next part section must run before part assembly, got %q: %#v", got, progress.Nodes)
	}
	if got := reportNodeState(progress.Nodes, "part-1"); got != "pending" {
		t.Fatalf("part assembly must remain pending until every section completes, got %q: %#v", got, progress.Nodes)
	}
	wantOrder := []string{"plan", "section-1-1", "section-2-1", "part-1", "part-2", "final", "artifact"}
	if len(progress.Nodes) != len(wantOrder) {
		t.Fatalf("unexpected node count: %#v", progress.Nodes)
	}
	for i, want := range wantOrder {
		if progress.Nodes[i].ID != want {
			t.Fatalf("node %d = %q, want %q: %#v", i, progress.Nodes[i].ID, want, progress.Nodes)
		}
	}
}

func TestProjectReportProgressTimesPartAssemblyAfterEverySection(t *testing.T) {
	base := time.Date(2026, 7, 14, 1, 0, 0, 0, time.UTC)
	events := []Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "plan": map[string]any{"parts": []any{
			map[string]any{"sections": []any{"part one"}},
			map[string]any{"sections": []any{"part two"}},
		}}}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section_1", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(20 * time.Second)},
		{EventID: "evt_section_2", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 2, "section_index": 1}), CreatedAt: base.Add(30 * time.Second)},
		{EventID: "evt_part_1", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(40 * time.Second)},
	}

	progress := ProjectReportProgress(events)
	nodes := map[string]ReportProgressNode{}
	for _, node := range progress.Nodes {
		nodes[node.ID] = node
	}
	assertNodeTiming(t, nodes["section-2-1"], base.Add(20*time.Second), 10_000)
	assertNodeTiming(t, nodes["part-1"], base.Add(30*time.Second), 10_000)
	if nodes["part-2"].State != "running" || nodes["part-2"].StartedAt == nil || !nodes["part-2"].StartedAt.Equal(base.Add(40*time.Second)) {
		t.Fatalf("second part must start after first part completion: %#v", nodes["part-2"])
	}
}

func reportNodeState(nodes []ReportProgressNode, id string) string {
	for _, node := range nodes {
		if node.ID == id {
			return node.State
		}
	}
	return ""
}

func mustReportPayload(t *testing.T, payload map[string]any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func assertNodeTiming(t *testing.T, node ReportProgressNode, startedAt time.Time, durationMS int64) {
	t.Helper()
	if node.StartedAt == nil || !node.StartedAt.Equal(startedAt) || node.DurationMS == nil || *node.DurationMS != durationMS {
		t.Fatalf("unexpected timing: %#v", node)
	}
}

func TestProjectReportProgressProjectsNonDraftReportOperations(t *testing.T) {
	progress := ProjectReportProgress([]Event{
		reportEvent("evt_h5", "report.humanize.pending", map[string]any{"report_mode": "planned"}),
		reportEvent("evt_h5_done", "report.artifact.exported", map[string]any{"pending_event_id": "evt_h5"}),
	})
	if progress.State != "completed" || len(progress.Nodes) != 3 || progress.Nodes[0].ID != "start" || progress.Nodes[2].ID != "artifact" || progress.Nodes[2].State != "completed" {
		t.Fatalf("humanize operation should project as a completed compact pipeline: %#v", progress)
	}
}

func TestProjectReportProgressRejectsConflictingTerminalOutcomes(t *testing.T) {
	progress := ProjectReportProgress([]Event{
		reportEvent("evt_patch", "report.patch.pending", map[string]any{}),
		reportEvent("evt_patch_failed", "report.patch.failed", map[string]any{"pending_event_id": "evt_patch"}),
		reportEvent("evt_patch_done", "report.artifact.created", map[string]any{"pending_event_id": "evt_patch"}),
	})
	if progress.State != "unknown" || progress.Retry.ReasonCode != "invalid_lineage" {
		t.Fatalf("conflicting terminal outcomes must fail closed: %#v", progress)
	}
}

func TestProjectReportProgressTreatsPatchFinalizedAsIntermediate(t *testing.T) {
	progress := ProjectReportProgress([]Event{
		reportEvent("evt_patch", "report.patch.pending", map[string]any{}),
		reportEvent("evt_finalized", "report.patch.finalized", map[string]any{"pending_event_id": "evt_patch"}),
		reportEvent("evt_artifact", "report.artifact.created", map[string]any{"pending_event_id": "evt_patch"}),
	})
	if progress.State != "completed" {
		t.Fatalf("patch finalize must remain intermediate: %#v", progress)
	}
}

func TestProjectReportProgressMarksEveryCanceledOperationSkipped(t *testing.T) {
	for _, item := range []struct{ pending, failed, kind string }{{"report.draft.pending", "report.draft.failed", "report_draft_canceled"}, {"report.design.pending", "report.design.failed", "designed_html_report_canceled"}, {"report.humanize.pending", "report.humanize.failed", "humanized_markdown_report_canceled"}, {"report.patch.pending", "report.patch.failed", "report_patch_canceled"}} {
		progress := ProjectReportProgress([]Event{reportEvent("evt_pending", item.pending, map[string]any{}), reportEvent("evt_canceled", item.failed, map[string]any{"pending_event_id": "evt_pending", "kind": item.kind})})
		if progress.State != "skipped" || progress.Retry.Restart || progress.Retry.ResumeFailed {
			t.Fatalf("canceled %s must be skipped without retry: %#v", item.pending, progress)
		}
	}
}

func TestProjectReportProgressDoesNotCompleteRetryFromAncestorArtifact(t *testing.T) {
	events := []Event{
		reportEvent("evt_root", "report.draft.pending", map[string]any{"report_mode": "long_form"}),
		reportEvent("evt_root_artifact", "report.artifact.created", map[string]any{"pending_event_id": "evt_root"}),
		reportEvent("evt_retry", "report.draft.pending", map[string]any{"report_mode": "long_form", "origin_pending_event_id": "evt_root", "retry_of_pending_event_id": "evt_root", "attempt_number": 2}),
	}
	progress := ProjectReportProgress(events)
	if progress.State != "running" {
		t.Fatalf("ancestor artifact completed retry: %#v", progress)
	}
	for _, node := range progress.Nodes {
		if node.ID == "artifact" && node.State == "completed" {
			t.Fatal("ancestor artifact must not complete selected attempt")
		}
	}
}

func TestProjectReportProgressRejectsCorruptLineage(t *testing.T) {
	for _, events := range [][]Event{
		{reportEvent("evt_a", "report.draft.pending", map[string]any{"report_mode": "long_form", "origin_pending_event_id": "evt_a", "retry_of_pending_event_id": "evt_b"}), reportEvent("evt_b", "report.draft.pending", map[string]any{"report_mode": "long_form", "origin_pending_event_id": "evt_a", "retry_of_pending_event_id": "evt_a"})},
		{reportEvent("evt_a", "report.draft.pending", map[string]any{"report_mode": "long_form", "origin_pending_event_id": "evt_missing", "retry_of_pending_event_id": "evt_missing"})},
	} {
		if got := ProjectReportProgress(events); got.State != "unknown" || got.Retry.ReasonCode != "invalid_lineage" {
			t.Fatalf("unsafe projection: %#v", got)
		}
	}
}
