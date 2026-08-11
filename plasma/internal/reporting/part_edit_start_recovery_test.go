package reporting_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestLoadCurrentPartEditStartValidatesExactCurrentStart(t *testing.T) {
	ctx := context.Background()
	t.Run("valid current start", func(t *testing.T) {
		svc, closeStore, binding := newPartEditFixture(t, ctx)
		defer closeStore()
		startPartEdit(t, ctx, svc, binding)

		recovered, ok, err := reporting.LoadCurrentPartEditStart(ctx, svc, partEditStartContract(binding))
		if err != nil || !ok || recovered != binding {
			t.Fatalf("recovered=%#v ok=%t err=%v", recovered, ok, err)
		}
	})

	for _, tc := range []struct {
		name   string
		mutate func(*app.AppendEventRequest)
	}{
		{name: "malformed missing artifact", mutate: func(req *app.AppendEventRequest) {
			payload := requestPayload(t, *req)
			delete(payload, "artifact_id")
			req.Payload = testJSON(payload)
		}},
		{name: "provider drift", mutate: func(req *app.AppendEventRequest) {
			payload := requestPayload(t, *req)
			payload["provider_session_id"] = "provider-drift"
			payload["previous_provider_session_id"] = "provider-drift"
			req.Producer = app.Producer{Type: "agent_session", ID: "provider-drift"}
			req.Payload = testJSON(payload)
		}},
		{name: "fork source drift", mutate: func(req *app.AppendEventRequest) {
			payload := requestPayload(t, *req)
			payload["fork_source_agent_session_id"] = "provider-other-source"
			req.Payload = testJSON(payload)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, closeStore, binding := newPartEditFixture(t, ctx)
			defer closeStore()
			req := reporting.BuildPartEditStartedAppendRequest("evt_stored_start", binding)
			tc.mutate(&req)
			if _, err := svc.AppendEvent(ctx, req); err != nil {
				t.Fatal(err)
			}
			_, _, err := reporting.LoadCurrentPartEditStart(ctx, svc, partEditStartContract(binding))
			if !errors.Is(err, app.ErrConflict) {
				t.Fatalf("error=%v, want conflict", err)
			}
		})
	}
}

func TestLoadCurrentPartEditStartRejectsDuplicateAndAncestorPartialStarts(t *testing.T) {
	ctx := context.Background()
	t.Run("duplicate current start", func(t *testing.T) {
		svc, closeStore, binding := newPartEditFixture(t, ctx)
		defer closeStore()
		if _, err := svc.AppendEvents(ctx, binding.MissionID, []app.AppendEventRequest{
			reporting.BuildPartEditStartedAppendRequest("evt_start_one", binding),
			reporting.BuildPartEditStartedAppendRequest("evt_start_two", binding),
		}); err != nil {
			t.Fatal(err)
		}
		_, _, err := reporting.LoadCurrentPartEditStart(ctx, svc, partEditStartContract(binding))
		if !errors.Is(err, app.ErrConflict) {
			t.Fatalf("error=%v, want conflict", err)
		}
	})

	t.Run("resume and restart do not reuse ancestor partial start", func(t *testing.T) {
		svc, closeStore, binding := newPartEditFixture(t, ctx)
		defer closeStore()
		startPartEdit(t, ctx, svc, binding)
		if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
			EventID: "evt_root_failed", MissionID: binding.MissionID, EventType: "report.draft.failed",
			Producer: app.Producer{Type: "agent", ID: "codex"},
			Payload:  testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "kind": "report_draft_failed"}),
		}); err != nil {
			t.Fatal(err)
		}
		for _, req := range []app.AppendEventRequest{
			{EventID: "evt_pending_resume", MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: testJSON(map[string]any{
				"report_mode": "long_form", "origin_pending_event_id": binding.PendingEventID,
				"retry_of_pending_event_id": binding.PendingEventID, "retry_strategy": "resume_failed", "attempt_number": 2,
			})},
			{EventID: "evt_pending_restart", MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: testJSON(map[string]any{
				"report_mode": "long_form", "origin_pending_event_id": binding.PendingEventID,
				"retry_of_pending_event_id": binding.PendingEventID, "retry_strategy": "restart", "attempt_number": 2,
			})},
		} {
			if _, err := svc.AppendEvent(ctx, req); err != nil {
				t.Fatal(err)
			}
		}
		for _, pendingID := range []string{"evt_pending_resume", "evt_pending_restart"} {
			contract := partEditStartContract(binding)
			contract.CurrentPendingEventID = pendingID
			contract.IdempotencyKey = "report-part-edit:" + pendingID + ":" + binding.PlanEventID + ":1"
			recovered, ok, err := reporting.LoadCurrentPartEditStart(ctx, svc, contract)
			if err != nil || ok {
				t.Fatalf("%s recovered ancestor start: recovered=%#v ok=%t err=%v", pendingID, recovered, ok, err)
			}
		}
	})
}

func TestLoadCurrentPartEditStartRejectsInvalidCompletionArtifact(t *testing.T) {
	ctx := context.Background()
	svc, closeStore, binding := newPartEditFixture(t, ctx)
	defer closeStore()
	startPartEdit(t, ctx, svc, binding)
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: binding.EditedArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: binding.Filename,
		Producer: app.Producer{Type: "agent_session", ID: "provider-unrelated"},
		Content:  []byte("# Part 1\n\nInvalid completion artifact.\n"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, partEditedRecoveryEvent(binding)); err != nil {
		t.Fatal(err)
	}

	recovered, ok, err := reporting.LoadCurrentPartEditStart(ctx, svc, partEditStartContract(binding))
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("recovered=%#v ok=%t error=%v, want conflict", recovered, ok, err)
	}
}

