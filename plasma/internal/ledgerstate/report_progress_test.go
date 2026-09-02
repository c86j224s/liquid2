package ledgerstate

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
)

func reportEvent(id, kind string, payload map[string]any) Event {
	b, _ := json.Marshal(payload)
	return Event{EventID: id, EventType: kind, Payload: b}
}

func TestProjectReportProgressExperimentalStagesPreservePendingMetadataAndTiming(t *testing.T) {
	base := time.Date(2026, 8, 21, 1, 0, 0, 0, time.UTC)
	pendingID := "evt_experimental_pending"
	events := []Event{{EventID: pendingID, EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{
		"origin_pending_event_id": pendingID, "attempt_number": 7,
		"report_mode": "planned", "pipeline_family": "report_il_experimental", "pipeline_graph": "report_il_source_anchored_images_v6",
	}), CreatedAt: base}}
	stages := []string{"source_packet", "il_source_selection", "il_editorial_memory", "il_narrative", "il_reader", "il_continuity", "il_images", "il_document", "il_render", "il_store"}
	for index, stage := range stages {
		start := base.Add(time.Duration(index+1) * time.Minute)
		completed := start.Add(10 * time.Second)
		events = append(events,
			Event{EventID: "evt_" + stage + "_started", EventType: "report." + stage + ".started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID}), CreatedAt: start},
			Event{EventID: "evt_" + stage + "_completed", EventType: "report." + stage + ".completed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID}), CreatedAt: completed},
		)
	}
	events = append(events,
		Event{EventID: "evt_final", EventType: "report.artifact.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID}), CreatedAt: base.Add(10 * time.Minute)},
	)
	progress := ProjectReportProgress(events)
	wantOrder := []string{"start", "source_packet", "il_source_selection", "il_editorial_memory", "il_narrative", "il_reader", "il_continuity", "il_images", "il_document", "il_render", "il_store", "final", "artifact"}
	if len(progress.Nodes) != len(wantOrder) {
		t.Fatalf("node count = %d, want %d: %#v", len(progress.Nodes), len(wantOrder), progress.Nodes)
	}
	for index, want := range wantOrder {
		if progress.Nodes[index].ID != want {
			t.Fatalf("node %d = %q, want %q: %#v", index, progress.Nodes[index].ID, want, progress.Nodes)
		}
	}
	if progress.OriginID != pendingID || progress.Attempt != 7 || progress.AttemptID != pendingID || progress.State != "completed" {
		t.Fatalf("pending metadata/state overwritten: %#v", progress)
	}
	for index, stage := range stages {
		node := progress.Nodes[index+1]
		if node.State != "completed" || node.AttemptID != pendingID || node.StartedAt == nil || !node.StartedAt.Equal(base.Add(time.Duration(index+1)*time.Minute)) || node.DurationMS == nil || *node.DurationMS != 10_000 {
			t.Fatalf("stage %s projection = %#v", stage, node)
		}
	}
}

func TestProjectReportProgressLongFormExperimentalRemainsStageBased(t *testing.T) {
	pendingID := "evt_long_form_il_pending"
	progress := ProjectReportProgress([]Event{{
		EventID: pendingID, EventType: "report.draft.pending",
		Payload: mustReportPayload(t, map[string]any{
			"report_mode": "long_form", "pipeline_family": reportpipeline.ExperimentalIL,
			"pipeline_graph": reportpipeline.ExperimentalILValidationProfilesGraph,
			"rigor_level":    "strict",
		}),
	}})
	want := []string{
		"start", "source_packet", "il_editorial_memory", "il_narrative",
		"il_long_form_plan", "il_long_form_sections", "il_long_form_parts", "il_long_form_final",
		"il_reader", "il_continuity", "il_images", "il_document", "il_render", "il_store", "final", "artifact",
	}
	if len(progress.Nodes) != len(want) {
		t.Fatalf("long-form IL node count = %d, want %d: %#v", len(progress.Nodes), len(want), progress.Nodes)
	}
	for index, id := range want {
		if progress.Nodes[index].ID != id {
			t.Fatalf("long-form IL node %d = %q, want %q: %#v", index, progress.Nodes[index].ID, id, progress.Nodes)
		}
	}
	for _, node := range progress.Nodes {
		if node.Kind == "part" || node.Kind == "section" || node.Kind == "part_edit" {
			t.Fatalf("long-form IL fabricated classic sectional progress: %#v", progress.Nodes)
		}
	}
}

func TestProjectReportProgressLongFormILProjectsDurableSectionFanout(t *testing.T) {
	base := time.Date(2026, 8, 31, 1, 0, 0, 0, time.UTC)
	pendingID := "evt_long_form_fanout_pending"
	plan := map[string]any{"parts": []any{
		map[string]any{"sections": []any{map[string]any{"title": "Section 1"}, map[string]any{"title": "Section 2"}}},
		map[string]any{"sections": []any{map[string]any{"title": "Section 3"}}},
	}}
	events := []Event{
		{EventID: pendingID, EventType: "report.draft.pending", CreatedAt: base, Payload: mustReportPayload(t, map[string]any{
			"report_mode": "long_form", "pipeline_family": reportpipeline.ExperimentalIL,
			"pipeline_graph": reportpipeline.ExperimentalILValidationProfilesGraph, "rigor_level": "strict",
		})},
		{EventID: "evt_plan_stage_started", EventType: "report.il_long_form_plan.started", CreatedAt: base.Add(time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID})},
		{EventID: "evt_plan_created", EventType: "report.il_long_form_plan.created", CreatedAt: base.Add(2 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "plan": plan})},
		{EventID: "evt_plan_stage_completed", EventType: "report.il_long_form_plan.completed", CreatedAt: base.Add(3 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID})},
		{EventID: "evt_sections_started", EventType: "report.il_long_form_sections.started", CreatedAt: base.Add(4 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID})},
		{EventID: "evt_section_1_started", EventType: "report.il_long_form_section.started", CreatedAt: base.Add(5 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "part_index": 1, "section_index": 1})},
		{EventID: "evt_section_2_started", EventType: "report.il_long_form_section.started", CreatedAt: base.Add(5 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "part_index": 1, "section_index": 2})},
		{EventID: "evt_section_3_started", EventType: "report.il_long_form_section.started", CreatedAt: base.Add(5 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "part_index": 2, "section_index": 1})},
		{EventID: "evt_section_1_completed", EventType: "report.il_long_form_section.completed", CreatedAt: base.Add(7 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "part_index": 1, "section_index": 1})},
		{EventID: "evt_section_2_completed", EventType: "report.il_long_form_section.completed", CreatedAt: base.Add(8 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "part_index": 1, "section_index": 2})},
		{EventID: "evt_section_3_completed", EventType: "report.il_long_form_section.completed", CreatedAt: base.Add(9 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "part_index": 2, "section_index": 1})},
		{EventID: "evt_sections_completed", EventType: "report.il_long_form_sections.completed", CreatedAt: base.Add(10 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID})},
		{EventID: "evt_parts_started", EventType: "report.il_long_form_parts.started", CreatedAt: base.Add(11 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID})},
		{EventID: "evt_part_1_started", EventType: "report.il_long_form_part.started", CreatedAt: base.Add(12 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "part_index": 1})},
		{EventID: "evt_part_1_completed", EventType: "report.il_long_form_part.completed", CreatedAt: base.Add(14 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "part_index": 1})},
		{EventID: "evt_part_2_started", EventType: "report.il_long_form_part.started", CreatedAt: base.Add(15 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "part_index": 2})},
	}

	progress := ProjectReportProgress(events)
	nodes := map[string]ReportProgressNode{}
	for _, node := range progress.Nodes {
		nodes[node.ID] = node
	}
	for _, id := range []string{"section-1-1", "section-1-2", "section-2-1", "part-edit-1", "part-edit-2"} {
		if _, ok := nodes[id]; !ok {
			t.Fatalf("long-form IL detail node %q missing: %#v", id, progress.Nodes)
		}
	}
	if nodes["section-1-1"].State != "completed" || nodes["section-1-2"].State != "completed" || nodes["section-2-1"].State != "completed" ||
		nodes["part-edit-1"].State != "completed" || nodes["part-edit-2"].State != "running" {
		t.Fatalf("long-form IL detail states = %#v", nodes)
	}
	for _, id := range []string{"section-1-1", "section-1-2", "section-2-1"} {
		if nodes[id].StartedAt == nil || !nodes[id].StartedAt.Equal(base.Add(5*time.Second)) {
			t.Fatalf("parallel Section %q timing = %#v", id, nodes[id])
		}
	}
	if nodes["part-edit-1"].StartedAt == nil || nodes["part-edit-1"].DurationMS == nil || *nodes["part-edit-1"].DurationMS != 2_000 ||
		nodes["part-edit-2"].StartedAt == nil || !nodes["part-edit-2"].StartedAt.Equal(base.Add(15*time.Second)) {
		t.Fatalf("sequential Part timing = %#v / %#v", nodes["part-edit-1"], nodes["part-edit-2"])
	}
}

