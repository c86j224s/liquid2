package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestFinalEditStageReplayRejectsChangedInputForSameIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "replay")

	_, err := SubmitFinalEditStage(ctx, svc, reader.Binding, "evt_reader_replay_submit_again", string(reader.Artifact.Content), 1)
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("reader replay operation mismatch error=%v, want conflict", err)
	}
	_, err = SubmitFinalEditStage(ctx, svc, reader.Binding, "evt_reader_replay_submit_changed", string(reader.Artifact.Content)+"\nChanged.\n", 0)
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("reader replay markdown mismatch error=%v, want conflict", err)
	}
}

func TestFinalEditStageDisabledReplayUsesExistingSubmissionBeforeDuplicateSourceRead(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "disabled_read_surface")

	guarded := &finalEditStageDuplicateReadGuardStore{Service: svc, failAfterEventLists: 1}
	result, err := SubmitFinalEditStage(ctx, guarded, reader.Binding, "evt_reader_disabled_read_surface_replay", string(reader.Artifact.Content), 0)
	if err != nil {
		t.Fatalf("disabled reader replay performed duplicate source read: %v", err)
	}
	if result.Artifact.ArtifactID != reader.Artifact.ArtifactID {
		t.Fatalf("reader replay artifact=%q, want %q", result.Artifact.ArtifactID, reader.Artifact.ArtifactID)
	}
}

func TestFinalEditGateReplayRejectsChangedFindingsForStoredSubmission(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "gate_replay")
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_gate_replay_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	manuscript := "# Report\n\nCorrected final manuscript.\n"
	finding := FinalEditGateFinding{
		Statement:      "Unsupported claim.",
		Classification: FinalEditGateClassUnverifiedExternalFact,
		RepairAction:   FinalEditRepairRemove,
	}
	if _, err := SubmitFinalEditGate(ctx, svc, FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       binding,
		StageEventID:       "evt_gate_replay_submit",
		CanonicalEventID:   "evt_gate_replay_final",
		ManuscriptMarkdown: manuscript,
		OperationCount:     1,
		Findings:           []FinalEditGateFinding{finding},
	}); err != nil {
		t.Fatal(err)
	}

	changedFinding := finding
	changedFinding.RepairAction = FinalEditRepairRetainWithFootnote
	_, err := SubmitFinalEditGate(ctx, svc, FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       binding,
		StageEventID:       "evt_gate_replay_submit_again",
		CanonicalEventID:   "evt_gate_replay_final_again",
		ManuscriptMarkdown: manuscript,
		OperationCount:     1,
		Findings:           []FinalEditGateFinding{changedFinding},
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("gate finding replay mismatch error=%v, want conflict", err)
	}
}

func TestSubmitFinalEditGateWritesNothingWhenFinalLineagePrevalidationFails(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "prevalidation")
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_gate_prevalidation_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	beforeEvents, err := svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	beforeArtifacts, err := svc.ListRawArtifacts(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	badFinalBinding := binding
	badFinalBinding.SectionArtifactIDs = []string{"art_other_section"}
	_, err = SubmitFinalEditGate(ctx, svc, FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       badFinalBinding,
		StageEventID:       "evt_gate_prevalidation_submit",
		CanonicalEventID:   "evt_gate_prevalidation_final",
		ManuscriptMarkdown: "# Report\n\nChanged content.\n",
		OperationCount:     1,
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("prevalidation error=%v, want conflict", err)
	}
	afterEvents, err := svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	afterArtifacts, err := svc.ListRawArtifacts(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterEvents) != len(beforeEvents) || len(afterArtifacts) != len(beforeArtifacts) {
		t.Fatalf("prevalidation wrote data: events %d->%d artifacts %d->%d", len(beforeEvents), len(afterEvents), len(beforeArtifacts), len(afterArtifacts))
	}
}

