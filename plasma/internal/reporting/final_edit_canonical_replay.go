package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type finalEditCanonicalFields struct {
	Pipeline         string
	GateFindings     []StoredFinalEditGateFinding
	SemanticReview   FinalEditSemanticAttestation
	ActualArtifactID string
	GateEventID      string
	GateChanged      bool
}

func loadCanonicalArtifactForBinding(ctx context.Context, store LongFormFinalizationStore, events []app.LedgerEvent, binding LongFormFinalizeBinding, event app.LedgerEvent) (app.RawArtifact, error) {
	payload := eventPayload(event)
	plan, ok, err := longFormPlanPipeline(events, binding.PendingEventID, binding.PlanEventID)
	if err != nil {
		return app.RawArtifact{}, err
	}
	payloadPipeline := payloadString(payload, "final_edit_pipeline")
	if ok && isSupportedFinalEditPipeline(plan.Pipeline) && payloadPipeline != plan.Pipeline {
		return app.RawArtifact{}, fmt.Errorf("%w: canonical final edit pipeline marker is missing", app.ErrConflict)
	}
	if payloadPipeline != "" {
		if !isSupportedFinalEditPipeline(payloadPipeline) {
			return app.RawArtifact{}, fmt.Errorf("%w: canonical final edit pipeline differs", app.ErrConflict)
		}
		if err := validateLongFormCanonicalGatePayload(ctx, store, events, binding, payload); err != nil {
			return app.RawArtifact{}, err
		}
		artifactID := payloadString(payload, "artifact_id")
		if event.CorrelationID != binding.IdempotencyKey || !canonicalMatchesBindingForActualArtifact(event, payload, binding, artifactID, true) {
			return app.RawArtifact{}, fmt.Errorf("%w: canonical long-form finalization binding differs", app.ErrConflict)
		}
		artifact, err := store.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return app.RawArtifact{}, err
		}
		if err := validateCanonicalArtifactEnvelope(artifact, binding, payload); err != nil {
			return app.RawArtifact{}, err
		}
		return artifact, nil
	}
	if event.CorrelationID != binding.IdempotencyKey || !canonicalMatchesBinding(event, payload, binding) {
		return app.RawArtifact{}, fmt.Errorf("%w: canonical long-form finalization binding differs", app.ErrConflict)
	}
	artifact, err := store.GetRawArtifact(ctx, binding.ArtifactID)
	if err != nil {
		return app.RawArtifact{}, err
	}
	if artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" || artifact.Filename != binding.Filename || artifact.Producer != binding.Producer {
		return app.RawArtifact{}, fmt.Errorf("%w: canonical long-form final artifact differs", app.ErrConflict)
	}
	return artifact, nil
}

func validateLongFormCanonicalReplayRequest(ctx context.Context, store LongFormFinalizationStore, events []app.LedgerEvent, binding LongFormFinalizeBinding, event app.LedgerEvent, req LongFormFinalizeRequest) error {
	payload := eventPayload(event)
	pipeline := strings.TrimSpace(req.FinalEditPipeline)
	if err := validateFinalEditCanonicalReplayPayload(payload, pipeline, req.GateFindings, req.SemanticReview); err != nil {
		return err
	}
	if isSupportedFinalEditPipeline(pipeline) {
		if strings.TrimSpace(req.FinalEditActualArtifactID) != "" && payloadString(payload, "artifact_id") != strings.TrimSpace(req.FinalEditActualArtifactID) {
			return fmt.Errorf("%w: canonical actual artifact differs", app.ErrConflict)
		}
		if strings.TrimSpace(req.FinalEditGateEventID) != "" && payloadString(payload, "final_edit_gate_event_id") != strings.TrimSpace(req.FinalEditGateEventID) {
			return fmt.Errorf("%w: canonical gate event differs", app.ErrConflict)
		}
		if changed, ok := payloadBoolStrict(payload, "final_edit_gate_changed"); !ok || changed != req.FinalEditGateChanged {
			return fmt.Errorf("%w: canonical gate changed flag differs", app.ErrConflict)
		}
		return validateLongFormCanonicalGatePayload(ctx, store, events, binding, payload)
	}
	if isSupportedFinalEditPipeline(payloadString(payload, "final_edit_pipeline")) {
		return validateLongFormCanonicalGatePayload(ctx, store, events, binding, payload)
	}
	return nil
}