func TestProjectReportProgressLongFormILRejectsMalformedDetailCoordinates(t *testing.T) {
	pendingID := "evt_long_form_invalid_detail_pending"
	plan := map[string]any{"parts": []any{map[string]any{"sections": []any{map[string]any{"title": "Section"}}}}}
	progress := ProjectReportProgress([]Event{
		{EventID: pendingID, EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{
			"report_mode": "long_form", "pipeline_family": reportpipeline.ExperimentalIL,
			"pipeline_graph": reportpipeline.ExperimentalILValidationProfilesGraph, "rigor_level": "strict",
		})},
		{EventID: "evt_plan_created", EventType: "report.il_long_form_plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": pendingID, "plan": plan,
		})},
		{EventID: "evt_unknown_section", EventType: "report.il_long_form_section.started", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": pendingID, "part_index": 4, "section_index": 7,
		})},
	})
	if progress.State != "unknown" || progress.Retry.ReasonCode != "invalid_lineage" {
		t.Fatalf("malformed long-form detail coordinate projection = %#v", progress)
	}
}

func TestProjectReportProgressTimesLongFormILStages(t *testing.T) {
	base := time.Date(2026, 8, 30, 1, 0, 0, 0, time.UTC)
	pendingID := "evt_long_form_timing_pending"
	events := []Event{
		{EventID: pendingID, EventType: "report.draft.pending", CreatedAt: base, Payload: mustReportPayload(t, map[string]any{
			"report_mode": "long_form", "pipeline_family": reportpipeline.ExperimentalIL,
			"pipeline_graph": reportpipeline.ExperimentalILValidationProfilesGraph, "rigor_level": "unverified",
		})},
		{EventID: "evt_plan_started", EventType: "report.il_long_form_plan.started", CreatedAt: base.Add(time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID})},
		{EventID: "evt_plan_completed", EventType: "report.il_long_form_plan.completed", CreatedAt: base.Add(4 * time.Second), Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID})},
	}

	progress := ProjectReportProgress(events)
	nodes := map[string]ReportProgressNode{}
	for _, node := range progress.Nodes {
		nodes[node.ID] = node
	}
	assertNodeTiming(t, nodes["il_long_form_plan"], base.Add(time.Second), 3_000)
}

func TestProjectReportProgressLongFormStageFailure(t *testing.T) {
	pendingID := "evt_long_form_failure_pending"
	progress := ProjectReportProgress([]Event{
		{EventID: pendingID, EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{
			"report_mode": "long_form", "pipeline_family": reportpipeline.ExperimentalIL,
			"pipeline_graph": reportpipeline.ExperimentalILValidationProfilesGraph, "rigor_level": "strict",
		})},
		{EventID: "evt_part_started", EventType: "report.il_long_form_parts.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID})},
		{EventID: "evt_part_failed", EventType: "report.il_long_form_parts.failed", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": pendingID, "stage_kind": "il_long_form_parts", "stage_id": "il_long_form_parts",
		})},
		{EventID: "evt_terminal", EventType: "report.draft.failed", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": pendingID, "failed_stage_kind": "il_long_form_parts", "failed_stage_id": "il_long_form_parts",
		})},
	})
	if progress.State != "failed" || reportNodeState(progress.Nodes, "il_long_form_parts") != "failed" {
		t.Fatalf("long-form failure projection = %#v", progress)
	}
}

