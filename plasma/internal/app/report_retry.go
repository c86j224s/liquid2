package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

const reportRetryLineageLimit = 64

// ReportRetryRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
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
	PipelineFamily string `json:"pipeline_family"`
	Attempt        int    `json:"attempt_number"`
}

type reportTerminalPayload struct {
	PendingID string `json:"pending_event_id"`
	Kind      string `json:"kind"`
}

type existingReportRetry struct{ event LedgerEvent }

// Error는 호출자에게 노출 가능한 안정적인 오류 문자열을 반환하며, 민감한 원문이나 provider 응답을 포함하지 않아야 한다.
func (err existingReportRetry) Error() string { return "existing report retry" }

// RequestReportRetry는 리포트 재시도 지속 명령 경계다. 조건부 append는 변할 수
// 있는 모든 조건을 하나의 장부 snapshot에 대해 검사해야 한다.
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
		family := strings.TrimSpace(target.PipelineFamily)
		if family != "" && family != "report_il_experimental" {
			return nil, fmt.Errorf("%w: independent report retries are not supported", ErrInvalidInput)
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
		if family == "report_il_experimental" && req.Strategy == "resume_failed" &&
			!reportILLineageCheckpointAvailable(events, attempts, target) &&
			!reportILLegacyCheckpointAvailable(events, target.EventID) {
			return nil, fmt.Errorf("%w: report IL retry has no recoverable checkpoint", ErrInvalidInput)
		}
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
func reportILLegacyCheckpointAvailable(events []LedgerEvent, pendingID string) bool {
	failedAtReader := false
	finalCompleted := false
	memoryArtifact := false
	finalArtifact := false
	for _, event := range events {
		var payload struct {
			PendingID string `json:"pending_event_id"`
			Failed    string `json:"failed_stage_kind"`
			ToolName  string `json:"tool_name"`
			Success   bool   `json:"success"`
			IOMetrics struct {
				Stage      string `json:"report_il_stage"`
				ArtifactID string `json:"artifact_id"`
				SHA256     string `json:"sha256"`
				ByteSize   int    `json:"byte_size"`
				Revision   int    `json:"revision"`
				Accounts   int    `json:"accounts"`
				Finalized  bool   `json:"finalized"`
			} `json:"io_metrics"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		if payload.PendingID == pendingID {
			switch event.EventType {
			case "report.draft.failed":
				failedAtReader = payload.Failed == "il_reader"
			case "report.il_long_form_final.completed":
				finalCompleted = true
			}
		}
		if event.EventType != "mcp.tool.called" || !payload.Success || !payload.IOMetrics.Finalized ||
			payload.IOMetrics.ArtifactID == "" || len(payload.IOMetrics.SHA256) != 64 ||
			payload.IOMetrics.ByteSize < 1 || payload.IOMetrics.Revision < 1 {
			continue
		}
		switch {
		case payload.ToolName == reportilcontract.EditorialMemoryFinalizeTool &&
			payload.IOMetrics.Stage == "il_editorial_memory" && payload.IOMetrics.Accounts > 0:
			memoryArtifact = true
		case payload.ToolName == reportilcontract.LongFormDocumentFinalizeTool &&
			payload.IOMetrics.Stage == "il_long_form_final":
			finalArtifact = true
		}
	}
	return failedAtReader && finalCompleted && memoryArtifact && finalArtifact
}

func reportILLineageCheckpointAvailable(events []LedgerEvent, attempts map[string]reportAttempt, target reportAttempt) bool {
	current := target
	for depth := 0; depth < reportRetryLineageLimit; depth++ {
		if reportILCheckpointAvailable(events, current.EventID) {
			return true
		}
		if current.RetryOf == "" {
			return false
		}
		parent, ok := attempts[current.RetryOf]
		if !ok {
			return false
		}
		current = parent
	}
	return false
}

func reportILCheckpointAvailable(events []LedgerEvent, pendingID string) bool {
	for _, event := range events {
		if event.EventType != reportILCheckpointEventType || event.CausationEventID != pendingID {
			continue
		}
		var payload struct {
			PendingID  string `json:"pending_event_id"`
			Checkpoint struct {
				Stage string `json:"stage"`
			} `json:"checkpoint"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil &&
			payload.PendingID == pendingID && reportILResumeStageOrder(payload.Checkpoint.Stage) > 0 {
			return true
		}
	}
	return false
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
