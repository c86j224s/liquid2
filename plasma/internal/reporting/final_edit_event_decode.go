package reporting

import (
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func finalEditStartedEventMatches(event ledger.Event, binding FinalEditStageBinding) bool {
	stored, ok := finalEditStageBindingFromStartEvent(event)
	return ok && stored == normalizeFinalEditStageBinding(binding)
}

func finalEditSubmittedEventMatches(event ledger.Event, binding FinalEditStageBinding) bool {
	stored, ok := finalEditStageBindingFromSubmittedEvent(event)
	if !ok || stored != normalizeFinalEditStageBinding(binding) {
		return false
	}
	payload := eventPayload(event)
	artifactID := payloadString(payload, "artifact_id")
	changed, ok := payloadBoolStrict(payload, "changed")
	return ok && ((!changed && artifactID == binding.SourceArtifactID) || (changed && artifactID == binding.EditedArtifactID))
}

func finalEditStageBindingFromStartEvent(event ledger.Event) (FinalEditStageBinding, bool) {
	binding, err := decodeFinalEditStageBindingFromEvent(event, false)
	return binding, err == nil
}

func finalEditStageBindingFromSubmittedEvent(event ledger.Event) (FinalEditStageBinding, bool) {
	binding, err := decodeFinalEditStageBindingFromEvent(event, true)
	return binding, err == nil
}

func decodeFinalEditStageBindingFromEvent(event ledger.Event, submitted bool) (FinalEditStageBinding, error) {
	payload := eventPayload(event)
	binding := finalEditStageBindingFromPayload(event, payload)
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
		payloadString(payload, "final_edit_pipeline") != FinalEditPipelineReaderStyleGateV1 ||
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
	return binding, nil
}

func finalEditStageBindingFromPayload(event ledger.Event, payload map[string]any) FinalEditStageBinding {
	artifactID := payloadString(payload, "artifact_id")
	binding := FinalEditStageBinding{
		MissionID: event.MissionID, PendingEventID: payloadString(payload, "pending_event_id"),
		PlanEventID: payloadString(payload, "plan_event_id"), Title: payloadString(payload, "title"),
		Stage: payloadString(payload, "stage"), SourceArtifactID: payloadString(payload, "source_artifact_id"),
		EditedArtifactID: artifactID, Filename: payloadString(payload, "filename"),
		ToolSessionID: payloadString(payload, "tool_session_id"), ProviderSessionID: payloadString(payload, "provider_session_id"),
		PreviousProviderSessionID:    payloadString(payload, "previous_provider_session_id"),
		IdempotencyKey:               payloadString(payload, "idempotency_key"),
		AgentExecutor:                payloadString(payload, "agent_executor"),
		AgentModel:                   payloadString(payload, "agent_model"),
		AgentReasoningEffort:         payloadString(payload, "agent_reasoning_effort"),
		AgentSelectionSource:         payloadString(payload, "agent_selection_source"),
		MCPMode:                      payloadString(payload, "mcp_mode"),
		RigorLevel:                   payloadString(payload, "rigor_level"),
		RigorLabel:                   payloadString(payload, "rigor_label"),
		ReportSessionPolicy:          payloadString(payload, "report_session_policy"),
		ReportSessionPolicySelection: payloadString(payload, "report_session_policy_selection"),
		PostReportHumanize:           payloadString(payload, "post_report_humanize"),
		GenerationGuidanceProfile:    payloadString(payload, "generation_guidance_profile"),
		GenerationGuidanceSHA256:     payloadString(payload, "generation_guidance_sha256"),
		SessionChainKind:             payloadString(payload, "session_chain_kind"),
		PreReportResearchSessionID:   payloadString(payload, "pre_report_research_session_id"),
		ReportPlanSessionID:          payloadString(payload, "report_plan_session_id"),
		ForkSourceAgentSessionID:     payloadString(payload, "fork_source_agent_session_id"),
		Producer:                     event.Producer,
	}
	if payloadString(payload, "edited_artifact_id") != "" {
		binding.EditedArtifactID = payloadString(payload, "edited_artifact_id")
	}
	return normalizeFinalEditStageBinding(binding)
}
