package reportplan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reporting/reportdocument"
	"io"
	"strings"
	"sync"
)

// Handler owns strict plan execution and request-local parsed-call accounting.
type Handler struct {
	Capability            func() (Service, bool)
	Binding               func() Binding
	MissionID             func() string
	Available             func() bool
	ArgumentsHash         func(json.RawMessage) (string, error)
	NewID                 func(string) string
	ErrorResult           func(string, string, string, string, bool, []string) wire.ToolResult
	mu                    sync.Mutex
	reportPlanParsedCalls int
}
type Binding struct {
	PendingEventID            string
	ReportMode                string
	IdempotencyKey            string
	ToolSessionID             string
	PreviousProviderSessionID string
	AgentExecutor             string
	AgentModel                string
	AgentReasoningEffort      string
	RequireWritingContract    bool
}

type Service interface {
	SubmitReportPlan(context.Context, app.ReportPlanSubmissionRequest) (app.ReportPlanSubmission, error)
	ValidateReportPlanRefs(context.Context, string, []reportdocument.ReportBlockSourceRefs) error
}

type reportPlanSubmitInput struct {
	MissionID      string          `json:"mission_id"`
	SessionID      string          `json:"session_id"`
	PendingEventID string          `json:"pending_event_id"`
	ReportMode     string          `json:"report_mode"`
	IdempotencyKey string          `json:"idempotency_key"`
	Producer       ledger.Producer `json:"producer"`
	Plan           json.RawMessage `json:"plan"`
}

func (server *Handler) Submit(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding := server.Binding()
	if !server.Available() {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", "report plan tool binding is incomplete", false, nil)
	}
	attempt, allowed := server.consumeReportPlanParsedCall()
	if !allowed {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "report plan parsed-call limit is exhausted", false, nil)
	}
	var input reportPlanSubmitInput
	if err := decodeJSON(call.Arguments, &input); err != nil {
		return server.reportPlanValidationError(call.Name, server.MissionID(), "report plan arguments are invalid")
	}
	if !reportPlanBindingFieldsComplete(input) {
		return server.reportPlanValidationError(call.Name, server.MissionID(), "report plan arguments are missing required binding fields")
	}
	if input.MissionID != server.MissionID() || input.SessionID != binding.ToolSessionID || input.PendingEventID != binding.PendingEventID || input.IdempotencyKey != binding.IdempotencyKey || strings.TrimSpace(input.Producer.Type) != "agent_session" || strings.TrimSpace(input.Producer.ID) != binding.ToolSessionID {
		return server.ErrorResult(call.Name, input.MissionID, "binding", "report plan call does not match the runner binding", false, nil)
	}
	if input.ReportMode != "planned" && input.ReportMode != "long_form" {
		return server.reportPlanValidationError(call.Name, input.MissionID, "unsupported report_mode")
	}
	if input.ReportMode != binding.ReportMode {
		return server.ErrorResult(call.Name, input.MissionID, "binding", "report mode does not match the runner binding", false, nil)
	}
	planPayload := unwrapStringWrappedReportPlan(input.Plan)
	var plan any
	if input.ReportMode == "planned" {
		var value reporting.ReportPlan
		if decodeJSON(planPayload, &value) != nil {
			return server.reportPlanValidationError(call.Name, input.MissionID, "planned report plan is invalid")
		}
		normalized, err := reporting.NormalizeReportPlan(value)
		if err != nil {
			return server.reportPlanValidationError(call.Name, input.MissionID, "planned report plan is invalid")
		}
		plan = normalized
	} else {
		var value reporting.SectionalReportPlan
		if decodeJSON(planPayload, &value) != nil {
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
	svc, ok := server.Capability()
	if !ok {
		return server.ErrorResult(call.Name, input.MissionID, "capability", "durable report plan service is unavailable", false, nil)
	}
	if err := svc.ValidateReportPlanRefs(ctx, input.MissionID, reporting.ReportPlanRefs(plan)); err != nil {
		return server.reportPlanValidationError(call.Name, input.MissionID, "report plan references are invalid")
	}
	planHash, encoded, err := reporting.ReportPlanHash(plan)
	if err != nil {
		return server.reportPlanValidationError(call.Name, input.MissionID, "report plan is invalid")
	}
	argumentsHash, err := server.ArgumentsHash(call.Arguments)
	if err != nil {
		return server.reportPlanValidationError(call.Name, input.MissionID, "report plan arguments are invalid")
	}
	result, err := svc.SubmitReportPlan(ctx, app.ReportPlanSubmissionRequest{
		EventID: server.NewID("evt"), MissionID: input.MissionID, PendingEventID: input.PendingEventID, ReportMode: input.ReportMode,
		ToolSessionID: binding.ToolSessionID, PreviousProviderSessionID: binding.PreviousProviderSessionID, AgentExecutor: binding.AgentExecutor,
		AgentModel: binding.AgentModel, AgentReasoningEffort: binding.AgentReasoningEffort,
		IdempotencyKey: input.IdempotencyKey, ArgumentsHash: argumentsHash, PlanHash: planHash, Plan: encoded, Attempt: attempt, ToolProducer: input.Producer,
	})
	if err != nil {
		kind := "storage"
		if errors.Is(err, app.ErrConflict) {
			kind = "conflict"
		}
		return server.ErrorResult(call.Name, input.MissionID, kind, "report plan submission was rejected", false, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: input.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: map[string]any{"submission_event_id": result.Event.EventID, "plan_hash": planHash, "replay": result.Replay}}
}

func reportPlanBindingFieldsComplete(input reportPlanSubmitInput) bool {
	return strings.TrimSpace(input.MissionID) != "" &&
		strings.TrimSpace(input.SessionID) != "" &&
		strings.TrimSpace(input.PendingEventID) != "" &&
		strings.TrimSpace(input.IdempotencyKey) != "" &&
		strings.TrimSpace(input.Producer.Type) != "" &&
		strings.TrimSpace(input.Producer.ID) != ""
}

func unwrapStringWrappedReportPlan(payload json.RawMessage) json.RawMessage {
	var encoded string
	if decodeJSON(payload, &encoded) != nil {
		return payload
	}
	return json.RawMessage(encoded)
}

func (server *Handler) reportPlanValidationError(tool, missionID, message string) wire.ToolResult {
	server.mu.Lock()
	attempt := server.reportPlanParsedCalls
	server.mu.Unlock()
	return server.ErrorResult(tool, missionID, "validation", message, attempt < 3, nil)
}

func (server *Handler) AttemptCount() int {
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.reportPlanParsedCalls
}

func (server *Handler) consumeReportPlanParsedCall() (int, bool) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.reportPlanParsedCalls++
	return server.reportPlanParsedCalls, server.reportPlanParsedCalls <= 3
}

func decodeJSON(payload json.RawMessage, target any) error {
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
