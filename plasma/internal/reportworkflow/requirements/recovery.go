package requirements

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const maxReviewedEvents = 256

type recoveredState struct {
	hasMap                bool
	hasStage              bool
	hasValidatedWorkStart bool
	event                 ledger.Event
	requirementMap        reporting.ReportRequirementMap
}

func recoverState(events []ledger.Event, pendingEventID, planEventID string, plan reporting.SectionalReportPlan) (recoveredState, error) {
	var state recoveredState
	lineage, err := recoveryLineage(events, pendingEventID)
	if err != nil {
		return recoveredState{}, err
	}
	for _, attemptID := range lineage {
		for _, event := range events {
			switch event.EventType {
			case reporting.ReportRequirementsStartedEventType:
				if stageMatches(attemptID, planEventID, event) {
					state.hasStage = true
				}
			case reporting.ReportRequirementsMappedEventType:
				if err := applyMap(attemptID, planEventID, event, plan, &state); err != nil {
					return recoveredState{}, err
				}
			case "report.section.started":
				if sectionStartedMatches(attemptID, planEventID, event) {
					state.hasValidatedWorkStart = true
				}
			}
		}
	}
	return state, nil
}

func stageMatches(pendingEventID, planEventID string, event ledger.Event) bool {
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
		PlanEventID    string `json:"plan_event_id"`
	}
	return json.Unmarshal(event.Payload, &payload) == nil &&
		strings.TrimSpace(payload.PendingEventID) == pendingEventID &&
		strings.TrimSpace(payload.PlanEventID) == planEventID
}

func sectionStartedMatches(pendingEventID, planEventID string, event ledger.Event) bool {
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
		PlanEventID    string `json:"plan_event_id"`
		PartIndex      int    `json:"part_index"`
		SectionIndex   int    `json:"section_index"`
	}
	return json.Unmarshal(event.Payload, &payload) == nil &&
		strings.TrimSpace(payload.PendingEventID) == pendingEventID &&
		strings.TrimSpace(payload.PlanEventID) == planEventID &&
		payload.PartIndex > 0 &&
		payload.SectionIndex > 0
}

func applyMap(pendingEventID, planEventID string, event ledger.Event, plan reporting.SectionalReportPlan, state *recoveredState) error {
	var payload struct {
		PendingEventID string                         `json:"pending_event_id"`
		PlanEventID    string                         `json:"plan_event_id"`
		RequirementMap reporting.ReportRequirementMap `json:"requirement_map"`
	}
	if json.Unmarshal(event.Payload, &payload) != nil {
		return nil
	}
	if strings.TrimSpace(payload.PendingEventID) != pendingEventID || strings.TrimSpace(payload.PlanEventID) != planEventID {
		return nil
	}
	normalized, err := reporting.NormalizeReportRequirementMap(payload.RequirementMap, plan)
	if err != nil {
		return fmt.Errorf("%w: recovered report requirement map is invalid", producterror.ErrConflict)
	}
	if state.hasMap {
		currentHash, _, _ := reporting.ReportRequirementMapHash(state.requirementMap)
		nextHash, _, _ := reporting.ReportRequirementMapHash(normalized)
		if currentHash != nextHash {
			return fmt.Errorf("%w: recovered report requirement maps conflict", producterror.ErrConflict)
		}
		return nil
	}
	state.hasMap = true
	state.event = event
	state.requirementMap = normalized
	return nil
}

func reviewEventIDs(events []ledger.Event, pendingEventID string) ([]string, error) {
	pendingEventID = strings.TrimSpace(pendingEventID)
	if pendingEventID == "" {
		return nil, fmt.Errorf("%w: report requirement pending event is required", producterror.ErrInvalidInput)
	}
	result := make([]string, 0)
	foundPending := false
	for _, event := range events {
		if event.EventID == pendingEventID {
			if event.EventType != "report.draft.pending" || event.Producer.Type != "user" {
				return nil, fmt.Errorf("%w: report requirement pending event is invalid", producterror.ErrInvalidInput)
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
		return nil, fmt.Errorf("%w: report requirement pending event is missing", producterror.ErrInvalidInput)
	}
	if len(result) > maxReviewedEvents {
		return nil, fmt.Errorf("%w: too many user events for report requirement review", producterror.ErrInvalidInput)
	}
	return result, nil
}
