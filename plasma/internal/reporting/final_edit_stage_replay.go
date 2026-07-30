package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type FinalEditStageStartContract struct {
	FinalBinding LongFormFinalizeBinding
	Stage        string
}

type FinalEditStageStartResult struct {
	Binding        FinalEditStageBinding
	SourceArtifact app.RawArtifact
	Event          app.LedgerEvent
}

type FinalEditStageProgress struct {
	Binding        FinalEditStageBinding
	SourceArtifact app.RawArtifact
	StartEvent     app.LedgerEvent
	Submission     *FinalEditStageResult
}

func LoadFinalEditStageProgress(ctx context.Context, store FinalEditStageStore, contract FinalEditStageStartContract) (FinalEditStageProgress, bool, error) {
	finalBinding := normalizeLongFormFinalizeBinding(contract.FinalBinding)
	if err := validateLongFormFinalizeBinding(finalBinding); err != nil {
		return FinalEditStageProgress{}, false, err
	}
	stage := normalizeFinalEditStageBinding(FinalEditStageBinding{Stage: contract.Stage}).Stage
	if finalEditStartedEventType(stage) == "" {
		return FinalEditStageProgress{}, false, fmt.Errorf("%w: unsupported final edit stage", app.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, finalBinding.MissionID)
	if err != nil {
		return FinalEditStageProgress{}, false, err
	}
	acceptedPending, err := longFormPendingLineage(events, finalBinding.PendingEventID)
	if err != nil {
		return FinalEditStageProgress{}, false, err
	}
	plan, ok, err := longFormPlanPipeline(events, finalBinding.PendingEventID, finalBinding.PlanEventID)
	if err != nil {
		return FinalEditStageProgress{}, false, err
	}
	if !ok || !isSupportedFinalEditPipeline(plan.Pipeline) {
		return FinalEditStageProgress{}, false, fmt.Errorf("%w: final edit plan is not active", app.ErrConflict)
	}
	finalBinding.PostReportHumanize = plan.PostReportHumanize
	var progress FinalEditStageProgress
	count := 0
	for _, event := range events {
		if event.EventType != finalEditStartedEventType(stage) ||
			event.MissionID != finalBinding.MissionID ||
			!acceptedPending[payloadString(eventPayload(event), "pending_event_id")] {
			continue
		}
		binding, ok := finalEditStageBindingFromStartEventForPipeline(event, plan.Pipeline)
		if !ok {
			return FinalEditStageProgress{}, false, fmt.Errorf("%w: stored final edit start is invalid", app.ErrConflict)
		}
		if binding.PendingEventID != finalBinding.PendingEventID ||
			binding.PlanEventID != finalBinding.PlanEventID ||
			binding.Stage != stage {
			continue
		}
		submitted, submittedOK, err := finalEditStageSubmittedEvent(events, binding)
		if err != nil {
			return FinalEditStageProgress{}, false, err
		}
		if err := validateFinalEditStageLineage(ctx, store, events, binding, submittedOK); err != nil {
			return FinalEditStageProgress{}, false, err
		}
		if err := finalEditStageStartMatchesFinalBinding(binding, finalBinding, plan); err != nil {
			return FinalEditStageProgress{}, false, err
		}
		source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
		if err != nil {
			return FinalEditStageProgress{}, false, err
		}
		if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.Filename != binding.Filename || source.SHA256 != contentSHA256(source.Content) {
			return FinalEditStageProgress{}, false, fmt.Errorf("%w: final edit start source artifact is invalid", app.ErrConflict)
		}
		progress = FinalEditStageProgress{Binding: binding, SourceArtifact: source, StartEvent: event}
		if submittedOK {
			result, err := finalEditStageResultFromEvent(ctx, store, progress.Binding, submitted, true)
			if err != nil {
				return FinalEditStageProgress{}, false, err
			}
			progress.Submission = &result
		}
		count++
	}
	if count > 1 {
		return FinalEditStageProgress{}, false, fmt.Errorf("%w: multiple final edit starts match current pending", app.ErrConflict)
	}
	if count == 0 {
		return FinalEditStageProgress{}, false, nil
	}
	return progress, true, nil
}

func finalEditStageStartMatchesFinalBinding(binding FinalEditStageBinding, finalBinding LongFormFinalizeBinding, plan FinalEditPipelinePlanState) error {
	if binding.MissionID != finalBinding.MissionID ||
		binding.PendingEventID != finalBinding.PendingEventID ||
		binding.PlanEventID != finalBinding.PlanEventID ||
		binding.Filename != finalBinding.Filename ||
		binding.Title != finalBinding.Title ||
		binding.AgentExecutor != finalBinding.AgentExecutor ||
		binding.AgentModel != finalBinding.AgentModel ||
		binding.AgentReasoningEffort != finalBinding.AgentReasoningEffort ||
		binding.AgentSelectionSource != finalBinding.AgentSelectionSource ||
		binding.MCPMode != finalBinding.MCPMode ||
		binding.RigorLevel != finalBinding.RigorLevel ||
		binding.RigorLabel != finalBinding.RigorLabel ||
		binding.ReportSessionPolicy != finalBinding.ReportSessionPolicy ||
		binding.ReportSessionPolicySelection != finalBinding.ReportSessionPolicySelection ||
		binding.PostReportHumanize != plan.PostReportHumanize ||
		binding.GenerationGuidanceProfile != finalBinding.GenerationGuidanceProfile ||
		binding.GenerationGuidanceSHA256 != finalBinding.GenerationGuidanceSHA256 ||
		binding.SessionChainKind != finalBinding.SessionChainKind ||
		binding.PreReportResearchSessionID != finalBinding.PreReportResearchSessionID ||
		binding.ReportPlanSessionID != finalBinding.ReportPlanSessionID {
		return fmt.Errorf("%w: final edit start binding differs from final binding", app.ErrConflict)
	}
	return nil
}

func loadLongFormCanonicalResult(ctx context.Context, store LongFormFinalizationStore, binding LongFormFinalizeBinding, event app.LedgerEvent) (LongFormFinalizeResult, error) {
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	artifact, err := loadCanonicalArtifactForBinding(ctx, store, events, binding, event)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	return LongFormFinalizeResult{Artifact: artifact, Event: event, Replay: true}, nil
}

func replayLongFormFinalizeForRequest(ctx context.Context, store LongFormFinalizationStore, binding LongFormFinalizeBinding, event app.LedgerEvent, req LongFormFinalizeRequest) (LongFormFinalizeResult, error) {
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if err := validateLongFormCanonicalReplayRequest(ctx, store, events, binding, event, req); err != nil {
		return LongFormFinalizeResult{}, err
	}
	artifact, err := loadCanonicalArtifactForBinding(ctx, store, events, binding, event)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	expected, err := longFormMarkdownForRequest(ctx, store, binding, req)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if artifact.SHA256 != contentSHA256([]byte(expected)) || string(artifact.Content) != expected {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: canonical long-form finalization content differs", app.ErrConflict)
	}
	return LongFormFinalizeResult{Artifact: artifact, Event: event, Replay: true}, nil
}

func finalEditStageStartedEvent(events []app.LedgerEvent, binding FinalEditStageBinding) (app.LedgerEvent, bool, error) {
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	var found app.LedgerEvent
	count := 0
	for _, event := range events {
		if event.EventType != finalEditStartedEventType(binding.Stage) || event.CorrelationID != binding.IdempotencyKey {
			continue
		}
		if !finalEditStartedEventMatchesPipeline(event, binding, plan.Pipeline) {
			return app.LedgerEvent{}, false, fmt.Errorf("%w: final edit stage start binding differs", app.ErrConflict)
		}
		found, count = event, count+1
	}
	if count > 1 {
		return app.LedgerEvent{}, false, fmt.Errorf("%w: multiple final edit stage starts match binding", app.ErrConflict)
	}
	return found, count == 1, nil
}

func finalEditStageSubmittedEvent(events []app.LedgerEvent, binding FinalEditStageBinding) (app.LedgerEvent, bool, error) {
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	var found app.LedgerEvent
	count := 0
	for _, event := range events {
		if event.EventType != finalEditSubmittedEventType(binding.Stage) || event.CorrelationID != binding.IdempotencyKey {
			continue
		}
		if !finalEditSubmittedEventMatchesPipeline(event, binding, plan.Pipeline) {
			return app.LedgerEvent{}, false, fmt.Errorf("%w: final edit stage submission binding differs", app.ErrConflict)
		}
		found, count = event, count+1
	}
	if count > 1 {
		return app.LedgerEvent{}, false, fmt.Errorf("%w: multiple final edit stage submissions match binding", app.ErrConflict)
	}
	return found, count == 1, nil
}

func finalEditStageResultFromEvent(ctx context.Context, store LongFormFinalizationStore, binding FinalEditStageBinding, event app.LedgerEvent, replay bool) (FinalEditStageResult, error) {
	payload := eventPayload(event)
	if !finalEditSubmittedEventMatchesPipeline(event, binding, finalEditStagePayloadPipeline(binding)) {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage submission binding differs", app.ErrConflict)
	}
	artifactID := payloadString(payload, "artifact_id")
	changed, ok := payloadBoolStrict(payload, "changed")
	if artifactID == "" || !ok {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage submission artifact fields are invalid", app.ErrConflict)
	}
	operationCount, ok := payloadIntStrict(payload, "operation_count")
	if !ok || operationCount < 0 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage operation count is invalid", app.ErrConflict)
	}
	findings, err := decodeStoredFinalEditGateFindingsPayload(payload["gate_findings"])
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
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage source artifact differs from binding", app.ErrConflict)
	}
	if payloadString(payload, "source_sha256") != source.SHA256 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage source sha differs", app.ErrConflict)
	}
	artifact, err := store.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" || artifact.Filename != binding.Filename || artifact.SHA256 != contentSHA256(artifact.Content) {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage artifact differs from binding", app.ErrConflict)
	}
	if !changed && artifact.ArtifactID != binding.SourceArtifactID {
		return FinalEditStageResult{}, fmt.Errorf("%w: no-op final edit stage must reuse source artifact", app.ErrConflict)
	}
	if changed && (artifact.ArtifactID != binding.EditedArtifactID || artifact.Producer != (app.Producer{Type: "agent_session", ID: binding.ProviderSessionID})) {
		return FinalEditStageResult{}, fmt.Errorf("%w: edited final edit stage artifact differs from binding", app.ErrConflict)
	}
	if changed && (artifact.SHA256 == source.SHA256 || string(artifact.Content) == string(source.Content)) {
		return FinalEditStageResult{}, fmt.Errorf("%w: changed final edit stage artifact must differ from source", app.ErrConflict)
	}
	if payloadString(payload, "artifact_sha256") != artifact.SHA256 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage artifact sha differs", app.ErrConflict)
	}
	return FinalEditStageResult{
		Binding:        finalEditStageBindingForCompare(binding),
		Artifact:       artifact,
		Event:          event,
		Replay:         replay,
		OperationCount: operationCount,
		Changed:        changed,
		GateFindings:   findings,
	}, nil
}

