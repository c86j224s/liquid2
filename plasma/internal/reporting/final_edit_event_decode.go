package reporting

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func finalEditStartedEventMatches(event app.LedgerEvent, binding FinalEditStageBinding) bool {
	stored, ok := finalEditStageBindingFromStartEvent(event)
	return ok && stored == normalizeFinalEditStageBinding(binding)
}

func finalEditSubmittedEventMatches(event app.LedgerEvent, binding FinalEditStageBinding) bool {
	stored, ok := finalEditStageBindingFromSubmittedEvent(event)
	if !ok || stored != normalizeFinalEditStageBinding(binding) {
		return false
	}
	payload := eventPayload(event)
	artifactID := payloadString(payload, "artifact_id")
	changed, ok := payloadBoolStrict(payload, "changed")
	return ok && ((!changed && artifactID == binding.SourceArtifactID) || (changed && artifactID == binding.EditedArtifactID))
}

func finalEditStageBindingFromStartEvent(event app.LedgerEvent) (FinalEditStageBinding, bool) {
	binding, err := decodeFinalEditStageBindingFromEvent(event, false)
	return binding, err == nil
}

func finalEditStageBindingFromSubmittedEvent(event app.LedgerEvent) (FinalEditStageBinding, bool) {
	binding, err := decodeFinalEditStageBindingFromEvent(event, true)
	return binding, err == nil
}

func decodeFinalEditStageBindingFromEvent(event app.LedgerEvent, submitted bool) (FinalEditStageBinding, error) {
	payload := eventPayload(event)
	binding := finalEditStageBindingFromPayload(event, payload)
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
		payloadString(payload, "final_edit_pipeline") != FinalEditPipelineReaderStyleGateV1 ||
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
	return binding, nil
}

func finalEditStageBindingFromPayload(event app.LedgerEvent, payload map[string]any) FinalEditStageBinding {
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

func decodeStoredFinalEditGateFindingsPayload(value any) ([]StoredFinalEditGateFinding, error) {
	if value == nil {
		return nil, nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: final edit gate findings payload is invalid", app.ErrConflict)
	}
	out := make([]StoredFinalEditGateFinding, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		raw, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: final edit gate finding payload is invalid", app.ErrConflict)
		}
		for key := range raw {
			switch key {
			case "statement_sha256", "classification", "repair_action", "evidence_ids":
			default:
				return nil, fmt.Errorf("%w: final edit gate finding contains unsupported field", app.ErrConflict)
			}
		}
		statementSHA := payloadString(raw, "statement_sha256")
		classification := payloadString(raw, "classification")
		repairAction := payloadString(raw, "repair_action")
		if !validStoredFinalEditStatementSHA256(statementSHA) || !finalEditGateClasses[classification] || seen[statementSHA] {
			return nil, fmt.Errorf("%w: final edit gate finding payload is invalid", app.ErrConflict)
		}
		seen[statementSHA] = true
		evidenceIDs, err := stringSlicePayload(raw["evidence_ids"])
		if err != nil {
			return nil, err
		}
		if classification == FinalEditGateClassUnverifiedExternalFact {
			if !validFinalEditRepairAction(repairAction) {
				return nil, fmt.Errorf("%w: final edit gate finding repair action is invalid", app.ErrConflict)
			}
		} else if repairAction != "" {
			return nil, fmt.Errorf("%w: final edit gate finding repair action is invalid", app.ErrConflict)
		}
		if len(evidenceIDs) > 0 && repairAction != FinalEditRepairAttachApprovedEvidence {
			return nil, fmt.Errorf("%w: final edit gate finding evidence refs are invalid", app.ErrConflict)
		}
		if repairAction == FinalEditRepairAttachApprovedEvidence && len(evidenceIDs) == 0 {
			return nil, fmt.Errorf("%w: final edit gate finding evidence refs are invalid", app.ErrConflict)
		}
		out = append(out, StoredFinalEditGateFinding{
			StatementSHA256: statementSHA,
			Classification:  classification,
			RepairAction:    repairAction,
			EvidenceIDs:     evidenceIDs,
		})
	}
	return out, nil
}

func payloadBoolStrict(payload map[string]any, key string) (bool, bool) {
	value, ok := payload[key].(bool)
	return value, ok
}

func payloadIntStrict(payload map[string]any, key string) (int, bool) {
	number, ok := payload[key].(float64)
	if !ok || number < 0 || float64(int(number)) != number {
		return 0, false
	}
	return int(number), true
}

func stringSlicePayload(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: string slice payload is invalid", app.ErrConflict)
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		text, ok := value.(string)
		if !ok || text == "" || seen[text] {
			return nil, fmt.Errorf("%w: string slice payload is invalid", app.ErrConflict)
		}
		seen[text] = true
		out = append(out, text)
	}
	return out, nil
}
