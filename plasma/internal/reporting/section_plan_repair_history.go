package reporting

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

type sectionPlanRepairAttempts struct {
	first   int
	second  int
	created int
}

// validateRepairTargetHistory proves that every amended coordinate exhausted
// the original two-attempt evidence budget without producing a Section.
func validateRepairTargetHistory(events []ledger.Event, pendingEventID, planEventID string, coordinates []ReportSectionCoordinate, stopBeforeEventID string) error {
	states := map[ReportSectionCoordinate]*sectionPlanRepairAttempts{}
	for _, coordinate := range coordinates {
		states[coordinate] = &sectionPlanRepairAttempts{}
	}
	for _, event := range events {
		if stopBeforeEventID != "" && event.EventID == stopBeforeEventID {
			break
		}
		if event.EventType != "report.section.evidence_gap" && event.EventType != "report.section.created" {
			continue
		}
		var payload struct {
			PendingEventID string `json:"pending_event_id"`
			PlanEventID    string `json:"plan_event_id"`
			PartIndex      int    `json:"part_index"`
			SectionIndex   int    `json:"section_index"`
			Attempt        int    `json:"attempt_number"`
			ReasonCode     string `json:"reason_code"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil ||
			strings.TrimSpace(payload.PendingEventID) != strings.TrimSpace(pendingEventID) ||
			strings.TrimSpace(payload.PlanEventID) != strings.TrimSpace(planEventID) {
			continue
		}
		state := states[ReportSectionCoordinate{PartIndex: payload.PartIndex, SectionIndex: payload.SectionIndex}]
		if state == nil {
			continue
		}
		if event.EventType == "report.section.created" {
			state.created++
			continue
		}
		if strings.TrimSpace(payload.ReasonCode) != "inadequate_section_evidence" {
			return fmt.Errorf("%w: Section plan repair gap reason is invalid", producterror.ErrConflict)
		}
		switch payload.Attempt {
		case 1:
			state.first++
		case 2:
			state.second++
		default:
			return fmt.Errorf("%w: Section plan repair gap attempt is invalid", producterror.ErrConflict)
		}
	}
	for _, state := range states {
		if state.first != 1 || state.second != 1 || state.created != 0 {
			return fmt.Errorf("%w: Section plan repair target lacks one terminal evidence-gap sequence", producterror.ErrConflict)
		}
	}
	return nil
}
