package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const reportRetryLineageLimit = 64

type ReportRetryRequest struct {
	EventID, MissionID, FailedPendingEventID, Strategy, RetryRequestID string
	Producer                                                           Producer
}

type reportAttemptPayload struct {
	OriginID       string `json:"origin_pending_event_id"`
	RetryOf        string `json:"retry_of_pending_event_id"`
	RetryStrategy  string `json:"retry_strategy"`
	RetryRequestID string `json:"retry_request_id"`
	ReportMode     string `json:"report_mode"`
	Attempt        int    `json:"attempt_number"`
}

type reportTerminalPayload struct {
	PendingID string `json:"pending_event_id"`
	Kind      string `json:"kind"`
}

type existingReportRetry struct{ event LedgerEvent }

func (err existingReportRetry) Error() string { return "existing report retry" }

// RequestReportRetry is the durable command boundary for report retry. Its
// conditional append checks all mutable conditions against one ledger snapshot.
func (s *Service) RequestReportRetry(ctx context.Context, req ReportRetryRequest) (LedgerEvent, error) {
	if err := validateReportRetryRequest(req); err != nil {
		return LedgerEvent{}, err
	}
	var existing LedgerEvent
	appended, err := s.appendLedgerEventsConditionally(ctx, req.MissionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		attempts, terminals, err := reportAttempts(events)
		if err != nil {
			return nil, err
		}
		if matched, ok := retryIdempotencyMatch(attempts, req); ok {
			existing = matched
			return nil, existingReportRetry{event: matched}
		}
		if retryRequestIDTaken(attempts, req.RetryRequestID) {
			return nil, fmt.Errorf("%w: retry_request_id is already used with different input", ErrConflict)
		}
		if err := validateNoActiveAgentWork(events); err != nil {
			return nil, fmt.Errorf("%w: report retry is unavailable while work is active", ErrConflict)
		}
		target, ok := attempts[req.FailedPendingEventID]
		if !ok || terminals[target.EventID] != "failed" {
			return nil, fmt.Errorf("%w: retry target must be a failed terminal report attempt", ErrInvalidInput)
		}
		if target.ReportMode != "long_form" {
			return nil, fmt.Errorf("%w: report retry requires a long-form failed attempt", ErrInvalidInput)
		}
		if err := validateRetryLeafAndLineage(attempts, target); err != nil {
			return nil, err
		}
		payload := copyRetryPayload(target.Payload)
		payload["origin_pending_event_id"] = target.OriginID
		payload["retry_of_pending_event_id"] = target.EventID
		payload["attempt_number"] = target.Attempt + 1
		payload["retry_strategy"] = req.Strategy
		payload["retry_request_id"] = req.RetryRequestID
		payload["resume_stage"] = retryResumeStage(events, target.EventID, req.Strategy)
		if req.Strategy == "restart" {
			delete(payload, "resume_stage_artifact_ids")
			delete(payload, "report_session_id")
		}
		event, err := buildLedgerEvent(AppendEventRequest{EventID: req.EventID, MissionID: req.MissionID, EventType: "report.draft.pending", Producer: req.Producer, CausationEventID: target.EventID, Payload: mustJSON(payload)})
		if err != nil {
			return nil, err
		}
		return []LedgerEvent{event}, nil
	})
	if _, ok := err.(existingReportRetry); ok {
		return existing, nil
	}
	if err != nil {
		return LedgerEvent{}, err
	}
	if len(appended) != 1 {
		return LedgerEvent{}, fmt.Errorf("%w: retry append failed", ErrInvalidInput)
	}
	return appended[0], nil
}

type reportAttempt struct {
	LedgerEvent
	reportAttemptPayload
	Payload map[string]any
}

