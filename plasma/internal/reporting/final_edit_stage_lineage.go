package reporting

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func validateFinalEditStageLineage(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, binding FinalEditStageBinding, allowCanonical bool) error {
	acceptedPending, err := longFormPendingLineage(events, binding.PendingEventID)
	if err != nil {
		return err
	}
	if !acceptedPending[binding.PendingEventID] {
		return fmt.Errorf("%w: final edit pending event is missing", app.ErrConflict)
	}
	plan, ok, err := longFormPlanPipeline(events, binding.PendingEventID, binding.PlanEventID)
	if err != nil {
		return err
	}
	if !ok || !isSupportedFinalEditPipeline(plan.Pipeline) || plan.ReportMode != ModeLongForm {
		return fmt.Errorf("%w: final edit stage requires an active long-form final edit plan", app.ErrConflict)
	}
	if binding.PostReportHumanize != plan.PostReportHumanize {
		return fmt.Errorf("%w: final edit stage humanize setting differs from plan", app.ErrConflict)
	}
	if binding.FinalEditPipeline != "" && binding.FinalEditPipeline != plan.Pipeline {
		return fmt.Errorf("%w: final edit stage pipeline differs from plan", app.ErrConflict)
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	for _, event := range events {
		payload := eventPayload(event)
		pending := payloadString(payload, "pending_event_id")
		if !acceptedPending[pending] || pending != binding.PendingEventID {
			continue
		}
		if event.EventType == "report.draft.failed" || event.EventType == "report.final.failed" || (!allowCanonical && event.EventType == "report.artifact.created") {
			return fmt.Errorf("%w: final edit stage is terminal", app.ErrConflict)
		}
	}
	if err := validateFinalEditStageSessionContract(binding, plan); err != nil {
		return err
	}
	switch binding.Stage {
	case FinalEditStageWriter:
		return validateFinalEditWriterLineage(ctx, store, events, binding, plan)
	case FinalEditStageReader:
		return validateFinalEditReaderLineage(ctx, store, events, binding, plan)
	case FinalEditStageStyle:
		return validateFinalEditStyleLineage(ctx, store, events, binding, plan)
	case FinalEditStageGate:
		return validateFinalEditGateLineage(ctx, store, events, binding, plan)
	default:
		return fmt.Errorf("%w: unsupported final edit stage", app.ErrInvalidInput)
	}
}

func finalEditStagePlanForBinding(events []app.LedgerEvent, binding FinalEditStageBinding) (FinalEditPipelinePlanState, error) {
	plan, ok, err := longFormPlanPipeline(events, binding.PendingEventID, binding.PlanEventID)
	if err != nil {
		return FinalEditPipelinePlanState{}, err
	}
	if !ok || !isSupportedFinalEditPipeline(plan.Pipeline) || plan.ReportMode != ModeLongForm {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: final edit stage requires an active long-form final edit plan", app.ErrConflict)
	}
	if binding.PostReportHumanize != "" && binding.PostReportHumanize != plan.PostReportHumanize {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: final edit stage humanize setting differs from plan", app.ErrConflict)
	}
	if binding.FinalEditPipeline != "" && binding.FinalEditPipeline != plan.Pipeline {
		return FinalEditPipelinePlanState{}, fmt.Errorf("%w: final edit stage pipeline differs from plan", app.ErrConflict)
	}
	return plan, nil
}

func finalEditStageBindingForPlan(binding FinalEditStageBinding, plan FinalEditPipelinePlanState) FinalEditStageBinding {
	binding = normalizeFinalEditStageBinding(binding)
	if plan.Pipeline == FinalEditPipelineAssemblyWriterReaderStyleGateV2 {
		binding.FinalEditPipeline = plan.Pipeline
	} else if binding.FinalEditPipeline == FinalEditPipelineReaderStyleGateV1 {
		binding.FinalEditPipeline = ""
	}
	return binding
}

func validateFinalEditStageSessionContract(binding FinalEditStageBinding, plan FinalEditPipelinePlanState) error {
	switch binding.Stage {
	case FinalEditStageWriter:
		if plan.Pipeline != FinalEditPipelineAssemblyWriterReaderStyleGateV2 ||
			binding.ProviderSessionID == binding.ReportPlanSessionID ||
			binding.ForkSourceAgentSessionID != binding.ReportPlanSessionID {
			return fmt.Errorf("%w: writer final edit session chain differs from contract", app.ErrConflict)
		}
	case FinalEditStageReader, FinalEditStageGate:
		if binding.ProviderSessionID == binding.ReportPlanSessionID || binding.ForkSourceAgentSessionID != binding.ReportPlanSessionID {
			return fmt.Errorf("%w: reader/gate final edit session chain differs from contract", app.ErrConflict)
		}
	}
	return nil
}

func validateFinalEditWriterLineage(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, binding FinalEditStageBinding, plan FinalEditPipelinePlanState) error {
	if plan.Pipeline != FinalEditPipelineAssemblyWriterReaderStyleGateV2 {
		return fmt.Errorf("%w: final writer requires assembly_writer_reader_style_gate_v2 plan", app.ErrConflict)
	}
	request, assembly, err := finalEditAssemblyRequest(ctx, store, events, binding)
	if err != nil {
		return err
	}
	if binding.EditedArtifactID == binding.SourceArtifactID || binding.EditedArtifactID == plan.ArtifactID {
		return fmt.Errorf("%w: final writer target artifact differs from contract", app.ErrConflict)
	}
	artifact, err := store.GetRawArtifact(ctx, request.ArtifactID)
	if err != nil {
		return err
	}
	if err := validateFinalEditAssemblyArtifact(artifact, request); err != nil {
		return err
	}
	if _, ok, err := finalEditAssemblyCreatedEvent(events, binding, request, assembly.PartArtifactIDs); err != nil || !ok {
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: final writer requires deterministic final assembly", app.ErrConflict)
	}
	return nil
}

func validateFinalEditReaderLineage(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, binding FinalEditStageBinding, plan FinalEditPipelinePlanState) error {
	if plan.Pipeline == FinalEditPipelineAssemblyWriterReaderStyleGateV2 {
		writer, ok, err := loadFinalEditStageSubmissionForChain(ctx, store, events, binding, plan, FinalEditStageWriter)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: reader edit requires final writer submission", app.ErrConflict)
		}
		if binding.SourceArtifactID != writer.Artifact.ArtifactID || binding.EditedArtifactID == binding.SourceArtifactID || binding.EditedArtifactID == plan.ArtifactID {
			return fmt.Errorf("%w: reader edit source chain differs from contract", app.ErrConflict)
		}
		return nil
	}
	request, err := finalEditReaderSourceRequest(ctx, store, events, binding)
	if err != nil {
		return err
	}
	if binding.EditedArtifactID == binding.SourceArtifactID || binding.EditedArtifactID == plan.ArtifactID {
		return fmt.Errorf("%w: reader edit target artifact differs from contract", app.ErrConflict)
	}
	if existing, err := store.GetRawArtifact(ctx, request.ArtifactID); err == nil {
		if err := validateFinalEditReaderSourceArtifact(existing, request); err != nil {
			return err
		}
	}
	return nil
}

