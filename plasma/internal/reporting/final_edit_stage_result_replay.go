package reporting

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

func finalEditStageStartedEvent(events []ledger.Event, binding FinalEditStageBinding) (ledger.Event, bool, error) {
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return ledger.Event{}, false, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	var found ledger.Event
	count := 0
	for _, event := range events {
		if event.EventType != finalEditStartedEventType(binding.Stage) || event.CorrelationID != binding.IdempotencyKey {
			continue
		}
		if !finalEditStartedEventMatchesPipeline(event, binding, plan.Pipeline) {
			return ledger.Event{}, false, fmt.Errorf("%w: final edit stage start binding differs", producterror.ErrConflict)
		}
		found, count = event, count+1
	}
	if count > 1 {
		return ledger.Event{}, false, fmt.Errorf("%w: multiple final edit stage starts match binding", producterror.ErrConflict)
	}
	return found, count == 1, nil
}

func finalEditStageSubmittedEvent(events []ledger.Event, binding FinalEditStageBinding) (ledger.Event, bool, error) {
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return ledger.Event{}, false, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	var found ledger.Event
	count := 0
	for _, event := range events {
		if event.EventType != finalEditSubmittedEventType(binding.Stage) || event.CorrelationID != binding.IdempotencyKey {
			continue
		}
		if !finalEditSubmittedEventMatchesPipeline(event, binding, plan.Pipeline) {
			return ledger.Event{}, false, fmt.Errorf("%w: final edit stage submission binding differs", producterror.ErrConflict)
		}
		found, count = event, count+1
	}
	if count > 1 {
		return ledger.Event{}, false, fmt.Errorf("%w: multiple final edit stage submissions match binding", producterror.ErrConflict)
	}
	return found, count == 1, nil
}