func finalEditStartedEventMatchesPipeline(event app.LedgerEvent, binding FinalEditStageBinding, pipeline string) bool {
	stored, ok := finalEditStageBindingFromStartEventForPipeline(event, pipeline)
	return ok && finalEditStageBindingsEqual(stored, binding)
}

func finalEditSubmittedEventMatchesPipeline(event app.LedgerEvent, binding FinalEditStageBinding, pipeline string) bool {
	stored, ok := finalEditStageBindingFromSubmittedEventForPipeline(event, pipeline)
	if !ok || !finalEditStageBindingsEqual(stored, binding) {
		return false
	}
	payload := eventPayload(event)
	artifactID := payloadString(payload, "artifact_id")
	changed, ok := payloadBoolStrict(payload, "changed")
	return ok && ((!changed && artifactID == binding.SourceArtifactID) || (changed && artifactID == binding.EditedArtifactID))
}

func finalEditStageBindingFromStartEventForPipeline(event app.LedgerEvent, pipeline string) (FinalEditStageBinding, bool) {
	binding, err := decodeFinalEditStageBindingFromEventForPipeline(event, false, pipeline)
	return binding, err == nil
}

func finalEditStageBindingFromSubmittedEventForPipeline(event app.LedgerEvent, pipeline string) (FinalEditStageBinding, bool) {
	binding, err := decodeFinalEditStageBindingFromEventForPipeline(event, true, pipeline)
	return binding, err == nil
}

