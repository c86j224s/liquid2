package ledgerstate

import (
	"encoding/json"
	"strings"
	"time"
)

// OpenAgentPendingEvent는 아직 terminal 응답을 갖지 않은 최신 에이전트 턴을
// 반환한다. 같은 user_event_id에 대한 turn.agent.response가 이미 있으면 그
// pending 이벤트는 닫힌 것으로 본다.
func OpenAgentPendingEvent(events []Event) (Event, bool) {
	completed := CompletedUserEventIDs(events)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "turn.agent.pending" {
			continue
		}
		userEventID := userEventIDFromPayload(event.Payload)
		if userEventID != "" {
			if _, done := completed[userEventID]; !done {
				return event, true
			}
		}
	}
	return Event{}, false
}

// Event는 원장 상태 판정에 필요한 최소 이벤트 형태다. Payload는 원본 JSON을
// 보존해야 하며, 이 패키지는 필요한 필드를 읽을 수 없는 이벤트를 상태 변화로
// 간주하지 않는다.
type Event struct {
	EventID   string
	Sequence  int64
	EventType string
	Payload   json.RawMessage
	CreatedAt time.Time
}

// HasOpenAgentPending은 미완료 에이전트 턴이 하나라도 남아 있는지 판정한다.
func HasOpenAgentPending(events []Event) bool {
	_, ok := OpenAgentPendingEvent(events)
	return ok
}

// HasAgentTerminalEventForUser는 특정 사용자 이벤트에 대해 terminal 응답이
// 기록됐는지 판정한다. 빈 userEventID는 호출자가 더 진행하지 않도록 이미
// 닫힌 상태로 취급한다.
func HasAgentTerminalEventForUser(events []Event, userEventID string) bool {
	userEventID = strings.TrimSpace(userEventID)
	if userEventID == "" {
		return true
	}
	_, ok := CompletedUserEventIDs(events)[userEventID]
	return ok
}

// ValidateWorkflowStartAfterEvent는 workflow 재개 기준 이벤트가 이 미션의
// 열린 사용자 턴인지 검증한다. 반환 문자열이 비어 있으면 재개 가능하고,
// 비어 있지 않으면 호출자가 그대로 사용자에게 전달할 수 있는 안정된 오류다.
func ValidateWorkflowStartAfterEvent(events []Event, startAfterEventID string) string {
	startAfterEventID = strings.TrimSpace(startAfterEventID)
	if startAfterEventID == "" {
		return ""
	}
	hasUserTurn := false
	hasPending := false
	hasTerminal := false
	for _, event := range events {
		switch event.EventType {
		case "turn.user":
			if event.EventID == startAfterEventID {
				hasUserTurn = true
			}
		case "turn.agent.pending":
			if userEventIDFromPayload(event.Payload) == startAfterEventID {
				hasPending = true
			}
		case "turn.agent.response":
			if userEventIDFromPayload(event.Payload) == startAfterEventID {
				hasTerminal = true
			}
		}
	}
	if !hasUserTurn {
		return "start_after_event_id must reference a turn.user event in this mission"
	}
	if hasTerminal {
		return "start_after_event_id already has a terminal agent response"
	}
	if !hasPending {
		return "start_after_event_id must reference an open agent turn"
	}
	return ""
}

// CompletedUserEventIDs는 turn.agent.response가 닫은 user_event_id 집합을
// 만든다. payload를 읽을 수 없는 응답은 닫힘 근거로 사용하지 않는다.
func CompletedUserEventIDs(events []Event) map[string]struct{} {
	completed := map[string]struct{}{}
	for _, event := range events {
		if event.EventType != "turn.agent.response" {
			continue
		}
		userEventID := userEventIDFromPayload(event.Payload)
		if userEventID != "" {
			completed[userEventID] = struct{}{}
		}
	}
	return completed
}

// HasOpenReportPending은 아직 terminal report 이벤트가 연결되지 않은 보고서
// 작업이 남아 있는지 판정한다.
func HasOpenReportPending(events []Event) bool {
	_, ok := OpenReportPendingEvent(events)
	return ok
}

// OpenReportPendingEvent는 terminal 이벤트가 아직 연결되지 않은 최신 보고서
// 작업을 반환한다. pending 이벤트의 EventID가 완료/실패/스킵 이벤트 payload에
// 참조되면 닫힌 것으로 본다.
func OpenReportPendingEvent(events []Event) (Event, bool) {
	completed := CompletedReportPendingEventIDs(events)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.draft.pending" && event.EventType != "report.design.pending" && event.EventType != "report.humanize.pending" && event.EventType != "report.patch.pending" {
			continue
		}
		if _, ok := completed[event.EventID]; !ok {
			return event, true
		}
	}
	return Event{}, false
}

// CompletedReportPendingEventIDs는 보고서 terminal 이벤트가 닫은 pending
// 이벤트 ID 집합을 만든다. 과거 payload 필드명이 여러 개라서 ReportPendingEventID
// 계약을 통해 호환 필드를 한 곳에서 해석한다.
func CompletedReportPendingEventIDs(events []Event) map[string]struct{} {
	completed := map[string]struct{}{}
	for _, event := range events {
		switch event.EventType {
		case "report.drafted", "report.artifact.created", "report.artifact.exported":
			if pendingEventID := ReportPendingEventID(event); pendingEventID != "" {
				completed[pendingEventID] = struct{}{}
			}
		case "report.draft.failed", "report.design.failed", "report.humanize.failed", "report.humanize.skipped", "report.patch.failed":
			var payload struct {
				PendingEventID string `json:"pending_event_id"`
			}
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				continue
			}
			if pendingEventID := strings.TrimSpace(payload.PendingEventID); pendingEventID != "" {
				completed[pendingEventID] = struct{}{}
			}
		}
	}
	return completed
}

// ReportPendingEventID는 보고서 terminal 이벤트 payload에서 원래 pending
// 이벤트 ID를 추출한다. 새 필드와 과거 호환 필드를 모두 읽되, 찾지 못하면
// 빈 문자열로 돌려 상태 변경 근거에서 제외한다.
func ReportPendingEventID(event Event) string {
	var payload struct {
		PendingEventID       string `json:"pending_event_id"`
		PendingReportEventID string `json:"pending_report_event_id"`
		Generation           struct {
			PendingEventID string `json:"pending_event_id"`
		} `json:"generation"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return ""
	}
	return firstNonEmptyText(
		payload.PendingEventID,
		payload.PendingReportEventID,
		payload.Generation.PendingEventID,
	)
}

func userEventIDFromPayload(payload json.RawMessage) string {
	var typed struct {
		UserEventID string `json:"user_event_id"`
	}
	if json.Unmarshal(payload, &typed) != nil {
		return ""
	}
	return strings.TrimSpace(typed.UserEventID)
}

func firstNonEmptyText(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
