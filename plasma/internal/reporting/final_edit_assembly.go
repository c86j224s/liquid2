package reporting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

const (
	FinalEditAssemblyCreatedEventType = "report.final_assembly.created"
	FinalEditAssemblyKind             = "final_assembly"
	FinalEditAssemblyProducerID       = "reporting_final_assembly"
	FinalEditAssemblySchema           = "plasma.final_assembly.v1"
)

var finalEditAssemblyProducer = ledger.Producer{Type: "system", ID: FinalEditAssemblyProducerID}

// FinalEditAssemblyResult는 최종 조립 artifact와 조립 metadata를 함께 반환한다.
type FinalEditAssemblyResult struct {
	Artifact artifactcontract.Raw
	Event    ledger.Event
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

// FinalEditAssemblyArtifactID는 final edit binding에서 deterministic artifact ID를 만든다.
func FinalEditAssemblyArtifactID(planEventID string, partArtifactIDs []string) string {
	encoded, _ := json.Marshal(finalEditAssemblyIdentity{
		Schema:          FinalEditAssemblySchema,
		PlanEventID:     strings.TrimSpace(planEventID),
		PartArtifactIDs: append([]string(nil), partArtifactIDs...),
	})
	sum := sha256.Sum256(encoded)
	return "art_" + hex.EncodeToString(sum[:])
}

// FinalEditAssemblyIdempotencyKey는 final edit assembly 재실행을 같은 결과로 묶는 key를 만든다.
func FinalEditAssemblyIdempotencyKey(planEventID string, partArtifactIDs []string) string {
	return "report-final-assembly:" + FinalEditAssemblyArtifactID(planEventID, partArtifactIDs)
}

// EnsureFinalEditAssembly는 final edit 결과를 같은 binding 기준의 canonical artifact로 확정한다.
func EnsureFinalEditAssembly(ctx context.Context, store FinalEditStageStore, eventID string, binding FinalEditStageBinding) (FinalEditAssemblyResult, bool, error) {
	binding = normalizeFinalEditStageBinding(binding)
	if err := validateFinalEditStageBinding(binding); err != nil {
		return FinalEditAssemblyResult{}, false, err
	}
	if binding.Stage != FinalEditStageWriter {
		return FinalEditAssemblyResult{}, false, fmt.Errorf("%w: final assembly requires final writer stage binding", producterror.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return FinalEditAssemblyResult{}, false, err
	}
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return FinalEditAssemblyResult{}, false, err
	}
	if !isFinalEditAssemblyPipeline(plan.Pipeline) {
		return FinalEditAssemblyResult{}, false, fmt.Errorf("%w: final assembly requires assembly final edit plan", producterror.ErrConflict)
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
	artifact, event, created, err := store.CreateRawArtifactWithEventConditionally(ctx, artifactReq, func(events []ledger.Event, artifact artifactcontract.Raw) (ledger.AppendRequest, ledger.Event, bool, error) {
		plan, err := finalEditStagePlanForBinding(events, binding)
		if err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if !isFinalEditAssemblyPipeline(plan.Pipeline) {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: final assembly requires assembly final edit plan", producterror.ErrConflict)
		}
		bound := finalEditStageBindingForPlan(binding, plan)
		request, assembly, err := finalEditAssemblyRequest(ctx, store, events, bound)
		if err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if err := validateFinalEditAssemblyArtifact(artifact, request); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if existing, ok, err := finalEditAssemblyCreatedEvent(events, bound, request, assembly.PartArtifactIDs); ok || err != nil {
			return ledger.AppendRequest{}, existing, false, err
		}
		return buildFinalEditAssemblyCreatedAppendRequest(strings.TrimSpace(eventID), bound, artifact, assembly), ledger.Event{}, true, nil
	})
	if err != nil {
		return FinalEditAssemblyResult{}, false, err
	}
	if err := validateFinalEditAssemblyArtifact(artifact, artifactReq); err != nil {
		return FinalEditAssemblyResult{}, false, err
	}
	return FinalEditAssemblyResult{Artifact: artifact, Event: event, Replay: !created}, created, nil
}

func finalEditAssemblyRequest(ctx context.Context, store FinalEditStageStore, events []ledger.Event, binding FinalEditStageBinding) (artifactcontract.CreateRequest, finalEditPartAssembly, error) {
	assembly, err := finalEditPartAssemblyForBinding(ctx, store, events, binding)
	if err != nil {
		return artifactcontract.CreateRequest{}, finalEditPartAssembly{}, err
	}
	artifactID := FinalEditAssemblyArtifactID(binding.PlanEventID, assembly.PartArtifactIDs)
	if binding.SourceArtifactID != artifactID {
		return artifactcontract.CreateRequest{}, finalEditPartAssembly{}, fmt.Errorf("%w: final assembly artifact id differs from deterministic contract", producterror.ErrConflict)
	}
	return artifactcontract.CreateRequest{
		ArtifactID: artifactID,
		MissionID:  binding.MissionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   binding.Filename,
		Producer:   finalEditAssemblyProducer,
		Content:    []byte(assembly.Markdown),
	}, assembly, nil
}

func finalEditPartAssemblyForBinding(ctx context.Context, store FinalEditStageStore, events []ledger.Event, binding FinalEditStageBinding) (finalEditPartAssembly, error) {
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

func buildFinalEditAssemblyCreatedAppendRequest(eventID string, binding FinalEditStageBinding, artifact artifactcontract.Raw, assembly finalEditPartAssembly) ledger.AppendRequest {
	payload := map[string]any{
		"kind":                FinalEditAssemblyKind,
		"schema":              FinalEditAssemblySchema,
		"pending_event_id":    binding.PendingEventID,
		"plan_event_id":       binding.PlanEventID,
		"final_edit_pipeline": binding.FinalEditPipeline,
		"title":               binding.Title,
		"artifact_id":         artifact.ArtifactID,
		"filename":            binding.Filename,
		"producer_id":         FinalEditAssemblyProducerID,
		"part_artifact_ids":   append([]string(nil), assembly.PartArtifactIDs...),
		"source_word_count":   len(strings.Fields(assembly.Markdown)),
		"artifact_sha256":     artifact.SHA256,
		"text":                "장문 리포트 최종 조립 artifact를 결정론적으로 생성했습니다.",
	}
	return ledger.AppendRequest{
		EventID:          strings.TrimSpace(eventID),
		MissionID:        binding.MissionID,
		EventType:        FinalEditAssemblyCreatedEventType,
		Producer:         finalEditAssemblyProducer,
		CausationEventID: binding.PlanEventID,
		CorrelationID:    FinalEditAssemblyIdempotencyKey(binding.PlanEventID, assembly.PartArtifactIDs),
		Payload:          mustJSON(payload),
	}
}
