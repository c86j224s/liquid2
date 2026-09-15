package reporting

import (
	"context"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// LoadFinalEditStageSubmission는 stage 제출 이벤트와 artifact를 장부에서 복원한다.
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
	if err := validateFinalEditStageLineage(ctx, store, events, binding, finalEditStageAllowsCanonicalLoad(binding.Stage)); err != nil {
		return FinalEditStageResult{}, false, err
	}
	event, ok, err := finalEditStageSubmittedEvent(events, binding)
	if err != nil || !ok {
		return FinalEditStageResult{}, ok, err
	}
	result, err := finalEditStageResultFromEvent(ctx, store, binding, event, true)
	return result, err == nil, err
}

func submitFinalEditStageNoOp(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, source artifactcontract.Raw, operationCount int, diagnoses []FinalEditStyleOperationDiagnosis, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) (FinalEditStageResult, error) {
	event, created, err := store.AppendEventConditionally(ctx, binding.MissionID, func(events []ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, finalEditStageAllowsCanonicalLoad(binding.Stage)); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if _, ok, err := finalEditStageStartedEvent(events, binding); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		} else if !ok {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: matching final edit stage start is missing", producterror.ErrConflict)
		}
		if existing, ok, err := finalEditStageSubmittedEvent(events, binding); ok || err != nil {
			return ledger.AppendRequest{}, existing, false, err
		}
		return buildFinalEditSubmittedAppendRequestWithStyleDiagnoses(eventID, binding, source, source, operationCount, false, diagnoses, findings, semanticReview), ledger.Event{}, true, nil
	})
	if err != nil {
		return FinalEditStageResult{}, err
	}
	result, err := finalEditStageResultFromEvent(ctx, store, binding, event, !created)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if err := validateFinalEditStageReplayInput(result, string(source.Content), operationCount, diagnoses, findings, semanticReview); err != nil {
		return FinalEditStageResult{}, err
	}
	return result, nil
}

func submitFinalEditStageExistingArtifact(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, eventID string, source artifactcontract.Raw, artifact artifactcontract.Raw, markdown string, operationCount int, changed bool, diagnoses []FinalEditStyleOperationDiagnosis, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) (FinalEditStageResult, error) {
	event, created, err := store.AppendEventConditionally(ctx, binding.MissionID, func(events []ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error) {
		if err := validateFinalEditStageLineage(ctx, store, events, binding, finalEditStageAllowsCanonicalLoad(binding.Stage)); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if _, ok, err := finalEditStageStartedEvent(events, binding); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		} else if !ok {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: matching final edit stage start is missing", producterror.ErrConflict)
		}
		if existing, ok, err := finalEditStageSubmittedEvent(events, binding); ok || err != nil {
			return ledger.AppendRequest{}, existing, false, err
		}
		return buildFinalEditSubmittedAppendRequestWithStyleDiagnoses(eventID, binding, source, artifact, operationCount, changed, diagnoses, findings, semanticReview), ledger.Event{}, true, nil
	})
	if err != nil {
		return FinalEditStageResult{}, err
	}
	result, err := finalEditStageResultFromEvent(ctx, store, binding, event, !created)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if err := validateFinalEditStageReplayInput(result, markdown, operationCount, diagnoses, findings, semanticReview); err != nil {
		return FinalEditStageResult{}, err
	}
	return result, nil
}

func finalEditStageAllowsCanonicalLoad(stage string) bool {
	return stage == FinalEditStageGate || stage == FinalEditStageEvidenceGate
}

func validateFinalEditStageReplayInput(result FinalEditStageResult, markdown string, operationCount int, diagnoses []FinalEditStyleOperationDiagnosis, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) error {
	if string(result.Artifact.Content) != markdown || result.OperationCount != operationCount || !equalStoredFinalEditGateFindings(result.GateFindings, findings) || !equalStoredFinalEditSemanticAcceptance(result.SemanticReview.Records, semanticReview.Records) || result.SemanticReview.Digest != semanticReview.Digest || result.SemanticReview.Count != semanticReview.Count {
		return fmt.Errorf("%w: final edit stage replay input differs", producterror.ErrConflict)
	}
	if result.StyleOperationDiagnosesPresent && !equalFinalEditStyleOperationDiagnosesForReplay(result.StyleOperationDiagnoses, diagnoses) {
		return fmt.Errorf("%w: final edit stage replay input differs", producterror.ErrConflict)
	}
	return nil
}
