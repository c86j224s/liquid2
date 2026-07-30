package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func partEditStartEventMatches(event app.LedgerEvent, binding PartEditBinding) bool {
	stored, ok := partEditBindingFromStartEvent(event)
	return ok && stored == normalizePartEditBinding(binding)
}

func partEditStartedEventMatches(events []app.LedgerEvent, acceptedPending map[string]bool, binding PartEditBinding) bool {
	count := 0
	for _, event := range events {
		if event.EventType != PartEditStartedEventType || event.CorrelationID != binding.IdempotencyKey {
			continue
		}
		payload := eventPayload(event)
		if !acceptedPending[payloadString(payload, "pending_event_id")] || !partEditStartEventMatches(event, binding) {
			continue
		}
		count++
	}
	return count == 1
}

func partEditBindingFromStartEvent(event app.LedgerEvent) (PartEditBinding, bool) {
	payload := eventPayload(event)
	binding := PartEditBinding{
		MissionID:                    strings.TrimSpace(event.MissionID),
		PendingEventID:               payloadString(payload, "pending_event_id"),
		PlanEventID:                  payloadString(payload, "plan_event_id"),
		SourcePartEventID:            payloadString(payload, "source_part_event_id"),
		SourceArtifactID:             payloadString(payload, "source_artifact_id"),
		EditedArtifactID:             payloadString(payload, "artifact_id"),
		Filename:                     payloadString(payload, "filename"),
		ToolSessionID:                payloadString(payload, "tool_session_id"),
		ProviderSessionID:            payloadString(payload, "provider_session_id"),
		PreviousProviderSessionID:    payloadString(payload, "previous_provider_session_id"),
		IdempotencyKey:               payloadString(payload, "idempotency_key"),
		PartIndex:                    jsonInt(payload["part_index"]),
		RequirementMapEventID:        payloadString(payload, "requirement_map_event_id"),
		RequirementMapHash:           payloadString(payload, "requirement_map_hash"),
		AgentExecutor:                payloadString(payload, "agent_executor"),
		AgentModel:                   payloadString(payload, "agent_model"),
		AgentReasoningEffort:         payloadString(payload, "agent_reasoning_effort"),
		AgentSelectionSource:         payloadString(payload, "agent_selection_source"),
		MCPMode:                      payloadString(payload, "mcp_mode"),
		ReportSessionPolicy:          payloadString(payload, "report_session_policy"),
		ReportSessionPolicySelection: payloadString(payload, "report_session_policy_selection"),
		GenerationGuidanceProfile:    payloadString(payload, "generation_guidance_profile"),
		GenerationGuidanceSHA256:     payloadString(payload, "generation_guidance_sha256"),
		SessionChainKind:             payloadString(payload, "session_chain_kind"),
		ReportPlanSessionID:          payloadString(payload, "report_plan_session_id"),
		ForkSourceAgentSessionID:     payloadString(payload, "fork_source_agent_session_id"),
	}
	binding = normalizePartEditBinding(binding)
	if payloadString(payload, "kind") != "sectional_markdown_report_part_edit_started" ||
		payloadString(payload, "stage_kind") != "part_edit" ||
		payloadString(payload, "stage_id") != fmt.Sprintf("part-edit-%d", binding.PartIndex) ||
		event.Producer != (app.Producer{Type: "agent_session", ID: binding.ProviderSessionID}) ||
		strings.TrimSpace(event.CausationEventID) != binding.SourcePartEventID ||
		strings.TrimSpace(event.CorrelationID) != binding.IdempotencyKey ||
		ValidatePartEditBinding(binding) != nil {
		return PartEditBinding{}, false
	}
	return binding, true
}

func validatePartEditRequirementMap(events []app.LedgerEvent, binding PartEditBinding) error {
	acceptedPending, err := longFormPendingLineage(events, binding.PendingEventID)
	if err != nil {
		return err
	}
	if !partEditRequirementMapMatches(events, acceptedPending, binding) {
		return fmt.Errorf("%w: Part edit requirement map differs from binding", app.ErrConflict)
	}
	return nil
}

func validatePartEditLineage(events []app.LedgerEvent, binding PartEditBinding) error {
	acceptedPending, err := longFormPendingLineage(events, binding.PendingEventID)
	if err != nil {
		return err
	}
	count := 0
	for _, event := range events {
		if event.EventID != binding.SourcePartEventID || event.EventType != "report.part.created" {
			continue
		}
		payload := eventPayload(event)
		pendingID := payloadString(payload, "pending_event_id")
		if !acceptedPending[pendingID] || payloadString(payload, "plan_event_id") != binding.PlanEventID || payloadString(payload, "artifact_id") != binding.SourceArtifactID || jsonInt(payload["part_index"]) != binding.PartIndex {
			return fmt.Errorf("%w: source part lineage differs from edit binding", app.ErrConflict)
		}
		count++
	}
	if count != 1 {
		return fmt.Errorf("%w: source part lineage is missing or duplicated", app.ErrConflict)
	}
	return nil
}

