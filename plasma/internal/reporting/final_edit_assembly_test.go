package reporting_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestFinalEditAssemblyCreatesDeterministicArtifactAndReplays(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithFinalEditPipeline(t, ctx, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, reporting.FinalEditHumanizeDisabled)
	defer closeStore()
	binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeDisabled)
	assemblyID := reporting.FinalEditAssemblyArtifactID(binding.PlanEventID, binding.PartArtifactIDs)
	writer := longFormFinalEditStageBinding(binding, reporting.FinalEditStageWriter, assemblyID, "art_writer_assembly", "")

	result, created, err := reporting.EnsureFinalEditAssembly(ctx, svc, "evt_assembly_created", writer)
	if err != nil || !created {
		t.Fatalf("assembly created=%t result=%#v err=%v", created, result, err)
	}
	wantMarkdown := reporting.AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nPreserved body.\n"})
	if result.Artifact.ArtifactID != assemblyID ||
		result.Artifact.Producer != (app.Producer{Type: "system", ID: reporting.FinalEditAssemblyProducerID}) ||
		result.Artifact.MediaType != "text/markdown; charset=utf-8" ||
		result.Artifact.Filename != binding.Filename ||
		string(result.Artifact.Content) != wantMarkdown ||
		result.Replay {
		t.Fatalf("assembly artifact contract mismatch: %#v", result)
	}
	if result.Event.EventType != reporting.FinalEditAssemblyCreatedEventType ||
		result.Event.Producer != (app.Producer{Type: "system", ID: reporting.FinalEditAssemblyProducerID}) ||
		result.Event.CausationEventID != binding.PlanEventID ||
		result.Event.CorrelationID != reporting.FinalEditAssemblyIdempotencyKey(binding.PlanEventID, binding.PartArtifactIDs) {
		t.Fatalf("assembly event envelope mismatch: %#v", result.Event)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(result.Event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["kind"] != reporting.FinalEditAssemblyKind ||
		payload["schema"] != reporting.FinalEditAssemblySchema ||
		payload["final_edit_pipeline"] != reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 ||
		payload["producer_id"] != reporting.FinalEditAssemblyProducerID ||
		payload["artifact_id"] != assemblyID ||
		payload["artifact_sha256"] != result.Artifact.SHA256 ||
		payload["source_word_count"] != float64(len(strings.Fields(wantMarkdown))) {
		t.Fatalf("assembly payload mismatch: %#v", payload)
	}

	replayed, created, err := reporting.EnsureFinalEditAssembly(ctx, svc, "evt_assembly_created_again", writer)
	if err != nil || created || !replayed.Replay ||
		replayed.Artifact.ArtifactID != result.Artifact.ArtifactID ||
		replayed.Event.EventID != result.Event.EventID {
		t.Fatalf("assembly replay changed identity created=%t result=%#v err=%v", created, replayed, err)
	}
	events, err := svc.ListEvents(ctx, binding.MissionID)
	if err != nil {
		t.Fatal(err)
	}
	if countEventType(events, reporting.FinalEditAssemblyCreatedEventType) != 1 {
		t.Fatalf("assembly replay appended duplicate events: %#v", events)
	}
}

func TestFinalEditAssemblyReplayRejectsTamperedSourceWordCount(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newLongFormFinalizeFixtureWithFinalEditPipeline(t, ctx, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, reporting.FinalEditHumanizeDisabled)
	defer closeStore()
	binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeDisabled)
	assemblyID := reporting.FinalEditAssemblyArtifactID(binding.PlanEventID, binding.PartArtifactIDs)
	writer := longFormFinalEditStageBinding(binding, reporting.FinalEditStageWriter, assemblyID, "art_writer_tampered_metric", "")
	markdown := reporting.AssembleLongFormFinalMarkdown(binding.Title, "", "", []string{"# Part 1\n\nPreserved body.\n"})
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: assemblyID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: app.Producer{Type: "system", ID: reporting.FinalEditAssemblyProducerID}, Content: []byte(markdown),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_assembly_tampered_metric", MissionID: binding.MissionID,
		EventType:        reporting.FinalEditAssemblyCreatedEventType,
		Producer:         app.Producer{Type: "system", ID: reporting.FinalEditAssemblyProducerID},
		CausationEventID: binding.PlanEventID,
		CorrelationID:    reporting.FinalEditAssemblyIdempotencyKey(binding.PlanEventID, binding.PartArtifactIDs),
		Payload: testJSON(map[string]any{
			"kind":                reporting.FinalEditAssemblyKind,
			"schema":              reporting.FinalEditAssemblySchema,
			"pending_event_id":    binding.PendingEventID,
			"plan_event_id":       binding.PlanEventID,
			"final_edit_pipeline": reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2,
			"title":               binding.Title,
			"artifact_id":         artifact.ArtifactID,
			"filename":            binding.Filename,
			"producer_id":         reporting.FinalEditAssemblyProducerID,
			"part_artifact_ids":   binding.PartArtifactIDs,
			"source_word_count":   len(strings.Fields(markdown)) + 1,
			"artifact_sha256":     artifact.SHA256,
			"text":                "장문 리포트 최종 조립 artifact를 결정론적으로 생성했습니다.",
		}),
	}); err != nil {
		t.Fatal(err)
	}

	_, _, err = reporting.EnsureFinalEditAssembly(ctx, svc, "evt_assembly_tampered_metric_replay", writer)
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("tampered source_word_count replay error=%v, want conflict", err)
	}
}

