package reporting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	FinalEditAssemblyCreatedEventType = "report.final_assembly.created"
	FinalEditAssemblyKind             = "final_assembly"
	FinalEditAssemblyProducerID       = "reporting_final_assembly"
	FinalEditAssemblySchema           = "plasma.final_assembly.v1"
)

var finalEditAssemblyProducer = app.Producer{Type: "system", ID: FinalEditAssemblyProducerID}

type FinalEditAssemblyResult struct {
	Artifact app.RawArtifact
	Event    app.LedgerEvent
	Replay   bool
}

type finalEditAssemblyIdentity struct {
	Schema          string   `json:"schema"`
	PlanEventID     string   `json:"plan_event_id"`
	PartArtifactIDs []string `json:"part_artifact_ids"`
}

type finalEditPartAssembly struct {
	Markdown        string
	PartArtifactIDs []string
}

func FinalEditAssemblyArtifactID(planEventID string, partArtifactIDs []string) string {
	encoded, _ := json.Marshal(finalEditAssemblyIdentity{
		Schema:          FinalEditAssemblySchema,
		PlanEventID:     strings.TrimSpace(planEventID),
		PartArtifactIDs: append([]string(nil), partArtifactIDs...),
	})
	sum := sha256.Sum256(encoded)
	return "art_" + hex.EncodeToString(sum[:])
}

func FinalEditAssemblyIdempotencyKey(planEventID string, partArtifactIDs []string) string {
	return "report-final-assembly:" + FinalEditAssemblyArtifactID(planEventID, partArtifactIDs)
}

