package reporting_test

import (
	"context"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestFinalEditStageLineageRequiresEnabledStyleAndGateFinalTarget(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithFinalEditPipeline(t, ctx, reporting.FinalEditPipelineReaderStyleGateV1, reporting.FinalEditHumanizeDisabled)
	defer closeStore()
	binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeDisabled)
	reader := startAndSubmitReaderStage(t, ctx, svc, binding, "lineage")

	styleBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageStyle, reader.Artifact.ArtifactID, "art_style_disabled", "")
	if _, _, err := reporting.StartFinalEditStage(ctx, svc, "evt_style_disabled_start", styleBinding); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("disabled style start error=%v, want conflict", err)
	}

	wrongGate := longFormFinalEditStageBinding(binding, reporting.FinalEditStageGate, reader.Artifact.ArtifactID, "art_wrong_final", "")
	if _, _, err := reporting.StartFinalEditStage(ctx, svc, "evt_gate_wrong_target_start", wrongGate); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("gate wrong target error=%v, want conflict", err)
	}

	gateBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID, "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_gate_lineage_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
}

type finalEditReaderStageTestResult struct {
	Binding  reporting.FinalEditStageBinding
	Artifact app.RawArtifact
}

func startAndSubmitReaderStage(t *testing.T, ctx context.Context, svc *app.Service, binding reporting.LongFormFinalizeBinding, suffix string) finalEditReaderStageTestResult {
	t.Helper()
	sourceID := reporting.FinalEditReaderSourceArtifactID(binding.PlanEventID, []string{"art_part"})
	readerBinding := longFormFinalEditStageBinding(binding, reporting.FinalEditStageReader, sourceID, "art_reader_"+suffix, "")
	if _, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_reader_"+suffix+"_start", readerBinding); err != nil || !created {
		t.Fatalf("reader start created=%t err=%v", created, err)
	}
	markdown := reporting.AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nPreserved body.\n"})
	result, err := reporting.SubmitFinalEditStage(ctx, svc, readerBinding, "evt_reader_"+suffix+"_submit", markdown, 0)
	if err != nil {
		t.Fatal(err)
	}
	return finalEditReaderStageTestResult{Binding: readerBinding, Artifact: result.Artifact}
}
