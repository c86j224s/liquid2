package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestFinalEditGateRequiresSemanticAcceptanceForEveryStyleChangedParagraph(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeEnabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "semantic_reader")
	styleBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyle, reader.Artifact.ArtifactID, "art_style_semantic")
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_style_semantic_start", styleBinding); err != nil || !created {
		t.Fatalf("style start created=%t err=%v", created, err)
	}
	styleMarkdown := strings.Replace(string(reader.Artifact.Content), "Preserved body.", "Preserved body!", 1)
	style, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_semantic_submit", styleMarkdown, 1, finalEditStyleDiagnosesForTest(1))
	if err != nil {
		t.Fatal(err)
	}
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, style.Artifact.ArtifactID, binding.ArtifactID)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_gate_semantic_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	_, err = SubmitFinalEditGate(ctx, svc, FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       binding,
		StageEventID:       "evt_gate_semantic_missing_submit",
		CanonicalEventID:   "evt_gate_semantic_missing_final",
		ManuscriptMarkdown: styleMarkdown,
		OperationCount:     0,
		Findings:           nil,
		SemanticAcceptance: nil,
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("missing semantic acceptance error=%v, want conflict", err)
	}
	comparison, err := FinalEditSemanticComparison(ctx, svc, gateBinding, styleMarkdown)
	if err != nil || len(comparison) != 1 {
		t.Fatalf("comparison=%#v err=%v", comparison, err)
	}
	review := FinalEditSemanticAcceptance{
		ParagraphOrdinal:      comparison[0].ParagraphOrdinal,
		FinalParagraphOrdinal: comparison[0].ParagraphOrdinal,
		Verdict:               FinalEditSemanticAcceptedEquivalent,
	}
	result, err := SubmitFinalEditGate(ctx, svc, FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       binding,
		StageEventID:       "evt_gate_semantic_submit",
		CanonicalEventID:   "evt_gate_semantic_final",
		ManuscriptMarkdown: styleMarkdown,
		OperationCount:     0,
		Findings:           nil,
		SemanticAcceptance: []FinalEditSemanticAcceptance{review},
	})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(result.Event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["final_edit_semantic_acceptance_digest"] == "" || payload["final_edit_semantic_acceptance_count"] != float64(1) {
		t.Fatalf("canonical semantic attestation missing: %#v", payload)
	}
	if strings.Contains(string(result.Event.Payload), "Preserved body!") {
		t.Fatalf("raw paragraph leaked into canonical payload: %s", result.Event.Payload)
	}
}

func TestFinalEditSemanticAcceptanceRejectsDuplicateMismatchedAndUnresolvedReviews(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeEnabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "semantic_reject")
	styleBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyle, reader.Artifact.ArtifactID, "art_style_reject")
	if _, _, err := StartFinalEditStage(ctx, svc, "evt_style_reject_start", styleBinding); err != nil {
		t.Fatal(err)
	}
	styleMarkdown := strings.Replace(string(reader.Artifact.Content), "Preserved body.", "Preserved body!", 1)
	style, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_reject_submit", styleMarkdown, 1, finalEditStyleDiagnosesForTest(1))
	if err != nil {
		t.Fatal(err)
	}
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, style.Artifact.ArtifactID, binding.ArtifactID)
	comparison, err := FinalEditSemanticComparison(ctx, svc, gateBinding, styleMarkdown)
	if err != nil || len(comparison) != 1 {
		t.Fatalf("comparison=%#v err=%v", comparison, err)
	}
	base := FinalEditSemanticAcceptance{
		ParagraphOrdinal:      comparison[0].ParagraphOrdinal,
		FinalParagraphOrdinal: comparison[0].ParagraphOrdinal,
		Verdict:               FinalEditSemanticAcceptedEquivalent,
	}
	for name, reviews := range map[string][]FinalEditSemanticAcceptance{
		"duplicate":  {base, base},
		"mismatch":   {{ParagraphOrdinal: 3, FinalParagraphOrdinal: 3, Verdict: FinalEditSemanticAcceptedEquivalent}},
		"unresolved": {{ParagraphOrdinal: base.ParagraphOrdinal, FinalParagraphOrdinal: base.FinalParagraphOrdinal, Verdict: FinalEditSemanticRevertedToReader}},
	} {
		if _, err := ValidateFinalEditSemanticAcceptance(ctx, svc, gateBinding, styleMarkdown, reviews); !errors.Is(err, app.ErrConflict) && !errors.Is(err, app.ErrInvalidInput) {
			t.Fatalf("%s error=%v, want closed failure", name, err)
		}
	}
}

