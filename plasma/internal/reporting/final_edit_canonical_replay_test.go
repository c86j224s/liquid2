package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestLoadLongFormFinalizationRejectsMissingPipelineMarkerForBoundPlan(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "missing_marker")
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_gate_missing_marker_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	manuscript := "# Report\n\nCorrected canonical content.\n"
	finalArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: binding.ArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: binding.Producer, Content: []byte(manuscript),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, buildFinalEditSubmittedAppendRequest("evt_gate_missing_marker_submit", gateBinding, reader.Artifact, finalArtifact, 1, true, nil)); err != nil {
		t.Fatal(err)
	}
	canonicalReq := longFormCanonicalRequest("evt_final_missing_marker", binding, finalArtifact, len(strings.Fields(manuscript)))
	if _, err := svc.AppendEvent(ctx, canonicalReq); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := LoadLongFormFinalization(ctx, svc, binding); ok || !errors.Is(err, app.ErrConflict) {
		t.Fatalf("load ok=%t err=%v, want missing pipeline conflict", ok, err)
	}
}

func TestLoadLongFormFinalizationRejectsCanonicalArtifactSHAMismatch(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "sha_mismatch")
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_gate_sha_mismatch_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	manuscript := "# Report\n\nCorrected canonical content.\n"
	finalArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: binding.ArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: binding.Producer, Content: []byte(manuscript),
	})
	if err != nil {
		t.Fatal(err)
	}
	gateEvent, err := svc.AppendEvent(ctx, buildFinalEditSubmittedAppendRequest("evt_gate_sha_mismatch_submit", gateBinding, reader.Artifact, finalArtifact, 1, true, nil))
	if err != nil {
		t.Fatal(err)
	}
	canonicalReq := longFormCanonicalRequestForFinalEdit("evt_final_sha_mismatch", binding, finalArtifact, len(strings.Fields(manuscript)), LongFormFinalizeRequest{
		FinalEditPipeline:         FinalEditPipelineReaderStyleGateV1,
		FinalEditActualArtifactID: finalArtifact.ArtifactID,
		FinalEditGateEventID:      gateEvent.EventID,
		FinalEditGateChanged:      true,
	})
	finalEditCanonicalReplayMutatePayload(t, &canonicalReq, func(payload map[string]any) {
		payload["artifact_sha256"] = strings.Repeat("0", 64)
	})
	if _, err := svc.AppendEvent(ctx, canonicalReq); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := LoadLongFormFinalization(ctx, svc, binding); ok || !errors.Is(err, app.ErrConflict) {
		t.Fatalf("load ok=%t err=%v, want SHA conflict", ok, err)
	}
}

func TestLoadLongFormFinalizationRejectsSecondMatchingGateSubmission(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newFinalEditStageStoreFixture(t, ctx, FinalEditHumanizeDisabled)
	defer closeStore()
	binding := finalEditStageStoreFinalBinding(FinalEditHumanizeDisabled)
	reader := startAndSubmitFinalEditReaderForStoreTest(t, ctx, svc, binding, "gate_duplicate")
	gateBinding := finalEditStageStoreStageBinding(binding, FinalEditStageGate, reader.Artifact.ArtifactID, binding.ArtifactID)
	if _, created, err := StartFinalEditStage(ctx, svc, "evt_gate_duplicate_start", gateBinding); err != nil || !created {
		t.Fatalf("gate start created=%t err=%v", created, err)
	}
	manuscript := "# Report\n\nCorrected canonical content.\n"
	finalArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: binding.ArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: binding.Producer, Content: []byte(manuscript),
	})
	if err != nil {
		t.Fatal(err)
	}
	firstGate, err := svc.AppendEvent(ctx, buildFinalEditSubmittedAppendRequest("evt_gate_duplicate_submit", gateBinding, reader.Artifact, finalArtifact, 1, true, nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, buildFinalEditSubmittedAppendRequest("evt_gate_duplicate_submit_tampered", gateBinding, reader.Artifact, finalArtifact, 1, true, nil)); err != nil {
		t.Fatal(err)
	}
	canonicalReq := longFormCanonicalRequestForFinalEdit("evt_final_gate_duplicate", binding, finalArtifact, len(strings.Fields(manuscript)), LongFormFinalizeRequest{
		FinalEditPipeline:         FinalEditPipelineReaderStyleGateV1,
		FinalEditActualArtifactID: finalArtifact.ArtifactID,
		FinalEditGateEventID:      firstGate.EventID,
		FinalEditGateChanged:      true,
	})
	if _, err := svc.AppendEvent(ctx, canonicalReq); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := LoadLongFormFinalization(ctx, svc, binding); ok || !errors.Is(err, app.ErrConflict) {
		t.Fatalf("load ok=%t err=%v, want duplicate gate submission conflict", ok, err)
	}
}

func finalEditCanonicalReplayMutatePayload(t *testing.T, request *app.AppendEventRequest, mutate func(map[string]any)) {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(request.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	mutate(payload)
	request.Payload = mustJSON(payload)
}
