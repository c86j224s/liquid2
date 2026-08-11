package reportrun

import "strings"

type pendingLineage struct {
	roots         map[string]string
	invalid       map[string]bool
	ambiguousRuns map[string]bool
}

type pendingRetryLink struct {
	event    Event
	origin   string
	parent   string
	strategy string
	retry    bool
}

type retryTerminal struct {
	kind     string
	sequence int64
}

const (
	retryTerminalFailed    = "failed"
	retryTerminalCanceled  = "canceled"
	retryTerminalCompleted = "completed"
)

func buildPendingLineage(events []Event, payloads map[string]eventPayload) pendingLineage {
	links := pendingRetryLinks(events, payloads)
	terminals := pendingRetryTerminals(events, payloads)
	roots := map[string]string{}
	invalid := map[string]bool{}
	for id := range links {
		root, ok := validatePendingRetryLineage(id, links, terminals, map[string]bool{}, 0)
		if ok {
			roots[id] = root
			continue
		}
		invalid[id] = true
	}
	ambiguous := map[string]bool{}
	for id := range invalid {
		link := links[id]
		if _, ok := links[link.origin]; ok {
			ambiguous[link.origin] = true
		}
		if root := roots[link.parent]; root != "" {
			ambiguous[root] = true
		} else if _, ok := links[link.parent]; ok {
			ambiguous[link.parent] = true
		}
	}
	return pendingLineage{roots: roots, invalid: invalid, ambiguousRuns: ambiguous}
}

func pendingRetryLinks(events []Event, payloads map[string]eventPayload) map[string]pendingRetryLink {
	links := map[string]pendingRetryLink{}
	for _, event := range events {
		if event.EventType != "report.draft.pending" {
			continue
		}
		payload := payloads[event.EventID]
		origin := payloadString(payload, "origin_pending_event_id")
		parent := payloadString(payload, "retry_of_pending_event_id")
		strategy := payloadString(payload, "retry_strategy")
		if origin == "" {
			origin = event.EventID
		}
		links[event.EventID] = pendingRetryLink{
			event: event, origin: origin, parent: parent, strategy: strategy,
			retry: origin != event.EventID || parent != "" || strategy != "",
		}
	}
	return links
}

func pendingRetryTerminals(events []Event, payloads map[string]eventPayload) map[string][]retryTerminal {
	terminals := map[string][]retryTerminal{}
	for _, event := range events {
		payload := payloads[event.EventID]
		pendingID := firstNonEmpty(payloadString(payload, "pending_event_id"), nestedPayloadString(payload, "generation", "pending_event_id"))
		if pendingID == "" {
			continue
		}
		switch event.EventType {
		case "report.draft.failed":
			if eventRole(event.EventType, payload) == "canceled" {
				terminals[pendingID] = append(terminals[pendingID], retryTerminal{kind: retryTerminalCanceled, sequence: event.Sequence})
			} else {
				terminals[pendingID] = append(terminals[pendingID], retryTerminal{kind: retryTerminalFailed, sequence: event.Sequence})
			}
		case "report.drafted", "report.artifact.created":
			terminals[pendingID] = append(terminals[pendingID], retryTerminal{kind: retryTerminalCompleted, sequence: event.Sequence})
		}
	}
	return terminals
}

func validatePendingRetryLineage(id string, links map[string]pendingRetryLink, terminals map[string][]retryTerminal, seen map[string]bool, depth int) (string, bool) {
	if depth >= 64 || seen[id] {
		return "", false
	}
	link, ok := links[id]
	if !ok {
		return "", false
	}
	if !link.retry {
		if link.origin != id || link.parent != "" || link.strategy != "" {
			return "", false
		}
		return id, true
	}
	if link.strategy != "resume_failed" && link.strategy != "restart" {
		return "", false
	}
	if link.origin == "" || link.parent == "" {
		return "", false
	}
	nextSeen := copyStringBoolMap(seen)
	nextSeen[id] = true
	root, ok := validatePendingRetryLineage(link.parent, links, terminals, nextSeen, depth+1)
	parent := links[link.parent]
	if !ok || root != link.origin || !validFailedRetryParent(parent.event.Sequence, terminals[link.parent], link.event.Sequence) {
		return "", false
	}
	return root, true
}

func validFailedRetryParent(parentSequence int64, terminals []retryTerminal, retrySequence int64) bool {
	if parentSequence <= 0 || retrySequence <= 0 || len(terminals) != 1 {
		return false
	}
	terminal := terminals[0]
	return terminal.kind == retryTerminalFailed &&
		terminal.sequence > parentSequence &&
		terminal.sequence < retrySequence
}

func copyStringBoolMap(values map[string]bool) map[string]bool {
	out := make(map[string]bool, len(values)+1)
	for key, value := range values {
		out[key] = value
	}
	return out
}

func nestedPayloadString(payload eventPayload, objectKey string, key string) string {
	object, ok := payload[objectKey].(map[string]any)
	if !ok {
		return ""
	}
	value, _ := object[key].(string)
	return strings.TrimSpace(value)
}