func TestProjectReportProgressValidationProfileStages(t *testing.T) {
	for _, tc := range []struct {
		profile string
		want    []string
	}{
		{profile: "unverified", want: []string{"start", "source_packet", "il_narrative", "il_images", "il_document", "il_render", "il_store", "final", "artifact"}},
		{profile: "exploratory", want: []string{"start", "source_packet", "il_editorial_memory", "il_narrative", "il_reader", "il_images", "il_document", "il_render", "il_store", "final", "artifact"}},
		{profile: "strict", want: []string{"start", "source_packet", "il_editorial_memory", "il_narrative", "il_reader", "il_continuity", "il_images", "il_document", "il_render", "il_store", "final", "artifact"}},
	} {
		t.Run(tc.profile, func(t *testing.T) {
			pendingID := "evt_" + tc.profile + "_pending"
			progress := ProjectReportProgress([]Event{{
				EventID: pendingID, EventType: "report.draft.pending",
				Payload: mustReportPayload(t, map[string]any{
					"report_mode": "planned", "pipeline_family": "report_il_experimental",
					"pipeline_graph": reportpipeline.ExperimentalILValidationProfilesGraph,
					"rigor_level":    tc.profile,
				}),
			}})
			if len(progress.Nodes) != len(tc.want) {
				t.Fatalf("%s node count = %d, want %d: %#v", tc.profile, len(progress.Nodes), len(tc.want), progress.Nodes)
			}
			for index, want := range tc.want {
				if progress.Nodes[index].ID != want {
					t.Fatalf("%s node %d = %q, want %q: %#v", tc.profile, index, progress.Nodes[index].ID, want, progress.Nodes)
				}
			}
		})
	}
}

func TestProjectReportProgressValidationProfilePreservesObservedLegacyStages(t *testing.T) {
	pendingID := "evt_unverified_legacy_pending"
	progress := ProjectReportProgress([]Event{
		{EventID: pendingID, EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{
			"report_mode": "planned", "pipeline_family": "report_il_experimental",
			"pipeline_graph": reportpipeline.ExperimentalILValidationProfilesGraph,
			"rigor_level":    "unverified",
		})},
		{EventID: "evt_memory_started", EventType: "report.il_editorial_memory.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID})},
		{EventID: "evt_reader_started", EventType: "report.il_reader.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID})},
		{EventID: "evt_continuity_started", EventType: "report.il_continuity.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID})},
	})
	for _, stage := range []string{"il_editorial_memory", "il_reader", "il_continuity"} {
		if reportNodeState(progress.Nodes, stage) != "running" {
			t.Fatalf("observed legacy stage %q was dropped: %#v", stage, progress.Nodes)
		}
	}
}

func TestProjectReportProgressLegacyExperimentalTerminalWithoutStageEvidenceKeepsFlowGraph(t *testing.T) {
	pendingID := "evt_legacy_experimental_pending"
	progress := ProjectReportProgress([]Event{
		{EventID: pendingID, EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{
			"report_mode": "planned", "pipeline_family": "report_il_experimental",
		})},
		{EventID: "evt_legacy_artifact", EventType: "report.artifact.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": pendingID,
		})},
	})
	wantOrder := []string{"start", "source_packet", "il_narrative", "il_document", "il_flow", "il_render", "il_store", "final", "artifact"}
	if len(progress.Nodes) != len(wantOrder) {
		t.Fatalf("node count = %d, want %d: %#v", len(progress.Nodes), len(wantOrder), progress.Nodes)
	}
	for index, want := range wantOrder {
		if progress.Nodes[index].ID != want {
			t.Fatalf("node %d = %q, want %q: %#v", index, progress.Nodes[index].ID, want, progress.Nodes)
		}
	}
}

func TestProjectReportProgressReaderEventOverridesMissingGraphMetadata(t *testing.T) {
	pendingID := "evt_reader_compat_pending"
	progress := ProjectReportProgress([]Event{
		{EventID: pendingID, EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{
			"report_mode": "planned", "pipeline_family": "report_il_experimental",
		})},
		{EventID: "evt_reader_started", EventType: "report.il_reader.started", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": pendingID,
		})},
	})
	if reportNodeState(progress.Nodes, "il_reader") != "running" || reportNodeState(progress.Nodes, "il_flow") != "" {
		t.Fatalf("reader evidence did not select reader graph: %#v", progress.Nodes)
	}
}