func TestFinalEditStyleSemanticValidationV3BuildsResolvedMarkdownDeterministically(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixtureWithPipeline(t, ctx, filepath.Join(t.TempDir(), "plasma.db"), FinalEditHumanizeEnabled, FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeEnabled)
	assemblyID := FinalEditAssemblyArtifactID(binding.PlanEventID, binding.PartArtifactIDs)
	writerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageWriter, assemblyID, "art_v3_semantic_writer")
	writerBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, created, err := EnsureFinalEditAssembly(ctx, svc, "evt_v3_semantic_assembly", writerBinding); err != nil || !created {
		t.Fatalf("assembly created=%t err=%v", created, err)
	}
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_v3_semantic_writer_start", writerBinding); err != nil || !created {
		t.Fatalf("writer start created=%t err=%v", created, err)
	}
	writerMarkdown := AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nPreserved body.\n"})
	writer, err := SubmitFinalEditStage(ctx, svc, writerBinding, "evt_v3_semantic_writer_submit", writerMarkdown, 0)
	if err != nil {
		t.Fatal(err)
	}
	readerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageReader, writer.Artifact.ArtifactID, "art_v3_semantic_reader")
	readerBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_v3_semantic_reader_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	reader, err := SubmitFinalEditStage(ctx, svc, readerBinding, "evt_v3_semantic_reader_submit", string(writer.Artifact.Content), 0)
	if err != nil {
		t.Fatal(err)
	}
	styleBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyle, reader.Artifact.ArtifactID, "art_v3_semantic_style")
	styleBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	styleBinding.ForkSourceAgentSessionID = reader.Binding.ProviderSessionID
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_v3_semantic_style_start", styleBinding); err != nil || !created {
		t.Fatalf("style start created=%t err=%v", created, err)
	}
	styleMarkdown := strings.Replace(string(reader.Artifact.Content), "Preserved body.", "Preserved body!", 1)
	style, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_v3_semantic_style_submit", styleMarkdown, 1, finalEditStyleDiagnosesForTest(1))
	if err != nil {
		t.Fatal(err)
	}
	semanticBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyleSemanticValidation, style.Artifact.ArtifactID, "art_v3_semantic_validated")
	semanticBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_v3_semantic_validation_start", semanticBinding); err != nil || !created {
		t.Fatalf("semantic validation start created=%t err=%v", created, err)
	}
	comparison, err := FinalEditSemanticComparison(ctx, svc, semanticBinding, styleMarkdown)
	if err != nil || len(comparison) != 1 {
		t.Fatalf("comparison=%#v err=%v", comparison, err)
	}

	rejected, err := SubmitFinalEditStyleSemanticValidation(ctx, svc, semanticBinding, "evt_v3_semantic_validation_submit", []FinalEditSemanticAcceptance{{
		ParagraphOrdinal: comparison[0].ParagraphOrdinal,
		Verdict:          FinalEditSemanticRejectedRevertToReader,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if string(rejected.Artifact.Content) != string(reader.Artifact.Content) {
		t.Fatalf("rejected verdict did not revert to reader markdown:\n%s", rejected.Artifact.Content)
	}
	if rejected.OperationCount != 0 || rejected.SemanticReview.Count != 1 || rejected.SemanticReview.Records[0].FinalParagraphOrdinal != comparison[0].ParagraphOrdinal {
		t.Fatalf("semantic validation attestation differs: %#v", rejected)
	}
	if rejected.SemanticReview.Records[0].FinalSHA256 != rejected.SemanticReview.Records[0].ReaderSHA256 {
		t.Fatalf("rejected verdict final hash must point to reader block: %#v", rejected.SemanticReview.Records[0])
	}

	_, err = SubmitFinalEditStyleSemanticValidation(ctx, svc, semanticBinding, "evt_v3_semantic_validation_bad_final_ordinal", []FinalEditSemanticAcceptance{{
		ParagraphOrdinal:      comparison[0].ParagraphOrdinal,
		FinalParagraphOrdinal: comparison[0].ParagraphOrdinal,
		Verdict:               FinalEditSemanticAcceptedEquivalent,
	}})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("style semantic validation accepted agent-supplied final ordinal err=%v", err)
	}
}

func TestFinalEditStyleSemanticValidationV3ReplayValidAcceptAndRevert(t *testing.T) {
	ctx := context.Background()
	for name, verdict := range map[string]string{
		"accept": FinalEditSemanticAcceptedEquivalent,
		"revert": FinalEditSemanticRejectedRevertToReader,
	} {
		t.Run(name, func(t *testing.T) {
			svc, closeStore, semanticBinding, comparison := seededV3StyleSemanticValidationReplayFixture(t, ctx, name)
			defer closeStore()
			submitted, err := SubmitFinalEditStyleSemanticValidation(ctx, svc, semanticBinding, "evt_v3_style_semantic_"+name+"_submit", []FinalEditSemanticAcceptance{{
				ParagraphOrdinal: comparison[0].ParagraphOrdinal,
				Verdict:          verdict,
			}})
			if err != nil {
				t.Fatal(err)
			}
			loaded, ok, err := LoadFinalEditStageSubmission(ctx, svc, semanticBinding)
			if err != nil || !ok {
				t.Fatalf("load ok=%t err=%v", ok, err)
			}
			if loaded.OperationCount != 0 || loaded.Artifact.ArtifactID != submitted.Artifact.ArtifactID || !loaded.Replay {
				t.Fatalf("loaded semantic validation replay differs: %#v", loaded)
			}
		})
	}
}

func TestFinalEditStyleSemanticValidationV3ReplayRejectsTamperedDurablePayloads(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, ctx context.Context, svc *app.Service, binding FinalEditStageBinding, reader app.RawArtifact, style app.RawArtifact, comparison []FinalEditSemanticComparisonParagraph) (app.RawArtifact, int, bool, FinalEditSemanticAttestation)
	}{
		{
			name: "arbitrary_third_manuscript",
			mutate: func(t *testing.T, ctx context.Context, svc *app.Service, binding FinalEditStageBinding, reader app.RawArtifact, style app.RawArtifact, comparison []FinalEditSemanticComparisonParagraph) (app.RawArtifact, int, bool, FinalEditSemanticAttestation) {
				_, attestation, err := BuildFinalEditStyleSemanticValidation(ctx, svc, binding, []FinalEditSemanticAcceptance{{ParagraphOrdinal: comparison[0].ParagraphOrdinal, Verdict: FinalEditSemanticAcceptedEquivalent}})
				if err != nil {
					t.Fatal(err)
				}
				artifact := createFinalEditReplayArtifact(t, ctx, svc, binding, "art_v3_semantic_third_"+tcSafeName(binding.ProviderSessionID), "# Report\n\nArbitrary third manuscript.\n", binding.Producer)
				return artifact, 0, true, attestation
			},
		},
		{
			name: "missing_attestation",
			mutate: func(t *testing.T, ctx context.Context, svc *app.Service, binding FinalEditStageBinding, reader app.RawArtifact, style app.RawArtifact, comparison []FinalEditSemanticComparisonParagraph) (app.RawArtifact, int, bool, FinalEditSemanticAttestation) {
				return reader, 0, true, FinalEditSemanticAttestation{}
			},
		},
		{
			name: "partial_attestation",
			mutate: func(t *testing.T, ctx context.Context, svc *app.Service, binding FinalEditStageBinding, reader app.RawArtifact, style app.RawArtifact, comparison []FinalEditSemanticComparisonParagraph) (app.RawArtifact, int, bool, FinalEditSemanticAttestation) {
				return reader, 0, true, FinalEditSemanticAttestation{Count: 1}
			},
		},
		{
			name: "repaired_by_gate",
			mutate: func(t *testing.T, ctx context.Context, svc *app.Service, binding FinalEditStageBinding, reader app.RawArtifact, style app.RawArtifact, comparison []FinalEditSemanticComparisonParagraph) (app.RawArtifact, int, bool, FinalEditSemanticAttestation) {
				record := StoredFinalEditSemanticAcceptance{
					ParagraphOrdinal:      comparison[0].ParagraphOrdinal,
					FinalParagraphOrdinal: comparison[0].ParagraphOrdinal,
					Verdict:               FinalEditSemanticRepairedByGate,
					ReaderSHA256:          comparison[0].ReaderSHA256,
					StyleSHA256:           comparison[0].StyleSHA256,
					FinalSHA256:           comparison[0].StyleSHA256,
				}
				digest, err := finalEditSemanticAcceptanceDigest([]StoredFinalEditSemanticAcceptance{record})
				if err != nil {
					t.Fatal(err)
				}
				return style, 0, false, FinalEditSemanticAttestation{Records: []StoredFinalEditSemanticAcceptance{record}, Digest: digest, Count: 1}
			},
		},
		{
			name: "wrong_artifact",
			mutate: func(t *testing.T, ctx context.Context, svc *app.Service, binding FinalEditStageBinding, reader app.RawArtifact, style app.RawArtifact, comparison []FinalEditSemanticComparisonParagraph) (app.RawArtifact, int, bool, FinalEditSemanticAttestation) {
				_, attestation, err := BuildFinalEditStyleSemanticValidation(ctx, svc, binding, []FinalEditSemanticAcceptance{{ParagraphOrdinal: comparison[0].ParagraphOrdinal, Verdict: FinalEditSemanticRejectedRevertToReader}})
				if err != nil {
					t.Fatal(err)
				}
				return style, 0, false, attestation
			},
		},
		{
			name: "nonzero_operation_count",
			mutate: func(t *testing.T, ctx context.Context, svc *app.Service, binding FinalEditStageBinding, reader app.RawArtifact, style app.RawArtifact, comparison []FinalEditSemanticComparisonParagraph) (app.RawArtifact, int, bool, FinalEditSemanticAttestation) {
				_, attestation, err := BuildFinalEditStyleSemanticValidation(ctx, svc, binding, []FinalEditSemanticAcceptance{{ParagraphOrdinal: comparison[0].ParagraphOrdinal, Verdict: FinalEditSemanticAcceptedEquivalent}})
				if err != nil {
					t.Fatal(err)
				}
				return style, 1, false, attestation
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, closeStore, semanticBinding, comparison := seededV3StyleSemanticValidationReplayFixture(t, ctx, tc.name)
			defer closeStore()
			style, ok, err := finalEditStyleSubmissionForGate(ctx, svc, mustListFinalEditReplayEvents(t, ctx, svc, semanticBinding.MissionID), semanticBinding)
			if err != nil || !ok {
				t.Fatalf("style lineage ok=%t err=%v", ok, err)
			}
			artifact, operationCount, changed, attestation := tc.mutate(t, ctx, svc, semanticBinding, style.SourceArtifact, style.Artifact, comparison)
			if _, err := svc.AppendEvent(ctx, buildFinalEditSubmittedAppendRequest("evt_v3_style_semantic_"+tc.name+"_submit", semanticBinding, style.Artifact, artifact, operationCount, changed, nil, attestation)); err != nil {
				t.Fatal(err)
			}
			_, ok, err = LoadFinalEditStageSubmission(ctx, svc, semanticBinding)
			if !errors.Is(err, app.ErrConflict) || ok {
				t.Fatalf("tampered semantic validation replay ok=%t err=%v, want conflict", ok, err)
			}
		})
	}
}