func finalEditStageResultFromEvent(ctx context.Context, store LongFormFinalizationStore, binding FinalEditStageBinding, event ledger.Event, replay bool) (FinalEditStageResult, error) {
	payload := eventPayload(event)
	if !finalEditSubmittedEventMatchesPipeline(event, binding, finalEditStagePayloadPipeline(binding)) {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage submission binding differs", producterror.ErrConflict)
	}
	artifactID := payloadString(payload, "artifact_id")
	changed, ok := payloadBoolStrict(payload, "changed")
	if artifactID == "" || !ok {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage submission artifact fields are invalid", producterror.ErrConflict)
	}
	operationCount, ok := payloadIntStrict(payload, "operation_count")
	if !ok || operationCount < 0 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage operation count is invalid", producterror.ErrConflict)
	}
	diagnoses, diagnosesPresent, err := decodeFinalEditStyleOperationDiagnosesForStage(payload, binding.Stage, operationCount)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if binding.Stage == FinalEditStageStyle && diagnosesPresent && !changed && (operationCount != 0 || len(diagnoses) != 0) {
		return FinalEditStageResult{}, fmt.Errorf("%w: unchanged style edit replay must have empty operation diagnoses", producterror.ErrConflict)
	}
	if binding.Stage == FinalEditStageStyle && diagnosesPresent && changed && operationCount <= 0 {
		return FinalEditStageResult{}, fmt.Errorf("%w: changed style edit replay requires operation diagnoses", producterror.ErrConflict)
	}
	pipeline := finalEditStagePayloadPipeline(binding)
	findings, err := decodeStoredFinalEditGateFindingsPayloadForStage(payload["gate_findings"], pipeline, binding.Stage)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	semanticReview, err := decodeFinalEditSemanticAcceptancePayload(payload)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if err := validateStoredFinalEditGateFindingEvidence(ctx, store, binding.MissionID, findings); err != nil {
		return FinalEditStageResult{}, err
	}
	source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.Filename != binding.Filename || source.SHA256 != contentSHA256(source.Content) {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage source artifact differs from binding", producterror.ErrConflict)
	}
	if payloadString(payload, "source_sha256") != source.SHA256 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage source sha differs", producterror.ErrConflict)
	}
	artifact, err := store.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" || artifact.Filename != binding.Filename || artifact.SHA256 != contentSHA256(artifact.Content) {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage artifact differs from binding", producterror.ErrConflict)
	}
	if !changed && artifact.ArtifactID != binding.SourceArtifactID {
		return FinalEditStageResult{}, fmt.Errorf("%w: no-op final edit stage must reuse source artifact", producterror.ErrConflict)
	}
	if changed && binding.Stage == FinalEditStageStyleSemanticValidation && artifact.ArtifactID == binding.SourceArtifactID {
		return FinalEditStageResult{}, fmt.Errorf("%w: edited final edit stage artifact differs from binding", producterror.ErrConflict)
	}
	if changed && binding.Stage != FinalEditStageStyleSemanticValidation && (artifact.ArtifactID != binding.EditedArtifactID || artifact.Producer != (ledger.Producer{Type: "agent_session", ID: binding.ProviderSessionID})) {
		return FinalEditStageResult{}, fmt.Errorf("%w: edited final edit stage artifact differs from binding", producterror.ErrConflict)
	}
	if changed && (artifact.SHA256 == source.SHA256 || string(artifact.Content) == string(source.Content)) {
		return FinalEditStageResult{}, fmt.Errorf("%w: changed final edit stage artifact must differ from source", producterror.ErrConflict)
	}
	if payloadString(payload, "artifact_sha256") != artifact.SHA256 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage artifact sha differs", producterror.ErrConflict)
	}
	if pipeline == FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		switch binding.Stage {
		case FinalEditStageStyleSemanticValidation:
			stageStore, ok := store.(FinalEditStageStore)
			if !ok {
				return FinalEditStageResult{}, fmt.Errorf("%w: semantic validation replay requires final edit stage store", producterror.ErrConflict)
			}
			if err := validateV3StyleSemanticValidationReplay(ctx, stageStore, binding, source, artifact, operationCount, changed, findings, semanticReview); err != nil {
				return FinalEditStageResult{}, err
			}
		case FinalEditStageEvidenceGate:
			if err := validateV3EvidenceGateReplay(source, artifact, operationCount, changed, findings, semanticReview); err != nil {
				return FinalEditStageResult{}, err
			}
		}
	}
	if semanticReview.Count > 0 || len(semanticReview.Records) > 0 || strings.TrimSpace(semanticReview.Digest) != "" {
		stageStore, ok := store.(FinalEditStageStore)
		if !ok {
			return FinalEditStageResult{}, fmt.Errorf("%w: semantic acceptance replay requires final edit stage store", producterror.ErrConflict)
		}
		if err := validateStoredFinalEditSemanticAcceptanceAgainstLineage(ctx, stageStore, binding, string(artifact.Content), semanticReview); err != nil {
			return FinalEditStageResult{}, err
		}
	}
	return FinalEditStageResult{
		Binding:                        finalEditStageBindingForCompare(binding),
		Artifact:                       artifact,
		Event:                          event,
		Replay:                         replay,
		OperationCount:                 operationCount,
		Changed:                        changed,
		GateFindings:                   findings,
		SemanticReview:                 semanticReview,
		StyleOperationDiagnoses:        diagnoses,
		StyleOperationDiagnosesPresent: diagnosesPresent,
	}, nil
}

func decodeFinalEditStyleOperationDiagnosesForStage(payload map[string]any, stage string, operationCount int) ([]FinalEditStyleOperationDiagnosis, bool, error) {
	value, present := payload[FinalEditStyleOperationDiagnosesField]
	_, versionPresent := payload[FinalEditStyleOperationDiagnosesVersionField]
	if stage != FinalEditStageStyle {
		if present || versionPresent {
			return nil, false, fmt.Errorf("%w: style operation diagnoses are only valid for style edit", producterror.ErrConflict)
		}
		return nil, false, nil
	}
	if versionPresent {
		version, ok := payloadIntStrict(payload, FinalEditStyleOperationDiagnosesVersionField)
		if !ok || version != FinalEditStyleOperationDiagnosesVersion {
			return nil, false, fmt.Errorf("%w: style operation diagnoses version is invalid", producterror.ErrConflict)
		}
		if !present {
			return nil, false, fmt.Errorf("%w: style operation diagnoses payload is invalid", producterror.ErrConflict)
		}
		return decodeFinalEditStyleOperationDiagnosesPayload(value, operationCount, true)
	}
	if !present {
		return nil, false, nil
	}
	return decodeLegacyFinalEditStyleOperationDiagnosesPayload(value, operationCount)
}