func validateFinalEditStyleLineage(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, binding FinalEditStageBinding, plan FinalEditPipelinePlanState) error {
	if plan.PostReportHumanize != FinalEditHumanizeEnabled {
		return fmt.Errorf("%w: style edit is disabled for this final edit plan", app.ErrConflict)
	}
	reader, ok, err := loadFinalEditStageSubmissionForChain(ctx, store, events, binding, plan, FinalEditStageReader)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: style edit requires reader submission", app.ErrConflict)
	}
	if binding.SourceArtifactID != reader.Artifact.ArtifactID || binding.EditedArtifactID == binding.SourceArtifactID || binding.EditedArtifactID == plan.ArtifactID {
		return fmt.Errorf("%w: style edit source chain differs from contract", app.ErrConflict)
	}
	if plan.Pipeline == FinalEditPipelineAssemblyWriterReaderStyleGateV2 &&
		(binding.ProviderSessionID == reader.Binding.ProviderSessionID || binding.ForkSourceAgentSessionID != reader.Binding.ProviderSessionID) {
		return fmt.Errorf("%w: style edit session chain differs from contract", app.ErrConflict)
	}
	return nil
}

func validateFinalEditGateLineage(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, binding FinalEditStageBinding, plan FinalEditPipelinePlanState) error {
	priorStage := FinalEditStageReader
	if plan.PostReportHumanize == FinalEditHumanizeEnabled {
		priorStage = FinalEditStageStyle
	}
	prior, ok, err := loadFinalEditStageSubmissionForChain(ctx, store, events, binding, plan, priorStage)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: corrective gate requires prior final edit submission", app.ErrConflict)
	}
	if binding.SourceArtifactID != prior.Artifact.ArtifactID || binding.EditedArtifactID != plan.ArtifactID {
		return fmt.Errorf("%w: corrective gate source or target differs from contract", app.ErrConflict)
	}
	return nil
}

