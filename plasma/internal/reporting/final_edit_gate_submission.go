package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type FinalEditGateSubmitRequest struct {
	StageBinding       FinalEditStageBinding
	FinalBinding       LongFormFinalizeBinding
	StageEventID       string
	CanonicalEventID   string
	ManuscriptMarkdown string
	OperationCount     int
	Findings           []FinalEditGateFinding
}

func SubmitFinalEditGate(ctx context.Context, store FinalEditStageStore, req FinalEditGateSubmitRequest) (LongFormFinalizeResult, error) {
	stageBinding := normalizeFinalEditStageBinding(req.StageBinding)
	finalBinding := normalizeLongFormFinalizeBinding(req.FinalBinding)
	findings, err := NormalizeFinalEditGateFindings(ctx, store, finalBinding.MissionID, req.Findings)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	plan, err := validateFinalEditGateSubmitRequest(ctx, store, stageBinding, finalBinding, req.ManuscriptMarkdown, req.OperationCount, findings)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	stage, err := submitFinalEditStage(ctx, store, stageBinding, req.StageEventID, req.ManuscriptMarkdown, req.OperationCount, findings)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	return finalizeLongForm(ctx, store, LongFormFinalizeRequest{
		Binding:                   finalBinding,
		EventID:                   strings.TrimSpace(req.CanonicalEventID),
		ManuscriptMarkdown:        string(stage.Artifact.Content),
		FinalEditPipeline:         plan.Pipeline,
		GateFindings:              stage.GateFindings,
		FinalEditActualArtifactID: stage.Artifact.ArtifactID,
		FinalEditGateEventID:      stage.Event.EventID,
		FinalEditGateChanged:      stage.Changed,
	}, true)
}

func validateFinalEditGateSubmitRequest(ctx context.Context, store FinalEditStageStore, stageBinding FinalEditStageBinding, finalBinding LongFormFinalizeBinding, markdown string, operationCount int, findings []StoredFinalEditGateFinding) (FinalEditPipelinePlanState, error) {
	if err := validateFinalEditStageBinding(stageBinding); err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	if err := validateLongFormFinalizeBinding(finalBinding); err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	if stageBinding.Stage != FinalEditStageGate {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: final edit gate submit requires corrective gate stage", app.ErrInvalidInput)
	}
	if strings.TrimSpace(markdown) == "" || operationCount < 0 {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: final edit gate Markdown is invalid", app.ErrInvalidInput)
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
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: final edit gate requires active final edit plan", app.ErrConflict)
	}
	stageBinding = finalEditStageBindingForPlan(stageBinding, plan)
	if err := validateLongFormFinalPipeline(events, finalBinding, true); err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	canonical, canonicalCount := longFormCanonical(events, finalBinding.PendingEventID)
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
	if _, ok, err := finalEditStageStartedEvent(events, stageBinding); err != nil {
		return FinalEditPipelinePlanState{}, err
	} else if !ok {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: matching corrective gate start is missing", app.ErrConflict)
	}
	source, err := store.GetRawArtifact(ctx, stageBinding.SourceArtifactID)
	if err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	if source.MissionID != stageBinding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.SHA256 != contentSHA256(source.Content) {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: corrective gate source artifact is foreign or not Markdown", app.ErrConflict)
	}
	if submitted, ok, err := finalEditStageSubmittedEvent(events, stageBinding); err != nil {
		return FinalEditPipelinePlanState{}, err
	} else if ok {
		result, err := finalEditStageResultFromEvent(ctx, store, stageBinding, submitted, true)
		if err != nil {
			return FinalEditPipelinePlanState{}, err
		}
		if err := validateFinalEditStageReplayInput(result, markdown, operationCount, findings); err != nil {
			return FinalEditPipelinePlanState{}, err
		}
	}
	if canonicalCount == 1 {
		if err := validateLongFormCanonicalGatePayload(ctx, store, events, finalBinding, eventPayload(canonical)); err != nil {
			return FinalEditPipelinePlanState{}, err
		}
		if err := validateFinalEditCanonicalReplayPayload(eventPayload(canonical), plan.Pipeline, findings); err != nil {
			return FinalEditPipelinePlanState{}, err
		}
	}
	return plan, nil
}

func validateFinalEditGateBindingMatchesFinal(stageBinding FinalEditStageBinding, finalBinding LongFormFinalizeBinding, plan FinalEditPipelinePlanState) error {
	if err := ValidateFinalEditGateBindingsCompatible(stageBinding, finalBinding); err != nil {
		return err
	}
	if finalBinding.ArtifactID != plan.ArtifactID {
		return fmt.Errorf("%w: corrective gate binding differs from final binding", app.ErrConflict)
	}
	return nil
}

func submitFinalEditStageChanged(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, source app.RawArtifact, markdown string, operationCount int, findings []StoredFinalEditGateFinding) (FinalEditStageResult, error) {
	producer := app.Producer{Type: "agent_session", ID: binding.ProviderSessionID}
	artifactReq := app.CreateRawArtifactRequest{
		ArtifactID: binding.EditedArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: producer, Content: []byte(markdown),
	}
	artifact, event, created, err := store.CreateRawArtifactWithEventConditionally(ctx, artifactReq, func(events []app.LedgerEvent, artifact app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error) {
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
		return buildFinalEditSubmittedAppendRequest(eventID, binding, source, artifact, operationCount, true, findings), app.LedgerEvent{}, true, nil
	})
	if err != nil {
		if existing, ok, loadErr := LoadFinalEditStageSubmission(ctx, store, binding); ok && loadErr == nil {
			if replayErr := validateFinalEditStageReplayInput(existing, markdown, operationCount, findings); replayErr != nil {
				return FinalEditStageResult{}, replayErr
			}
			return existing, nil
		}
		return FinalEditStageResult{}, err
	}
	if created {
		return FinalEditStageResult{Artifact: artifact, Event: event, OperationCount: operationCount, Changed: true, GateFindings: append([]StoredFinalEditGateFinding(nil), findings...)}, nil
	}
	result, err := finalEditStageResultFromEvent(ctx, store, binding, event, true)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if err := validateFinalEditStageReplayInput(result, markdown, operationCount, findings); err != nil {
		return FinalEditStageResult{}, err
	}
	return result, nil
}