func TestFinalEditCanonicalReplayRejectsGateFindingMismatch(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "canonical_mismatch")
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_gate_canonical_mismatch_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	manuscript := "# Report\n\nCorrected final manuscript.\n"
	finalArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: binding.ArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: binding.Producer, Content: []byte(manuscript),
	})
	if err != nil {
		t.Fatal(err)
	}
	gateFindings := []StoredFinalEditGateFinding{{
		StatementSHA256: contentSHA256([]byte("Unsupported claim.")),
		Classification:  FinalEditGateClassUnverifiedExternalFact,
		RepairAction:    FinalEditRepairRemove,
	}}
	gateSubmitted, err := svc.AppendEvent(ctx, buildFinalEditSubmittedAppendRequest("evt_gate_canonical_mismatch_submit", gateBinding, reader.Artifact, finalArtifact, 1, true, gateFindings, FinalEditSemanticAttestation{}))
	if err != nil {
		t.Fatal(err)
	}
	wrongFindings := []StoredFinalEditGateFinding{{
		StatementSHA256: contentSHA256([]byte("Unsupported claim.")),
		Classification:  FinalEditGateClassUnverifiedExternalFact,
		RepairAction:    FinalEditRepairRetainWithFootnote,
	}}
	canonicalReq := longFormCanonicalRequestForFinalEdit("evt_final_canonical_mismatch", binding, finalArtifact, len(strings.Fields(manuscript)), LongFormFinalizeRequest{
		FinalEditPipeline:         FinalEditPipelineReaderStyleGateV1,
		GateFindings:              wrongFindings,
		FinalEditActualArtifactID: finalArtifact.ArtifactID,
		FinalEditGateEventID:      gateSubmitted.EventID,
		FinalEditGateChanged:      true,
	})
	if _, err := svc.AppendEvent(ctx, canonicalReq); err != nil {
		t.Fatal(err)
	}

	if _, ok, err := LoadLongFormFinalization(ctx, svc, binding); !errors.Is(err, app.ErrConflict) || ok {
		t.Fatalf("canonical/gate mismatch load ok=%t err=%v, want conflict", ok, err)
	}
}

