package ledgerstate

import (
	"encoding/json"
	"strings"
	"time"
)

// OpenAgentPendingEvent returns the newest agent turn that has no terminal response.
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

type Event struct {
	EventID   string
	Sequence  int64
	EventType string
	Payload   json.RawMessage
	CreatedAt time.Time
}

func HasOpenAgentPending(events []Event) bool {
	_, ok := OpenAgentPendingEvent(events)
	return ok
}

func HasAgentTerminalEventForUser(events []Event, userEventID string) bool {
	userEventID = strings.TrimSpace(userEventID)
	if userEventID == "" {
		return true
	}
	_, ok := CompletedUserEventIDs(events)[userEventID]
	return ok
}

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

func HasOpenReportPending(events []Event) bool {
	_, ok := OpenReportPendingEvent(events)
	return ok
}

// OpenReportPendingEvent returns the newest report operation that has no terminal event.
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