func TestProjectReportProgressExperimentalFailureRequiresCheckpointForResume(t *testing.T) {
	pendingID := "evt_experimental_retry_pending"
	progress := ProjectReportProgress([]Event{
		{EventID: pendingID, EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form", "pipeline_family": "report_il_experimental"})},
		{EventID: "evt_failed", EventType: "report.draft.failed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "failed_stage_kind": "source_packet"})},
	})
	if progress.Retry.ResumeFailed || !progress.Retry.Restart || progress.Retry.ReasonCode != "resume_checkpoint_missing" {
		t.Fatalf("experimental retry capability without a checkpoint = %#v", progress.Retry)
	}
}

func TestProjectReportProgressAcceptsPartCheckpointForResume(t *testing.T) {
	pendingID := "evt_parts_checkpoint_pending"
	progress := ProjectReportProgress([]Event{
		{EventID: pendingID, EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form", "pipeline_family": "report_il_experimental"})},
		{EventID: "evt_parts_checkpoint", EventType: "report.il.checkpoint.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "checkpoint": map[string]any{"stage": "il_long_form_parts"}})},
		{EventID: "evt_failed", EventType: "report.draft.failed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "failed_stage_kind": "il_long_form_final"})},
	})
	if !progress.Retry.ResumeFailed || !progress.Retry.Restart {
		t.Fatalf("Part checkpoint did not enable resume: %#v", progress.Retry)
	}
}

func TestProjectReportProgressExperimentalFailureCompanionAndDraftTerminal(t *testing.T) {
	base := time.Date(2026, 8, 21, 2, 0, 0, 0, time.UTC)
	pendingID := "evt_experimental_failure_pending"
	events := []Event{
		{EventID: pendingID, EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"origin_pending_event_id": pendingID, "attempt_number": 3, "report_mode": "planned", "pipeline_family": "report_il_experimental", "pipeline_graph": "report_il_editorial_v2"}), CreatedAt: base},
		{EventID: "evt_source_done", EventType: "report.source_packet.completed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID}), CreatedAt: base.Add(time.Minute)},
		{EventID: "evt_reader_start", EventType: "report.il_reader.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID}), CreatedAt: base.Add(2 * time.Minute)},
		{EventID: "evt_reader_failed", EventType: "report.il_reader.failed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "stage_kind": "il_reader", "stage_id": "il_reader", "safe_error_message": "safe reader failure"}), CreatedAt: base.Add(2*time.Minute + 12*time.Second)},
	}
	progress := ProjectReportProgress(events)
	if progress.State != "running" || progress.AttemptID != pendingID || progress.OriginID != pendingID {
		t.Fatalf("companion unexpectedly closed attempt: %#v", progress)
	}
	nodes := map[string]ReportProgressNode{}
	for _, node := range progress.Nodes {
		nodes[node.ID] = node
	}
	if nodes["il_reader"].State != "failed" || nodes["il_reader"].Error != "safe reader failure" || nodes["il_reader"].StartedAt == nil || nodes["il_reader"].DurationMS == nil || *nodes["il_reader"].DurationMS != 12_000 {
		t.Fatalf("reader failure projection = %#v", nodes["il_reader"])
	}
	if nodes["il_document"].State != "pending" || nodes["il_render"].State != "pending" || nodes["il_store"].State != "pending" {
		t.Fatalf("downstream stage states = %#v", progress.Nodes)
	}
	terminal := Event{EventID: "evt_draft_failed", EventType: "report.draft.failed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": pendingID, "failed_stage_kind": "il_reader", "failed_stage_id": "il_reader", "safe_error_message": "safe reader failure"}), CreatedAt: base.Add(3 * time.Minute)}
	progress = ProjectReportProgress(append(events, terminal))
	if progress.State != "failed" || progress.AttemptID != pendingID {
		t.Fatalf("draft terminal did not close attempt: %#v", progress)
	}
	readerNode := map[string]ReportProgressNode{}
	for _, node := range progress.Nodes {
		readerNode[node.ID] = node
	}
	if readerNode["il_reader"].State != "failed" || readerNode["il_reader"].Error != "safe reader failure" {
		t.Fatalf("terminal reader failure node = %#v", readerNode["il_reader"])
	}
}

func TestProjectReportProgressClassicPendingDoesNotGainExperimentalNodes(t *testing.T) {
	progress := ProjectReportProgress([]Event{{EventID: "evt_classic", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "planned"})}})
	for _, node := range progress.Nodes {
		if node.ID == "source_packet" || node.ID == "il_source_selection" || node.ID == "il_editorial_memory" || node.ID == "il_narrative" || node.ID == "il_continuity" || node.ID == "il_reader" || node.ID == "il_document" || node.ID == "il_flow" || node.ID == "il_render" || node.ID == "il_store" {
			t.Fatalf("classic pending gained IL node: %#v", progress.Nodes)
		}
	}
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

func TestProjectReportProgressProjectsParallelSectionStarts(t *testing.T) {
	base := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	events := []Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "plan": map[string]any{"parts": []any{
			map[string]any{"sections": []any{"one"}},
			map[string]any{"sections": []any{"two"}},
		}}}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section_1_start", EventType: "report.section.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(12 * time.Second)},
		{EventID: "evt_section_2_start", EventType: "report.section.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 2, "section_index": 1}), CreatedAt: base.Add(13 * time.Second)},
	}

	progress := ProjectReportProgress(events)
	nodes := map[string]ReportProgressNode{}
	running := 0
	for _, node := range progress.Nodes {
		nodes[node.ID] = node
		if node.State == "running" {
			running++
		}
	}
	if running != 2 || nodes["section-1-1"].State != "running" || nodes["section-2-1"].State != "running" || nodes["part-1"].State != "pending" {
		t.Fatalf("parallel section starts should be the only running nodes: %#v", progress.Nodes)
	}
	if nodes["section-1-1"].StartedAt == nil || !nodes["section-1-1"].StartedAt.Equal(base.Add(12*time.Second)) {
		t.Fatalf("first section start time not projected: %#v", nodes["section-1-1"])
	}
	if nodes["section-2-1"].StartedAt == nil || !nodes["section-2-1"].StartedAt.Equal(base.Add(13*time.Second)) {
		t.Fatalf("second section start time not projected: %#v", nodes["section-2-1"])
	}
}

func TestProjectReportProgressEvidenceGapStaysOnSectionNode(t *testing.T) {
	events := []Event{
		reportEvent("evt_pending", "report.draft.pending", map[string]any{"report_mode": "long_form"}),
		reportEvent("evt_plan", "report.plan.created", map[string]any{"pending_event_id": "evt_pending", "plan": map[string]any{"parts": []any{
			map[string]any{"sections": []any{"one"}},
		}}}),
		reportEvent("evt_gap", "report.section.evidence_gap", map[string]any{
			"pending_event_id": "evt_pending", "plan_event_id": "evt_plan",
			"part_index": 1, "section_index": 1, "attempt_number": 1,
			"reason_code": "inadequate_section_evidence",
		}),
	}

	progress := ProjectReportProgress(events)
	if got := reportNodeState(progress.Nodes, "section-1-1"); got != "running" {
		t.Fatalf("evidence gap should keep Section node running, got %q: %#v", got, progress.Nodes)
	}
	for _, node := range progress.Nodes {
		if node.Kind == "evidence_gap" {
			t.Fatalf("evidence gap must not create a visible node: %#v", progress.Nodes)
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

func TestProjectReportProgressIncludesPartEditStageWhenPlanEnablesIt(t *testing.T) {
	base := time.Date(2026, 7, 18, 1, 0, 0, 0, time.UTC)
	events := []Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_pending", "part_edit_enabled": true,
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}},
		}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(20 * time.Second)},
		{EventID: "evt_part", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(30 * time.Second)},
		{EventID: "evt_part_edit_start", EventType: "report.part_edit.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(35 * time.Second)},
		{EventID: "evt_part_edit", EventType: "report.part.edited", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(50 * time.Second)},
	}

	progress := ProjectReportProgress(events)
	wantOrder := []string{"plan", "section-1-1", "part-1", "part-edit-1", "final", "artifact"}
	if len(progress.Nodes) != len(wantOrder) {
		t.Fatalf("unexpected nodes: %#v", progress.Nodes)
	}
	nodes := map[string]ReportProgressNode{}
	for index, want := range wantOrder {
		if progress.Nodes[index].ID != want {
			t.Fatalf("node %d = %q, want %q: %#v", index, progress.Nodes[index].ID, want, progress.Nodes)
		}
		nodes[progress.Nodes[index].ID] = progress.Nodes[index]
	}
	if nodes["part-edit-1"].Kind != "part_edit" || nodes["part-edit-1"].State != "completed" {
		t.Fatalf("Part edit node not completed: %#v", nodes["part-edit-1"])
	}
	assertNodeTiming(t, nodes["part-edit-1"], base.Add(35*time.Second), 15_000)
	if nodes["final"].State != "running" {
		t.Fatalf("final should be the next running node: %#v", progress.Nodes)
	}
}

