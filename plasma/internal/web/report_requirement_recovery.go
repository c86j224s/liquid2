package web

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func reportRequirementStageMatches(pendingEventID, planEventID string, event app.LedgerEvent) bool {
	var payload struct {
		PendingEventID string `json:"pending_event_id"`
		PlanEventID    string `json:"plan_event_id"`
	}
	return json.Unmarshal(event.Payload, &payload) == nil && strings.TrimSpace(payload.PendingEventID) == pendingEventID && strings.TrimSpace(payload.PlanEventID) == planEventID
}

func applyReportRequirementMapProgress(pendingEventID, planEventID string, event app.LedgerEvent, progress *sectionalReportProgress) error {
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
	normalized, err := reporting.NormalizeReportRequirementMap(payload.RequirementMap, progress.plan)
	if err != nil {
		return fmt.Errorf("%w: recovered report requirement map is invalid", app.ErrConflict)
	}
	if progress.hasRequirementMap {
		currentHash, _, _ := reporting.ReportRequirementMapHash(progress.requirementMap)
		nextHash, _, _ := reporting.ReportRequirementMapHash(normalized)
		if currentHash != nextHash {
			return fmt.Errorf("%w: recovered report requirement maps conflict", app.ErrConflict)
		}
		return nil
	}
	progress.hasRequirementMap = true
	progress.hasRequirementStage = true
	progress.requirementMapEvent = event
	progress.requirementMap = normalized
	return nil
}
