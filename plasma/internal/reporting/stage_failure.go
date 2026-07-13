package reporting

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type StageFailureError struct {
	Kind, PlanEventID, ErrorClass, Message, EventID string
	PartIndex, SectionIndex                         int
	Retryable                                       bool
	Cause                                           error
}

func (err *StageFailureError) Error() string { return fmt.Sprintf("report %s stage failed", err.Kind) }
func (err *StageFailureError) Unwrap() error { return err.Cause }
func (err *StageFailureError) ID() string {
	return stageFailureID(err.Kind, err.PartIndex, err.SectionIndex)
}
func (err *StageFailureError) AppendRequest(missionID, pendingID, terminalID string, producer app.Producer) app.AppendEventRequest {
	payload := map[string]any{"pending_event_id": pendingID, "plan_event_id": err.PlanEventID, "stage_kind": err.Kind, "stage_id": err.ID(), "part_index": err.PartIndex, "section_index": err.SectionIndex, "safe_error_class": err.ErrorClass, "safe_error_message": safeStageMessage(err.Message), "retryable": err.Retryable, "terminal_event_id": terminalID}
	return app.AppendEventRequest{EventID: err.EventID, MissionID: missionID, EventType: "report." + err.Kind + ".failed", Producer: producer, CorrelationID: terminalID, Payload: mustJSON(payload)}
}
func NewStageFailure(kind, planID string, part, section int, cause error) *StageFailureError {
	return &StageFailureError{Kind: kind, PlanEventID: planID, PartIndex: part, SectionIndex: section, ErrorClass: "report_stage_failed", Message: "리포트 생성 단계가 실패했습니다.", Retryable: true, Cause: cause}
}

type StageFailureRequest struct {
	MissionID, PendingEventID, PlanEventID, StageKind string
	PartIndex, SectionIndex                           int
	ErrorClass, Message                               string
	Retryable                                         bool
	Producer                                          app.Producer
}

// AppendStageFailure records safe coordinates before the existing draft
// terminal closure. It does not contain prompts, provider payloads, or IDs.
func (runner Runner) AppendStageFailure(ctx context.Context, req StageFailureRequest) (app.LedgerEvent, error) {
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
	return runner.Service.AppendEvent(ctx, app.AppendEventRequest{EventID: runner.id("evt"), MissionID: req.MissionID, EventType: eventType, Producer: req.Producer, Payload: mustJSON(payload)})
}

func stageFailureID(kind string, part, section int) string {
	switch kind {
	case "plan":
		return "plan"
	case "section":
		return "section-" + strconv.Itoa(part) + "-" + strconv.Itoa(section)
	case "part":
		return "part-" + strconv.Itoa(part)
	case "artifact", "export":
		return "artifact"
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