func TestFinalEditStyleSemanticValidationV3ReplayRejectsMixedArtifactForeignProducer(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixtureWithPipeline(t, ctx, filepath.Join(t.TempDir(), "plasma.db"), FinalEditHumanizeEnabled, FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeEnabled)
	assemblyID := FinalEditAssemblyArtifactID(binding.PlanEventID, binding.PartArtifactIDs)
	writerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageWriter, assemblyID, "art_v3_mixed_writer")
	writerBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, created, err := EnsureFinalEditAssembly(ctx, svc, "evt_v3_mixed_assembly", writerBinding); err != nil || !created {
		t.Fatalf("assembly created=%t err=%v", created, err)
	}
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_v3_mixed_writer_start", writerBinding); err != nil || !created {
		t.Fatalf("writer start created=%t err=%v", created, err)
	}
	writerMarkdown := AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nFirst body.\n\nSecond body.\n"})
	writer, err := SubmitFinalEditStage(ctx, svc, writerBinding, "evt_v3_mixed_writer_submit", writerMarkdown, 1)
	if err != nil {
		t.Fatal(err)
	}
	readerBinding := finalEditStageStoreStageBinding(binding, FinalEditStageReader, writer.Artifact.ArtifactID, "art_v3_mixed_reader")
	readerBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_v3_mixed_reader_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	reader, err := SubmitFinalEditStage(ctx, svc, readerBinding, "evt_v3_mixed_reader_submit", string(writer.Artifact.Content), 0)
	if err != nil {
		t.Fatal(err)
	}
	styleBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyle, reader.Artifact.ArtifactID, "art_v3_mixed_style")
	styleBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	styleBinding.ForkSourceAgentSessionID = reader.Binding.ProviderSessionID
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_v3_mixed_style_start", styleBinding); err != nil || !created {
		t.Fatalf("style start created=%t err=%v", created, err)
	}
	styleMarkdown := strings.ReplaceAll(string(reader.Artifact.Content), "First body.", "First body!")
	styleMarkdown = strings.ReplaceAll(styleMarkdown, "Second body.", "Second body!")
	style, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_v3_mixed_style_submit", styleMarkdown, 2, finalEditStyleDiagnosesForTest(2))
	if err != nil {
		t.Fatal(err)
	}
	semanticBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyleSemanticValidation, style.Artifact.ArtifactID, "art_v3_mixed_validated")
	semanticBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_v3_mixed_semantic_start", semanticBinding); err != nil || !created {
		t.Fatalf("semantic validation start created=%t err=%v", created, err)
	}
	comparison, err := FinalEditSemanticComparison(ctx, svc, semanticBinding, "")
	if err != nil || len(comparison) != 2 {
		t.Fatalf("comparison=%#v err=%v", comparison, err)
	}
	reviews := []FinalEditSemanticAcceptance{
		{ParagraphOrdinal: comparison[0].ParagraphOrdinal, Verdict: FinalEditSemanticAcceptedEquivalent},
		{ParagraphOrdinal: comparison[1].ParagraphOrdinal, Verdict: FinalEditSemanticRejectedRevertToReader},
	}
	resolved, attestation, err := BuildFinalEditStyleSemanticValidation(ctx, svc, semanticBinding, reviews)
	if err != nil {
		t.Fatal(err)
	}
	if resolved == string(style.Artifact.Content) || resolved == string(reader.Artifact.Content) {
		t.Fatalf("test did not produce mixed resolved markdown:\n%s", resolved)
	}
	forged := createFinalEditReplayArtifact(t, ctx, svc, semanticBinding, semanticBinding.EditedArtifactID, resolved, app.Producer{Type: "agent_session", ID: "provider-foreign"})
	if _, err := svc.AppendEvent(ctx, buildFinalEditSubmittedAppendRequest("evt_v3_mixed_semantic_submit", semanticBinding, style.Artifact, forged, 0, true, nil, attestation)); err != nil {
		t.Fatal(err)
	}
	_, ok, err := LoadFinalEditStageSubmission(ctx, svc, semanticBinding)
	if ok || !errors.Is(err, app.ErrConflict) {
		t.Fatalf("foreign producer mixed replay ok=%t err=%v, want conflict", ok, err)
	}
}

