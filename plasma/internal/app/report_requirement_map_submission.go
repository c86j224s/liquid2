package app

import (
	"context"
	"encoding/json"
	"fmt"
)

func (s *Service) SubmitReportRequirementMap(ctx context.Context, req ReportRequirementMapSubmissionRequest) (ReportRequirementMapSubmission, error) {
	store, ok := s.store.(ConditionalLedgerStore)
	if !ok {
		return ReportRequirementMapSubmission{}, fmt.Errorf("%w: conditional ledger store is required for report requirement mapping", ErrInvalidInput)
	}
	if err := validateReportRequirementMapRequest(req); err != nil {
		return ReportRequirementMapSubmission{}, err
	}
	var replay LedgerEvent
	appended, err := store.AppendLedgerEventsConditionally(ctx, req.MissionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		pendingIndex, planIndex, err := validateReportRequirementMapSlot(events, req)
		if err != nil {
			return nil, err
		}
		if err := validateReviewedReportRequirementEvents(events, pendingIndex, req); err != nil {
			return nil, err
		}
		for _, event := range events {
			if event.EventType != "report.requirements.mapped" {
				continue
			}
			var payload reportRequirementMapPayload
			if json.Unmarshal(event.Payload, &payload) != nil {
				continue
			}
			sameSlot := payload.PendingEventID == req.PendingEventID && payload.PlanEventID == req.PlanEventID
			if !sameSlot && payload.IdempotencyKey != req.IdempotencyKey {
				continue
			}
			if !sameReportRequirementMapBinding(payload, req) {
				return nil, fmt.Errorf("%w: report requirement mapping differs from existing mapping", ErrConflict)
			}
			replay = event
			return nil, nil
		}
		if reportStagesStartedAfterPlan(events, planIndex, req.PendingEventID, req.PlanEventID) {
			return nil, fmt.Errorf("%w: report requirement mapping must precede section generation", ErrConflict)
		}
		payload := reportRequirementMapPayload{
			SchemaVersion:             ReportRequirementMapSubmissionSchemaVersion,
			Kind:                      "sectional_markdown_report_requirement_map",
			PendingEventID:            req.PendingEventID,
			PlanEventID:               req.PlanEventID,
			ToolSessionID:             req.ToolSessionID,
			PreviousProviderSessionID: req.PreviousProviderSessionID,
			AgentExecutor:             req.AgentExecutor,
			AgentModel:                req.AgentModel,
			AgentReasoningEffort:      req.AgentReasoningEffort,
			IdempotencyKey:            req.IdempotencyKey,
			ArgumentsHash:             req.ArgumentsHash,
			RequirementMapHash:        req.RequirementMapHash,
			RequirementMap:            append(json.RawMessage(nil), req.RequirementMap...),
			Attempt:                   req.Attempt,
			Text:                      "확정된 장문 개요에 사용자 출력 요구를 연결했습니다.",
		}
		encoded, _ := json.Marshal(payload)
		event, err := buildLedgerEvent(AppendEventRequest{
			EventID: req.EventID, MissionID: req.MissionID, EventType: "report.requirements.mapped",
			Producer: req.ToolProducer, CausationEventID: req.PlanEventID, CorrelationID: req.PendingEventID, Payload: encoded,
		})
		if err != nil {
			return nil, err
		}
		return []LedgerEvent{event}, nil
	})
	if err != nil {
		return ReportRequirementMapSubmission{}, err
	}
	if replay.EventID != "" {
		return ReportRequirementMapSubmission{Event: replay, Replay: true}, nil
	}
	if len(appended) != 1 {
		return ReportRequirementMapSubmission{}, fmt.Errorf("%w: report requirement mapping was not appended", ErrConflict)
	}
	return ReportRequirementMapSubmission{Event: appended[0]}, nil
}

func (s *Service) SelectReportRequirementMap(ctx context.Context, query ReportRequirementMapQuery) (ReportRequirementMapSelection, error) {
	events, err := s.ListEvents(ctx, query.MissionID)
	if err != nil {
		return ReportRequirementMapSelection{}, err
	}
	matches := []ReportRequirementMapSelection{}
	for _, event := range events {
		if event.EventType != "report.requirements.mapped" {
			continue
		}
		var payload reportRequirementMapPayload
		if json.Unmarshal(event.Payload, &payload) != nil {
			continue
		}
		if payload.PendingEventID == query.PendingEventID && payload.PlanEventID == query.PlanEventID && payload.ToolSessionID == query.ToolSessionID && payload.PreviousProviderSessionID == query.PreviousProviderSessionID && payload.AgentExecutor == query.AgentExecutor && payload.AgentModel == query.AgentModel && payload.AgentReasoningEffort == query.AgentReasoningEffort && payload.IdempotencyKey == query.IdempotencyKey {
			matches = append(matches, ReportRequirementMapSelection{Event: event, RequirementMapHash: payload.RequirementMapHash, RequirementMap: append(json.RawMessage(nil), payload.RequirementMap...)})
		}
	}
	if len(matches) != 1 {
		return ReportRequirementMapSelection{}, fmt.Errorf("%w: expected exactly one current report requirement mapping", ErrConflict)
	}
	return matches[0], nil
}
