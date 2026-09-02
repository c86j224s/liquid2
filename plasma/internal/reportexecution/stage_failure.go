package reportexecution

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

// StageFailureError is the stable error envelope returned when a long-form report stage fails.
type StageFailureError struct {
	Kind, PlanEventID, ErrorClass, Message, EventID string
	PartIndex, SectionIndex                         int
	Retryable                                       bool
	SafeFailureReason                               ProviderFailureReason
	SafeValidationCode                              ProviderValidationCode
	ProviderUsage                                   *ProviderFailureUsageReceipt
	Cause                                           error
}

// Error returns a stable user-safe failure string.
func (err *StageFailureError) Error() string { return fmt.Sprintf("report %s stage failed", err.Kind) }

// Unwrap exposes the underlying cause for programmatic inspection without changing the safe public message.
func (err *StageFailureError) Unwrap() error { return err.Cause }

// FailurePayload preserves safe nested metadata and a bounded internal
// diagnostic for non-provider store failures. Public error surfaces remain redacted.
func (err *StageFailureError) FailurePayload() map[string]any {
	if err.Cause == nil {
		return nil
	}
	payload := map[string]any{}
	var provider FailurePayloadProvider
	hasNestedPayload := errors.As(err.Cause, &provider)
	if hasNestedPayload {
		for key, value := range provider.FailurePayload() {
			if allowedFailurePayloadKey(key) && value != nil {
				payload[key] = value
			}
		}
	}
	if err.Kind == "il_store" && !hasNestedPayload {
		message := strings.TrimSpace(err.Cause.Error())
		if len(message) > 500 {
			message = message[:500]
		}
		payload["internal_failure_detail"] = message
	}
	if len(payload) == 0 {
		return nil
	}
	return payload
}

// ID returns the stable stage coordinate used by retry and display surfaces.
func (err *StageFailureError) ID() string {
	return stageFailureID(err.Kind, err.PartIndex, err.SectionIndex)
}

// AppendRequest builds the stage failure append request; it does not write durable state.
func (err *StageFailureError) AppendRequest(missionID, pendingID, terminalID string, producer ledger.Producer) ledger.AppendRequest {
	payload := map[string]any{"pending_event_id": pendingID, "plan_event_id": err.PlanEventID, "stage_kind": err.Kind, "stage_id": err.ID(), "part_index": err.PartIndex, "section_index": err.SectionIndex, "safe_error_class": err.ErrorClass, "safe_error_message": safeStageMessage(err.Message), "retryable": err.Retryable, "terminal_event_id": terminalID}
	err.appendProviderFailurePayload(payload)
	return ledger.AppendRequest{EventID: err.EventID, MissionID: missionID, EventType: "report." + err.Kind + ".failed", Producer: producer, CorrelationID: terminalID, Payload: mustJSON(payload)}
}

func (err *StageFailureError) appendProviderFailurePayload(payload map[string]any) {
	if err.SafeFailureReason.Valid() {
		payload["safe_failure_reason"] = err.SafeFailureReason
	}
	if err.SafeValidationCode.Valid() {
		payload["safe_validation_code"] = err.SafeValidationCode
	}
	if err.ProviderUsage != nil {
		payload["provider_usage_receipt"] = NewProviderFailureUsageReceipt(err.ProviderUsage.Attempts)
	}
}

// NewStageFailure wraps a stage failure with the default safe class and retryable state.
func NewStageFailure(kind, planID string, part, section int, cause error) *StageFailureError {
	retryable := true
	switch kind {
	case "source_packet", "il_source_selection", "il_editorial_memory", "il_narrative", "il_long_form_plan", "il_long_form_sections", "il_long_form_parts", "il_long_form_final", "il_continuity", "il_reader", "il_images", "il_document", "il_flow", "il_render", "il_store":
		retryable = false
	}
	return &StageFailureError{Kind: kind, PlanEventID: planID, PartIndex: part, SectionIndex: section, ErrorClass: "report_stage_failed", Message: "리포트 생성 단계가 실패했습니다.", Retryable: retryable, Cause: cause}
}

// StageFailureRequest carries the safe stage coordinates to append before draft terminal closure.
type StageFailureRequest struct {
	MissionID, PendingEventID, PlanEventID, StageKind string
	PartIndex, SectionIndex                           int
	ErrorClass, Message                               string
	Retryable                                         bool
	Producer                                          ledger.Producer
}

// AppendStageFailure writes only safe stage coordinates and redacted failure metadata.
func (runner Runner) AppendStageFailure(ctx context.Context, req StageFailureRequest) (ledger.Event, error) {
	kind := strings.TrimSpace(req.StageKind)
	eventType := "report." + kind + ".failed"
	payload := map[string]any{
		"pending_event_id": req.PendingEventID, "plan_event_id": req.PlanEventID,
		"stage_kind": kind, "stage_id": stageFailureID(kind, req.PartIndex, req.SectionIndex),
		"part_index": req.PartIndex, "section_index": req.SectionIndex,
		"safe_error_class":   firstNonEmpty(strings.TrimSpace(req.ErrorClass), "report_stage_failed"),
		"safe_error_message": safeStageMessage(req.Message), "retryable": req.Retryable,
		"terminal_pending_event_id": req.PendingEventID,
		"failed_at":                 time.Now().UTC().Format(time.RFC3339Nano),
	}
	return runner.Service.AppendEvent(ctx, ledger.AppendRequest{EventID: runner.id("evt"), MissionID: req.MissionID, EventType: eventType, Producer: req.Producer, Payload: mustJSON(payload)})
}

func stageFailureID(kind string, part, section int) string {
	switch kind {
	case "plan":
		return "plan"
	case "requirements":
		return "requirements"
	case "section":
		return "section-" + strconv.Itoa(part) + "-" + strconv.Itoa(section)
	case "part":
		return "part-" + strconv.Itoa(part)
	case "part_plan":
		return "part-plan-" + strconv.Itoa(part)
	case "part_edit":
		return "part-edit-" + strconv.Itoa(part)
	case "artifact", "export":
		return "artifact"
	case "source_packet", "il_source_selection", "il_editorial_memory", "il_narrative", "il_long_form_plan", "il_long_form_sections", "il_long_form_parts", "il_long_form_final", "il_continuity", "il_reader", "il_images", "il_document", "il_flow", "il_render", "il_store":
		return kind
	default:
		return "final"
	}
}

func safeStageMessage(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "보고서 단계가 실패했습니다."
	}
	if len(value) > 240 {
		return value[:240]
	}
	return value
}
