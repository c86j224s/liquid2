package reportrequirements

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"sync"
)

// Handler owns requirement submission execution and parsed-call accounting.
// Root supplies the protocol availability gate and exact error encoding.
type Handler struct {
	ListEvents                   func(context.Context, string) ([]ledger.Event, error)
	Capability                   func() (Service, bool)
	Binding                      func() reporting.ReportRequirementMapBinding
	MissionID                    func() string
	Available                    func() bool
	Decode                       func(json.RawMessage, any) error
	ArgumentsHash                func(json.RawMessage) (string, error)
	NewID                        func(string) string
	ErrorResult                  func(string, string, string, string, bool, []string) wire.ToolResult
	mu                           sync.Mutex
	reportRequirementParsedCalls int
}
type Service interface {
	SubmitReportRequirementMap(context.Context, app.ReportRequirementMapSubmissionRequest) (app.ReportRequirementMapSubmission, error)
}

type reportRequirementsSubmitInput struct {
	MissionID      string                         `json:"mission_id"`
	SessionID      string                         `json:"session_id"`
	PendingEventID string                         `json:"pending_event_id"`
	PlanEventID    string                         `json:"plan_event_id"`
	IdempotencyKey string                         `json:"idempotency_key"`
	Producer       ledger.Producer                `json:"producer"`
	RequirementMap reporting.ReportRequirementMap `json:"requirement_map"`
}

func (server *Handler) Submit(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding := server.Binding()
	if !server.Available() {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", "report requirement tool binding is incomplete", false, nil)
	}
	attempt, allowed := server.consumeReportRequirementParsedCall()
	if !allowed {
		return server.ErrorResult(call.Name, server.MissionID(), "validation", "report requirement parsed-call limit is exhausted", false, nil)
	}
	var input reportRequirementsSubmitInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.reportRequirementValidationError(call.Name, server.MissionID(), "report requirement arguments are invalid")
	}
	if input.MissionID != binding.MissionID || input.SessionID != binding.ToolSessionID || input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID || input.IdempotencyKey != binding.IdempotencyKey || input.Producer != binding.Producer {
		return server.ErrorResult(call.Name, input.MissionID, "binding", "report requirement call does not match the runner binding", false, nil)
	}
	plan, err := server.boundSectionalReportPlan(ctx, binding)
	if err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "binding", "bound report plan is unavailable", false, nil)
	}
	requirementMap, err := reporting.NormalizeReportRequirementMap(input.RequirementMap, plan)
	if err != nil {
		return server.reportRequirementValidationError(call.Name, input.MissionID, "report requirement map is invalid")
	}
	mapHash, encoded, err := reporting.ReportRequirementMapHash(requirementMap)
	if err != nil {
		return server.reportRequirementValidationError(call.Name, input.MissionID, "report requirement map is invalid")
	}
	argumentsHash, err := server.ArgumentsHash(call.Arguments)
	if err != nil {
		return server.reportRequirementValidationError(call.Name, input.MissionID, "report requirement arguments are invalid")
	}
	svc, ok := server.Capability()
	if !ok {
		return server.ErrorResult(call.Name, input.MissionID, "capability", "durable report requirement service is unavailable", false, nil)
	}
	result, err := svc.SubmitReportRequirementMap(ctx, app.ReportRequirementMapSubmissionRequest{
		EventID: server.NewID("evt"), MissionID: input.MissionID, PendingEventID: input.PendingEventID, PlanEventID: input.PlanEventID,
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
		return server.ErrorResult(call.Name, input.MissionID, kind, "report requirement mapping was rejected", false, nil)
	}
	return wire.ToolResult{
		ToolName: call.Name, MissionID: input.MissionID, CreatedEventIDs: []string{result.Event.EventID},
		Content: map[string]any{"requirement_map_event_id": result.Event.EventID, "requirement_map_hash": mapHash, "replay": result.Replay},
	}
}

func (server *Handler) boundSectionalReportPlan(ctx context.Context, binding reporting.ReportRequirementMapBinding) (reporting.SectionalReportPlan, error) {
	events, err := server.ListEvents(ctx, binding.MissionID)
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

func (server *Handler) reportRequirementValidationError(tool, missionID, message string) wire.ToolResult {
	server.mu.Lock()
	attempt := server.reportRequirementParsedCalls
	server.mu.Unlock()
	return server.ErrorResult(tool, missionID, "validation", message, attempt < 3, nil)
}

func (server *Handler) consumeReportRequirementParsedCall() (int, bool) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.reportRequirementParsedCalls++
	return server.reportRequirementParsedCalls, server.reportRequirementParsedCalls <= 3
}
