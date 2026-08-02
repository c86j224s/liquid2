package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type FinalEditGateResumeRequest struct {
	StageBinding     FinalEditStageBinding
	FinalBinding     LongFormFinalizeBinding
	CanonicalEventID string
}

func LoadCurrentFinalEditStageStart(ctx context.Context, store FinalEditStageStore, contract FinalEditStageStartContract) (FinalEditStageStartResult, bool, error) {
	finalBinding := normalizeLongFormFinalizeBinding(contract.FinalBinding)
	if err := validateLongFormFinalizeBinding(finalBinding); err != nil {
		return FinalEditStageStartResult{}, false, err
	}
	stage := normalizeFinalEditStageBinding(FinalEditStageBinding{Stage: contract.Stage}).Stage
	if finalEditStartedEventType(stage) == "" {
		return FinalEditStageStartResult{}, false, fmt.Errorf("%w: unsupported final edit stage", app.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, finalBinding.MissionID)
	if err != nil {
		return FinalEditStageStartResult{}, false, err
	}
	acceptedPending, err := longFormPendingLineage(events, finalBinding.PendingEventID)
	if err != nil {
		return FinalEditStageStartResult{}, false, err
	}
	plan, ok, err := longFormPlanPipeline(events, finalBinding.PendingEventID, finalBinding.PlanEventID)
	if err != nil {
		return FinalEditStageStartResult{}, false, err
	}
	if !ok || !isSupportedFinalEditPipeline(plan.Pipeline) {
		return FinalEditStageStartResult{}, false, fmt.Errorf("%w: final edit plan is not active", app.ErrConflict)
	}
	var found FinalEditStageStartResult
	count := 0
	for _, event := range events {
		if event.EventType != finalEditStartedEventType(stage) ||
			event.MissionID != finalBinding.MissionID ||
			!acceptedPending[payloadString(eventPayload(event), "pending_event_id")] {
			continue
		}
		binding, ok := finalEditStageBindingFromStartEventForPipeline(event, plan.Pipeline)
		if !ok {
			return FinalEditStageStartResult{}, false, fmt.Errorf("%w: stored final edit start is invalid", app.ErrConflict)
		}
		if binding.PendingEventID != finalBinding.PendingEventID ||
			binding.PlanEventID != finalBinding.PlanEventID ||
			binding.Stage != stage {
			continue
		}
		if err := validateFinalEditStageLineage(ctx, store, events, binding, false); err != nil {
			return FinalEditStageStartResult{}, false, err
		}
		if err := finalEditStageStartMatchesFinalBinding(binding, finalBinding, plan); err != nil {
			return FinalEditStageStartResult{}, false, err
		}
		if submitted, ok, err := finalEditStageSubmittedEvent(events, binding); err != nil {
			return FinalEditStageStartResult{}, false, err
		} else if ok {
			if _, err := finalEditStageResultFromEvent(ctx, store, binding, submitted, true); err != nil {
				return FinalEditStageStartResult{}, false, err
			}
			continue
		}
		source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
		if err != nil {
			return FinalEditStageStartResult{}, false, err
		}
		if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.Filename != binding.Filename || source.SHA256 != contentSHA256(source.Content) {
			return FinalEditStageStartResult{}, false, fmt.Errorf("%w: final edit start source artifact is invalid", app.ErrConflict)
		}
		found = FinalEditStageStartResult{Binding: binding, SourceArtifact: source, Event: event}
		count++
	}
	if count > 1 {
		return FinalEditStageStartResult{}, false, fmt.Errorf("%w: multiple open final edit starts match current pending", app.ErrConflict)
	}
	return found, count == 1, nil
}

func ResumeFinalEditGate(ctx context.Context, store FinalEditStageStore, req FinalEditGateResumeRequest) (LongFormFinalizeResult, error) {
	stageBinding := normalizeFinalEditStageBinding(req.StageBinding)
	finalBinding := normalizeLongFormFinalizeBinding(req.FinalBinding)
	if strings.TrimSpace(req.CanonicalEventID) == "" {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: canonical event id is required", app.ErrInvalidInput)
	}
	plan, err := validateFinalEditGateResumeRequest(ctx, store, stageBinding, finalBinding)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	stage, ok, err := LoadFinalEditStageSubmission(ctx, store, stageBinding)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if !ok {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: final edit gate submission is missing", app.ErrConflict)
	}
	return finalizeLongForm(ctx, store, LongFormFinalizeRequest{
		Binding:                   finalBinding,
		EventID:                   strings.TrimSpace(req.CanonicalEventID),
		ManuscriptMarkdown:        string(stage.Artifact.Content),
		FinalEditPipeline:         plan.Pipeline,
		GateFindings:              stage.GateFindings,
		SemanticReview:            stage.SemanticReview,
		FinalEditActualArtifactID: stage.Artifact.ArtifactID,
		FinalEditGateEventID:      stage.Event.EventID,
		FinalEditGateChanged:      stage.Changed,
	}, true)
}

func validateFinalEditGateResumeRequest(ctx context.Context, store FinalEditStageStore, stageBinding FinalEditStageBinding, finalBinding LongFormFinalizeBinding) (FinalEditPipelinePlanState, error) {
	if err := validateFinalEditStageBinding(stageBinding); err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	if err := validateLongFormFinalizeBinding(finalBinding); err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	if stageBinding.Stage != FinalEditStageGate && stageBinding.Stage != FinalEditStageEvidenceGate {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: final edit gate resume requires a gate stage", app.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, finalBinding.MissionID)
	if err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	plan, ok, err := longFormPlanPipeline(events, finalBinding.PendingEventID, finalBinding.PlanEventID)
	if err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	if !ok || !isSupportedFinalEditPipeline(plan.Pipeline) {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: final edit gate resume requires active final edit plan", app.ErrConflict)
	}
	if stageBinding.Stage == FinalEditStageEvidenceGate && plan.Pipeline != FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: evidence gate resume requires active v3 final edit plan", app.ErrConflict)
	}
	if stageBinding.Stage == FinalEditStageGate && plan.Pipeline == FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: v3 final edit resume requires evidence gate stage", app.ErrConflict)
	}
	stageBinding = finalEditStageBindingForPlan(stageBinding, plan)
	if err := validateLongFormFinalPipeline(events, finalBinding, true); err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	_, canonicalCount := longFormCanonical(events, finalBinding.PendingEventID)
	if canonicalCount > 1 {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: multiple canonical long-form finalizations", app.ErrConflict)
	}
	if canonicalCount == 0 {
		if err := validateLongFormLineage(ctx, store, events, finalBinding); err != nil {
			return FinalEditPipelinePlanState{}, err
		}
	}
	if err := validateFinalEditGateBindingMatchesFinal(stageBinding, finalBinding, plan); err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	if err := validateFinalEditStageLineage(ctx, store, events, stageBinding, true); err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	return plan, nil
}