func loadFinalEditStageSubmissionForChain(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, binding FinalEditStageBinding, plan FinalEditPipelinePlanState, stage string) (FinalEditStageResult, bool, error) {
	key := FinalEditStageIdempotencyKey(stage, binding.PendingEventID, binding.PlanEventID)
	var foundBinding FinalEditStageBinding
	var found app.LedgerEvent
	count := 0
	for _, event := range events {
		if event.EventType != finalEditSubmittedEventType(stage) || event.CorrelationID != key {
			continue
		}
		candidate, ok := finalEditStageBindingFromSubmittedEventForPipeline(event, plan.Pipeline)
		if !ok {
			return FinalEditStageResult{}, false, fmt.Errorf("%w: stored final edit submission is invalid", app.ErrConflict)
		}
		if !finalEditStageSharesContract(candidate, binding) {
			return FinalEditStageResult{}, false, fmt.Errorf("%w: final edit source chain binding differs", app.ErrConflict)
		}
		foundBinding, found, count = candidate, event, count+1
	}
	if count > 1 {
		return FinalEditStageResult{}, false, fmt.Errorf("%w: multiple prior final edit submissions match chain", app.ErrConflict)
	}
	if count == 0 {
		return FinalEditStageResult{}, false, nil
	}
	if err := validateFinalEditStageLineage(ctx, store, events, foundBinding, true); err != nil {
		return FinalEditStageResult{}, false, err
	}
	result, err := finalEditStageResultFromEvent(ctx, store, foundBinding, found, true)
	return result, err == nil, err
}

func finalEditStageSharesContract(left, right FinalEditStageBinding) bool {
	left = finalEditStageBindingForCompare(left)
	right = finalEditStageBindingForCompare(right)
	return left.MissionID == right.MissionID &&
		left.PendingEventID == right.PendingEventID &&
		left.PlanEventID == right.PlanEventID &&
		left.Filename == right.Filename &&
		left.Title == right.Title &&
		left.AgentExecutor == right.AgentExecutor &&
		left.AgentModel == right.AgentModel &&
		left.AgentReasoningEffort == right.AgentReasoningEffort &&
		left.AgentSelectionSource == right.AgentSelectionSource &&
		left.MCPMode == right.MCPMode &&
		left.RigorLevel == right.RigorLevel &&
		left.RigorLabel == right.RigorLabel &&
		left.ReportSessionPolicy == right.ReportSessionPolicy &&
		left.ReportSessionPolicySelection == right.ReportSessionPolicySelection &&
		left.PostReportHumanize == right.PostReportHumanize &&
		left.GenerationGuidanceProfile == right.GenerationGuidanceProfile &&
		left.GenerationGuidanceSHA256 == right.GenerationGuidanceSHA256 &&
		left.SessionChainKind == right.SessionChainKind &&
		left.PreReportResearchSessionID == right.PreReportResearchSessionID &&
		left.ReportPlanSessionID == right.ReportPlanSessionID
}