func seededV3StyleSemanticValidationReplayFixture(t *testing.T, ctx context.Context, suffix string) (*app.Service, func(), FinalEditStageBinding, []FinalEditSemanticComparisonParagraph) {
	t.Helper()
	safeSuffix := tcSafeName(suffix)
	svc, closeStore := newFinalEditStageStoreFixtureWithPipeline(t, ctx, filepath.Join(t.TempDir(), "plasma.db"), FinalEditHumanizeEnabled, FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3)
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeEnabled)
	reader := startAndSubmitV3FinalEditReaderForStoreTest(t, ctx, svc, binding, "style_semantic_"+safeSuffix)
	styleBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyle, reader.Artifact.ArtifactID, "art_v3_style_"+safeSuffix)
	styleBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	styleBinding.ForkSourceAgentSessionID = reader.Binding.ProviderSessionID
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_v3_style_"+safeSuffix+"_start", styleBinding); err != nil || !created {
		t.Fatalf("style start created=%t err=%v", created, err)
	}
	styleMarkdown := strings.Replace(string(reader.Artifact.Content), "Preserved body.", "Preserved body!", 1)
	style, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_v3_style_"+safeSuffix+"_submit", styleMarkdown, 1, finalEditStyleDiagnosesForTest(1))
	if err != nil {
		t.Fatal(err)
	}
	semanticBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyleSemanticValidation, style.Artifact.ArtifactID, "art_v3_style_semantic_"+safeSuffix)
	semanticBinding.FinalEditPipeline = FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_v3_style_semantic_"+safeSuffix+"_start", semanticBinding); err != nil || !created {
		t.Fatalf("semantic validation start created=%t err=%v", created, err)
	}
	comparison, err := FinalEditSemanticComparison(ctx, svc, semanticBinding, "")
	if err != nil || len(comparison) != 1 {
		t.Fatalf("comparison=%#v err=%v", comparison, err)
	}
	return svc, closeStore, semanticBinding, comparison
}

