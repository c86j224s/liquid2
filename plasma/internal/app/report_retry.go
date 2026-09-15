package app

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

type existingReportRetry struct{ event ledger.Event }

// Error는 호출자에게 노출 가능한 안정적인 오류 문자열을 반환하며, 민감한 원문이나 provider 응답을 포함하지 않아야 한다.
func (err existingReportRetry) Error() string { return "existing report retry" }

// RequestReportRetry는 리포트 재시도 지속 명령 경계다. 조건부 append는 변할 수
// 있는 모든 조건을 하나의 장부 snapshot에 대해 검사해야 한다.
func (s *Service) RequestReportRetry(ctx context.Context, req reportexecution.ReportRetryRequest) (ledger.Event, error) {
	if err := reportexecution.ValidateReportRetryRequest(req); err != nil {
		return ledger.Event{}, err
	}
	var existing ledger.Event
	appended, err := s.appendLedgerEventsConditionally(ctx, req.MissionID, func(events []ledger.Event) ([]ledger.Event, error) {
		attempts, terminals, err := reportexecution.ReportAttempts(events)
		if err != nil {
			return nil, err
		}
		if matched, ok := reportexecution.RetryIdempotencyMatch(attempts, req); ok {
			existing = matched
			return nil, existingReportRetry{event: matched}
		}
		if reportexecution.RetryRequestIDTaken(attempts, req.RetryRequestID) {
			return nil, fmt.Errorf("%w: retry_request_id is already used with different input", ErrConflict)
		}
		if err := validateNoActiveAgentWork(events); err != nil {
			return nil, fmt.Errorf("%w: report retry is unavailable while work is active", ErrConflict)
		}
		target, ok := attempts[req.FailedPendingEventID]
		if !ok || terminals[target.EventID] != "failed" {
			return nil, fmt.Errorf("%w: retry target must be a failed terminal report attempt", ErrInvalidInput)
		}
		family := strings.TrimSpace(target.PipelineFamily)
		if family != "" && family != "report_il_experimental" {
			return nil, fmt.Errorf("%w: independent report retries are not supported", ErrInvalidInput)
		}
		if target.ReportMode != "long_form" {
			return nil, fmt.Errorf("%w: report retry requires a long-form failed attempt", ErrInvalidInput)
		}
		if err := reportexecution.ValidateRetryLeafAndLineage(attempts, target); err != nil {
			return nil, err
		}
		payload := reportexecution.CopyRetryPayload(target.Payload)
		payload["origin_pending_event_id"] = target.OriginID
		payload["retry_of_pending_event_id"] = target.EventID
		payload["attempt_number"] = target.Attempt + 1
		payload["retry_strategy"] = req.Strategy
		payload["retry_request_id"] = req.RetryRequestID
		payload["resume_stage"] = reportexecution.RetryResumeStage(events, target.EventID, req.Strategy)
		if family == "report_il_experimental" && req.Strategy == "resume_failed" &&
			!reportexecution.ReportILLineageCheckpointAvailable(events, attempts, target) &&
			!reportexecution.ReportILSourceSelectionCheckpointAvailable(events, target.EventID) &&
			!reportexecution.ReportILLegacyCheckpointAvailable(events, target.EventID) {
			return nil, fmt.Errorf("%w: report IL retry has no recoverable checkpoint", ErrInvalidInput)
		}
		if req.Strategy == "restart" {
			delete(payload, "resume_stage_artifact_ids")
			delete(payload, "report_session_id")
		}
		event, err := buildLedgerEvent(ledger.AppendRequest{EventID: req.EventID, MissionID: req.MissionID, EventType: "report.draft.pending", Producer: req.Producer, CausationEventID: target.EventID, Payload: mustJSON(payload)})
		if err != nil {
			return nil, err
		}
		return []ledger.Event{event}, nil
	})
	if _, ok := err.(existingReportRetry); ok {
		return existing, nil
	}
	if err != nil {
		return ledger.Event{}, err
	}
	if len(appended) != 1 {
		return ledger.Event{}, fmt.Errorf("%w: retry append failed", ErrInvalidInput)
	}
	return appended[0], nil
}
