package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type reportRequirementMCPService struct {
	*fakeMCPService
	request app.ReportRequirementMapSubmissionRequest
}

func (service *reportRequirementMCPService) SubmitReportRequirementMap(_ context.Context, req app.ReportRequirementMapSubmissionRequest) (app.ReportRequirementMapSubmission, error) {
	service.request = req
	return app.ReportRequirementMapSubmission{Event: app.LedgerEvent{EventID: req.EventID, MissionID: req.MissionID, EventType: reporting.ReportRequirementsMappedEventType}}, nil
}

func TestReportRequirementsSubmitIsBoundToFixedPlan(t *testing.T) {
	binding := reporting.ReportRequirementMapBinding{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_tool",
		PreviousProviderSessionID: "ses_plan", IdempotencyKey: "rrk_once", AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "high",
		Producer: app.Producer{Type: "agent_session", ID: "ses_tool"},
	}
	service := &reportRequirementMCPService{fakeMCPService: &fakeMCPService{ledgerEvents: []app.LedgerEvent{{
		EventID: "evt_plan", MissionID: "mis_1", EventType: "report.plan.created",
		Payload: json.RawMessage(`{"pending_event_id":"evt_pending","plan":{"parts":[{"title":"Part","sections":[{"title":"Section"}]}]}}`),
	}}}}
	server := NewServer(service,
		WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_tool", AgentExecutor: "codex"}),
		WithReportRequirementMapBinding(binding), WithEnabledTools([]string{ToolReportRequirementsSubmit}),
	)
	if !containsString(toolNames(server.ListTools()), ToolReportRequirementsSubmit) {
		t.Fatal("bound report requirement tool was not exposed")
	}
	result := server.Call(context.Background(), ToolCall{Name: ToolReportRequirementsSubmit, Arguments: mustArgs(t, map[string]any{
		"mission_id": "mis_1", "session_id": "ses_tool", "pending_event_id": "evt_pending", "plan_event_id": "evt_plan", "idempotency_key": "rrk_once",
		"producer": map[string]any{"type": "agent_session", "id": "ses_tool"},
		"requirement_map": map[string]any{
			"reviewed_event_ids": []string{"evt_pending"},
			"requirements":       []any{map[string]any{"requirement_id": "req_table", "instruction": "include a table", "source_event_ids": []string{"evt_pending"}, "owner": map[string]any{"part_index": 1, "section_index": 1}}},
		},
	})})
	if result.Error != nil {
		t.Fatalf("submit returned tool error: %#v", result.Error)
	}
	if service.request.PlanEventID != "evt_plan" || service.request.PreviousProviderSessionID != "ses_plan" || service.request.RequirementMapHash == "" || len(service.request.ReviewedEventIDs) != 1 {
		t.Fatalf("unexpected durable request: %#v", service.request)
	}
}

func TestReportRequirementsSubmitRejectsOutlineMutationAndBindingMismatch(t *testing.T) {
	binding := reporting.ReportRequirementMapBinding{MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_tool", IdempotencyKey: "rrk", AgentExecutor: "codex", Producer: app.Producer{Type: "agent_session", ID: "ses_tool"}}
	service := &reportRequirementMCPService{fakeMCPService: &fakeMCPService{ledgerEvents: []app.LedgerEvent{{EventID: "evt_plan", MissionID: "mis_1", EventType: "report.plan.created", Payload: json.RawMessage(`{"pending_event_id":"evt_pending","plan":{"parts":[{"title":"Part","sections":[{"title":"Section"}]}]}}`)}}}}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_tool", AgentExecutor: "codex"}), WithReportRequirementMapBinding(binding), WithEnabledTools([]string{ToolReportRequirementsSubmit}))
	base := map[string]any{
		"mission_id": "mis_1", "session_id": "ses_tool", "pending_event_id": "evt_pending", "plan_event_id": "evt_plan", "idempotency_key": "rrk",
		"producer":        map[string]any{"type": "agent_session", "id": "ses_tool"},
		"requirement_map": map[string]any{"reviewed_event_ids": []string{"evt_pending"}, "requirements": []any{map[string]any{"requirement_id": "req_one", "instruction": "one", "source_event_ids": []string{"evt_pending"}, "owner": map[string]any{"part_index": 1, "section_index": 2}}}},
	}
	result := server.Call(context.Background(), ToolCall{Name: ToolReportRequirementsSubmit, Arguments: mustArgs(t, base)})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("out-of-outline owner was accepted: %#v", result)
	}
	base["plan_event_id"] = "evt_other"
	result = server.Call(context.Background(), ToolCall{Name: ToolReportRequirementsSubmit, Arguments: mustArgs(t, base)})
	if result.Error == nil || result.Error.ErrorKind != "binding" {
		t.Fatalf("binding mismatch was accepted: %#v", result)
	}
}

func TestReportRequirementsToolRequiresCompleteBinding(t *testing.T) {
	server := NewServer(&fakeMCPService{}, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_tool", AgentExecutor: "codex"}), WithEnabledTools([]string{ToolReportRequirementsSubmit}))
	if containsString(toolNames(server.ListTools()), ToolReportRequirementsSubmit) {
		t.Fatal("unbound report requirement tool was exposed")
	}
	if !json.Valid(schemaReportRequirementsSubmit) {
		t.Fatal("report requirement schema is invalid JSON")
	}
}

func TestReportRequirementsToolDoesNotFallbackToReportPlanBinding(t *testing.T) {
	server := NewServer(&fakeMCPService{},
		WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_tool", AgentExecutor: "codex"}),
		WithReportPlanBinding(ReportPlanBinding{
			PendingEventID: "evt_pending", ReportMode: "long_form", IdempotencyKey: "rrk_plan",
			ToolSessionID: "ses_tool", AgentExecutor: "codex",
		}),
		WithEnabledTools([]string{ToolReportRequirementsSubmit}),
	)
	if containsString(toolNames(server.ListTools()), ToolReportRequirementsSubmit) {
		t.Fatal("report requirement tool fell back to report plan binding")
	}
	result := server.Call(context.Background(), ToolCall{Name: ToolReportRequirementsSubmit, Arguments: mustArgs(t, map[string]any{
		"mission_id": "mis_1", "session_id": "ses_tool", "pending_event_id": "evt_pending", "plan_event_id": "evt_plan", "idempotency_key": "rrk_plan",
		"producer":        map[string]any{"type": "agent_session", "id": "ses_tool"},
		"requirement_map": map[string]any{"reviewed_event_ids": []string{"evt_pending"}},
	})})
	if result.Error == nil || result.Error.ErrorKind != "binding" {
		t.Fatalf("unbound requirement submission was accepted: %#v", result)
	}
}
