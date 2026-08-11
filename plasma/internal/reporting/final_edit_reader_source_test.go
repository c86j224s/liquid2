package reporting_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestFinalEditReaderSourceStartMaterializesDeterministicArtifactAndRecoversOpenStart(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithFinalEditPipeline(t, ctx, reporting.FinalEditPipelineReaderStyleGateV1, reporting.FinalEditHumanizeDisabled)
	defer closeStore()
	binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeDisabled)
	sourceID := reporting.FinalEditReaderSourceArtifactID(binding.PlanEventID, []string{"art_part"})
	readerBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageReader, sourceID, "art_reader_recovery", "")

	started, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_reader_recovery_start", readerBinding)
	if err != nil || !created {
		t.Fatalf("reader start created=%t event=%#v err=%v", created, started, err)
	}
	source, err := svc.GetRawArtifact(ctx, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	wantMarkdown := reporting.AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nPreserved body.\n"})
	if source.Filename != binding.Filename ||
		source.Producer != (app.Producer{Type: "system", ID: "reporting_reader_assembly"}) ||
		string(source.Content) != wantMarkdown {
		t.Fatalf("reader source contract mismatch: %#v", source)
	}

	loaded, ok, err := reporting.LoadCurrentFinalEditStageStart(ctx, svc, reporting.FinalEditStageStartContract{
		FinalBinding: binding,
		Stage:        reporting.FinalEditStageReader,
	})
	if err != nil || !ok {
		t.Fatalf("open reader start not recovered: loaded=%#v ok=%t err=%v", loaded, ok, err)
	}
	if loaded.Binding != readerBinding || loaded.SourceArtifact.ArtifactID != sourceID || loaded.Event.EventID != started.EventID {
		t.Fatalf("recovered reader start differs: %#v", loaded)
	}

	if _, err := reporting.SubmitFinalEditStage(ctx, svc, readerBinding, "evt_reader_recovery_submit", wantMarkdown, 0); err != nil {
		t.Fatal(err)
	}
	if loaded, ok, err := reporting.LoadCurrentFinalEditStageStart(ctx, svc, reporting.FinalEditStageStartContract{
		FinalBinding: binding,
		Stage:        reporting.FinalEditStageReader,
	}); err != nil || ok {
		t.Fatalf("completed reader start still recovered: loaded=%#v ok=%t err=%v", loaded, ok, err)
	}
}

func TestFinalEditReaderSourceUsesOrderedMultiplePartsAndReplays(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeDisabled)
	binding.MissionID = "mis_reader_multi"
	binding.PendingEventID = "evt_reader_multi_pending"
	binding.PlanEventID = "evt_reader_multi_plan"
	binding.ArtifactID = "art_reader_multi_final"
	binding.PartArtifactIDs = []string{"art_reader_multi_part_1", "art_reader_multi_part_2"}
	binding.SectionArtifactIDs = []string{"art_reader_multi_section_1", "art_reader_multi_section_2"}
	binding.SectionWordCount = 4
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: binding.MissionID, Title: "reader multi"}); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range []app.CreateRawArtifactRequest{
		{ArtifactID: binding.PartArtifactIDs[0], MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: "part-1.md", Producer: binding.Producer, Content: []byte("# Part 1\n\nFirst body.\n")},
		{ArtifactID: binding.PartArtifactIDs[1], MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: "part-2.md", Producer: binding.Producer, Content: []byte("# Part 2\n\nSecond body.\n")},
		{ArtifactID: binding.SectionArtifactIDs[0], MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: "section-1.md", Producer: binding.Producer, Content: []byte("# Section 1\n\nFirst.\n")},
		{ArtifactID: binding.SectionArtifactIDs[1], MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: "section-2.md", Producer: binding.Producer, Content: []byte("# Section 2\n\nSecond.\n")},
	} {
		if _, err := svc.CreateRawArtifact(ctx, artifact); err != nil {
			t.Fatal(err)
		}
	}
	for _, event := range []app.AppendEventRequest{
		{EventID: binding.PendingEventID, MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: testJSON(map[string]any{"report_mode": reporting.ModeLongForm})},
		{EventID: binding.PlanEventID, MissionID: binding.MissionID, EventType: "report.plan.created", Producer: binding.Producer, Payload: testJSON(map[string]any{
			"pending_event_id": binding.PendingEventID, "report_mode": reporting.ModeLongForm, "artifact_id": binding.ArtifactID,
			"final_edit_pipeline": reporting.FinalEditPipelineReaderStyleGateV1, "post_report_humanize": reporting.FinalEditHumanizeDisabled,
			"plan": map[string]any{"parts": []any{
				map[string]any{"sections": []any{"section 1"}},
				map[string]any{"sections": []any{"section 2"}},
			}},
		})},
		{EventID: "evt_reader_multi_part_2", MissionID: binding.MissionID, EventType: "report.part.created", Producer: binding.Producer, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.PartArtifactIDs[1], "part_index": 2})},
		{EventID: "evt_reader_multi_part_1", MissionID: binding.MissionID, EventType: "report.part.created", Producer: binding.Producer, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.PartArtifactIDs[0], "part_index": 1})},
		{EventID: "evt_reader_multi_section_2", MissionID: binding.MissionID, EventType: "report.section.created", Producer: binding.Producer, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.SectionArtifactIDs[1], "part_index": 2, "section_index": 1})},
		{EventID: "evt_reader_multi_section_1", MissionID: binding.MissionID, EventType: "report.section.created", Producer: binding.Producer, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.SectionArtifactIDs[0], "part_index": 1, "section_index": 1})},
	} {
		if _, err := svc.AppendEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	sourceID := reporting.FinalEditReaderSourceArtifactID(binding.PlanEventID, binding.PartArtifactIDs)
	readerBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageReader, sourceID, "art_reader_multi_reader", "")
	started, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_reader_multi_start", readerBinding)
	if err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	replayed, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_reader_multi_start_again", readerBinding)
	if err != nil || created || replayed.EventID != started.EventID {
		t.Fatalf("reader replay created=%t event=%#v err=%v", created, replayed, err)
	}
	source, err := svc.GetRawArtifact(ctx, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	want := reporting.AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nFirst body.\n", "# Part 2\n\nSecond body.\n"})
	if source.Filename != binding.Filename || string(source.Content) != want {
		t.Fatalf("reader source differs:\n%s", source.Content)
	}
}

