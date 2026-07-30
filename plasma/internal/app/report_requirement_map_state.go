package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
)

func validateReportRequirementMapRequest(req ReportRequirementMapSubmissionRequest) error {
	if validateID("mis_", req.MissionID) != nil || !strings.HasPrefix(req.EventID, "evt_") || !strings.HasPrefix(req.PendingEventID, "evt_") || !strings.HasPrefix(req.PlanEventID, "evt_") || req.ToolSessionID == "" || req.AgentExecutor == "" || req.IdempotencyKey == "" || req.ArgumentsHash == "" || req.RequirementMapHash == "" || !json.Valid(req.RequirementMap) || len(req.ReviewedEventIDs) == 0 || req.Attempt < 1 {
		return fmt.Errorf("%w: incomplete report requirement mapping", ErrInvalidInput)
	}
	if req.ToolProducer.Type != "agent_session" || req.ToolProducer.ID != req.ToolSessionID {
		return fmt.Errorf("%w: report requirement mapping producer mismatch", ErrInvalidInput)
	}
	return nil
}

func validateReportRequirementMapSlot(events []LedgerEvent, req ReportRequirementMapSubmissionRequest) (int, int, error) {
	pendingIndex, planIndex := -1, -1
	for index, event := range events {
		if event.EventID == req.PendingEventID {
			if event.EventType != "report.draft.pending" {
				return -1, -1, fmt.Errorf("%w: report requirement pending event is invalid", ErrInvalidInput)
			}
			var payload struct {
				ReportMode    string `json:"report_mode"`
				AgentExecutor string `json:"agent_executor"`
			}
			if json.Unmarshal(event.Payload, &payload) != nil || payload.ReportMode != "long_form" || strings.TrimSpace(payload.AgentExecutor) != strings.TrimSpace(req.AgentExecutor) {
				return -1, -1, fmt.Errorf("%w: report requirement pending binding mismatch", ErrConflict)
			}
			pendingIndex = index
		}
		if event.EventID == req.PlanEventID {
			if event.EventType != "report.plan.created" {
				return -1, -1, fmt.Errorf("%w: report requirement plan event is invalid", ErrInvalidInput)
			}
			var payload struct {
				PendingEventID string `json:"pending_event_id"`
			}
			if json.Unmarshal(event.Payload, &payload) != nil || payload.PendingEventID != req.PendingEventID {
				return -1, -1, fmt.Errorf("%w: report requirement plan binding mismatch", ErrConflict)
			}
			planIndex = index
		}
	}
	if pendingIndex < 0 || planIndex <= pendingIndex {
		return -1, -1, fmt.Errorf("%w: report requirement mapping requires a canonical plan", ErrInvalidInput)
	}
	if _, closed := ledgerstate.CompletedReportPendingEventIDs(ledgerStateEventsFromApp(events))[req.PendingEventID]; closed {
		return -1, -1, fmt.Errorf("%w: report pending event is finalized", ErrConflict)
	}
	return pendingIndex, planIndex, nil
}

func validateReviewedReportRequirementEvents(events []LedgerEvent, pendingIndex int, req ReportRequirementMapSubmissionRequest) error {
	eligible, err := ReportRequirementReviewEventIDs(events[:pendingIndex+1], req.PendingEventID)
	if err != nil {
		return err
	}
	eligibleSet := make(map[string]struct{}, len(eligible))
	for _, eventID := range eligible {
		eligibleSet[eventID] = struct{}{}
	}
	seen := map[string]struct{}{}
	for _, eventID := range req.ReviewedEventIDs {
		eventID = strings.TrimSpace(eventID)
		if _, duplicate := seen[eventID]; duplicate {
			return fmt.Errorf("%w: duplicate reviewed report requirement event", ErrInvalidInput)
		}
		seen[eventID] = struct{}{}
		if _, ok := eligibleSet[eventID]; !ok {
			return fmt.Errorf("%w: report requirement sources must be prior user turns or the current report request", ErrInvalidInput)
		}
	}
	if _, ok := seen[req.PendingEventID]; !ok {
		return fmt.Errorf("%w: current report request must be reviewed", ErrInvalidInput)
	}
	return nil
}

func reportStagesStartedAfterPlan(events []LedgerEvent, planIndex int, pendingEventID, planEventID string) bool {
	for _, event := range events[planIndex+1:] {
		switch event.EventType {
		case "report.section.started", "report.section.created", "report.part.created", "report.artifact.created":
		default:
			continue
		}
		var payload struct {
			PendingEventID string `json:"pending_event_id"`
			PlanEventID    string `json:"plan_event_id"`
		}
		if json.Unmarshal(event.Payload, &payload) == nil && payload.PendingEventID == pendingEventID && (payload.PlanEventID == "" || payload.PlanEventID == planEventID) {
			return true
		}
	}
	return false
}

func sameReportRequirementMapBinding(payload reportRequirementMapPayload, req ReportRequirementMapSubmissionRequest) bool {
	return payload.PendingEventID == req.PendingEventID && payload.PlanEventID == req.PlanEventID && payload.ToolSessionID == req.ToolSessionID && payload.PreviousProviderSessionID == req.PreviousProviderSessionID && payload.AgentExecutor == req.AgentExecutor && payload.AgentModel == req.AgentModel && payload.AgentReasoningEffort == req.AgentReasoningEffort && payload.IdempotencyKey == req.IdempotencyKey && payload.ArgumentsHash == req.ArgumentsHash && payload.RequirementMapHash == req.RequirementMapHash
}