func TestSubmitFinalEditEvidenceGateCanonicalLoadAndIdempotentReplay(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "plasma.db")
	svc, closeStore := newFinalEditStageStoreFixtureWithPipeline(t, ctx, path, FinalEditHumanizeDisabled, FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3)
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitV3FinalEditReaderForStoreTest(t, ctx, svc, binding, "evidence_canonical")
	evidenceBinding := finalEditStageStoreStageBinding(binding, FinalEditStageEvidenceGate, reader.Artifact.ArtifactID, binding.ArtifactID)
	finalBinding := finalEditStageStoreFinalBindingForStage(binding, evidenceBinding)
	approvedSvc := finalEditApprovedEvidenceStoreForFinalEditTest(svc, binding.MissionID, "evd_evidence_canonical")
	if _, created, err := StartFinalEditStage(ctx, approvedSvc, "evt_evidence_canonical_start", evidenceBinding); err != nil || !created {
		t.Fatalf("evidence gate start created=%t err=%v", created, err)
	}
	passages, err := FinalEditEvidenceGatePassages(string(reader.Artifact.Content))
	if err != nil {
		t.Fatal(err)
	}
	badFinding := FinalEditGateFinding{StatementSHA256: strings.Repeat("0", 64), Classification: FinalEditGateClassDerivedSynthesis}
	if _, err := SubmitFinalEditEvidenceGate(ctx, approvedSvc, FinalEditEvidenceGateSubmitRequest{
		StageBinding:     evidenceBinding,
		FinalBinding:     finalBinding,
		StageEventID:     "evt_evidence_bad_hash_submit",
		CanonicalEventID: "evt_evidence_bad_hash_final",
		Findings:         []FinalEditGateFinding{badFinding},
	}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("foreign evidence hash error=%v, want invalid input", err)
	}

	finding := FinalEditGateFinding{
		StatementSHA256: passages[0].StatementSHA256,
		Classification:  FinalEditGateClassUnverifiedExternalFact,
		EvidenceIDs:     []string{"evd_evidence_canonical"},
	}
	finalized, err := SubmitFinalEditEvidenceGate(ctx, approvedSvc, FinalEditEvidenceGateSubmitRequest{
		StageBinding:     evidenceBinding,
		FinalBinding:     finalBinding,
		StageEventID:     "evt_evidence_canonical_submit",
		CanonicalEventID: "evt_evidence_canonical_final",
		Findings:         []FinalEditGateFinding{finding},
	})
	if err != nil {
		t.Fatal(err)
	}
	if finalized.Artifact.ArtifactID != reader.Artifact.ArtifactID || string(finalized.Artifact.Content) != string(reader.Artifact.Content) {
		t.Fatalf("evidence gate canonicalized different content: %#v", finalized.Artifact)
	}
	loaded, ok, err := LoadFinalEditStageSubmission(ctx, approvedSvc, evidenceBinding)
	if err != nil || !ok {
		t.Fatalf("LoadFinalEditStageSubmission after canonical ok=%t err=%v", ok, err)
	}
	if loaded.Artifact.ArtifactID != reader.Artifact.ArtifactID || loaded.OperationCount != 0 || loaded.Changed {
		t.Fatalf("loaded evidence gate submission differs: %#v", loaded)
	}
	replayed, err := SubmitFinalEditEvidenceGate(ctx, approvedSvc, FinalEditEvidenceGateSubmitRequest{
		StageBinding:     evidenceBinding,
		FinalBinding:     finalBinding,
		StageEventID:     "evt_evidence_canonical_submit_again",
		CanonicalEventID: "evt_evidence_canonical_final_again",
		Findings:         []FinalEditGateFinding{finding},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replay || replayed.Event.EventID != finalized.Event.EventID || replayed.Artifact.ArtifactID != finalized.Artifact.ArtifactID {
		t.Fatalf("evidence gate idempotent replay differs: %#v", replayed)
	}
	closeStore()

	reopened, closeReopened := newFinalEditStageStoreFixtureFromExistingDB(t, ctx, path)
	defer closeReopened()
	reopenedApproved := finalEditApprovedEvidenceStoreForFinalEditTest(reopened, binding.MissionID, "evd_evidence_canonical")
	restartedStage, ok, err := LoadFinalEditStageSubmission(ctx, reopenedApproved, evidenceBinding)
	if err != nil || !ok || restartedStage.Artifact.ArtifactID != finalized.Artifact.ArtifactID {
		t.Fatalf("restarted evidence stage load ok=%t stage=%#v err=%v", ok, restartedStage, err)
	}
	restartedFinal, ok, err := LoadLongFormFinalization(ctx, reopenedApproved, finalBinding)
	if err != nil || !ok || restartedFinal.Artifact.ArtifactID != finalized.Artifact.ArtifactID {
		t.Fatalf("restarted canonical load ok=%t final=%#v err=%v", ok, restartedFinal, err)
	}
}

func TestFinalEditEvidenceGateReplayRejectsDurableRepairActionAndForeignHash(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name           string
		finding        StoredFinalEditGateFinding
		operationCount int
		changed        bool
		changedContent bool
		want           error
	}{
		{
			name: "repair_action",
			finding: StoredFinalEditGateFinding{
				StatementSHA256: contentSHA256([]byte("# Report")),
				Classification:  FinalEditGateClassUnverifiedExternalFact,
				RepairAction:    FinalEditRepairRemove,
			},
			want: app.ErrConflict,
		},
		{
			name: "foreign_hash",
			finding: StoredFinalEditGateFinding{
				StatementSHA256: strings.Repeat("0", 64),
				Classification:  FinalEditGateClassDerivedSynthesis,
			},
			want: app.ErrInvalidInput,
		},
		{
			name:           "changed_artifact",
			operationCount: 0,
			changed:        true,
			changedContent: true,
			want:           app.ErrConflict,
		},
		{
			name:           "nonzero_operation_count",
			operationCount: 1,
			want:           app.ErrConflict,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, closeStore := newFinalEditStageStoreFixtureWithPipeline(t, ctx, filepath.Join(t.TempDir(), "plasma.db"), FinalEditHumanizeDisabled, FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3)
			defer closeStore()
			binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
			reader := startAndSubmitV3FinalEditReaderForStoreTest(t, ctx, svc, binding, "evidence_reject_"+tc.name)
			evidenceBinding := finalEditStageStoreStageBinding(binding, FinalEditStageEvidenceGate, reader.Artifact.ArtifactID, binding.ArtifactID)
			evidenceBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
			if _, created, err := StartFinalEditStage(ctx, svc, "evt_evidence_reject_"+tc.name+"_start", evidenceBinding); err != nil || !created {
				t.Fatalf("evidence gate start created=%t err=%v", created, err)
			}
			artifact := reader.Artifact
			if tc.changedContent {
				artifact = createFinalEditReplayArtifact(t, ctx, svc, evidenceBinding, evidenceBinding.EditedArtifactID, "# Report\n\nChanged by forged evidence gate.\n", evidenceBinding.Producer)
			}
			findings := []StoredFinalEditGateFinding{tc.finding}
			if tc.finding.Classification == "" {
				findings = nil
			}
			if _, err := svc.AppendEvent(ctx, buildFinalEditSubmittedAppendRequest("evt_evidence_reject_"+tc.name+"_submit", evidenceBinding, reader.Artifact, artifact, tc.operationCount, tc.changed, findings, FinalEditSemanticAttestation{})); err != nil {
				t.Fatal(err)
			}
			_, ok, err := LoadFinalEditStageSubmission(ctx, svc, evidenceBinding)
			if !errors.Is(err, tc.want) || ok {
				t.Fatalf("durable evidence replay ok=%t err=%v, want %v", ok, err, tc.want)
			}
		})
	}
}