func appendLongFormCanonicalForExistingArtifact(ctx context.Context, store LongFormFinalizationStore, req LongFormFinalizeRequest, binding LongFormFinalizeBinding, artifact app.RawArtifact, markdown string, allowReaderStyleGate bool) (LongFormFinalizeResult, error) {
	if artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" || artifact.Filename != binding.Filename || artifact.SHA256 != contentSHA256([]byte(markdown)) || (artifact.ArtifactID == binding.ArtifactID && artifact.Producer != binding.Producer) {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: existing final artifact differs from binding", app.ErrConflict)
	}
	if artifact.ArtifactID != binding.ArtifactID && (!isSupportedFinalEditPipeline(strings.TrimSpace(req.FinalEditPipeline)) || req.FinalEditGateChanged || artifact.ArtifactID != strings.TrimSpace(req.FinalEditActualArtifactID)) {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: existing final artifact differs from binding", app.ErrConflict)
	}
	event, created, err := store.AppendEventConditionally(ctx, binding.MissionID, func(events []app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
		existing, count := longFormCanonical(events, binding.PendingEventID)
		if count > 1 {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: multiple canonical long-form finalizations", app.ErrConflict)
		}
		if count == 1 {
			payload := eventPayload(existing)
			canonicalMatches := canonicalMatchesBinding(existing, payload, binding)
			if isSupportedFinalEditPipeline(payloadString(payload, "final_edit_pipeline")) {
				canonicalMatches = canonicalMatchesBindingForActualArtifact(existing, payload, binding, payloadString(payload, "artifact_id"), true)
			}
			if existing.CorrelationID != binding.IdempotencyKey || !canonicalMatches {
				return app.AppendEventRequest{}, app.LedgerEvent{}, false, fmt.Errorf("%w: canonical long-form finalization binding differs", app.ErrConflict)
			}
			if err := validateLongFormCanonicalReplayRequest(ctx, store, events, binding, existing, req); err != nil {
				return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
			}
			return app.AppendEventRequest{}, existing, false, nil
		}
		if err := validateLongFormLineage(ctx, store, events, binding); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		if err := validateLongFormFinalPipeline(events, binding, allowReaderStyleGate); err != nil {
			return app.AppendEventRequest{}, app.LedgerEvent{}, false, err
		}
		return longFormCanonicalRequestForFinalEdit(strings.TrimSpace(req.EventID), binding, artifact, len(strings.Fields(markdown)), req), app.LedgerEvent{}, true, nil
	})
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if created {
		return LongFormFinalizeResult{Artifact: artifact, Event: event}, nil
	}
	return replayLongFormFinalizeForRequest(ctx, store, binding, event, req)
}
