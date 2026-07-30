package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type reportRequirementMapService interface {
	SubmitReportRequirementMap(context.Context, app.ReportRequirementMapSubmissionRequest) (app.ReportRequirementMapSubmission, error)
}

type reportRequirementsSubmitInput struct {
	MissionID      string                         `json:"mission_id"`
	SessionID      string                         `json:"session_id"`
	PendingEventID string                         `json:"pending_event_id"`
	PlanEventID    string                         `json:"plan_event_id"`
	IdempotencyKey string                         `json:"idempotency_key"`
	Producer       app.Producer                   `json:"producer"`
	RequirementMap reporting.ReportRequirementMap `json:"requirement_map"`
}

func (server *Server) callReportRequirementsSubmit(ctx context.Context, call ToolCall) ToolResult {
	binding := server.reportRequirementMapBinding
	if !server.reportRequirementToolAvailable() {
		return errorResult(call.Name, server.binding.MissionID, "binding", "report requirement tool binding is incomplete", false, nil)
	}
	attempt, allowed := server.consumeReportRequirementParsedCall()
	if !allowed {
		return errorResult(call.Name, server.binding.MissionID, "validation", "report requirement parsed-call limit is exhausted", false, nil)
	}
	var input reportRequirementsSubmitInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return server.reportRequirementValidationError(call.Name, server.binding.MissionID, "report requirement arguments are invalid")
	}
	if input.MissionID != binding.MissionID || input.SessionID != binding.ToolSessionID || input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID || input.IdempotencyKey != binding.IdempotencyKey || input.Producer != binding.Producer {
		return errorResult(call.Name, input.MissionID, "binding", "report requirement call does not match the runner binding", false, nil)
	}
	plan, err := server.boundSectionalReportPlan(ctx, binding)
	if err != nil {
		return errorResult(call.Name, input.MissionID, "binding", "bound report plan is unavailable", false, nil)
	}
	requirementMap, err := reporting.NormalizeReportRequirementMap(input.RequirementMap, plan)
	if err != nil {
		return server.reportRequirementValidationError(call.Name, input.MissionID, "report requirement map is invalid")
	}
	mapHash, encoded, err := reporting.ReportRequirementMapHash(requirementMap)
	if err != nil {
		return server.reportRequirementValidationError(call.Name, input.MissionID, "report requirement map is invalid")
	}
	argumentsHash, err := canonicalArgumentsHash(call.Arguments)
	if err != nil {
		return server.reportRequirementValidationError(call.Name, input.MissionID, "report requirement arguments are invalid")
	}
	svc, ok := server.service.(reportRequirementMapService)
	if !ok {
		return errorResult(call.Name, input.MissionID, "capability", "durable report requirement service is unavailable", false, nil)
	}
	result, err := svc.SubmitReportRequirementMap(ctx, app.ReportRequirementMapSubmissionRequest{
		EventID: newMCPID("evt"), MissionID: input.MissionID, PendingEventID: input.PendingEventID, PlanEventID: input.PlanEventID,
		ToolSessionID: binding.ToolSessionID, PreviousProviderSessionID: binding.PreviousProviderSessionID,
		AgentExecutor: binding.AgentExecutor, AgentModel: binding.AgentModel, AgentReasoningEffort: binding.AgentReasoningEffort,
		IdempotencyKey: input.IdempotencyKey, ArgumentsHash: argumentsHash, RequirementMapHash: mapHash,
		RequirementMap: encoded, ReviewedEventIDs: requirementMap.ReviewedEventIDs, Attempt: attempt, ToolProducer: input.Producer,
	})
	if err != nil {
		if errors.Is(err, app.ErrInvalidInput) {
			return server.reportRequirementValidationError(call.Name, input.MissionID, "reviewed events or requirement bindings are invalid")
		}
		kind := "storage"
		if errors.Is(err, app.ErrConflict) {
			kind = "conflict"
		}
		return errorResult(call.Name, input.MissionID, kind, "report requirement mapping was rejected", false, nil)
	}
	return ToolResult{
		ToolName: call.Name, MissionID: input.MissionID, CreatedEventIDs: []string{result.Event.EventID},
		Content: map[string]any{"requirement_map_event_id": result.Event.EventID, "requirement_map_hash": mapHash, "replay": result.Replay},
	}
}

func (server *Server) reportRequirementToolAvailable() bool {
	if !server.toolEnabled(ToolReportRequirementsSubmit) {
		return false
	}
	return ValidateReportRequirementMapBinding(server.binding, server.reportRequirementMapBinding) == nil
}

func (server *Server) boundSectionalReportPlan(ctx context.Context, binding reporting.ReportRequirementMapBinding) (reporting.SectionalReportPlan, error) {
	events, err := server.service.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return reporting.SectionalReportPlan{}, err
	}
	for _, event := range events {
		if event.EventID != binding.PlanEventID || event.EventType != "report.plan.created" {
			continue
		}
		var payload struct {
			PendingEventID string                        `json:"pending_event_id"`
			Plan           reporting.SectionalReportPlan `json:"plan"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil || payload.PendingEventID != binding.PendingEventID {
			break
		}
		return reporting.NormalizeSectionalReportPlan(payload.Plan)
	}
	return reporting.SectionalReportPlan{}, fmt.Errorf("%w: bound report plan was not found", app.ErrInvalidInput)
}

func (server *Server) reportRequirementValidationError(tool, missionID, message string) ToolResult {
	server.mu.Lock()
	attempt := server.reportRequirementParsedCalls
	server.mu.Unlock()
	return errorResult(tool, missionID, "validation", message, attempt < 3, nil)
}

func (server *Server) consumeReportRequirementParsedCall() (int, bool) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.reportRequirementParsedCalls++
	return server.reportRequirementParsedCalls, server.reportRequirementParsedCalls <= 3
}