type finalEditStageStoreReaderResult struct {
	Binding  FinalEditStageBinding
	Artifact app.RawArtifact
}

type finalEditStageDuplicateReadGuardStore struct {
	*app.Service
	eventLists          int
	failAfterEventLists int
}

func (s *finalEditStageDuplicateReadGuardStore) ListEvents(ctx context.Context, missionID string) ([]app.LedgerEvent, error) {
	s.eventLists++
	if s.failAfterEventLists > 0 && s.eventLists > s.failAfterEventLists {
		return nil, errors.New("unexpected duplicate lineage event read")
	}
	return s.Service.ListEvents(ctx, missionID)
}

func newFinalEditStageStoreFixture(t *testing.T, ctx context.Context, humanize string) (*app.Service, func()) {
	t.Helper()
	return newFinalEditStageStoreFixtureWithPipeline(t, ctx, filepath.Join(t.TempDir(), "plasma.db"), humanize, FinalEditPipelineReaderStyleGateV1)
}

func newFinalEditStageStoreFixtureAt(t *testing.T, ctx context.Context, path string, humanize string) (*app.Service, func()) {
	t.Helper()
	return newFinalEditStageStoreFixtureWithPipeline(t, ctx, path, humanize, FinalEditPipelineReaderStyleGateV1)
}

func newFinalEditStageStoreFixtureWithPipeline(t *testing.T, ctx context.Context, path string, humanize string, pipeline string) (*app.Service, func()) {
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
			"final_edit_pipeline": pipeline, "post_report_humanize": humanize,
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

func newFinalEditStageStoreFixtureFromExistingDB(t *testing.T, ctx context.Context, path string) (*app.Service, func()) {
	t.Helper()
	store, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	return app.NewService(store), func() { _ = store.Close() }
}

type finalEditApprovedEvidenceStore struct {
	*app.Service
	evidence map[string]app.EvidenceRecord
}

func finalEditApprovedEvidenceStoreForFinalEditTest(svc *app.Service, missionID string, evidenceIDs ...string) finalEditApprovedEvidenceStore {
	evidence := make(map[string]app.EvidenceRecord, len(evidenceIDs))
	for _, evidenceID := range evidenceIDs {
		evidence[evidenceID] = app.EvidenceRecord{EvidenceID: evidenceID, MissionID: missionID, State: "approved"}
	}
	return finalEditApprovedEvidenceStore{Service: svc, evidence: evidence}
}

func (s finalEditApprovedEvidenceStore) GetEvidenceRecord(ctx context.Context, evidenceID string) (app.EvidenceRecord, error) {
	if record, ok := s.evidence[evidenceID]; ok {
		return record, nil
	}
	return s.Service.GetEvidenceRecord(ctx, evidenceID)
}

func startAndSubmitFinalEditReaderForStoreTest(t *testing.T, ctx context.Context, svc *app.Service, binding LongFormFinalizeBinding, suffix string) finalEditStageStoreReaderResult {
	t.Helper()
	sourceID := FinalEditReaderSourceArtifactID(binding.PlanEventID, []string{"art_part"})
	readerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageReader, sourceID, "art_reader_"+suffix)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_reader_"+suffix+"_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	markdown := AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nPreserved body.\n"})
	result, err := SubmitFinalEditStage(ctx, svc, readerBinding, "evt_reader_"+suffix+"_submit", markdown, 0)
	if err != nil {
		t.Fatal(err)
	}
	return finalEditStageStoreReaderResult{Binding: readerBinding, Artifact: result.Artifact}
}

func startAndSubmitV3FinalEditReaderForStoreTest(t *testing.T, ctx context.Context, svc *app.Service, binding LongFormFinalizeBinding, suffix string) finalEditStageStoreReaderResult {
	t.Helper()
	assemblyID := FinalEditAssemblyArtifactID(binding.PlanEventID, binding.PartArtifactIDs)
	writerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageWriter, assemblyID, "art_writer_"+suffix)
	writerBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, created, err := EnsureFinalEditAssembly(ctx, svc, "evt_assembly_"+suffix, writerBinding); err != nil || !created {
		t.Fatalf("assembly created=%t err=%v", created, err)
	}
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_writer_"+suffix+"_start", writerBinding); err != nil || !created {
		t.Fatalf("writer start created=%t err=%v", created, err)
	}
	markdown := AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nPreserved body.\n"})
	writer, err := SubmitFinalEditStage(ctx, svc, writerBinding, "evt_writer_"+suffix+"_submit", markdown, 0)
	if err != nil {
		t.Fatal(err)
	}
	readerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageReader, writer.Artifact.ArtifactID, "art_reader_"+suffix)
	readerBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_reader_"+suffix+"_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	reader, err := SubmitFinalEditStage(ctx, svc, readerBinding, "evt_reader_"+suffix+"_submit", string(writer.Artifact.Content), 0)
	if err != nil {
		t.Fatal(err)
	}
	return finalEditStageStoreReaderResult{Binding: readerBinding, Artifact: reader.Artifact}
}