func TestProjectReportProgressIncludesReaderStyleGatePipelineStages(t *testing.T) {
	base := time.Date(2026, 7, 28, 1, 0, 0, 0, time.UTC)
	events := []Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_pending", "final_edit_pipeline": "reader_style_gate_v1", "post_report_humanize": "enabled",
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}},
		}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(20 * time.Second)},
		{EventID: "evt_part", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(30 * time.Second)},
		{EventID: "evt_reader_start", EventType: "report.final_edit.reader.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(35 * time.Second)},
		{EventID: "evt_reader", EventType: "report.final_edit.reader.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(45 * time.Second)},
		{EventID: "evt_style_start", EventType: "report.final_edit.style.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(50 * time.Second)},
		{EventID: "evt_style", EventType: "report.final_edit.style.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(65 * time.Second)},
		{EventID: "evt_gate_start", EventType: "report.final_edit.gate.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(70 * time.Second)},
		{EventID: "evt_gate", EventType: "report.final_edit.gate.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(80 * time.Second)},
		{EventID: "evt_artifact", EventType: "report.artifact.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(90 * time.Second)},
	}

	progress := ProjectReportProgress(events)
	if progress.State != "completed" {
		t.Fatalf("progress state=%q, want completed: %#v", progress.State, progress)
	}
	wantOrder := []string{"plan", "section-1-1", "part-1", "reader-edit", "style-edit", "corrective-gate", "final", "artifact"}
	if len(progress.Nodes) != len(wantOrder) {
		t.Fatalf("unexpected node count: %#v", progress.Nodes)
	}
	nodes := map[string]ReportProgressNode{}
	for index, want := range wantOrder {
		if progress.Nodes[index].ID != want {
			t.Fatalf("node %d = %q, want %q: %#v", index, progress.Nodes[index].ID, want, progress.Nodes)
		}
		nodes[want] = progress.Nodes[index]
	}
	for _, id := range wantOrder {
		if nodes[id].State != "completed" {
			t.Fatalf("%s state=%q, want completed: %#v", id, nodes[id].State, progress.Nodes)
		}
	}
	if reportNodeState(progress.Nodes, "final-assembly") != "" || reportNodeState(progress.Nodes, "final-write") != "" {
		t.Fatalf("legacy v1 progress synthesized v2 stages: %#v", progress.Nodes)
	}
	assertNodeTiming(t, nodes["reader-edit"], base.Add(35*time.Second), 10_000)
	assertNodeTiming(t, nodes["style-edit"], base.Add(50*time.Second), 15_000)
	assertNodeTiming(t, nodes["corrective-gate"], base.Add(70*time.Second), 10_000)

	disabled := ProjectReportProgress([]Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_pending", "final_edit_pipeline": "reader_style_gate_v1", "post_report_humanize": "disabled",
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}},
		}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(20 * time.Second)},
		{EventID: "evt_part", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(30 * time.Second)},
		{EventID: "evt_reader", EventType: "report.final_edit.reader.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(45 * time.Second)},
		{EventID: "evt_gate_start", EventType: "report.final_edit.gate.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(70 * time.Second)},
	})
	if reportNodeState(disabled.Nodes, "style-edit") != "" {
		t.Fatalf("disabled style should omit style node: %#v", disabled.Nodes)
	}
	if got := reportNodeState(disabled.Nodes, "corrective-gate"); got != "running" {
		t.Fatalf("gate state=%q, want running: %#v", got, disabled.Nodes)
	}
	if reportNodeState(disabled.Nodes, "final-assembly") != "" || reportNodeState(disabled.Nodes, "final-write") != "" {
		t.Fatalf("disabled legacy v1 progress synthesized v2 stages: %#v", disabled.Nodes)
	}
}

func TestProjectReportProgressIncludesAssemblyWriterReaderStyleGateV2Stages(t *testing.T) {
	base := time.Date(2026, 7, 28, 3, 0, 0, 0, time.UTC)
	events := []Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_pending", "final_edit_pipeline": "assembly_writer_reader_style_gate_v2", "post_report_humanize": "enabled",
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}},
		}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(20 * time.Second)},
		{EventID: "evt_part", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(30 * time.Second)},
		{EventID: "evt_assembly", EventType: "report.final_assembly.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(35 * time.Second)},
		{EventID: "evt_writer_start", EventType: "report.final_edit.writer.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(40 * time.Second)},
		{EventID: "evt_writer", EventType: "report.final_edit.writer.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(55 * time.Second)},
		{EventID: "evt_reader_start", EventType: "report.final_edit.reader.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(60 * time.Second)},
		{EventID: "evt_reader", EventType: "report.final_edit.reader.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(75 * time.Second)},
		{EventID: "evt_style_start", EventType: "report.final_edit.style.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(80 * time.Second)},
		{EventID: "evt_style", EventType: "report.final_edit.style.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(90 * time.Second)},
		{EventID: "evt_gate_start", EventType: "report.final_edit.gate.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(95 * time.Second)},
		{EventID: "evt_gate", EventType: "report.final_edit.gate.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(105 * time.Second)},
		{EventID: "evt_artifact", EventType: "report.artifact.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(110 * time.Second)},
	}

	progress := ProjectReportProgress(events)
	if progress.State != "completed" {
		t.Fatalf("progress state=%q, want completed: %#v", progress.State, progress)
	}
	nodes := assertReportNodeOrder(t, progress.Nodes, []string{"plan", "section-1-1", "part-1", "final-assembly", "final-write", "reader-edit", "style-edit", "corrective-gate", "final", "artifact"})
	for _, id := range []string{"plan", "section-1-1", "part-1", "final-assembly", "final-write", "reader-edit", "style-edit", "corrective-gate", "final", "artifact"} {
		if nodes[id].State != "completed" {
			t.Fatalf("%s state=%q, want completed: %#v", id, nodes[id].State, progress.Nodes)
		}
	}
	if nodes["final-assembly"].Kind != "final_assembly" || nodes["final-write"].Kind != "final_write" {
		t.Fatalf("v2 final stage kinds differ from UI label contract: %#v %#v", nodes["final-assembly"], nodes["final-write"])
	}
	assertNodeTiming(t, nodes["final-assembly"], base.Add(30*time.Second), 5_000)
	assertNodeTiming(t, nodes["final-write"], base.Add(40*time.Second), 15_000)
	assertNodeTiming(t, nodes["reader-edit"], base.Add(60*time.Second), 15_000)
	assertNodeTiming(t, nodes["style-edit"], base.Add(80*time.Second), 10_000)
	assertNodeTiming(t, nodes["corrective-gate"], base.Add(95*time.Second), 10_000)

	disabled := ProjectReportProgress([]Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_pending", "final_edit_pipeline": "assembly_writer_reader_style_gate_v2", "post_report_humanize": "disabled",
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}},
		}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(20 * time.Second)},
		{EventID: "evt_part", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(30 * time.Second)},
		{EventID: "evt_assembly", EventType: "report.final_assembly.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(35 * time.Second)},
		{EventID: "evt_writer", EventType: "report.final_edit.writer.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(55 * time.Second)},
		{EventID: "evt_reader", EventType: "report.final_edit.reader.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(75 * time.Second)},
		{EventID: "evt_gate_start", EventType: "report.final_edit.gate.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(95 * time.Second)},
	})
	assertReportNodeOrder(t, disabled.Nodes, []string{"plan", "section-1-1", "part-1", "final-assembly", "final-write", "reader-edit", "corrective-gate", "final", "artifact"})
	if reportNodeState(disabled.Nodes, "style-edit") != "" {
		t.Fatalf("disabled v2 style should omit style node: %#v", disabled.Nodes)
	}
	if got := reportNodeState(disabled.Nodes, "corrective-gate"); got != "running" {
		t.Fatalf("disabled v2 gate state=%q, want running: %#v", got, disabled.Nodes)
	}
}

func TestProjectReportProgressIncludesV3ReadOnlyValidationStages(t *testing.T) {
	base := time.Date(2026, 7, 28, 5, 0, 0, 0, time.UTC)
	events := []Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_pending", "final_edit_pipeline": "assembly_writer_reader_style_validation_evidence_gate_v3", "post_report_humanize": "enabled",
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}},
		}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(20 * time.Second)},
		{EventID: "evt_part", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(30 * time.Second)},
		{EventID: "evt_assembly", EventType: "report.final_assembly.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(35 * time.Second)},
		{EventID: "evt_writer", EventType: "report.final_edit.writer.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(45 * time.Second)},
		{EventID: "evt_reader", EventType: "report.final_edit.reader.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(55 * time.Second)},
		{EventID: "evt_style", EventType: "report.final_edit.style.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(65 * time.Second)},
		{EventID: "evt_style_semantic_start", EventType: "report.final_edit.style_semantic_validation.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(70 * time.Second)},
		{EventID: "evt_style_semantic", EventType: "report.final_edit.style_semantic_validation.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(78 * time.Second)},
		{EventID: "evt_evidence_gate_start", EventType: "report.final_edit.evidence_gate.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(80 * time.Second)},
	}

	progress := ProjectReportProgress(events)
	nodes := assertReportNodeOrder(t, progress.Nodes, []string{"plan", "section-1-1", "part-1", "final-assembly", "final-write", "reader-edit", "style-edit", "style-semantic-validation", "evidence-gate", "final", "artifact"})
	if nodes["style-semantic-validation"].Kind != "style_semantic_validation" || nodes["evidence-gate"].Kind != "evidence_gate" {
		t.Fatalf("v3 validation nodes not projected with stable kinds: %#v", progress.Nodes)
	}
	if nodes["style-semantic-validation"].State != "completed" || nodes["evidence-gate"].State != "running" {
		t.Fatalf("v3 validation node states differ: %#v", progress.Nodes)
	}
	assertNodeTiming(t, nodes["style-semantic-validation"], base.Add(70*time.Second), 8_000)

	disabled := ProjectReportProgress([]Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_pending", "final_edit_pipeline": "assembly_writer_reader_style_validation_evidence_gate_v3", "post_report_humanize": "disabled",
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}},
		}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(20 * time.Second)},
		{EventID: "evt_part", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(30 * time.Second)},
		{EventID: "evt_assembly", EventType: "report.final_assembly.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(35 * time.Second)},
		{EventID: "evt_writer", EventType: "report.final_edit.writer.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(45 * time.Second)},
		{EventID: "evt_reader", EventType: "report.final_edit.reader.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(55 * time.Second)},
		{EventID: "evt_evidence_gate_start", EventType: "report.final_edit.evidence_gate.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(80 * time.Second)},
	})
	assertReportNodeOrder(t, disabled.Nodes, []string{"plan", "section-1-1", "part-1", "final-assembly", "final-write", "reader-edit", "evidence-gate", "final", "artifact"})
	if reportNodeState(disabled.Nodes, "style-edit") != "" || reportNodeState(disabled.Nodes, "style-semantic-validation") != "" {
		t.Fatalf("disabled v3 progress must skip style nodes: %#v", disabled.Nodes)
	}
}

func TestProjectReportProgressMapsV2FinalWriteFailure(t *testing.T) {
	base := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	progress := ProjectReportProgress([]Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_pending", "final_edit_pipeline": "assembly_writer_reader_style_gate_v2", "post_report_humanize": "disabled",
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}},
		}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(20 * time.Second)},
		{EventID: "evt_part", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(30 * time.Second)},
		{EventID: "evt_assembly", EventType: "report.final_assembly.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(35 * time.Second)},
		{EventID: "evt_writer_start", EventType: "report.final_edit.writer.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(40 * time.Second)},
		{EventID: "evt_failed", EventType: "report.final.failed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "failed_stage_kind": "final_write", "safe_error_message": "writer provider unavailable"}), CreatedAt: base.Add(55 * time.Second)},
		{EventID: "evt_terminal", EventType: "report.draft.failed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "failed_stage_kind": "final_write", "safe_error_message": "writer provider unavailable"}), CreatedAt: base.Add(55 * time.Second)},
	})
	if progress.State != "failed" || !progress.Retry.ResumeFailed {
		t.Fatalf("v2 writer draft failure not retryable: %#v", progress)
	}
	nodes := assertReportNodeOrder(t, progress.Nodes, []string{"plan", "section-1-1", "part-1", "final-assembly", "final-write", "reader-edit", "corrective-gate", "final", "artifact"})
	if nodes["final-write"].State != "failed" || nodes["final-write"].Error != "writer provider unavailable" {
		t.Fatalf("writer failure not mapped: %#v", progress.Nodes)
	}
	if nodes["reader-edit"].State != "pending" || nodes["final"].State != "pending" {
		t.Fatalf("downstream/final nodes absorbed writer failure: %#v", progress.Nodes)
	}
	assertNodeTiming(t, nodes["final-write"], base.Add(40*time.Second), 15_000)
}

func TestProjectReportProgressMapsFinalEditFailedStageCompanionToDraftFailure(t *testing.T) {
	base := time.Date(2026, 7, 28, 2, 0, 0, 0, time.UTC)
	progress := ProjectReportProgress([]Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_pending", "final_edit_pipeline": "reader_style_gate_v1", "post_report_humanize": "enabled",
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}},
		}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(20 * time.Second)},
		{EventID: "evt_part", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(30 * time.Second)},
		{EventID: "evt_reader", EventType: "report.final_edit.reader.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(40 * time.Second)},
		{EventID: "evt_style_start", EventType: "report.final_edit.style.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(50 * time.Second)},
		{EventID: "evt_failed", EventType: "report.final.failed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "failed_stage_kind": "style_edit", "safe_error_message": "style provider unavailable"}), CreatedAt: base.Add(70 * time.Second)},
		{EventID: "evt_terminal", EventType: "report.draft.failed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "failed_stage_kind": "style_edit", "safe_error_message": "style provider unavailable"}), CreatedAt: base.Add(70 * time.Second)},
	})
	if progress.State != "failed" || !progress.Retry.ResumeFailed {
		t.Fatalf("draft failure not projected as retryable long-form failure: %#v", progress)
	}
	nodes := map[string]ReportProgressNode{}
	for _, node := range progress.Nodes {
		nodes[node.ID] = node
	}
	if nodes["style-edit"].State != "failed" || nodes["style-edit"].Error != "style provider unavailable" {
		t.Fatalf("style edit failure not mapped: %#v", progress.Nodes)
	}
	assertNodeTiming(t, nodes["style-edit"], base.Add(50*time.Second), 20_000)
	if nodes["final"].State != "pending" {
		t.Fatalf("final node should not absorb style failure: %#v", progress.Nodes)
	}
}

func TestProjectReportProgressDoesNotTreatFinalFailedAsTerminal(t *testing.T) {
	base := time.Date(2026, 7, 28, 2, 30, 0, 0, time.UTC)
	progress := ProjectReportProgress([]Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_pending", "final_edit_pipeline": "reader_style_gate_v1", "post_report_humanize": "disabled",
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}},
		}), CreatedAt: base.Add(10 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(20 * time.Second)},
		{EventID: "evt_part", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(30 * time.Second)},
		{EventID: "evt_reader", EventType: "report.final_edit.reader.submitted", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(40 * time.Second)},
		{EventID: "evt_gate_start", EventType: "report.final_edit.gate.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending"}), CreatedAt: base.Add(50 * time.Second)},
		{EventID: "evt_failed", EventType: "report.final.failed", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "failed_stage_kind": "corrective_gate", "safe_error_message": "gate unavailable"}), CreatedAt: base.Add(70 * time.Second)},
	})
	if progress.State == "failed" || progress.Retry.ResumeFailed || progress.Retry.Restart {
		t.Fatalf("final.failed companion must not close attempt: %#v", progress)
	}
	if got := reportNodeState(progress.Nodes, "corrective-gate"); got != "failed" {
		t.Fatalf("gate companion stage state=%q, nodes=%#v", got, progress.Nodes)
	}
}