func TestLoadCurrentPartEditStartRedactsInvalidCompletionArtifactError(t *testing.T) {
	binding := partEditBinding()
	store := sensitivePartEditRecoveryStore{binding: binding, events: []app.LedgerEvent{
		partEditRecoveryLedger(app.AppendEventRequest{EventID: binding.PendingEventID, MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: testJSON(map[string]any{"report_mode": "long_form"})}),
		partEditRecoveryLedger(app.AppendEventRequest{EventID: binding.PlanEventID, MissionID: binding.MissionID, EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: binding.ReportPlanSessionID}, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "report_mode": "long_form", "artifact_id": "art_final"})}),
		partEditRecoveryLedger(app.AppendEventRequest{EventID: binding.SourcePartEventID, MissionID: binding.MissionID, EventType: "report.part.created", Producer: app.Producer{Type: "agent_session", ID: "provider-part"}, Payload: testJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.SourceArtifactID, "part_index": binding.PartIndex})}),
		partEditRecoveryLedger(reporting.BuildPartEditStartedAppendRequest("evt_recovery_start", binding)),
		partEditRecoveryLedger(partEditedRecoveryEvent(binding)),
	}}

	_, _, err := reporting.LoadCurrentPartEditStart(context.Background(), store, partEditStartContract(binding))
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("error=%v, want conflict", err)
	}
	if strings.Contains(err.Error(), "SECRET_TOKEN") {
		t.Fatalf("recovery leaked inner artifact error: %v", err)
	}
}

func partEditStartContract(binding reporting.PartEditBinding) reporting.PartEditStartContract {
	return reporting.PartEditStartContract{
		MissionID: binding.MissionID, CurrentPendingEventID: binding.PendingEventID, PlanEventID: binding.PlanEventID,
		SourcePartEventID: binding.SourcePartEventID, SourceArtifactID: binding.SourceArtifactID, PartIndex: binding.PartIndex,
		IdempotencyKey: binding.IdempotencyKey, RequirementMapEventID: binding.RequirementMapEventID, RequirementMapHash: binding.RequirementMapHash,
		AgentExecutor: binding.AgentExecutor, AgentModel: binding.AgentModel, AgentReasoningEffort: binding.AgentReasoningEffort,
		AgentSelectionSource: binding.AgentSelectionSource, MCPMode: binding.MCPMode,
		ReportSessionPolicy: binding.ReportSessionPolicy, ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
		GenerationGuidanceProfile: binding.GenerationGuidanceProfile, GenerationGuidanceSHA256: binding.GenerationGuidanceSHA256,
		SessionChainKind: binding.SessionChainKind, ReportPlanSessionID: binding.ReportPlanSessionID,
		ForkSourceAgentSessionID:   binding.ForkSourceAgentSessionID,
		ExpectedProviderSessionID:  binding.ProviderSessionID,
		ExcludedProviderSessionIDs: []string{binding.ReportPlanSessionID},
	}
}

type sensitivePartEditRecoveryStore struct {
	binding reporting.PartEditBinding
	events  []app.LedgerEvent
}

func (store sensitivePartEditRecoveryStore) ListEvents(context.Context, string) ([]app.LedgerEvent, error) {
	return store.events, nil
}

func (store sensitivePartEditRecoveryStore) GetRawArtifact(_ context.Context, artifactID string) (app.RawArtifact, error) {
	if artifactID == store.binding.SourceArtifactID {
		return app.RawArtifact{
			ArtifactID: artifactID, MissionID: store.binding.MissionID,
			MediaType: "text/markdown; charset=utf-8", Producer: app.Producer{Type: "agent_session", ID: "provider-part"},
			Content: []byte("# Part 1\n\nSource body.\n"),
		}, nil
	}
	return app.RawArtifact{}, errors.New("SECRET_TOKEN=inner-artifact-error")
}

func partEditRecoveryLedger(req app.AppendEventRequest) app.LedgerEvent {
	return app.LedgerEvent{
		EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Producer: req.Producer,
		CausationEventID: req.CausationEventID, CorrelationID: req.CorrelationID, Payload: req.Payload,
	}
}

func partEditedRecoveryEvent(binding reporting.PartEditBinding) app.AppendEventRequest {
	return app.AppendEventRequest{
		EventID:          "evt_part_edit_invalid_completion",
		MissionID:        binding.MissionID,
		EventType:        reporting.PartEditedEventType,
		Producer:         app.Producer{Type: "agent_session", ID: binding.ProviderSessionID},
		CausationEventID: binding.SourcePartEventID,
		CorrelationID:    binding.IdempotencyKey,
		Payload: testJSON(map[string]any{
			"kind":                            reporting.PartEditedKind,
			"pending_event_id":                binding.PendingEventID,
			"plan_event_id":                   binding.PlanEventID,
			"source_part_event_id":            binding.SourcePartEventID,
			"source_artifact_id":              binding.SourceArtifactID,
			"artifact_id":                     binding.EditedArtifactID,
			"tool_session_id":                 binding.ToolSessionID,
			"provider_session_id":             binding.ProviderSessionID,
			"previous_provider_session_id":    binding.PreviousProviderSessionID,
			"idempotency_key":                 binding.IdempotencyKey,
			"part_index":                      binding.PartIndex,
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
			"operation_count":                 1,
			"source_word_count":               4,
			"edited_word_count":               4,
			"changed":                         true,
			"text":                            "조립된 Part를 별도 편집 단계에서 검토하고 편집본으로 확정했습니다.",
		}),
	}
}

func requestPayload(t *testing.T, req app.AppendEventRequest) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}