func finalEditStageStoreFinalBinding(humanize string) LongFormFinalizeBinding {
	return LongFormFinalizeBinding{
		MissionID: "mis_final_edit_store", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ArtifactID: "art_final", Filename: "report.md", Title: "Report",
		ToolSessionID: "ses_corrective_gate", IdempotencyKey: "final-key", ProviderSessionID: "provider-corrective-gate", PreviousProviderSessionID: "provider-corrective-gate",
		PartArtifactIDs: []string{"art_part"}, SectionArtifactIDs: []string{"art_section"}, SectionWordCount: 3,
		CompositionStrategy: LongFormCompositionNarrativeEdit,
		AgentExecutor:       "codex", AgentModel: "model", AgentReasoningEffort: "high", AgentSelectionSource: "request", MCPMode: "auto",
		RigorLevel: "standard", RigorLabel: "Standard", ReportSessionPolicy: "same_session", ReportSessionPolicySelection: "default",
		PostReportHumanize: humanize, GenerationGuidanceProfile: "default", GenerationGuidanceSHA256: "guidance-sha",
		SessionChainKind: "same_session_report", PreReportResearchSessionID: "provider-research", ReportPlanSessionID: "provider-plan",
		ForkSourceAgentSessionID: "provider-plan", PlanToolSessionID: "ses_plan", Producer: app.Producer{Type: "agent_session", ID: "provider-corrective-gate"},
	}
}

func finalEditStageStoreStageBinding(binding LongFormFinalizeBinding, stage string, sourceArtifactID string, editedArtifactID string) FinalEditStageBinding {
	providerSessionID := "provider-" + strings.ReplaceAll(stage, "_", "-")
	filename := binding.Filename
	if stage == FinalEditStageGate {
		providerSessionID = binding.ProviderSessionID
	}
	return FinalEditStageBinding{
		MissionID: binding.MissionID, PendingEventID: binding.PendingEventID, PlanEventID: binding.PlanEventID,
		Title: binding.Title, Stage: stage, SourceArtifactID: sourceArtifactID, EditedArtifactID: editedArtifactID,
		Filename: filename, ToolSessionID: "ses_" + stage, ProviderSessionID: providerSessionID, PreviousProviderSessionID: providerSessionID,
		IdempotencyKey: FinalEditStageIdempotencyKey(stage, binding.PendingEventID, binding.PlanEventID),
		AgentExecutor:  binding.AgentExecutor, AgentModel: binding.AgentModel, AgentReasoningEffort: binding.AgentReasoningEffort,
		AgentSelectionSource: binding.AgentSelectionSource, MCPMode: binding.MCPMode, RigorLevel: binding.RigorLevel, RigorLabel: binding.RigorLabel,
		ReportSessionPolicy: binding.ReportSessionPolicy, ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
		PostReportHumanize: binding.PostReportHumanize, GenerationGuidanceProfile: binding.GenerationGuidanceProfile, GenerationGuidanceSHA256: binding.GenerationGuidanceSHA256,
		SessionChainKind: binding.SessionChainKind, PreReportResearchSessionID: binding.PreReportResearchSessionID, ReportPlanSessionID: binding.ReportPlanSessionID,
		ForkSourceAgentSessionID: binding.ReportPlanSessionID, Producer: app.Producer{Type: "agent_session", ID: providerSessionID},
	}
}

func finalEditStageStoreFinalBindingForStage(binding LongFormFinalizeBinding, stage FinalEditStageBinding) LongFormFinalizeBinding {
	binding.ToolSessionID = stage.ToolSessionID
	binding.ProviderSessionID = stage.ProviderSessionID
	binding.PreviousProviderSessionID = stage.PreviousProviderSessionID
	binding.ForkSourceAgentSessionID = stage.ForkSourceAgentSessionID
	binding.Producer = stage.Producer
	binding.PostReportHumanize = stage.PostReportHumanize
	return binding
}

func finalEditStageStoreJSON(value any) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}
