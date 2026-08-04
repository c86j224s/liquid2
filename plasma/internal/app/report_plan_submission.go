package app

import (
	"context"
	"encoding/json"
	"fmt"
)

// ReportPlanSubmissionSchemaVersion은 report plan MCP 제출 payload의 schema version이다.
const ReportPlanSubmissionSchemaVersion = "plasma.report_plan_submission.v1"

// ReportPlanSubmissionRequest는 report planning agent가 MCP tool로 제출한 계획을
// 장부에 기록하기 위한 입력이다.
//
// PendingEventID, ToolSessionID, IdempotencyKey가 같은 실행 slot을 식별한다.
type ReportPlanSubmissionRequest struct {
	EventID                   string
	MissionID                 string
	PendingEventID            string
	ReportMode                string
	ToolSessionID             string
	PreviousProviderSessionID string
	AgentExecutor             string
	AgentModel                string
	AgentReasoningEffort      string
	IdempotencyKey            string
	ArgumentsHash             string
	PlanHash                  string
	Plan                      json.RawMessage
	Attempt                   int
	ToolProducer              Producer
}

// ReportPlanSubmission은 제출 event와 replay 여부를 반환한다.
type ReportPlanSubmission struct {
	Event  LedgerEvent
	Replay bool
}

// ReportPlanSubmissionQuery는 이미 제출된 report plan을 정확히 하나 선택하기 위한
// binding query다.
type ReportPlanSubmissionQuery struct {
	MissionID, PendingEventID, ReportMode, ToolSessionID, PreviousProviderSessionID string
	AgentExecutor, AgentModel, AgentReasoningEffort, IdempotencyKey                 string
}

// ReportPlanSubmissionSelection은 선택된 report plan event와 hash metadata다.
type ReportPlanSubmissionSelection struct {
	EventID, ArgumentsHash, PlanHash string
	Plan                             json.RawMessage
}

// SelectReportPlanSubmission은 binding이 정확히 일치하는 report plan 제출 하나를 찾는다.
//
// 0개 또는 여러 개가 나오면 runner가 어떤 계획을 promote해야 하는지 모호하므로
// conflict로 처리한다.
func (s *Service) SelectReportPlanSubmission(ctx context.Context, query ReportPlanSubmissionQuery) (ReportPlanSubmissionSelection, error) {
	events, err := s.ListEvents(ctx, query.MissionID)
	if err != nil {
		return ReportPlanSubmissionSelection{}, err
	}
	matches := []ReportPlanSubmissionSelection{}
	for _, event := range events {
		if event.EventType != "report.plan.submitted" {
			continue
		}
		var payload reportPlanSubmissionPayload
		if json.Unmarshal(event.Payload, &payload) != nil {
			continue
		}
		if payload.PendingEventID == query.PendingEventID && payload.ReportMode == query.ReportMode && payload.ToolSessionID == query.ToolSessionID && payload.PreviousProviderSessionID == query.PreviousProviderSessionID && payload.AgentExecutor == query.AgentExecutor && payload.AgentModel == query.AgentModel && payload.AgentReasoningEffort == query.AgentReasoningEffort && payload.IdempotencyKey == query.IdempotencyKey {
			matches = append(matches, ReportPlanSubmissionSelection{EventID: event.EventID, ArgumentsHash: payload.ArgumentsHash, PlanHash: payload.PlanHash, Plan: append(json.RawMessage(nil), payload.Plan...)})
		}
	}
	if len(matches) != 1 {
		return ReportPlanSubmissionSelection{}, fmt.Errorf("%w: expected exactly one current report plan submission", ErrConflict)
	}
	return matches[0], nil
}

// PromoteReportPlanRequest는 제출된 report plan을 canonical pending event로 승격할 때의
// 입력이다.
type PromoteReportPlanRequest struct {
	MissionID                 string
	PendingEventID            string
	ReportMode                string
	ToolSessionID             string
	PreviousProviderSessionID string
	AgentExecutor             string
	AgentModel                string
	AgentReasoningEffort      string
	IdempotencyKey            string
	ArgumentsHash             string
	PlanHash                  string
	SubmissionEventID         string
	Canonical                 AppendEventRequest
}

