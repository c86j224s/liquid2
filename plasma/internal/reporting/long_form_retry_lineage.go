package reporting

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type longFormRetryLink struct {
	origin   string
	parent   string
	strategy string
}

type longFormRetryTerminal string

const (
	longFormRetryTerminalFailed    longFormRetryTerminal = "failed"
	longFormRetryTerminalCanceled  longFormRetryTerminal = "canceled"
	longFormRetryTerminalCompleted longFormRetryTerminal = "completed"
)

func longFormPendingLineage(events []app.LedgerEvent, pendingID string) (map[string]bool, error) {
	links := longFormRetryLinks(events)
	terminals := longFormRetryTerminals(events)
	accepted := map[string]bool{}
	current, ok := links[pendingID]
	if !ok {
		return accepted, fmt.Errorf("%w: bound report pending event does not exist", app.ErrConflict)
	}
	origin := current.origin
	seen := map[string]bool{}
	for depth := 0; depth < 64; depth++ {
		if seen[pendingID] {
			return nil, fmt.Errorf("%w: long-form retry lineage cycle", app.ErrConflict)
		}
		seen[pendingID] = true
		item, ok := links[pendingID]
		if !ok || item.origin != origin {
			return nil, fmt.Errorf("%w: long-form retry lineage differs", app.ErrConflict)
		}
		accepted[pendingID] = true
		if item.strategy == "restart" {
			if err := validateLongFormRetryParent(links, terminals, origin, item.parent, "restart"); err != nil {
				return nil, err
			}
			return accepted, nil
		}
		if item.parent == "" {
			if item.origin != pendingID {
				return nil, fmt.Errorf("%w: long-form retry origin differs", app.ErrConflict)
			}
			return accepted, nil
		}
		if item.strategy != "resume_failed" {
			return nil, fmt.Errorf("%w: unsupported long-form retry lineage", app.ErrConflict)
		}
		if err := validateLongFormRetryParent(links, terminals, origin, item.parent, "resume_failed"); err != nil {
			return nil, err
		}
		pendingID = item.parent
	}
	return nil, fmt.Errorf("%w: long-form retry lineage is too deep", app.ErrConflict)
}

func longFormRetryLinks(events []app.LedgerEvent) map[string]longFormRetryLink {
	links := map[string]longFormRetryLink{}
	for _, event := range events {
		if event.EventType != "report.draft.pending" {
			continue
		}
		payload := eventPayload(event)
		origin, _ := payload["origin_pending_event_id"].(string)
		parent, _ := payload["retry_of_pending_event_id"].(string)
		strategy, _ := payload["retry_strategy"].(string)
		if origin == "" {
			origin = event.EventID
		}
		links[event.EventID] = longFormRetryLink{origin: origin, parent: parent, strategy: strategy}
	}
	return links
}

func longFormRetryTerminals(events []app.LedgerEvent) map[string][]longFormRetryTerminal {
	terminals := map[string][]longFormRetryTerminal{}
	for _, event := range events {
		payload := eventPayload(event)
		pendingID, _ := payload["pending_event_id"].(string)
		if pendingID == "" {
			continue
		}
		switch event.EventType {
		case "report.draft.failed":
			if payloadString(payload, "kind") == "report_draft_canceled" {
				terminals[pendingID] = append(terminals[pendingID], longFormRetryTerminalCanceled)
			} else {
				terminals[pendingID] = append(terminals[pendingID], longFormRetryTerminalFailed)
			}
		case "report.drafted", "report.artifact.created":
			terminals[pendingID] = append(terminals[pendingID], longFormRetryTerminalCompleted)
		}
	}
	return terminals
}

func validateLongFormRetryParent(links map[string]longFormRetryLink, terminals map[string][]longFormRetryTerminal, origin, parentID, strategy string) error {
	if parentID == "" {
		return fmt.Errorf("%w: long-form %s lineage is incomplete", app.ErrConflict, strategy)
	}
	parent, ok := links[parentID]
	if !ok || parent.origin != origin {
		return fmt.Errorf("%w: long-form %s lineage differs", app.ErrConflict, strategy)
	}
	outcomes := terminals[parentID]
	if len(outcomes) != 1 {
		return fmt.Errorf("%w: long-form retry parent terminal count differs", app.ErrConflict)
	}
	if outcomes[0] != longFormRetryTerminalFailed {
		return fmt.Errorf("%w: long-form retry parent was not failed", app.ErrConflict)
	}
	return nil
}