func createFinalEditReplayArtifact(t *testing.T, ctx context.Context, svc *app.Service, binding FinalEditStageBinding, artifactID string, markdown string, producer app.Producer) app.RawArtifact {
	t.Helper()
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: artifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: producer, Content: []byte(markdown),
	})
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func mustListFinalEditReplayEvents(t *testing.T, ctx context.Context, svc *app.Service, missionID string) []app.LedgerEvent {
	t.Helper()
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func tcSafeName(value string) string {
	return strings.NewReplacer("-", "_", "/", "_", " ", "_").Replace(value)
}

func TestFinalEditGateDisabledRejectsSemanticAcceptanceBeforeLineageRead(t *testing.T) {
	ctx := context.Background()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, "art_reader_disabled_semantic", binding.ArtifactID)
	_, err := SubmitFinalEditGate(ctx, finalEditGateNoReadStore{}, FinalEditGateSubmitRequest{
		StageBinding:       gateBinding,
		FinalBinding:       binding,
		StageEventID:       "evt_gate_disabled_semantic_submit",
		CanonicalEventID:   "evt_gate_disabled_semantic_final",
		ManuscriptMarkdown: "# Report\n\nPreserved body.\n",
		OperationCount:     0,
		Findings:           nil,
		SemanticAcceptance: []FinalEditSemanticAcceptance{{
			ParagraphOrdinal:      1,
			FinalParagraphOrdinal: 1,
			Verdict:               FinalEditSemanticAcceptedEquivalent,
		}},
	})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("disabled semantic acceptance err=%v, want invalid input before lineage read", err)
	}
}