func validateFinalEditCanonicalReplayPayload(payload map[string]any, pipeline string, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) error {
	storedPipeline := payloadString(payload, "final_edit_pipeline")
	if strings.TrimSpace(pipeline) == "" {
		if storedPipeline != "" {
			return fmt.Errorf("%w: canonical final edit pipeline differs", app.ErrConflict)
		}
		return nil
	}
	if !isSupportedFinalEditPipeline(pipeline) || storedPipeline != pipeline {
		return fmt.Errorf("%w: canonical final edit pipeline differs", app.ErrConflict)
	}
	storedFindings, err := decodeStoredFinalEditGateFindingsPayloadForPipeline(payload["final_edit_gate_findings"], pipeline)
	if err != nil {
		return err
	}
	if !equalStoredFinalEditGateFindings(storedFindings, findings) {
		return fmt.Errorf("%w: canonical final edit gate findings differ", app.ErrConflict)
	}
	storedSemantic, err := decodeFinalEditSemanticAcceptancePayload(map[string]any{
		"semantic_acceptance":        payload["final_edit_semantic_acceptance"],
		"semantic_acceptance_count":  payload["final_edit_semantic_acceptance_count"],
		"semantic_acceptance_digest": payload["final_edit_semantic_acceptance_digest"],
	})
	if err != nil {
		return err
	}
	if !equalStoredFinalEditSemanticAcceptance(storedSemantic.Records, semanticReview.Records) || storedSemantic.Digest != semanticReview.Digest || storedSemantic.Count != semanticReview.Count {
		return fmt.Errorf("%w: canonical semantic acceptance differs", app.ErrConflict)
	}
	return nil
}

func validateLongFormCanonicalGatePayload(ctx context.Context, store LongFormFinalizationStore, events []app.LedgerEvent, binding LongFormFinalizeBinding, payload map[string]any) error {
	plan, ok, err := longFormPlanPipeline(events, binding.PendingEventID, binding.PlanEventID)
	if err != nil {
		return err
	}
	if !ok || !isSupportedFinalEditPipeline(plan.Pipeline) {
		return fmt.Errorf("%w: canonical final edit plan differs", app.ErrConflict)
	}
	if payloadString(payload, "final_edit_pipeline") != plan.Pipeline ||
		payloadString(payload, "planned_final_artifact_id") != binding.ArtifactID ||
		payloadString(payload, "artifact_id") == "" ||
		payloadString(payload, "final_edit_gate_event_id") == "" ||
		payloadString(payload, "artifact_sha256") == "" {
		return fmt.Errorf("%w: canonical final edit gate payload is incomplete", app.ErrConflict)
	}
	canonicalFindings, err := decodeStoredFinalEditGateFindingsPayloadForPipeline(payload["final_edit_gate_findings"], plan.Pipeline)
	if err != nil {
		return err
	}
	gateChanged, ok := payloadBoolStrict(payload, "final_edit_gate_changed")
	if !ok {
		return fmt.Errorf("%w: canonical final edit gate changed flag is invalid", app.ErrConflict)
	}
	gateEventID := payloadString(payload, "final_edit_gate_event_id")
	gateStage := FinalEditStageGate
	if plan.Pipeline == FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		gateStage = FinalEditStageEvidenceGate
	}
	key := FinalEditStageIdempotencyKey(gateStage, binding.PendingEventID, binding.PlanEventID)
	var foundBinding FinalEditStageBinding
	var gateEvent app.LedgerEvent
	count := 0
	for _, event := range events {
		if event.EventType != finalEditSubmittedEventType(gateStage) || event.CorrelationID != key {
			continue
		}
		count++
		if event.EventID != gateEventID {
			continue
		}
		stageBinding, ok := finalEditStageBindingFromSubmittedEventForPipeline(event, plan.Pipeline)
		if !ok {
			return fmt.Errorf("%w: canonical final edit gate submission is invalid", app.ErrConflict)
		}
		foundBinding = stageBinding
		gateEvent = event
	}
	if count != 1 || gateEvent.EventID != gateEventID {
		return fmt.Errorf("%w: canonical final edit gate submission count differs", app.ErrConflict)
	}
	if err := validateFinalEditGateBindingMatchesFinal(foundBinding, binding, plan); err != nil {
		return err
	}
	gateResult, err := finalEditStageResultFromEvent(ctx, store, foundBinding, gateEvent, true)
	if err != nil {
		return err
	}
	if payloadString(payload, "artifact_id") != gateResult.Artifact.ArtifactID ||
		payloadString(payload, "artifact_sha256") != gateResult.Artifact.SHA256 ||
		gateChanged != gateResult.Changed ||
		!equalStoredFinalEditGateFindings(canonicalFindings, gateResult.GateFindings) {
		return fmt.Errorf("%w: canonical final edit gate payload differs", app.ErrConflict)
	}
	canonicalSemantic, err := decodeFinalEditSemanticAcceptancePayload(map[string]any{
		"semantic_acceptance":        payload["final_edit_semantic_acceptance"],
		"semantic_acceptance_count":  payload["final_edit_semantic_acceptance_count"],
		"semantic_acceptance_digest": payload["final_edit_semantic_acceptance_digest"],
	})
	if err != nil {
		return err
	}
	if !equalStoredFinalEditSemanticAcceptance(canonicalSemantic.Records, gateResult.SemanticReview.Records) || canonicalSemantic.Digest != gateResult.SemanticReview.Digest || canonicalSemantic.Count != gateResult.SemanticReview.Count {
		return fmt.Errorf("%w: canonical semantic acceptance differs", app.ErrConflict)
	}
	if gateChanged {
		if plan.Pipeline == FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
			return fmt.Errorf("%w: evidence gate canonical payload cannot be changed", app.ErrConflict)
		}
		if gateResult.Artifact.ArtifactID != binding.ArtifactID || gateResult.Artifact.Producer != binding.Producer {
			return fmt.Errorf("%w: changed corrective gate canonical artifact differs from binding", app.ErrConflict)
		}
	} else if gateResult.Artifact.ArtifactID != foundBinding.SourceArtifactID {
		return fmt.Errorf("%w: no-op corrective gate canonical artifact differs from source", app.ErrConflict)
	}
	if plan.Pipeline == FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 && payloadString(payload, "artifact_id") != foundBinding.SourceArtifactID {
		return fmt.Errorf("%w: evidence gate canonical artifact must be the source artifact", app.ErrConflict)
	}
	return nil
}