func reportAttempts(events []LedgerEvent) (map[string]reportAttempt, map[string]string, error) {
	attempts := map[string]reportAttempt{}
	terminals := map[string]string{}
	for _, event := range events {
		if event.EventType == "report.draft.pending" {
			var payload map[string]any
			if json.Unmarshal(event.Payload, &payload) != nil {
				return nil, nil, fmt.Errorf("%w: report pending payload is invalid", ErrInvalidInput)
			}
			var p reportAttemptPayload
			_ = json.Unmarshal(event.Payload, &p)
			if p.OriginID == "" {
				p.OriginID = event.EventID
			}
			if p.Attempt < 1 {
				p.Attempt = 1
			}
			attempts[event.EventID] = reportAttempt{LedgerEvent: event, reportAttemptPayload: p, Payload: payload}
		}
		if event.EventType == "report.draft.failed" {
			var p reportTerminalPayload
			_ = json.Unmarshal(event.Payload, &p)
			if p.PendingID != "" {
				if _, exists := terminals[p.PendingID]; exists {
					return nil, nil, fmt.Errorf("%w: conflicting report terminal outcomes", ErrInvalidInput)
				}
				if p.Kind == "report_draft_canceled" {
					terminals[p.PendingID] = "canceled"
				} else {
					terminals[p.PendingID] = "failed"
				}
			}
		}
		if event.EventType == "report.artifact.created" || event.EventType == "report.drafted" {
			var p reportTerminalPayload
			_ = json.Unmarshal(event.Payload, &p)
			if p.PendingID != "" {
				if _, exists := terminals[p.PendingID]; exists {
					return nil, nil, fmt.Errorf("%w: conflicting report terminal outcomes", ErrInvalidInput)
				}
				terminals[p.PendingID] = "completed"
			}
		}
	}
	return attempts, terminals, nil
}
func retryIdempotencyMatch(attempts map[string]reportAttempt, req ReportRetryRequest) (LedgerEvent, bool) {
	for _, a := range attempts {
		if a.RetryRequestID == req.RetryRequestID && a.RetryOf == req.FailedPendingEventID && a.RetryStrategy == req.Strategy {
			return a.LedgerEvent, true
		}
	}
	return LedgerEvent{}, false
}
func retryRequestIDTaken(attempts map[string]reportAttempt, id string) bool {
	for _, a := range attempts {
		if a.RetryRequestID == id {
			return true
		}
	}
	return false
}
func validateRetryLeafAndLineage(attempts map[string]reportAttempt, target reportAttempt) error {
	for _, child := range attempts {
		if child.RetryOf == target.EventID {
			return fmt.Errorf("%w: retry target has a newer attempt", ErrConflict)
		}
	}
	seen := map[string]bool{}
	current := target
	for depth := 0; depth < reportRetryLineageLimit; depth++ {
		if seen[current.EventID] {
			return fmt.Errorf("%w: report retry lineage cycle", ErrInvalidInput)
		}
		seen[current.EventID] = true
		if current.OriginID == "" {
			return fmt.Errorf("%w: report retry origin missing", ErrInvalidInput)
		}
		if current.RetryOf == "" {
			if current.OriginID != current.EventID {
				return fmt.Errorf("%w: report retry origin mismatch", ErrInvalidInput)
			}
			return nil
		}
		parent, ok := attempts[current.RetryOf]
		if !ok {
			return fmt.Errorf("%w: report retry ancestor missing", ErrInvalidInput)
		}
		if parent.OriginID != current.OriginID {
			return fmt.Errorf("%w: report retry lineage origin mismatch", ErrInvalidInput)
		}
		current = parent
	}
	return fmt.Errorf("%w: report retry lineage too deep", ErrInvalidInput)
}
func validateReportRetryRequest(req ReportRetryRequest) error {
	if err := validateID("mis_", req.MissionID); err != nil {
		return err
	}
	if err := validateID("evt_", req.EventID); err != nil {
		return err
	}
	if req.Strategy != "resume_failed" && req.Strategy != "restart" {
		return fmt.Errorf("%w: unsupported retry strategy", ErrInvalidInput)
	}
	if strings.TrimSpace(req.RetryRequestID) == "" {
		return fmt.Errorf("%w: retry_request_id is required", ErrInvalidInput)
	}
	return nil
}
func copyRetryPayload(payload map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range payload {
		out[k] = v
	}
	return out
}
func retryResumeStage(events []LedgerEvent, pendingID, strategy string) string {
	if strategy == "restart" {
		return "plan"
	}
	for _, e := range events {
		var p struct {
			PendingID     string `json:"pending_event_id"`
			FailedStage   string `json:"failed_stage_kind"`
			FailedStageID string `json:"failed_stage_id"`
		}
		_ = json.Unmarshal(e.Payload, &p)
		if e.EventType == "report.draft.failed" && p.PendingID == pendingID {
			if p.FailedStageID != "" {
				return p.FailedStageID
			}
			if p.FailedStage != "" {
				return p.FailedStage
			}
		}
	}
	return "plan"
}