func decodeFinalEditStageBindingFromEventForPipeline(event app.LedgerEvent, submitted bool, pipeline string) (FinalEditStageBinding, error) {
	pipeline = strings.TrimSpace(pipeline)
	if !isSupportedFinalEditPipeline(pipeline) {
		return FinalEditStageBinding{}, fmt.Errorf("%w: unsupported final edit pipeline", app.ErrConflict)
	}
	payload := eventPayload(event)
	binding := finalEditStageBindingFromPayload(event, payload)
	if pipeline == FinalEditPipelineAssemblyWriterReaderStyleGateV2 {
		binding.FinalEditPipeline = pipeline
	}
	if err := validateFinalEditStageBinding(binding); err != nil {
		return FinalEditStageBinding{}, err
	}
	if binding.RigorLevel == "" || binding.RigorLabel == "" {
		return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage event binding is incomplete", app.ErrConflict)
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
		return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage event envelope differs", app.ErrConflict)
	}
	if submitted {
		if payloadString(payload, "artifact_id") == "" || payloadString(payload, "edited_artifact_id") != binding.EditedArtifactID {
			return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage submitted artifact envelope differs", app.ErrConflict)
		}
		if _, ok := payloadIntStrict(payload, "operation_count"); !ok {
			return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage operation count is invalid", app.ErrConflict)
		}
		if _, ok := payloadBoolStrict(payload, "changed"); !ok {
			return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage changed flag is invalid", app.ErrConflict)
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

func validateStoredFinalEditGateFindingEvidence(ctx context.Context, store LongFormFinalizationStore, missionID string, findings []StoredFinalEditGateFinding) error {
	requiresEvidence := false
	for _, finding := range findings {
		requiresEvidence = requiresEvidence || len(finding.EvidenceIDs) > 0
	}
	if !requiresEvidence {
		return nil
	}
	validator, ok := store.(finalEditEvidenceStore)
	if !ok || validator == nil {
		return fmt.Errorf("%w: final edit gate evidence validator is required", app.ErrConflict)
	}
	missionID = strings.TrimSpace(missionID)
	for _, finding := range findings {
		for _, evidenceID := range finding.EvidenceIDs {
			record, err := validator.GetEvidenceRecord(ctx, evidenceID)
			if err != nil || record.MissionID != missionID || strings.TrimSpace(record.State) != "approved" {
				return fmt.Errorf("%w: final edit gate evidence ref is not approved", app.ErrConflict)
			}
		}
	}
	return nil
}