func EnsureFinalEditAssembly(ctx context.Context, store FinalEditStageStore, eventID string, binding FinalEditStageBinding) (FinalEditAssemblyResult, bool, error) {
	binding = normalizeFinalEditStageBinding(binding)
	if err := validateFinalEditStageBinding(binding); err != nil {
		return FinalEditAssemblyResult{}, false, err
	}
	if binding.Stage != FinalEditStageWriter {
		return FinalEditAssemblyResult{}, false, fmt.Errorf("%w: final assembly requires final writer stage binding", app.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return FinalEditAssemblyResult{}, false, err
	}
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return FinalEditAssemblyResult{}, false, err
	}
	if plan.Pipeline != FinalEditPipelineAssemblyWriterReaderStyleGateV2 {
		return FinalEditAssemblyResult{}, false, fmt.Errorf("%w: final assembly requires assembly_writer_reader_style_gate_v2 plan", app.ErrConflict)
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	artifactReq, assembly, err := finalEditAssemblyRequest(ctx, store, events, binding)
	if err != nil {
		return FinalEditAssemblyResult{}, false, err
	}
	if existing, ok, err := finalEditAssemblyCreatedEvent(events, binding, artifactReq, assembly.PartArtifactIDs); err != nil || ok {
		if err != nil {
			return FinalEditAssemblyResult{}, false, err
		}
		artifact, err := store.GetRawArtifact(ctx, artifactReq.ArtifactID)
		if err != nil {
			return FinalEditAssemblyResult{}, false, err
		}
		if err := validateFinalEditAssemblyArtifact(artifact, artifactReq); err != nil {
			return FinalEditAssemblyResult{}, false, err
		}
		return FinalEditAssemblyResult{Artifact: artifact, Event: existing, Replay: true}, false, nil
	}
	artifact, event, created, err := store.CreateRawArtifactWithEventConditionally(ctx, artifactReq, func(events []app.LedgerEvent, artifact app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		plan, err := finalEditStagePlanForBinding(events, binding)
		if err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if plan.Pipeline != FinalEditPipelineAssemblyWriterReaderStyleGateV2 {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: final assembly requires assembly_writer_reader_style_gate_v2 plan", app.ErrConflict)
		}
		bound := finalEditStageBindingForPlan(binding, plan)
		request, assembly, err := finalEditAssemblyRequest(ctx, store, events, bound)
		if err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if err := validateFinalEditAssemblyArtifact(artifact, request); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if existing, ok, err := finalEditAssemblyCreatedEvent(events, bound, request, assembly.PartArtifactIDs); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return buildFinalEditAssemblyCreatedAppendRequest(strings.TrimSpace(eventID), bound, artifact, assembly), app.LedgerEvent{}, true, nil
	})
	if err != nil {
		return FinalEditAssemblyResult{}, false, err
	}
	if err := validateFinalEditAssemblyArtifact(artifact, artifactReq); err != nil {
		return FinalEditAssemblyResult{}, false, err
	}
	return FinalEditAssemblyResult{Artifact: artifact, Event: event, Replay: !created}, created, nil
}

func finalEditAssemblyRequest(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, binding FinalEditStageBinding) (app.CreateRawArtifactRequest, finalEditPartAssembly, error) {
	assembly, err := finalEditPartAssemblyForBinding(ctx, store, events, binding)
	if err != nil {
		return app.CreateRawArtifactRequest{}, finalEditPartAssembly{}, err
	}
	artifactID := FinalEditAssemblyArtifactID(binding.PlanEventID, assembly.PartArtifactIDs)
	if binding.SourceArtifactID != artifactID {
		return app.CreateRawArtifactRequest{}, finalEditPartAssembly{}, fmt.Errorf("%w: final assembly artifact id differs from deterministic contract", app.ErrConflict)
	}
	return app.CreateRawArtifactRequest{
		ArtifactID: artifactID,
		MissionID:  binding.MissionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   binding.Filename,
		Producer:   finalEditAssemblyProducer,
		Content:    []byte(assembly.Markdown),
	}, assembly, nil
}

func finalEditPartAssemblyForBinding(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, binding FinalEditStageBinding) (finalEditPartAssembly, error) {
	parts, err := orderedLongFormPartArtifactsForFinalEdit(ctx, store, events, binding)
	if err != nil {
		return finalEditPartAssembly{}, err
	}
	ids := make([]string, 0, len(parts))
	markdownParts := make([]string, 0, len(parts))
	for _, part := range parts {
		ids = append(ids, part.ArtifactID)
		markdownParts = append(markdownParts, string(part.Content))
	}
	return finalEditPartAssembly{
		Markdown:        AssembleLongFormFinalMarkdown(binding.Title, "", "", markdownParts),
		PartArtifactIDs: ids,
	}, nil
}

func buildFinalEditAssemblyCreatedAppendRequest(eventID string, binding FinalEditStageBinding, artifact app.RawArtifact, assembly finalEditPartAssembly) app.AppendEventRequest {
	payload := map[string]any{
		"kind":                FinalEditAssemblyKind,
		"schema":              FinalEditAssemblySchema,
		"pending_event_id":    binding.PendingEventID,
		"plan_event_id":       binding.PlanEventID,
		"final_edit_pipeline": FinalEditPipelineAssemblyWriterReaderStyleGateV2,
		"title":               binding.Title,
		"artifact_id":         artifact.ArtifactID,
		"filename":            binding.Filename,
		"producer_id":         FinalEditAssemblyProducerID,
		"part_artifact_ids":   append([]string(nil), assembly.PartArtifactIDs...),
		"source_word_count":   len(strings.Fields(assembly.Markdown)),
		"artifact_sha256":     artifact.SHA256,
		"text":                "장문 리포트 최종 조립 artifact를 결정론적으로 생성했습니다.",
	}
	return app.AppendEventRequest{
		EventID:          strings.TrimSpace(eventID),
		MissionID:        binding.MissionID,
		EventType:        FinalEditAssemblyCreatedEventType,
		Producer:         finalEditAssemblyProducer,
		CausationEventID: binding.PlanEventID,
		CorrelationID:    FinalEditAssemblyIdempotencyKey(binding.PlanEventID, assembly.PartArtifactIDs),
		Payload:          mustJSON(payload),
	}
}

func finalEditAssemblyCreatedEvent(events []app.LedgerEvent, binding FinalEditStageBinding, request app.CreateRawArtifactRequest, partArtifactIDs []string) (app.LedgerEvent, bool, error) {
	key := FinalEditAssemblyIdempotencyKey(binding.PlanEventID, partArtifactIDs)
	var found app.LedgerEvent
	count := 0
	for _, event := range events {
		if event.EventType != FinalEditAssemblyCreatedEventType || event.CorrelationID != key {
			continue
		}
		if err := validateFinalEditAssemblyCreatedEvent(events, event, binding, request, partArtifactIDs); err != nil {
			return app.LedgerEvent{}, false, err
		}
		found, count = event, count+1
	}
	if count > 1 {
		return app.LedgerEvent{}, false, fmt.Errorf("%w: multiple final assembly events match binding", app.ErrConflict)
	}
	return found, count == 1, nil
}

func validateFinalEditAssemblyCreatedEvent(events []app.LedgerEvent, event app.LedgerEvent, binding FinalEditStageBinding, request app.CreateRawArtifactRequest, partArtifactIDs []string) error {
	acceptedPending, err := longFormPendingLineage(events, binding.PendingEventID)
	if err != nil {
		return err
	}
	payload := eventPayload(event)
	payloadParts, err := stringSlicePayload(payload["part_artifact_ids"])
	if err != nil {
		return err
	}
	sourceWordCount, ok := payloadIntStrict(payload, "source_word_count")
	expectedSourceWordCount := len(strings.Fields(string(request.Content)))
	if event.MissionID != binding.MissionID ||
		event.Producer != finalEditAssemblyProducer ||
		event.CausationEventID != binding.PlanEventID ||
		payloadString(payload, "kind") != FinalEditAssemblyKind ||
		payloadString(payload, "schema") != FinalEditAssemblySchema ||
		!acceptedPending[payloadString(payload, "pending_event_id")] ||
		payloadString(payload, "plan_event_id") != binding.PlanEventID ||
		payloadString(payload, "final_edit_pipeline") != FinalEditPipelineAssemblyWriterReaderStyleGateV2 ||
		payloadString(payload, "title") != binding.Title ||
		payloadString(payload, "artifact_id") != request.ArtifactID ||
		payloadString(payload, "filename") != binding.Filename ||
		payloadString(payload, "producer_id") != FinalEditAssemblyProducerID ||
		payloadString(payload, "artifact_sha256") != contentSHA256(request.Content) ||
		!ok || sourceWordCount != expectedSourceWordCount ||
		!equalStrings(payloadParts, partArtifactIDs) {
		return fmt.Errorf("%w: final assembly event differs from deterministic contract", app.ErrConflict)
	}
	return nil
}

func validateFinalEditAssemblyArtifact(artifact app.RawArtifact, request app.CreateRawArtifactRequest) error {
	expectedSHA := contentSHA256(request.Content)
	if artifact.ArtifactID != request.ArtifactID ||
		artifact.MissionID != request.MissionID ||
		artifact.MediaType != request.MediaType ||
		artifact.Filename != request.Filename ||
		artifact.Producer != request.Producer ||
		artifact.SHA256 != expectedSHA {
		return fmt.Errorf("%w: existing final assembly artifact differs from deterministic contract", app.ErrConflict)
	}
	return nil
}