func TestFinalEditWriterStageSubmitsChangedAndNoOpFromAssembly(t *testing.T) {
	for _, tc := range []struct {
		name    string
		changed bool
	}{
		{name: "no_op", changed: false},
		{name: "changed", changed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			svc, closeStore := newLongFormFinalizeFixtureWithFinalEditPipeline(t, ctx, reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2, reporting.FinalEditHumanizeDisabled)
			defer closeStore()
			binding := longFormFinalizeBindingForFinalEditPipeline(reporting.FinalEditHumanizeDisabled)
			assemblyID := reporting.FinalEditAssemblyArtifactID(binding.PlanEventID, binding.PartArtifactIDs)
			writer := longFormFinalEditStageBinding(binding, reporting.FinalEditStageWriter, assemblyID, "art_writer_"+tc.name, "")

			started, created, err := reporting.StartFinalEditStage(ctx, svc, "evt_writer_"+tc.name+"_start", writer)
			if err != nil || !created || started.EventType != reporting.FinalEditWriterStartedEventType {
				t.Fatalf("writer start created=%t event=%#v err=%v", created, started, err)
			}
			source, err := svc.GetRawArtifact(ctx, assemblyID)
			if err != nil {
				t.Fatal(err)
			}
			markdown := string(source.Content)
			operationCount := 0
			wantArtifactID := source.ArtifactID
			if tc.changed {
				markdown = strings.Replace(markdown, "Preserved body.", "Writer-level connective body.", 1)
				operationCount = 1
				wantArtifactID = writer.EditedArtifactID
			}
			submitted, err := reporting.SubmitFinalEditStage(ctx, svc, writer, "evt_writer_"+tc.name+"_submit", markdown, operationCount)
			if err != nil {
				t.Fatal(err)
			}
			if submitted.Changed != tc.changed ||
				submitted.Artifact.ArtifactID != wantArtifactID ||
				submitted.Event.EventType != reporting.FinalEditWriterSubmittedEventType ||
				submitted.OperationCount != operationCount {
				t.Fatalf("writer submission mismatch: %#v", submitted)
			}
			if !tc.changed {
				if _, err := svc.GetRawArtifact(ctx, writer.EditedArtifactID); err == nil {
					t.Fatal("no-op writer created duplicate edited artifact")
				}
			}
			events, err := svc.ListEvents(ctx, binding.MissionID)
			if err != nil {
				t.Fatal(err)
			}
			if countEventType(events, reporting.FinalEditAssemblyCreatedEventType) != 1 ||
				countEventType(events, reporting.FinalEditWriterStartedEventType) != 1 ||
				countEventType(events, reporting.FinalEditWriterSubmittedEventType) != 1 ||
				countEventType(events, "report.artifact.created") != 0 {
				t.Fatalf("writer stage emitted unexpected events: %#v", events)
			}
			payload := map[string]any{}
			if err := json.Unmarshal(submitted.Event.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["final_edit_pipeline"] != reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 ||
				payload["stage"] != reporting.FinalEditStageWriter ||
				payload["stage_id"] != "final-write" ||
				payload["source_artifact_id"] != assemblyID ||
				payload["artifact_id"] != wantArtifactID ||
				payload["edited_artifact_id"] != writer.EditedArtifactID {
				t.Fatalf("writer submitted payload mismatch: %#v", payload)
			}
		})
	}
}
