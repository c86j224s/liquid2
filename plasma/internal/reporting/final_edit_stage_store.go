package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type FinalEditStageStore interface {
	LongFormFinalizationStore
	GetEvidenceRecord(context.Context, string) (app.EvidenceRecord, error)
}

func StartFinalEditStage(ctx context.Context, store FinalEditStageStore, eventID string, binding FinalEditStageBinding) (app.LedgerEvent, bool, error) {
	binding = normalizeFinalEditStageBinding(binding)
	if err := validateFinalEditStageBinding(binding); err != nil {
		return app.LedgerEvent{}, false, err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	if binding.Stage == FinalEditStageWriter {
		return startFinalEditWriterStage(ctx, store, eventID, binding)
	}
	if binding.Stage == FinalEditStageReader && plan.Pipeline == FinalEditPipelineReaderStyleGateV1 {
		return startFinalEditReaderStage(ctx, store, eventID, binding)
	}
	if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
		return app.LedgerEvent{}, false, err
	}
	return store.AppendEventConditionally(ctx, binding.MissionID, func(events []app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if existing, ok, err := finalEditStageStartedEvent(events, binding); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return BuildFinalEditStageStartedAppendRequest(eventID, binding), app.LedgerEvent{}, true, nil
	})
}

func startFinalEditWriterStage(ctx context.Context, store FinalEditStageStore, eventID string, binding FinalEditStageBinding) (app.LedgerEvent, bool, error) {
	if _, _, err := EnsureFinalEditAssembly(ctx, store, newFinalEditAssemblyEventID(eventID), binding); err != nil {
		return app.LedgerEvent{}, false, err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
		return app.LedgerEvent{}, false, err
	}
	return store.AppendEventConditionally(ctx, binding.MissionID, func(events []app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if existing, ok, err := finalEditStageStartedEvent(events, binding); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return BuildFinalEditStageStartedAppendRequest(eventID, binding), app.LedgerEvent{}, true, nil
	})
}

func newFinalEditAssemblyEventID(stageEventID string) string {
	stageEventID = strings.TrimSpace(stageEventID)
	if strings.HasPrefix(stageEventID, "evt_") {
		return "evt_final_assembly_" + strings.TrimPrefix(stageEventID, "evt_")
	}
	return stageEventID
}

func startFinalEditReaderStage(ctx context.Context, store FinalEditStageStore, eventID string, binding FinalEditStageBinding) (app.LedgerEvent, bool, error) {
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
		return app.LedgerEvent{}, false, err
	}
	artifactReq, err := finalEditReaderSourceRequest(ctx, store, events, binding)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	if existing, err := store.GetRawArtifact(ctx, artifactReq.ArtifactID); err == nil {
		if err := validateFinalEditReaderSourceArtifact(existing, artifactReq); err != nil {
			return app.LedgerEvent{}, false, err
		}
	}
	artifact, event, created, err := store.CreateRawArtifactWithEventConditionally(ctx, artifactReq, func(events []app.LedgerEvent, artifact app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if err := validateFinalEditReaderSourceArtifact(artifact, artifactReq); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if existing, ok, err := finalEditStageStartedEvent(events, binding); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return BuildFinalEditStageStartedAppendRequest(eventID, binding), app.LedgerEvent{}, true, nil
	})
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	if err := validateFinalEditReaderSourceArtifact(artifact, artifactReq); err != nil {
		return app.LedgerEvent{}, false, err
	}
	return event, created, nil
}

func SubmitFinalEditStage(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, markdown string, operationCount int) (FinalEditStageResult, error) {
	if normalizeFinalEditStageBinding(binding).Stage == FinalEditStageGate {
		return FinalEditStageResult{}, fmt.Errorf("%w: corrective gate completion must use SubmitFinalEditGate", app.ErrInvalidInput)
	}
	return submitFinalEditStage(ctx, store, binding, eventID, markdown, operationCount, nil)
}