func canonicalMatchesBindingForActualArtifact(event app.LedgerEvent, payload map[string]any, binding LongFormFinalizeBinding, actualArtifactID string, requirePlanned bool) bool {
	if requirePlanned && payloadString(payload, "planned_final_artifact_id") != binding.ArtifactID {
		return false
	}
	return event.Producer == binding.Producer &&
		payload["pending_event_id"] == binding.PendingEventID && payload["plan_event_id"] == binding.PlanEventID &&
		payload["artifact_id"] == actualArtifactID && payload["title"] == binding.Title &&
		payload["tool_session_id"] == binding.ToolSessionID && payloadString(payload, "plan_tool_session_id") == binding.PlanToolSessionID &&
		payload["report_mode"] == ModeLongForm && payload["agent_executor"] == binding.AgentExecutor &&
		payloadString(payload, "agent_model") == binding.AgentModel && payloadString(payload, "agent_reasoning_effort") == binding.AgentReasoningEffort &&
		payloadString(payload, "agent_selection_source") == binding.AgentSelectionSource && payloadString(payload, "agent_session_id") == binding.ProviderSessionID &&
		payloadString(payload, "previous_agent_session_id") == binding.PreviousProviderSessionID && payloadString(payload, "returned_agent_session_id") == "" &&
		payloadString(payload, "report_session_id") == binding.ProviderSessionID && payloadString(payload, "mcp_mode") == binding.MCPMode &&
		payloadString(payload, "rigor_level") == binding.RigorLevel && payloadString(payload, "rigor_label") == binding.RigorLabel &&
		payloadString(payload, "report_session_policy") == binding.ReportSessionPolicy && payloadString(payload, "report_session_policy_selection") == binding.ReportSessionPolicySelection &&
		payloadString(payload, "post_report_humanize") == binding.PostReportHumanize && payloadBool(payload, "humanize_enabled") == (binding.PostReportHumanize != "disabled") &&
		payloadString(payload, "generation_guidance_profile") == binding.GenerationGuidanceProfile && payloadString(payload, "generation_guidance_sha256") == binding.GenerationGuidanceSHA256 &&
		payloadString(payload, "session_chain_kind") == binding.SessionChainKind && payloadString(payload, "pre_report_research_session_id") == binding.PreReportResearchSessionID &&
		payloadString(payload, "report_plan_session_id") == binding.ReportPlanSessionID && payloadString(payload, "fork_source_agent_session_id") == binding.ForkSourceAgentSessionID &&
		payloadString(payload, "composition_strategy") == binding.CompositionStrategy && payloadString(payload, "assembly_strategy") == longFormAssemblyStrategy(binding.CompositionStrategy) &&
		jsonInt(payload["part_count"]) == len(binding.PartArtifactIDs) && jsonInt(payload["section_count"]) == len(binding.SectionArtifactIDs) &&
		jsonInt(payload["section_word_count"]) == binding.SectionWordCount && equalJSONStrings(payload["part_artifact_ids"], binding.PartArtifactIDs) &&
		equalJSONStrings(payload["section_artifact_ids"], binding.SectionArtifactIDs)
}
