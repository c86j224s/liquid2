package reporting

import (
	"context"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestLoadFinalEditStageProgressStates(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	contract := FinalEditStageStartContract{FinalBinding: binding, Stage: FinalEditStageReader}

	if progress, ok, err := LoadFinalEditStageProgress(ctx, svc, contract); err != nil || ok {
		t.Fatalf("not-started progress=%#v ok=%t err=%v, want absent nil", progress, ok, err)
	}
	sourceID := FinalEditReaderSourceArtifactID(binding.PlanEventID, []string{"art_part"})
	readerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageReader, sourceID, "art_reader_progress")
	started, created, err := StartFinalEditStage(ctx, svc, "evt_reader_progress_start", readerBinding)
	if err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	progress, ok, err := LoadFinalEditStageProgress(ctx, svc, contract)
	if err != nil || !ok {
		t.Fatalf("open progress ok=%t err=%v", ok, err)
	}
	if progress.Binding != readerBinding || progress.StartEvent.EventID != started.EventID || progress.SourceArtifact.ArtifactID != sourceID || progress.Submission != nil {
		t.Fatalf("open progress differs: %#v", progress)
	}

	markdown := AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nPreserved body.\n"})
	submitted, err := SubmitFinalEditStage(ctx, svc, readerBinding, "evt_reader_progress_submit", markdown, 0)
	if err != nil {
		t.Fatal(err)
	}
	progress, ok, err = LoadFinalEditStageProgress(ctx, svc, contract)
	if err != nil || !ok || progress.Submission == nil {
		t.Fatalf("submitted progress ok=%t progress=%#v err=%v", ok, progress, err)
	}
	if progress.Submission.Artifact.ArtifactID != submitted.Artifact.ArtifactID || !progress.Submission.Replay {
		t.Fatalf("submitted result was not validated/replayed: %#v", progress.Submission)
	}
	if current, ok, err := LoadCurrentFinalEditStageStart(ctx, svc, contract); err != nil || ok {
		t.Fatalf("LoadCurrentFinalEditStageStart semantics changed: current=%#v ok=%t err=%v", current, ok, err)
	}
}

func TestLoadFinalEditStageProgressRejectsMalformedSubmissionPayload(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	sourceID := FinalEditReaderSourceArtifactID(binding.PlanEventID, []string{"art_part"})
	readerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageReader, sourceID, "art_reader_malformed_progress")
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_reader_malformed_progress_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	source, err := svc.GetRawArtifact(ctx, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	request := buildFinalEditSubmittedAppendRequest("evt_reader_malformed_progress_submit", readerBinding, source, source, 0, false, nil, FinalEditSemanticAttestation{})
	payload := eventPayload(app.LedgerEvent{Payload: request.Payload})
	payload["operation_count"] = "invalid"
	request.Payload = finalEditStageStoreJSON(payload)
	if _, err := svc.AppendEvent(ctx, request); err != nil {
		t.Fatal(err)
	}

	_, _, err = LoadFinalEditStageProgress(ctx, svc, FinalEditStageStartContract{FinalBinding: binding, Stage: FinalEditStageReader})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("malformed submission error=%v, want conflict", err)
	}
}

func TestLoadFinalEditStageProgressRejectsDuplicateStarts(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	sourceID := FinalEditReaderSourceArtifactID(binding.PlanEventID, []string{"art_part"})
	readerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageReader, sourceID, "art_reader_duplicate_progress")
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_reader_duplicate_progress_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	duplicate := BuildFinalEditStageStartedAppendRequest("evt_reader_duplicate_progress_start_2", readerBinding)
	if _, err := svc.AppendEvent(ctx, duplicate); err != nil {
		t.Fatal(err)
	}

	_, _, err := LoadFinalEditStageProgress(ctx, svc, FinalEditStageStartContract{FinalBinding: binding, Stage: FinalEditStageReader})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("duplicate start error=%v, want conflict", err)
	}
}

func TestLoadFinalEditStageProgressRejectsUnapprovedGateEvidence(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "progress_gate")
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_gate_progress_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_gate_progress_evidence", MissionID: binding.MissionID, EventType: "evidence.proposed",
		Producer: app.Producer{Type: "user", ID: "test"}, Payload: finalEditStageStoreJSON(map[string]any{"evidence_id": "evd_gate_progress"}),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateEvidenceRecord(ctx, app.CreateEvidenceRecordRequest{
		EvidenceID: "evd_gate_progress", MissionID: binding.MissionID, State: "proposed", Summary: "Unapproved gate evidence.",
		EvidenceType: "user_assertion", Producer: app.Producer{Type: "user", ID: "test"}, CreatedEventID: "evt_gate_progress_evidence",
	}); err != nil {
		t.Fatal(err)
	}
	finding := StoredFinalEditGateFinding{
		StatementSHA256: contentSHA256([]byte("Unsupported gate statement.")),
		Classification:  FinalEditGateClassUnverifiedExternalFact,
		RepairAction:    FinalEditRepairAttachApprovedEvidence,
		EvidenceIDs:     []string{"evd_gate_progress"},
	}
	source, err := svc.GetRawArtifact(ctx, gateBinding.SourceArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	submitted := buildFinalEditSubmittedAppendRequest("evt_gate_progress_submit", gateBinding, source, source, 0, false, []StoredFinalEditGateFinding{finding}, FinalEditSemanticAttestation{})
	if _, err := svc.AppendEvent(ctx, submitted); err != nil {
		t.Fatal(err)
	}

	_, _, err = LoadFinalEditStageProgress(ctx, svc, FinalEditStageStartContract{FinalBinding: binding, Stage: FinalEditStageGate})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("unapproved evidence error=%v, want conflict", err)
	}
}

func TestLoadFinalEditStageProgressRejectsOpenGateAfterCanonical(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "progress_open_gate_terminal")
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_gate_progress_open_terminal_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: binding.ArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: binding.Producer, Content: []byte("# Report\n\nCanonical final report.\n"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_gate_progress_open_terminal_final", MissionID: binding.MissionID, EventType: "report.artifact.created",
		Producer: binding.Producer, CorrelationID: binding.IdempotencyKey, Payload: finalEditStageStoreJSON(map[string]any{
			"pending_event_id": binding.PendingEventID,
			"plan_event_id":    binding.PlanEventID,
			"artifact_id":      binding.ArtifactID,
		}),
	}); err != nil {
		t.Fatal(err)
	}

	_, _, err := LoadFinalEditStageProgress(ctx, svc, FinalEditStageStartContract{FinalBinding: binding, Stage: FinalEditStageGate})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("open gate terminal error=%v, want conflict", err)
	}
}