func LoadFinalEditStageSubmission(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding) (FinalEditStageResult, bool, error) {
	binding = normalizeFinalEditStageBinding(binding)
	if err := validateFinalEditStageBinding(binding); err != nil {
		return FinalEditStageResult{}, false, err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return FinalEditStageResult{}, false, err
	}
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return FinalEditStageResult{}, false, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	if err := validateFinalEditStageLineage(ctx, store, events, binding, binding.Stage == FinalEditStageGate); err != nil {
		return FinalEditStageResult{}, false, err
	}
	event, ok, err := finalEditStageSubmittedEvent(events, binding)
	if err != nil || !ok {
		return FinalEditStageResult{}, ok, err
	}
	result, err := finalEditStageResultFromEvent(ctx, store, binding, event, true)
	return result, err == nil, err
}

func submitFinalEditStage(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, markdown string, operationCount int, findings []StoredFinalEditGateFinding) (FinalEditStageResult, error) {
	binding = normalizeFinalEditStageBinding(binding)
	if err := validateFinalEditStageBinding(binding); err != nil {
		return FinalEditStageResult{}, err
	}
	if strings.TrimSpace(markdown) == "" || operationCount < 0 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage Markdown is invalid", app.ErrInvalidInput)
	}
	if existing, ok, err := LoadFinalEditStageSubmission(ctx, store, binding); err != nil {
		return FinalEditStageResult{}, err
	} else if ok {
		if err := validateFinalEditStageReplayInput(existing, markdown, operationCount, findings); err != nil {
			return FinalEditStageResult{}, err
		}
		return existing, nil
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	if err := validateFinalEditStageLineage(ctx, store, events, binding, binding.Stage == FinalEditStageGate); err != nil {
		return FinalEditStageResult{}, err
	}
	if _, ok, err := finalEditStageStartedEvent(events, binding); err != nil {
		return FinalEditStageResult{}, err
	} else if !ok {
		return FinalEditStageResult{}, fmt.Errorf("%w: matching final edit stage start is missing", app.ErrConflict)
	}
	source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.SHA256 != contentSHA256(source.Content) {
		return FinalEditStageResult{}, fmt.Errorf("%w: source final edit artifact is foreign or not Markdown", app.ErrConflict)
	}
	if string(source.Content) == markdown {
		return submitFinalEditStageNoOp(ctx, store, binding, eventID, source, operationCount, findings)
	}
	return submitFinalEditStageChanged(ctx, store, binding, eventID, source, markdown, operationCount, findings)
}

func submitFinalEditStageNoOp(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, source app.RawArtifact, operationCount int, findings []StoredFinalEditGateFinding) (FinalEditStageResult, error) {
	event, created, err := store.AppendEventConditionally(ctx, binding.MissionID, func(events []app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, binding.Stage == FinalEditStageGate); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if _, ok, err := finalEditStageStartedEvent(events, binding); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		} else if !ok {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: matching final edit stage start is missing", app.ErrConflict)
		}
		if existing, ok, err := finalEditStageSubmittedEvent(events, binding); ok || err != nil {
			return app.AppendEventRequest{}, existing, false, err
		}
		return buildFinalEditSubmittedAppendRequest(eventID, binding, source, source, operationCount, false, findings), app.LedgerEvent{}, true, nil
	})
	if err != nil {
		return FinalEditStageResult{}, err
	}
	result, err := finalEditStageResultFromEvent(ctx, store, binding, event, !created)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if err := validateFinalEditStageReplayInput(result, string(source.Content), operationCount, findings); err != nil {
		return FinalEditStageResult{}, err
	}
	return result, nil
}

func validateFinalEditStageReplayInput(result FinalEditStageResult, markdown string, operationCount int, findings []StoredFinalEditGateFinding) error {
	if string(result.Artifact.Content) != markdown || result.OperationCount != operationCount || !equalStoredFinalEditGateFindings(result.GateFindings, findings) {
		return fmt.Errorf("%w: final edit stage replay input differs", app.ErrConflict)
	}
	return nil
}