func TestFinalEditReaderSourceAcceptsResumeFailedAncestorLineageAndPartEdit(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeDisabled)
	binding.MissionID = "mis_reader_resume"
	binding.PendingEventID = "evt_reader_resume_retry"
	binding.PlanEventID = "evt_reader_resume_plan"
	binding.ArtifactID = "art_reader_resume_final"
	binding.PartArtifactIDs = []string{"art_reader_resume_part_edit"}
	binding.SectionArtifactIDs = []string{"art_reader_resume_section"}
	rootPendingID := "evt_reader_resume_root"
	sourcePartID := "art_reader_resume_part"
	sourcePartEventID := "evt_reader_resume_part"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: binding.MissionID, Title: "reader resume"}); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range []app.CreateRawArtifactRequest{
		{ArtifactID: sourcePartID, MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: "part.md", Producer: binding.Producer, Content: []byte("# Part 1\n\nAncestor body.\n")},
		{ArtifactID: binding.SectionArtifactIDs[0], MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: "section.md", Producer: binding.Producer, Content: []byte("# Section 1\n\nAncestor section.\n")},
	} {
		if _, err := svc.CreateRawArtifact(ctx, artifact); err != nil {
			t.Fatal(err)
		}
	}
	for _, event := range []app.AppendEventRequest{
		{EventID: rootPendingID, MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: testJSON(map[string]any{"origin_pending_event_id": rootPendingID, "retry_strategy": "initial", "report_mode": reporting.ModeLongForm})},
		{EventID: binding.PlanEventID, MissionID: binding.MissionID, EventType: "report.plan.created", Producer: binding.Producer, Payload: testJSON(map[string]any{
			"pending_event_id": rootPendingID, "report_mode": reporting.ModeLongForm, "artifact_id": binding.ArtifactID,
			"final_edit_pipeline": reporting.FinalEditPipelineReaderStyleGateV1, "post_report_humanize": reporting.FinalEditHumanizeDisabled,
			"part_edit_enabled": true,
			"plan":              map[string]any{"parts": []any{map[string]any{"sections": []any{"section 1"}}}},
		})},
		{EventID: sourcePartEventID, MissionID: binding.MissionID, EventType: "report.part.created", Producer: binding.Producer, Payload: testJSON(map[string]any{"pending_event_id": rootPendingID, "plan_event_id": binding.PlanEventID, "artifact_id": sourcePartID, "part_index": 1})},
		{EventID: "evt_reader_resume_section", MissionID: binding.MissionID, EventType: "report.section.created", Producer: binding.Producer, Payload: testJSON(map[string]any{"pending_event_id": rootPendingID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.SectionArtifactIDs[0], "part_index": 1, "section_index": 1})},
	} {
		if _, err := svc.AppendEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	partEdit := reporting.PartEditBinding{
		MissionID: binding.MissionID, PendingEventID: rootPendingID, PlanEventID: binding.PlanEventID,
		SourcePartEventID: sourcePartEventID, SourceArtifactID: sourcePartID, EditedArtifactID: binding.PartArtifactIDs[0],
		Filename: "part-1-edited.md", ToolSessionID: "ses_part_edit", ProviderSessionID: "provider-part-edit", PreviousProviderSessionID: "provider-part-edit",
		IdempotencyKey: reporting.FinalEditStageIdempotencyKey("part_edit", rootPendingID, binding.PlanEventID), PartIndex: 1,
		AgentExecutor: binding.AgentExecutor, AgentModel: binding.AgentModel, AgentReasoningEffort: binding.AgentReasoningEffort,
		AgentSelectionSource: binding.AgentSelectionSource, MCPMode: binding.MCPMode,
		ReportSessionPolicy: binding.ReportSessionPolicy, ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
		GenerationGuidanceProfile: binding.GenerationGuidanceProfile, GenerationGuidanceSHA256: binding.GenerationGuidanceSHA256,
		SessionChainKind: binding.SessionChainKind, ReportPlanSessionID: binding.ReportPlanSessionID, ForkSourceAgentSessionID: binding.ReportPlanSessionID,
	}
	partEdit.IdempotencyKey = "report-part-edit:" + rootPendingID + ":" + binding.PlanEventID + ":1"
	if _, _, err := reporting.StartPartEdit(ctx, svc, "evt_reader_resume_part_edit_start", partEdit); err != nil {
		t.Fatal(err)
	}
	edited, err := reporting.FinalizePartEdit(ctx, svc, partEdit, "evt_reader_resume_part_edit_submit", "# Part 1\n\nEdited ancestor body.\n", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []app.AppendEventRequest{
		{EventID: "evt_reader_resume_root_failed", MissionID: binding.MissionID, EventType: "report.draft.failed", Producer: binding.Producer, Payload: testJSON(map[string]any{
			"pending_event_id": rootPendingID,
			"kind":             "report_draft_failed",
		})},
		{EventID: binding.PendingEventID, MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, CausationEventID: rootPendingID, Payload: testJSON(map[string]any{
			"origin_pending_event_id":   rootPendingID,
			"retry_of_pending_event_id": rootPendingID,
			"retry_strategy":            "resume_failed",
			"report_mode":               reporting.ModeLongForm,
		})},
	} {
		if _, err := svc.AppendEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	sourceID := reporting.FinalEditReaderSourceArtifactID(binding.PlanEventID, []string{edited.Artifact.ArtifactID})
	readerBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageReader, sourceID, "art_reader_resume_reader", "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_reader_resume_start", readerBinding); err != nil || !created {
		t.Fatalf("reader source did not accept ancestor lineage created=%t err=%v", created, err)
	}
	source, err := svc.GetRawArtifact(ctx, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	want := reporting.AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nEdited ancestor body.\n"})
	if string(source.Content) != want {
		t.Fatalf("reader source content=%q, want %q", source.Content, want)
	}
}

func TestFinalEditReaderSourceRejectsForeignBoundPlanPending(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithFinalEditPipeline(t, ctx, reporting.FinalEditPipelineReaderStyleGateV1, reporting.FinalEditHumanizeDisabled)
	defer closeStore()
	binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeDisabled)
	events, err := svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.EventID != binding.PlanEventID {
			continue
		}
		payload := map[string]any{}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		payload["pending_event_id"] = "evt_foreign_pending"
		if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
			ArtifactID: binding.ArtifactID, MissionID: binding.MissionID,
			MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
			Producer: binding.Producer, Content: []byte("# Report\n\nSynthetic foreign-bound final.\n"),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
			EventID: event.EventID + "_foreign", MissionID: binding.MissionID, EventType: event.EventType,
			Producer: event.Producer, CausationEventID: event.CausationEventID, CorrelationID: event.CorrelationID, Payload: testJSON(payload),
		}); err != nil {
			t.Fatal(err)
		}
		break
	}
	sourceID := reporting.FinalEditReaderSourceArtifactID(binding.PlanEventID+"_foreign", []string{"art_part"})
	readerBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageReader, sourceID, "art_reader_foreign", "")
	readerBinding.PlanEventID = binding.PlanEventID + "_foreign"
	readerBinding.IdempotencyKey = reporting.FinalEditStageIdempotencyKey(reporting.FinalEditStageReader, readerBinding.PendingEventID, readerBinding.PlanEventID)
	_, _, err = reporting.StartFinalEditStage(ctx, svc, "evt_reader_foreign_start", readerBinding)
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("err=%v, want foreign plan conflict", err)
	}
}
