package app

import (
	"fmt"
	"strings"
)

const maxReportRequirementReviewedEvents = 256

// ReportRequirementReviewEventIDs returns the only ledger events that may
// supply user-authored output requirements for one report request.
func ReportRequirementReviewEventIDs(events []LedgerEvent, pendingEventID string) ([]string, error) {
	pendingEventID = strings.TrimSpace(pendingEventID)
	if pendingEventID == "" {
		return nil, fmt.Errorf("%w: report requirement pending event is required", ErrInvalidInput)
	}

	result := make([]string, 0)
	foundPending := false
	for _, event := range events {
		if event.EventID == pendingEventID {
			if event.EventType != "report.draft.pending" || event.Producer.Type != "user" {
				return nil, fmt.Errorf("%w: report requirement pending event is invalid", ErrInvalidInput)
			}
			result = append(result, event.EventID)
			foundPending = true
			break
		}
		if event.EventType == "turn.user" && event.Producer.Type == "user" {
			result = append(result, event.EventID)
		}
	}
	if !foundPending {
		return nil, fmt.Errorf("%w: report requirement pending event is missing", ErrInvalidInput)
	}
	if len(result) > maxReportRequirementReviewedEvents {
		return nil, fmt.Errorf("%w: too many user events for report requirement review", ErrInvalidInput)
	}
	return result, nil
}
