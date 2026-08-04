package workflowstate

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// AppendEventRequest는 workflowstate builder가 workflowruns service에 넘기는 event
// append 요청의 최소 형태다.
type AppendEventRequest struct {
	EventID   string
	MissionID string
	EventType string
	Payload   json.RawMessage
}

// BuildTerminalAppendRequest는 workflow run을 terminal event 하나로 닫는 append 요청을
// 만든다.
//
// 이미 terminal event가 있는 run이면 ok=false를 반환해 idempotent하게 중복 기록을
// 막는다. 없는 run이나 지원하지 않는 terminal event type은 오류다.
func BuildTerminalAppendRequest(events []Event, eventID string, req WorkflowRunTerminalEventRequest, now time.Time) (AppendEventRequest, bool, error) {
	workflowRunID := strings.TrimSpace(req.WorkflowRunID)
	eventType := strings.TrimSpace(req.EventType)
	if workflowRunID == "" || eventType == "" {
		return AppendEventRequest{}, false, nil
	}
	if StatusForTerminalEvent(eventType) == "" {
		return AppendEventRequest{}, false, fmt.Errorf("unsupported workflow terminal event %q", eventType)
	}
	if HasTerminalEvent(events, workflowRunID) {
		return AppendEventRequest{}, false, nil
	}
	var run WorkflowRunView
	found := false
	for _, candidate := range ProjectRuns(events) {
		if candidate.WorkflowRunID == workflowRunID {
			run = candidate
			found = true
			break
		}
	}
	if !found {
		return AppendEventRequest{}, false, fmt.Errorf("workflow run not found")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	payload := WorkflowRunTerminalPayload{
		WorkflowRunID:      workflowRunID,
		MissionID:          strings.TrimSpace(req.MissionID),
		Reason:             strings.TrimSpace(req.Reason),
		Error:              strings.TrimSpace(req.Error),
		CompletedStepCount: run.CompletedStepCount,
		TerminalAt:         now.UTC().Format(time.RFC3339Nano),
	}
	if eventType == WorkflowRunStoppedEvent {
		payload.StopReason = strings.TrimSpace(req.Reason)
	}
	return AppendEventRequest{
		EventID:   strings.TrimSpace(eventID),
		MissionID: payload.MissionID,
		EventType: eventType,
		Payload:   mustJSONRaw(payload),
	}, true, nil
}

func mustJSONRaw(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}
