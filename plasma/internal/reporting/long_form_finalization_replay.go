package reporting

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func replayLongFormFinalize(ctx context.Context, store LongFormFinalizationStore, binding LongFormFinalizeBinding, event app.LedgerEvent, req LongFormFinalizeRequest) (LongFormFinalizeResult, error) {
	payload := eventPayload(event)
	artifact, err := store.GetRawArtifact(ctx, binding.ArtifactID)
	if err != nil {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: canonical final artifact is missing", app.ErrConflict)
	}
	expected, err := longFormMarkdownForRequest(ctx, store, binding, req)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" || artifact.Filename != binding.Filename || artifact.SHA256 != contentSHA256([]byte(expected)) || artifact.Producer != binding.Producer || event.CorrelationID != binding.IdempotencyKey || !canonicalMatchesBinding(event, payload, binding) {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: canonical long-form finalization binding differs", app.ErrConflict)
	}
	return LongFormFinalizeResult{Artifact: artifact, Event: event, Replay: true}, nil
}

func canonicalMatchesBinding(event app.LedgerEvent, payload map[string]any, binding LongFormFinalizeBinding) bool {
	return event.Producer == binding.Producer &&
		payload["pending_event_id"] == binding.PendingEventID && payload["plan_event_id"] == binding.PlanEventID &&
		payload["artifact_id"] == binding.ArtifactID && payload["title"] == binding.Title &&
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

func longFormAssemblyStrategy(composition string) string {
	if composition == LongFormCompositionNarrativeEdit {
		return "narrative_contract_final_edit"
	}
	return "c4_normalized_section_headings"
}

func longFormCanonical(events []app.LedgerEvent, pendingID string) (app.LedgerEvent, int) {
	var found app.LedgerEvent
	count := 0
	for _, event := range events {
		if event.EventType == "report.artifact.created" && eventPayload(event)["pending_event_id"] == pendingID {
			found, count = event, count+1
		}
	}
	return found, count
}
