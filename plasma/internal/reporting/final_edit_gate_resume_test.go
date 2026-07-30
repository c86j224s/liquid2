package reporting

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestResumeFinalEditGateUsesStoredSubmissionAfterRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := newFinalEditStageStoreFixtureAt(t, ctx, path, FinalEditHumanizeDisabled)
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "resume")
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_gate_resume_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	storedFindings := []StoredFinalEditGateFinding{{
		StatementSHA256: contentSHA256([]byte("Stored unsupported claim.")),
		Classification:  FinalEditGateClassUnverifiedExternalFact,
		RepairAction:    FinalEditRepairRemove,
	}}
	gate, err := submitFinalEditStage(ctx, svc, gateBinding, "evt_gate_resume_submit", string(reader.Artifact.Content), 0, storedFindings)
	if err != nil {
		t.Fatal(err)
	}
	closeStore()

	store, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	resumed, err := ResumeFinalEditGate(ctx, app.NewService(store), FinalEditGateResumeRequest{
		StageBinding:     gateBinding,
		FinalBinding:     binding,
		CanonicalEventID: "evt_gate_resume_final",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Artifact.ArtifactID != gate.Artifact.ArtifactID || resumed.Event.EventID != "evt_gate_resume_final" {
		t.Fatalf("resume result differs: %#v", resumed)
	}
	payload := eventPayload(resumed.Event)
	if payload["artifact_id"] != gate.Artifact.ArtifactID ||
		payload["planned_final_artifact_id"] != binding.ArtifactID ||
		payload["final_edit_gate_event_id"] != gate.Event.EventID ||
		payload["final_edit_gate_changed"] != false ||
		payload["artifact_sha256"] != gate.Artifact.SHA256 {
		t.Fatalf("canonical resume payload differs: %#v", payload)
	}
	if _, err := app.NewService(store).GetRawArtifact(ctx, binding.ArtifactID); err == nil {
		t.Fatal("no-op gate resume created duplicate planned final artifact")
	}
}

func TestResumeFinalEditGatePreservesV2PipelineAfterRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := newFinalEditStageStoreV2FixtureAt(t, ctx, path, FinalEditHumanizeDisabled)
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitV2ReaderForGateResumeTest(t, ctx, svc, binding, "resume_v2")
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_gate_resume_v2_start", gateBinding); err != nil || !created {
		t.Fatalf("v2 gate start created=%t err=%v", created, err)
	}
	gate, err := submitFinalEditStage(ctx, svc, gateBinding, "evt_gate_resume_v2_submit", string(reader.Artifact.Content), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	closeStore()

	store, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	resumed, err := ResumeFinalEditGate(ctx, app.NewService(store), FinalEditGateResumeRequest{
		StageBinding:     gateBinding,
		FinalBinding:     binding,
		CanonicalEventID: "evt_gate_resume_v2_final",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Artifact.ArtifactID != gate.Artifact.ArtifactID || resumed.Event.EventID != "evt_gate_resume_v2_final" {
		t.Fatalf("v2 resume result differs: %#v", resumed)
	}
	payload := eventPayload(resumed.Event)
	if payload["final_edit_pipeline"] != FinalEditPipelineAssemblyWriterReaderStyleGateV2 ||
		payload["artifact_id"] != gate.Artifact.ArtifactID ||
		payload["planned_final_artifact_id"] != binding.ArtifactID ||
		payload["final_edit_gate_event_id"] != gate.Event.EventID ||
		payload["final_edit_gate_changed"] != false ||
		payload["artifact_sha256"] != gate.Artifact.SHA256 {
		t.Fatalf("v2 canonical resume payload differs: %#v", payload)
	}
	if _, err := app.NewService(store).GetRawArtifact(ctx, binding.ArtifactID); err == nil {
		t.Fatal("v2 no-op gate resume created duplicate planned final artifact")
	}
}

func newFinalEditStageStoreV2FixtureAt(t *testing.T, ctx context.Context, path string, humanize string) (*app.Service, func()) {
	t.Helper()
	store, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	binding := finalEditStageStoreFinalBinding(humanize)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: binding.MissionID, Title: "final edit"}); err != nil {
		t.Fatal(err)
	}
	part, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_part", MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: "part.md",
		Producer: app.Producer{Type: "agent_session", ID: "provider-plan"}, Content: []byte("# Part 1\n\nPreserved body.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: binding.SectionArtifactIDs[0], MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: "section.md",
		Producer: app.Producer{Type: "agent_session", ID: "provider-plan"}, Content: []byte("# Section 1\n\nPreserved body.\n"),
	}); err != nil {
		t.Fatal(err)
	}
	events := []app.AppendEventRequest{
		{EventID: binding.PendingEventID, MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: finalEditStageStoreJSON(map[string]any{"report_mode": ModeLongForm})},
		{EventID: binding.PlanEventID, MissionID: binding.MissionID, EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "provider-plan"}, Payload: finalEditStageStoreJSON(map[string]any{
			"pending_event_id": binding.PendingEventID, "report_mode": ModeLongForm, "artifact_id": binding.ArtifactID,
			"final_edit_pipeline": FinalEditPipelineAssemblyWriterReaderStyleGateV2, "post_report_humanize": humanize,
			"plan": map[string]any{"parts": []any{map[string]any{"sections": []any{"section 1"}}}},
		})},
		{EventID: "evt_part", MissionID: binding.MissionID, EventType: "report.part.created", Producer: app.Producer{Type: "agent_session", ID: "provider-plan"}, Payload: finalEditStageStoreJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": part.ArtifactID, "part_index": 1})},
		{EventID: "evt_section", MissionID: binding.MissionID, EventType: "report.section.created", Producer: app.Producer{Type: "agent_session", ID: "provider-plan"}, Payload: finalEditStageStoreJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": "art_section", "part_index": 1, "section_index": 1})},
	}
	for _, event := range events {
		if _, err := svc.AppendEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	return svc, func() { _ = store.Close() }
}

func startAndSubmitV2ReaderForGateResumeTest(t *testing.T, ctx context.Context, svc *app.Service, binding LongFormFinalizeBinding, suffix string) finalEditStageStoreReaderResult {
	t.Helper()
	assemblyID := FinalEditAssemblyArtifactID(binding.PlanEventID, binding.PartArtifactIDs)
	writerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageWriter, assemblyID, "art_writer_"+suffix)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_writer_"+suffix+"_start", writerBinding); err != nil || !created {
		t.Fatalf("writer start created=%t err=%v", created, err)
	}
	writerMarkdown := strings.Replace(AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nPreserved body.\n"}), "Preserved body.", "Writer body.", 1)
	writer, err := SubmitFinalEditStage(ctx, svc, writerBinding, "evt_writer_"+suffix+"_submit", writerMarkdown, 1)
	if err != nil {
		t.Fatal(err)
	}
	readerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageReader, writer.Artifact.ArtifactID, "art_reader_"+suffix)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_reader_"+suffix+"_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	reader, err := SubmitFinalEditStage(ctx, svc, readerBinding, "evt_reader_"+suffix+"_submit", string(writer.Artifact.Content), 0)
	if err != nil {
		t.Fatal(err)
	}
	return finalEditStageStoreReaderResult{Binding: readerBinding, Artifact: reader.Artifact}
}