func finalEditStartedEventMatchesPipeline(event ledger.Event, binding FinalEditStageBinding, pipeline string) bool {
	stored, ok := finalEditStageBindingFromStartEventForPipeline(event, pipeline)
	return ok && finalEditStageBindingsEqual(stored, binding)
}

func finalEditSubmittedEventMatchesPipeline(event ledger.Event, binding FinalEditStageBinding, pipeline string) bool {
	stored, ok := finalEditStageBindingFromSubmittedEventForPipeline(event, pipeline)
	if !ok || !finalEditStageBindingsEqual(stored, binding) {
		return false
	}
	payload := eventPayload(event)
	artifactID := payloadString(payload, "artifact_id")
	changed, ok := payloadBoolStrict(payload, "changed")
	if ok && changed && binding.Stage == FinalEditStageStyleSemanticValidation {
		return artifactID != "" && artifactID != binding.SourceArtifactID
	}
	return ok && ((!changed && artifactID == binding.SourceArtifactID) || (changed && artifactID == binding.EditedArtifactID))
}

func finalEditStageBindingFromStartEventForPipeline(event ledger.Event, pipeline string) (FinalEditStageBinding, bool) {
	binding, err := decodeFinalEditStageBindingFromEventForPipeline(event, false, pipeline)
	return binding, err == nil
}

func finalEditStageBindingFromSubmittedEventForPipeline(event ledger.Event, pipeline string) (FinalEditStageBinding, bool) {
	binding, err := decodeFinalEditStageBindingFromEventForPipeline(event, true, pipeline)
	return binding, err == nil
}

func decodeFinalEditStageBindingFromEventForPipeline(event ledger.Event, submitted bool, pipeline string) (FinalEditStageBinding, error) {
	pipeline = strings.TrimSpace(pipeline)
	if !isSupportedFinalEditPipeline(pipeline) {
		return FinalEditStageBinding{}, fmt.Errorf("%w: unsupported final edit pipeline", producterror.ErrConflict)
	}
	payload := eventPayload(event)
	binding := finalEditStageBindingFromPayload(event, payload)
	if pipeline == FinalEditPipelineAssemblyWriterReaderStyleGateV2 || pipeline == FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		binding.FinalEditPipeline = pipeline
	}
	if err := validateFinalEditStageBinding(binding); err != nil {
		return FinalEditStageBinding{}, err
	}
	if binding.RigorLevel == "" || binding.RigorLabel == "" {
		return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage event binding is incomplete", producterror.ErrConflict)
	}
	expectedType := finalEditStartedEventType(binding.Stage)
	expectedKind := "long_form_final_edit_" + binding.Stage + "_started"
	if submitted {
		expectedType = finalEditSubmittedEventType(binding.Stage)
		expectedKind = "long_form_final_edit_" + binding.Stage + "_submitted"
	}
	if event.EventType != expectedType ||
		event.CorrelationID != binding.IdempotencyKey ||
		event.CausationEventID != binding.PlanEventID ||
		event.Producer != binding.Producer ||
		payloadString(payload, "final_edit_pipeline") != pipeline ||
		payloadString(payload, "kind") != expectedKind ||
		payloadString(payload, "stage_id") != finalEditStageID(binding.Stage) {
		return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage event envelope differs", producterror.ErrConflict)
	}
	if submitted {
		if payloadString(payload, "artifact_id") == "" || payloadString(payload, "edited_artifact_id") != binding.EditedArtifactID {
			return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage submitted artifact envelope differs", producterror.ErrConflict)
		}
		if _, ok := payloadIntStrict(payload, "operation_count"); !ok {
			return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage operation count is invalid", producterror.ErrConflict)
		}
		if _, ok := payloadBoolStrict(payload, "changed"); !ok {
			return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage changed flag is invalid", producterror.ErrConflict)
		}
	}
	return finalEditStageBindingForCompare(binding), nil
}

func finalEditStageBindingsEqual(left, right FinalEditStageBinding) bool {
	return finalEditStageBindingForCompare(left) == finalEditStageBindingForCompare(right)
}

func finalEditStageBindingForCompare(binding FinalEditStageBinding) FinalEditStageBinding {
	binding = normalizeFinalEditStageBinding(binding)
	if binding.FinalEditPipeline == FinalEditPipelineReaderStyleGateV1 {
		binding.FinalEditPipeline = ""
	}
	return binding
}
