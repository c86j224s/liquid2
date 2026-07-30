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
	gateSubmitted, err := svc.AppendEvent(ctx, buildFinalEditSubmittedAppendRequest("evt_gate_canonical_mismatch_submit", gateBinding, reader.Artifact, finalArtifact, 1, true, gateFindings))
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

type finalEditStageStoreReaderResult struct {
	Binding  FinalEditStageBinding
	Artifact app.RawArtifact
}

func newFinalEditStageStoreFixture(t *testing.T, ctx context.Context, humanize string) (*app.Service, func()) {
	t.Helper()
	return newFinalEditStageStoreFixtureAt(t, ctx, filepath.Join(t.TempDir(), "plasma.db"), humanize)
}

func newFinalEditStageStoreFixtureAt(t *testing.T, ctx context.Context, path string, humanize string) (*app.Service, func()) {
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
			"final_edit_pipeline": FinalEditPipelineReaderStyleGateV1, "post_report_humanize": humanize,
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

func finalEditStageStoreJSON(value any) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}