func TestProjectReportProgressShowsPartPlanningAndAuthorOnlyWithPlanPayload(t *testing.T) {
	base := time.Date(2026, 7, 27, 1, 0, 0, 0, time.UTC)
	events := []Event{
		{EventID: "evt_pending", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_edit_enabled": true, "part_planning_enabled": true, "plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}}}), CreatedAt: base.Add(time.Second)},
		{EventID: "evt_part_plan", EventType: "report.part_plan.created", Payload: mustReportPayload(t, map[string]any{"kind": "sectional_markdown_report_part_plan", "pending_event_id": "evt_pending", "plan_event_id": "evt_plan", "part_index": 1}), CreatedAt: base.Add(4 * time.Second)},
		{EventID: "evt_section", EventType: "report.section.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1, "section_index": 1}), CreatedAt: base.Add(6 * time.Second)},
		{EventID: "evt_part", EventType: "report.part.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(8 * time.Second)},
		{EventID: "evt_part_edit_start", EventType: "report.part_edit.started", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(9 * time.Second)},
		{EventID: "evt_part_edited", EventType: "report.part.edited", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_index": 1}), CreatedAt: base.Add(12 * time.Second)},
	}

	progress := ProjectReportProgress(events)
	if got := reportNodeState(progress.Nodes, "part-plan-1"); got != "completed" {
		t.Fatalf("part plan state = %q, nodes = %#v", got, progress.Nodes)
	}
	if got := reportNodeState(progress.Nodes, "part-author-1"); got != "completed" {
		t.Fatalf("part author state = %q, nodes = %#v", got, progress.Nodes)
	}
	if got := reportNodeState(progress.Nodes, "part-edit-1"); got != "" {
		t.Fatalf("part edit should be renamed to author under capability, got %q: %#v", got, progress.Nodes)
	}
	nodes := map[string]ReportProgressNode{}
	for _, node := range progress.Nodes {
		nodes[node.ID] = node
	}
	assertNodeTiming(t, nodes["part-plan-1"], base.Add(time.Second), 3000)
	assertNodeTiming(t, nodes["part-author-1"], base.Add(9*time.Second), 3000)
	wantOrder := []string{"plan", "part-plan-1", "section-1-1", "part-1", "part-author-1", "final", "artifact"}
	if len(progress.Nodes) != len(wantOrder) {
		t.Fatalf("unexpected node count: %#v", progress.Nodes)
	}
	for index, want := range wantOrder {
		if progress.Nodes[index].ID != want {
			t.Fatalf("node %d = %q, want %q: %#v", index, progress.Nodes[index].ID, want, progress.Nodes)
		}
	}

	legacy := ProjectReportProgress([]Event{
		events[0],
		{EventID: "evt_legacy_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{"pending_event_id": "evt_pending", "part_edit_enabled": true, "plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}}}), CreatedAt: base.Add(time.Second)},
		events[3], events[4], events[5], events[6],
	})
	if got := reportNodeState(legacy.Nodes, "part-edit-1"); got != "completed" {
		t.Fatalf("legacy Part edit state = %q, nodes = %#v", got, legacy.Nodes)
	}
	if reportNodeState(legacy.Nodes, "part-plan-1") != "" || reportNodeState(legacy.Nodes, "part-author-1") != "" {
		t.Fatalf("legacy progress synthesized W4 nodes: %#v", legacy.Nodes)
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

func assertReportNodeOrder(t *testing.T, nodes []ReportProgressNode, wantOrder []string) map[string]ReportProgressNode {
	t.Helper()
	if len(nodes) != len(wantOrder) {
		t.Fatalf("unexpected node count: %#v", nodes)
	}
	out := map[string]ReportProgressNode{}
	for index, want := range wantOrder {
		if nodes[index].ID != want {
			t.Fatalf("node %d = %q, want %q: %#v", index, nodes[index].ID, want, nodes)
		}
		out[want] = nodes[index]
	}
	return out
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

func TestProjectReportProgressProjectsPlannedReportLifecycle(t *testing.T) {
	running := ProjectReportProgress([]Event{
		reportEvent("evt_planned", "report.draft.pending", map[string]any{"report_mode": "planned"}),
		reportEvent("evt_plan", "report.plan.created", map[string]any{"pending_event_id": "evt_planned"}),
	})
	runningNodes := assertReportNodeOrder(t, running.Nodes, []string{"start", "final", "artifact"})
	if running.State != "running" || runningNodes["start"].State != "completed" || runningNodes["final"].State != "running" || runningNodes["artifact"].State != "pending" {
		t.Fatalf("planned report should expose compact running lifecycle: %#v", running)
	}

	completed := ProjectReportProgress([]Event{
		reportEvent("evt_planned", "report.draft.pending", map[string]any{"report_mode": "planned"}),
		reportEvent("evt_plan", "report.plan.created", map[string]any{"pending_event_id": "evt_planned"}),
		reportEvent("evt_artifact", "report.artifact.created", map[string]any{"pending_event_id": "evt_planned"}),
	})
	completedNodes := assertReportNodeOrder(t, completed.Nodes, []string{"start", "final", "artifact"})
	if completed.State != "completed" {
		t.Fatalf("planned report state=%q, want completed: %#v", completed.State, completed)
	}
	for _, id := range []string{"start", "final", "artifact"} {
		if completedNodes[id].State != "completed" {
			t.Fatalf("planned report %s state=%q, want completed: %#v", id, completedNodes[id].State, completed.Nodes)
		}
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

func TestProjectReportProgressResumedSectionReplacesAncestorFailure(t *testing.T) {
	base := time.Date(2026, 8, 10, 4, 0, 0, 0, time.UTC)
	resumedAt := base.Add(2*time.Hour + 40*time.Minute)
	events := []Event{
		{EventID: "evt_root", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{"report_mode": "long_form"}), CreatedAt: base},
		{EventID: "evt_plan", EventType: "report.plan.created", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_root",
			"plan":             map[string]any{"parts": []any{map[string]any{"sections": []any{"one"}}}},
		}), CreatedAt: base.Add(time.Minute)},
		{EventID: "evt_root_section_start", EventType: "report.section.started", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_root", "part_index": 1, "section_index": 1,
		}), CreatedAt: base.Add(2 * time.Minute)},
		{EventID: "evt_root_section_failed", EventType: "report.section.failed", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_root", "part_index": 1, "section_index": 1,
		}), CreatedAt: base.Add(3 * time.Minute)},
		{EventID: "evt_root_failed", EventType: "report.draft.failed", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_root", "failed_stage_kind": "section", "failed_stage_id": "section-1-1",
		}), CreatedAt: base.Add(3 * time.Minute)},
		{EventID: "evt_retry", EventType: "report.draft.pending", Payload: mustReportPayload(t, map[string]any{
			"report_mode": "long_form", "origin_pending_event_id": "evt_root", "retry_of_pending_event_id": "evt_root",
			"retry_strategy": "resume_failed", "attempt_number": 2,
		}), CreatedAt: resumedAt},
		{EventID: "evt_retry_section_start", EventType: "report.section.started", Payload: mustReportPayload(t, map[string]any{
			"pending_event_id": "evt_retry", "part_index": 1, "section_index": 1,
		}), CreatedAt: resumedAt.Add(time.Second)},
	}

	progress := ProjectReportProgress(events)
	nodes := map[string]ReportProgressNode{}
	for _, node := range progress.Nodes {
		nodes[node.ID] = node
	}
	section := nodes["section-1-1"]
	if progress.State != "running" || section.State != "running" || section.AttemptID != "evt_retry" {
		t.Fatalf("resumed Section must replace ancestor failure: %#v", progress)
	}
	if section.StartedAt == nil || !section.StartedAt.Equal(resumedAt.Add(time.Second)) || section.DurationMS != nil {
		t.Fatalf("resumed Section must use the retry start time: %#v", section)
	}
	part := nodes["part-1"]
	if part.State != "pending" || part.StartedAt != nil || part.DurationMS != nil {
		t.Fatalf("part assembly must remain pending while the resumed Section runs: %#v", part)
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
