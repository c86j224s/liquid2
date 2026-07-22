package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type reportPlanService interface {
	SubmitReportPlan(context.Context, app.ReportPlanSubmissionRequest) (app.ReportPlanSubmission, error)
	ValidateReportPlanRefs(context.Context, string, []app.ReportBlockSourceRefs) error
}

type reportPlanSubmitInput struct {
	MissionID      string          `json:"mission_id"`
	SessionID      string          `json:"session_id"`
	PendingEventID string          `json:"pending_event_id"`
	ReportMode     string          `json:"report_mode"`
	IdempotencyKey string          `json:"idempotency_key"`
	Producer       app.Producer    `json:"producer"`
	Plan           json.RawMessage `json:"plan"`
}

func (server *Server) callReportPlanSubmit(ctx context.Context, call ToolCall) ToolResult {
	binding := server.reportPlanBinding
	if !binding.complete() || !server.toolEnabled(ToolReportPlanSubmit) {
		return errorResult(call.Name, server.binding.MissionID, "binding", "report plan tool binding is incomplete", false, nil)
	}
	attempt, allowed := server.consumeReportPlanParsedCall()
	if !allowed {
		return errorResult(call.Name, server.binding.MissionID, "validation", "report plan parsed-call limit is exhausted", false, nil)
	}
	var input reportPlanSubmitInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return server.reportPlanValidationError(call.Name, server.binding.MissionID, "report plan arguments are invalid")
	}
	if input.MissionID != server.binding.MissionID || input.SessionID != binding.ToolSessionID || input.PendingEventID != binding.PendingEventID || input.IdempotencyKey != binding.IdempotencyKey || strings.TrimSpace(input.Producer.Type) != "agent_session" || strings.TrimSpace(input.Producer.ID) != binding.ToolSessionID {
		return errorResult(call.Name, input.MissionID, "binding", "report plan call does not match the runner binding", false, nil)
	}
	if input.ReportMode != "planned" && input.ReportMode != "long_form" {
		return server.reportPlanValidationError(call.Name, input.MissionID, "unsupported report_mode")
	}
	if input.ReportMode != binding.ReportMode {
		return errorResult(call.Name, input.MissionID, "binding", "report mode does not match the runner binding", false, nil)
	}
	planPayload := unwrapStringWrappedReportPlan(input.Plan)
	var plan any
	if input.ReportMode == "planned" {
		var value reporting.ReportPlan
		if decodeReportPlanJSON(planPayload, &value) != nil {
			return server.reportPlanValidationError(call.Name, input.MissionID, "planned report plan is invalid")
		}
		normalized, err := reporting.NormalizeReportPlan(value)
		if err != nil {
			return server.reportPlanValidationError(call.Name, input.MissionID, "planned report plan is invalid")
		}
		plan = normalized
	} else {
		var value reporting.SectionalReportPlan
		if decodeReportPlanJSON(planPayload, &value) != nil {
			return server.reportPlanValidationError(call.Name, input.MissionID, "long-form report plan is invalid")
		}
		normalized, err := reporting.NormalizeSectionalReportPlan(value)
		if err != nil {
			return server.reportPlanValidationError(call.Name, input.MissionID, "long-form report plan is incomplete")
		}
		plan = normalized
	}
	if binding.RequireWritingContract {
		if err := reporting.RequireReportWritingContract(plan); err != nil {
			return server.reportPlanValidationError(call.Name, input.MissionID, "report writing contract is required")
		}
	}
	svc, ok := server.service.(reportPlanService)
	if !ok {
		return errorResult(call.Name, input.MissionID, "capability", "durable report plan service is unavailable", false, nil)
	}
	if err := svc.ValidateReportPlanRefs(ctx, input.MissionID, reporting.ReportPlanRefs(plan)); err != nil {
		return server.reportPlanValidationError(call.Name, input.MissionID, "report plan references are invalid")
	}
	planHash, encoded, err := reporting.ReportPlanHash(plan)
	if err != nil {
		return server.reportPlanValidationError(call.Name, input.MissionID, "report plan is invalid")
	}
	argumentsHash, err := canonicalArgumentsHash(call.Arguments)
	if err != nil {
		return server.reportPlanValidationError(call.Name, input.MissionID, "report plan arguments are invalid")
	}
	result, err := svc.SubmitReportPlan(ctx, app.ReportPlanSubmissionRequest{
		EventID: newMCPID("evt"), MissionID: input.MissionID, PendingEventID: input.PendingEventID, ReportMode: input.ReportMode,
		ToolSessionID: binding.ToolSessionID, PreviousProviderSessionID: binding.PreviousProviderSessionID, AgentExecutor: binding.AgentExecutor,
		AgentModel: binding.AgentModel, AgentReasoningEffort: binding.AgentReasoningEffort,
		IdempotencyKey: input.IdempotencyKey, ArgumentsHash: argumentsHash, PlanHash: planHash, Plan: encoded, Attempt: attempt, ToolProducer: input.Producer,
	})
	if err != nil {
		kind := "storage"
		if errors.Is(err, app.ErrConflict) {
			kind = "conflict"
		}
		return errorResult(call.Name, input.MissionID, kind, "report plan submission was rejected", false, nil)
	}
	return ToolResult{ToolName: call.Name, MissionID: input.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: map[string]any{"submission_event_id": result.Event.EventID, "plan_hash": planHash, "replay": result.Replay}}
}

func unwrapStringWrappedReportPlan(payload json.RawMessage) json.RawMessage {
	var encoded string
	if decodeReportPlanJSON(payload, &encoded) != nil {
		return payload
	}
	return json.RawMessage(encoded)
}

func (server *Server) reportPlanValidationError(tool, missionID, message string) ToolResult {
	server.mu.Lock()
	attempt := server.reportPlanParsedCalls
	server.mu.Unlock()
	return errorResult(tool, missionID, "validation", message, attempt < 3, nil)
}

func (server *Server) reportPlanAttemptCount() int {
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.reportPlanParsedCalls
}

func (server *Server) consumeReportPlanParsedCall() (int, bool) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.reportPlanParsedCalls++
	return server.reportPlanParsedCalls, server.reportPlanParsedCalls <= 3
}

func decodeReportPlanJSON(payload json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
