package reporting_test

import (
	"context"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestStartPartEditPersistsFullNormalizedBindingPayload(t *testing.T) {
	ctx := context.Background()
	svc, closeStore, binding := newPartEditFixture(t, ctx)
	defer closeStore()

	event, created, err := reporting.StartPartEdit(ctx, svc, "evt_full_start", binding)
	if err != nil || !created {
		t.Fatalf("start created=%t event=%#v err=%v", created, event, err)
	}
	payload := partEditedPayload(t, event)
	for key, want := range map[string]any{
		"kind":                            "sectional_markdown_report_part_edit_started",
		"pending_event_id":                binding.PendingEventID,
		"plan_event_id":                   binding.PlanEventID,
		"source_part_event_id":            binding.SourcePartEventID,
		"source_artifact_id":              binding.SourceArtifactID,
		"artifact_id":                     binding.EditedArtifactID,
		"filename":                        binding.Filename,
		"tool_session_id":                 binding.ToolSessionID,
		"provider_session_id":             binding.ProviderSessionID,
		"previous_provider_session_id":    binding.PreviousProviderSessionID,
		"idempotency_key":                 binding.IdempotencyKey,
		"requirement_map_event_id":        binding.RequirementMapEventID,
		"requirement_map_hash":            binding.RequirementMapHash,
		"agent_executor":                  binding.AgentExecutor,
		"agent_model":                     binding.AgentModel,
		"agent_reasoning_effort":          binding.AgentReasoningEffort,
		"agent_selection_source":          binding.AgentSelectionSource,
		"mcp_mode":                        binding.MCPMode,
		"report_session_policy":           binding.ReportSessionPolicy,
		"report_session_policy_selection": binding.ReportSessionPolicySelection,
		"generation_guidance_profile":     binding.GenerationGuidanceProfile,
		"generation_guidance_sha256":      binding.GenerationGuidanceSHA256,
		"session_chain_kind":              binding.SessionChainKind,
		"report_plan_session_id":          binding.ReportPlanSessionID,
		"fork_source_agent_session_id":    binding.ForkSourceAgentSessionID,
		"stage_kind":                      "part_edit",
		"stage_id":                        "part-edit-1",
	} {
		if got := payload[key]; got != want {
			t.Fatalf("payload[%s]=%#v, want %#v; payload=%#v", key, got, want, payload)
		}
	}
	if _, exists := payload["edited_artifact_id"]; exists {
		t.Fatalf("start payload used non-canonical edited_artifact_id field: %#v", payload)
	}
}