func canonicalPartEditEvent(events []app.LedgerEvent, binding PartEditBinding) (app.LedgerEvent, bool, error) {
	var found app.LedgerEvent
	count := 0
	for _, candidate := range events {
		if candidate.EventType != PartEditedEventType || candidate.CorrelationID != binding.IdempotencyKey {
			continue
		}
		if !partEditEventMatches(candidate, binding) {
			return app.LedgerEvent{}, false, fmt.Errorf("%w: part edit replay binding differs", app.ErrConflict)
		}
		found, count = candidate, count+1
	}
	if count > 1 {
		return app.LedgerEvent{}, false, fmt.Errorf("%w: multiple part edits match binding", app.ErrConflict)
	}
	return found, count == 1, nil
}

func partEditResultFromEvent(ctx context.Context, store PartEditOutcomeStore, binding PartEditBinding, event app.LedgerEvent, replay bool) (PartEditResult, error) {
	artifactID := payloadString(eventPayload(event), "artifact_id")
	artifact, err := store.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return PartEditResult{}, err
	}
	if err := validatePartEditArtifact(artifact, binding, artifactID, payloadBool(eventPayload(event), "changed")); err != nil {
		return PartEditResult{}, err
	}
	return PartEditResult{Artifact: artifact, Event: event, Replay: replay}, nil
}

func partEditEventMatches(event app.LedgerEvent, binding PartEditBinding) bool {
	payload := eventPayload(event)
	artifactID := payloadString(payload, "artifact_id")
	changed := payloadBool(payload, "changed")
	artifactMatches := (!changed && artifactID == binding.SourceArtifactID) || (changed && artifactID != "" && artifactID != binding.SourceArtifactID)
	return event.Producer == (app.Producer{Type: "agent_session", ID: binding.ProviderSessionID}) &&
		event.CausationEventID == binding.SourcePartEventID && event.CorrelationID == binding.IdempotencyKey &&
		payloadString(payload, "kind") == PartEditedKind && payloadString(payload, "idempotency_key") == binding.IdempotencyKey &&
		payloadString(payload, "pending_event_id") == binding.PendingEventID && payloadString(payload, "plan_event_id") == binding.PlanEventID &&
		payloadString(payload, "source_part_event_id") == binding.SourcePartEventID && payloadString(payload, "source_artifact_id") == binding.SourceArtifactID && artifactMatches &&
		payloadString(payload, "tool_session_id") == binding.ToolSessionID &&
		payloadString(payload, "provider_session_id") == binding.ProviderSessionID && payloadString(payload, "previous_provider_session_id") == binding.PreviousProviderSessionID &&
		jsonInt(payload["part_index"]) == binding.PartIndex && payloadString(payload, "requirement_map_event_id") == binding.RequirementMapEventID &&
		payloadString(payload, "requirement_map_hash") == binding.RequirementMapHash && payloadString(payload, "agent_executor") == binding.AgentExecutor &&
		payloadString(payload, "agent_model") == binding.AgentModel && payloadString(payload, "agent_reasoning_effort") == binding.AgentReasoningEffort &&
		payloadString(payload, "agent_selection_source") == binding.AgentSelectionSource && payloadString(payload, "mcp_mode") == binding.MCPMode &&
		payloadString(payload, "report_session_policy") == binding.ReportSessionPolicy && payloadString(payload, "report_session_policy_selection") == binding.ReportSessionPolicySelection &&
		payloadString(payload, "generation_guidance_profile") == binding.GenerationGuidanceProfile && payloadString(payload, "generation_guidance_sha256") == binding.GenerationGuidanceSHA256 &&
		payloadString(payload, "session_chain_kind") == binding.SessionChainKind && payloadString(payload, "report_plan_session_id") == binding.ReportPlanSessionID &&
		payloadString(payload, "fork_source_agent_session_id") == binding.ForkSourceAgentSessionID
}

func validatePartEditArtifact(artifact app.RawArtifact, binding PartEditBinding, eventArtifactID string, changed bool) error {
	if artifact.ArtifactID != eventArtifactID || artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" {
		return fmt.Errorf("%w: edited part artifact differs from binding", app.ErrConflict)
	}
	if !changed {
		if artifact.ArtifactID != binding.SourceArtifactID {
			return fmt.Errorf("%w: unchanged part edit must reuse source artifact", app.ErrConflict)
		}
		return nil
	}
	if artifact.ArtifactID == binding.SourceArtifactID || artifact.Producer != (app.Producer{Type: "agent_session", ID: binding.ProviderSessionID}) {
		return fmt.Errorf("%w: edited part artifact differs from binding", app.ErrConflict)
	}
	return nil
}