// SubmitReportPlan은 report planning MCP 제출을 idempotent하게 장부에 기록한다.
//
// 같은 slot에 다른 binding/hash가 들어오면 충돌로 막고, 동일 제출은 replay로
// 반환한다.
func (s *Service) SubmitReportPlan(ctx context.Context, req ReportPlanSubmissionRequest) (ReportPlanSubmission, error) {
	store, ok := s.store.(ConditionalLedgerStore)
	if !ok {
		return ReportPlanSubmission{}, fmt.Errorf("%w: conditional ledger store is required for report plan submission", ErrInvalidInput)
	}
	if err := validateReportPlanSubmissionRequest(req); err != nil {
		return ReportPlanSubmission{}, err
	}
	var replay LedgerEvent
	appended, err := store.AppendLedgerEventsConditionally(ctx, req.MissionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		pending, err := openReportPlanPending(events, req.PendingEventID, req.ReportMode, req.AgentExecutor)
		if err != nil {
			return nil, err
		}
		_ = pending
		for _, event := range events {
			if event.EventType != "report.plan.submitted" {
				continue
			}
			var payload reportPlanSubmissionPayload
			if json.Unmarshal(event.Payload, &payload) != nil {
				continue
			}
			sameSlot := payload.PendingEventID == req.PendingEventID && payload.ToolSessionID == req.ToolSessionID
			if !sameSlot && payload.IdempotencyKey != req.IdempotencyKey {
				continue
			}
			if !sameReportPlanSubmissionBinding(payload, req) {
				return nil, fmt.Errorf("%w: report plan submission binding differs from existing submission", ErrConflict)
			}
			replay = event
			return nil, nil
		}
		payload := reportPlanSubmissionPayload{
			SchemaVersion: reqSchema(req), PendingEventID: req.PendingEventID, ReportMode: req.ReportMode,
			ToolSessionID: req.ToolSessionID, PreviousProviderSessionID: req.PreviousProviderSessionID, AgentExecutor: req.AgentExecutor,
			AgentModel: req.AgentModel, AgentReasoningEffort: req.AgentReasoningEffort, ToolProducer: req.ToolProducer,
			IdempotencyKey: req.IdempotencyKey, ArgumentsHash: req.ArgumentsHash, PlanHash: req.PlanHash,
			Plan: append(json.RawMessage(nil), req.Plan...), Attempt: req.Attempt,
		}
		encoded, _ := json.Marshal(payload)
		event, err := buildLedgerEvent(AppendEventRequest{EventID: req.EventID, MissionID: req.MissionID, EventType: "report.plan.submitted", Producer: Producer{Type: "mcp_server", ID: "plasma.report.plan.submit"}, CausationEventID: req.PendingEventID, CorrelationID: req.PendingEventID, Payload: encoded})
		if err != nil {
			return nil, err
		}
		return []LedgerEvent{event}, nil
	})
	if err != nil {
		return ReportPlanSubmission{}, err
	}
	if replay.EventID != "" {
		return ReportPlanSubmission{Event: replay, Replay: true}, nil
	}
	if len(appended) != 1 {
		return ReportPlanSubmission{}, fmt.Errorf("%w: report plan submission was not appended", ErrConflict)
	}
	return ReportPlanSubmission{Event: appended[0]}, nil
}

// PromoteReportPlan는 선택한 report plan을 현재 plan으로 승격한다.
func (s *Service) PromoteReportPlan(ctx context.Context, req PromoteReportPlanRequest) (LedgerEvent, error) {
	store, ok := s.store.(ConditionalLedgerStore)
	if !ok {
		return LedgerEvent{}, fmt.Errorf("%w: conditional ledger store is required for report plan promotion", ErrInvalidInput)
	}
	var existing LedgerEvent
	appended, err := store.AppendLedgerEventsConditionally(ctx, req.MissionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		if canonical := canonicalReportPlanEvent(events, req.PendingEventID); canonical.EventID != "" {
			submission, err := matchingReportPlanSubmission(events, req)
			if err != nil {
				return nil, err
			}
			var payload struct {
				Submission reportPlanCanonicalPointer `json:"plan_submission"`
			}
			if json.Unmarshal(canonical.Payload, &payload) == nil && payload.Submission == (reportPlanCanonicalPointer{
				SubmissionEventID: submission.EventID,
				PlanHash:          req.PlanHash,
				ArgumentsHash:     req.ArgumentsHash,
				ToolSessionID:     req.ToolSessionID,
				IdempotencyKey:    req.IdempotencyKey,
			}) {
				existing = canonical
				return nil, nil
			}
			return nil, fmt.Errorf("%w: report plan already promoted from another submission", ErrConflict)
		}
		if _, err := openReportPlanPending(events, req.PendingEventID, req.ReportMode, req.AgentExecutor); err != nil {
			return nil, err
		}
		submission, err := matchingReportPlanSubmission(events, req)
		if err != nil {
			return nil, err
		}
		if req.Canonical.EventType != "report.plan.created" || req.Canonical.MissionID != req.MissionID {
			return nil, fmt.Errorf("%w: invalid canonical report plan event", ErrInvalidInput)
		}
		var payload map[string]any
		if json.Unmarshal(req.Canonical.Payload, &payload) != nil {
			return nil, fmt.Errorf("%w: invalid canonical report plan payload", ErrInvalidInput)
		}
		payload["plan_submission"] = reportPlanCanonicalPointer{SubmissionEventID: submission.EventID, PlanHash: req.PlanHash, ArgumentsHash: req.ArgumentsHash, ToolSessionID: req.ToolSessionID, IdempotencyKey: req.IdempotencyKey}
		req.Canonical.Payload, _ = json.Marshal(payload)
		event, err := buildLedgerEvent(req.Canonical)
		if err != nil {
			return nil, err
		}
		return []LedgerEvent{event}, nil
	})
	if err != nil {
		return LedgerEvent{}, err
	}
	if existing.EventID != "" {
		return existing, nil
	}
	if len(appended) != 1 {
		return LedgerEvent{}, fmt.Errorf("%w: report plan promotion was not appended", ErrConflict)
	}
	return appended[0], nil
}
