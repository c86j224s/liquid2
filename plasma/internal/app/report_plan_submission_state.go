package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
)

type reportPlanSubmissionPayload struct {
	SchemaVersion             string          `json:"schema_version"`
	PendingEventID            string          `json:"pending_event_id"`
	ReportMode                string          `json:"report_mode"`
	ToolSessionID             string          `json:"tool_session_id"`
	PreviousProviderSessionID string          `json:"previous_provider_session_id,omitempty"`
	AgentExecutor             string          `json:"agent_executor"`
	AgentModel                string          `json:"agent_model,omitempty"`
	AgentReasoningEffort      string          `json:"agent_reasoning_effort,omitempty"`
	ToolProducer              Producer        `json:"tool_producer"`
	IdempotencyKey            string          `json:"idempotency_key"`
	ArgumentsHash             string          `json:"arguments_hash"`
	PlanHash                  string          `json:"plan_hash"`
	Plan                      json.RawMessage `json:"plan"`
	Attempt                   int             `json:"attempt"`
}

type reportPlanCanonicalPointer struct {
	SubmissionEventID string `json:"submission_event_id"`
	PlanHash          string `json:"plan_hash"`
	ArgumentsHash     string `json:"arguments_hash"`
	ToolSessionID     string `json:"tool_session_id"`
	IdempotencyKey    string `json:"idempotency_key"`
}

func validateReportPlanSubmissionRequest(req ReportPlanSubmissionRequest) error {
	if validateID("mis_", req.MissionID) != nil || !strings.HasPrefix(req.EventID, "evt_") || req.PendingEventID == "" || req.ToolSessionID == "" || req.AgentExecutor == "" || req.IdempotencyKey == "" || req.ArgumentsHash == "" || req.PlanHash == "" || !json.Valid(req.Plan) || req.Attempt < 1 {
		return fmt.Errorf("%w: incomplete report plan submission", ErrInvalidInput)
	}
	if req.ReportMode != "planned" && req.ReportMode != "long_form" {
		return fmt.Errorf("%w: unsupported report mode", ErrInvalidInput)
	}
	if req.ToolProducer.Type != "agent_session" || req.ToolProducer.ID != req.ToolSessionID {
		return fmt.Errorf("%w: report plan producer binding mismatch", ErrInvalidInput)
	}
	return nil
}

func openReportPlanPending(events []LedgerEvent, pendingID, mode, agentExecutor string) (LedgerEvent, error) {
	pending, found := reportPendingEvent(events, pendingID)
	if !found || pending.EventType != "report.draft.pending" {
		return LedgerEvent{}, fmt.Errorf("%w: report pending event does not exist", ErrInvalidInput)
	}
	var payload struct {
		ReportMode    string `json:"report_mode"`
		AgentExecutor string `json:"agent_executor"`
	}
	if json.Unmarshal(pending.Payload, &payload) != nil {
		return LedgerEvent{}, fmt.Errorf("%w: report pending mode mismatch", ErrConflict)
	}
	if strings.TrimSpace(payload.ReportMode) == "" {
		payload.ReportMode = "planned"
	}
	if payload.ReportMode != mode {
		return LedgerEvent{}, fmt.Errorf("%w: report pending mode mismatch", ErrConflict)
	}
	if expected := strings.TrimSpace(payload.AgentExecutor); expected != "" && expected != strings.TrimSpace(agentExecutor) {
		return LedgerEvent{}, fmt.Errorf("%w: report pending executor mismatch", ErrConflict)
	}
	if _, closed := ledgerstate.CompletedReportPendingEventIDs(ledgerStateEventsFromApp(events))[pendingID]; closed {
		return LedgerEvent{}, fmt.Errorf("%w: report pending event is finalized", ErrConflict)
	}
	if canonicalReportPlanEvent(events, pendingID).EventID != "" {
		return LedgerEvent{}, fmt.Errorf("%w: report plan is already canonical", ErrConflict)
	}
	return pending, nil
}

func canonicalReportPlanEvent(events []LedgerEvent, pendingID string) LedgerEvent {
	for _, event := range events {
		if event.EventType != "report.plan.created" {
			continue
		}
		var payload struct {
			PendingEventID string `json:"pending_event_id"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && payload.PendingEventID == pendingID {
			return event
		}
	}
	return LedgerEvent{}
}

func matchingReportPlanSubmission(events []LedgerEvent, req PromoteReportPlanRequest) (LedgerEvent, error) {
	for _, event := range events {
		if event.EventType != "report.plan.submitted" || event.EventID != req.SubmissionEventID {
			continue
		}
		var payload reportPlanSubmissionPayload
		if json.Unmarshal(event.Payload, &payload) != nil || !sameReportPlanPromotionBinding(payload, req) {
			return LedgerEvent{}, fmt.Errorf("%w: report plan submission binding mismatch", ErrConflict)
		}
		return event, nil
	}
	return LedgerEvent{}, fmt.Errorf("%w: matching report plan submission is missing", ErrConflict)
}

func reqSchema(ReportPlanSubmissionRequest) string { return ReportPlanSubmissionSchemaVersion }

func sameReportPlanSubmissionBinding(payload reportPlanSubmissionPayload, req ReportPlanSubmissionRequest) bool {
	return payload.PendingEventID == req.PendingEventID && payload.ReportMode == req.ReportMode &&
		payload.ToolSessionID == req.ToolSessionID && payload.PreviousProviderSessionID == req.PreviousProviderSessionID &&
		payload.AgentExecutor == req.AgentExecutor && payload.AgentModel == req.AgentModel &&
		payload.AgentReasoningEffort == req.AgentReasoningEffort && payload.ToolProducer == req.ToolProducer &&
		payload.IdempotencyKey == req.IdempotencyKey && payload.ArgumentsHash == req.ArgumentsHash && payload.PlanHash == req.PlanHash
}

func sameReportPlanPromotionBinding(payload reportPlanSubmissionPayload, req PromoteReportPlanRequest) bool {
	return payload.PendingEventID == req.PendingEventID && payload.ReportMode == req.ReportMode &&
		payload.ToolSessionID == req.ToolSessionID && payload.PreviousProviderSessionID == req.PreviousProviderSessionID &&
		payload.AgentExecutor == req.AgentExecutor && payload.AgentModel == req.AgentModel &&
		payload.AgentReasoningEffort == req.AgentReasoningEffort && payload.IdempotencyKey == req.IdempotencyKey &&
		payload.ArgumentsHash == req.ArgumentsHash && payload.PlanHash == req.PlanHash
}