func TestFinalEditSemanticAcceptanceAllowsGateInsertedFinalParagraph(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeEnabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "semantic_insert")
	styleBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyle, reader.Artifact.ArtifactID, "art_style_insert")
	if _, _, err := StartFinalEditStage(ctx, svc, "evt_style_insert_start", styleBinding); err != nil {
		t.Fatal(err)
	}
	styleMarkdown := strings.Replace(string(reader.Artifact.Content), "Preserved body.", "Preserved body!", 1)
	style, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_insert_submit", styleMarkdown, 1, finalEditStyleDiagnosesForTest(1))
	if err != nil {
		t.Fatal(err)
	}
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, style.Artifact.ArtifactID, binding.ArtifactID)
	finalMarkdown := strings.Replace(styleMarkdown, "Preserved body!", "Gate inserted context.\n\nPreserved body!", 1)
	comparison, err := FinalEditSemanticComparison(ctx, svc, gateBinding, finalMarkdown)
	if err != nil || len(comparison) != 1 {
		t.Fatalf("comparison=%#v err=%v", comparison, err)
	}
	remappedFinalOrdinal := comparison[0].ParagraphOrdinal + 1
	attestation, err := ValidateFinalEditSemanticAcceptance(ctx, svc, gateBinding, finalMarkdown, []FinalEditSemanticAcceptance{{
		ParagraphOrdinal:      comparison[0].ParagraphOrdinal,
		FinalParagraphOrdinal: remappedFinalOrdinal,
		Verdict:               FinalEditSemanticAcceptedEquivalent,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if attestation.Count != 1 || attestation.Digest == "" || attestation.Records[0].FinalParagraphOrdinal != remappedFinalOrdinal {
		t.Fatalf("inserted final paragraph broke semantic attestation: %#v", attestation)
	}
	_, err = ValidateFinalEditSemanticAcceptance(ctx, svc, gateBinding, finalMarkdown, []FinalEditSemanticAcceptance{{
		ParagraphOrdinal:      comparison[0].ParagraphOrdinal,
		FinalParagraphOrdinal: comparison[0].ParagraphOrdinal,
		Verdict:               FinalEditSemanticAcceptedEquivalent,
	}})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("stale final ordinal error=%v, want conflict", err)
	}
}

type finalEditGateNoReadStore struct{}

func (finalEditGateNoReadStore) ListEvents(context.Context, string) ([]app.LedgerEvent, error) {
	return nil, errors.New("unexpected lineage read")
}

func (finalEditGateNoReadStore) GetRawArtifact(context.Context, string) (app.RawArtifact, error) {
	return app.RawArtifact{}, errors.New("unexpected artifact read")
}

func (finalEditGateNoReadStore) AppendEventConditionally(context.Context, string, func([]app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.LedgerEvent, bool, error) {
	return app.LedgerEvent{}, false, errors.New("unexpected append")
}

func (finalEditGateNoReadStore) CreateRawArtifactWithEventConditionally(context.Context, app.CreateRawArtifactRequest, func([]app.LedgerEvent, app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.RawArtifact, app.LedgerEvent, bool, error) {
	return app.RawArtifact{}, app.LedgerEvent{}, false, errors.New("unexpected artifact create")
}

func (finalEditGateNoReadStore) GetEvidenceRecord(context.Context, string) (app.EvidenceRecord, error) {
	return app.EvidenceRecord{}, errors.New("unexpected evidence read")
}

func TestFinalEditSemanticReplayRejectsStoredHashTamperAndRawFields(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeEnabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeEnabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "semantic_tamper")
	styleBinding := finalEditStageStoreStageBinding(binding, FinalEditStageStyle, reader.Artifact.ArtifactID, "art_style_tamper")
	if _, _, err := StartFinalEditStage(ctx, svc, "evt_style_tamper_start", styleBinding); err != nil {
		t.Fatal(err)
	}
	styleMarkdown := strings.Replace(string(reader.Artifact.Content), "Preserved body.", "Preserved body!", 1)
	style, err := SubmitFinalEditStyleStage(ctx, svc, styleBinding, "evt_style_tamper_submit", styleMarkdown, 1, finalEditStyleDiagnosesForTest(1))
	if err != nil {
		t.Fatal(err)
	}
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, style.Artifact.ArtifactID, binding.ArtifactID)
	comparison, err := FinalEditSemanticComparison(ctx, svc, gateBinding, styleMarkdown)
	if err != nil || len(comparison) != 1 {
		t.Fatalf("comparison=%#v err=%v", comparison, err)
	}
	valid, err := ValidateFinalEditSemanticAcceptance(ctx, svc, gateBinding, styleMarkdown, []FinalEditSemanticAcceptance{{
		ParagraphOrdinal:      comparison[0].ParagraphOrdinal,
		FinalParagraphOrdinal: comparison[0].ParagraphOrdinal,
		Verdict:               FinalEditSemanticAcceptedEquivalent,
	}})
	if err != nil {
		t.Fatal(err)
	}
	tampered := valid
	tampered.Records = append([]StoredFinalEditSemanticAcceptance(nil), valid.Records...)
	tampered.Records[0].ReaderSHA256 = tampered.Records[0].StyleSHA256
	tampered.Digest, err = finalEditSemanticAcceptanceDigest(tampered.Records)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateStoredFinalEditSemanticAcceptanceAgainstLineage(ctx, svc, gateBinding, styleMarkdown, tampered); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("tampered self-consistent semantic record err=%v, want conflict", err)
	}
	rawPayload := map[string]any{
		"semantic_acceptance": []map[string]any{{
			"paragraph_ordinal": 1, "final_paragraph_ordinal": 1, "verdict": FinalEditSemanticAcceptedEquivalent,
			"reader_sha256": contentSHA256([]byte("reader")), "style_sha256": contentSHA256([]byte("style")), "final_sha256": contentSHA256([]byte("style")),
			"reader_text": "raw text must fail",
		}},
		"semantic_acceptance_count":  1,
		"semantic_acceptance_digest": "not reached",
	}
	if _, err := decodeFinalEditSemanticAcceptancePayload(rawPayload); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("raw semantic field err=%v, want conflict", err)
	}
	duplicate := []StoredFinalEditSemanticAcceptance{
		{ParagraphOrdinal: 1, FinalParagraphOrdinal: 2, Verdict: FinalEditSemanticAcceptedEquivalent, ReaderSHA256: contentSHA256([]byte("r1")), StyleSHA256: contentSHA256([]byte("s1")), FinalSHA256: contentSHA256([]byte("s1"))},
		{ParagraphOrdinal: 2, FinalParagraphOrdinal: 2, Verdict: FinalEditSemanticAcceptedEquivalent, ReaderSHA256: contentSHA256([]byte("r2")), StyleSHA256: contentSHA256([]byte("s2")), FinalSHA256: contentSHA256([]byte("s2"))},
	}
	digest, err := finalEditSemanticAcceptanceDigest(duplicate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeFinalEditSemanticAcceptancePayload(map[string]any{
		"semantic_acceptance": duplicate, "semantic_acceptance_count": 2, "semantic_acceptance_digest": digest,
	}); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("duplicate final ordinal err=%v, want conflict", err)
	}
}
