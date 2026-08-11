package reporting

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const (
	// LongFormSectionPlanRepairCompletedEventType records the one valid planner
	// outcome allowed after terminal Section evidence gaps.
	LongFormSectionPlanRepairCompletedEventType = "report.plan.section_repair.completed"
	longFormSectionPlanRepairKind               = "sectional_markdown_report_plan_repair"
	sectionPlanRepairOutcomeApplied             = "applied"
	sectionPlanRepairOutcomeUnrepairable        = "unrepairable"
)

// ReportSectionCoordinate identifies one stable Section slot in a long-form plan.
type ReportSectionCoordinate struct {
	PartIndex    int `json:"part_index"`
	SectionIndex int `json:"section_index"`
}

// ReportSectionPlanReplacement changes one Section without moving its stable slot.
type ReportSectionPlanReplacement struct {
	ReportSectionCoordinate
	Section ReportPlanSection `json:"section"`
}

// LongFormSectionPlanRepairEventRequest carries one bounded plan amendment and
// the provider lineage that produced it. It never carries source excerpts.
type LongFormSectionPlanRepairEventRequest struct {
	MarkdownReportEventBase
	PlanEventID  string
	Coordinates  []ReportSectionCoordinate
	Replacements []ReportSectionPlanReplacement
	Unrepairable bool
}

// LongFormSectionPlanRepairResult is the validated effective plan reconstructed
// from the immutable canonical plan plus one durable repair event.
type LongFormSectionPlanRepairResult struct {
	Event        ledger.Event
	Plan         SectionalReportPlan
	Coordinates  []ReportSectionCoordinate
	Replacements []ReportSectionPlanReplacement
	Unrepairable bool
}

// LongFormSectionPlanRepairStore provides the conditional append needed to
// guarantee at most one repair event for a canonical plan lineage.
type LongFormSectionPlanRepairStore interface {
	AppendEventConditionally(context.Context, string, func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error)) (ledger.Event, bool, error)
}

// FinalizeLongFormSectionPlanRepair validates and conditionally records one
// same-coordinate plan amendment. A concurrent existing event wins only when it
// repairs the same coordinate set.
func FinalizeLongFormSectionPlanRepair(ctx context.Context, service LongFormSectionPlanRepairStore, original SectionalReportPlan, req LongFormSectionPlanRepairEventRequest) (LongFormSectionPlanRepairResult, error) {
	normalized, replacements, err := applySectionPlanReplacements(original, req.Replacements)
	if err != nil {
		return LongFormSectionPlanRepairResult{}, err
	}
	req.Coordinates = coordinatesForReplacements(replacements)
	req.Replacements = replacements
	req.Unrepairable = false
	return finalizeLongFormSectionPlanRepairOutcome(ctx, service, original, normalized, req)
}

// FinalizeLongFormSectionPlanUnrepairable durably consumes the one repair round
// when the planner finds no supportable replacement for every failed slot.
func FinalizeLongFormSectionPlanUnrepairable(ctx context.Context, service LongFormSectionPlanRepairStore, original SectionalReportPlan, req LongFormSectionPlanRepairEventRequest) (LongFormSectionPlanRepairResult, error) {
	normalized, err := NormalizeSectionalReportPlan(original)
	if err != nil {
		return LongFormSectionPlanRepairResult{}, err
	}
	req.Coordinates, err = normalizeSectionPlanRepairCoordinates(normalized, req.Coordinates)
	if err != nil {
		return LongFormSectionPlanRepairResult{}, err
	}
	req.Replacements = nil
	req.Unrepairable = true
	return finalizeLongFormSectionPlanRepairOutcome(ctx, service, original, normalized, req)
}

func finalizeLongFormSectionPlanRepairOutcome(ctx context.Context, service LongFormSectionPlanRepairStore, original, effective SectionalReportPlan, req LongFormSectionPlanRepairEventRequest) (LongFormSectionPlanRepairResult, error) {
	if service == nil {
		return LongFormSectionPlanRepairResult{}, fmt.Errorf("%w: Section plan repair store is required", producterror.ErrInvalidInput)
	}
	base := req.MarkdownReportEventBase
	var stored LongFormSectionPlanRepairResult
	event, _, err := service.AppendEventConditionally(ctx, base.MissionID, func(events []ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error) {
		parent, err := sectionPlanRepairParent(events, base.MissionID, req.PlanEventID, original)
		if err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		lineage, err := longFormPendingLineage(events, base.PendingEventID)
		if err != nil || !lineage[parent.pendingEventID] {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: Section plan repair pending lineage differs", producterror.ErrConflict)
		}
		if err := validateSectionPlanRepairRequest(req, parent); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		matches, err := matchingSectionPlanRepairs(events, req.PlanEventID, lineage)
		if err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		if len(matches) > 1 {
			return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: multiple Section plan repairs match one report plan", producterror.ErrConflict)
		}
		if len(matches) == 1 {
			result, err := decodeSectionPlanRepair(events, matches[0], parent, lineage)
			if err != nil {
				return ledger.AppendRequest{}, ledger.Event{}, false, err
			}
			if !sameRepairCoordinates(result.Coordinates, req.Coordinates) {
				return ledger.AppendRequest{}, ledger.Event{}, false, fmt.Errorf("%w: stored Section plan repair targets differ", producterror.ErrConflict)
			}
			stored = result
			return ledger.AppendRequest{}, matches[0], false, nil
		}
		if err := validateRepairTargetHistory(events, base.PendingEventID, req.PlanEventID, req.Coordinates, ""); err != nil {
			return ledger.AppendRequest{}, ledger.Event{}, false, err
		}
		request := BuildLongFormSectionPlanRepairCompletedAppendRequest(req)
		return request, ledger.Event{}, true, nil
	})
	if err != nil {
		return LongFormSectionPlanRepairResult{}, err
	}
	if stored.Event.EventID != "" {
		return stored, nil
	}
	return LongFormSectionPlanRepairResult{
		Event: event, Plan: effective, Coordinates: req.Coordinates,
		Replacements: req.Replacements, Unrepairable: req.Unrepairable,
	}, nil
}

// RecoverLongFormSectionPlanRepair returns the one validated amendment in the
// current retry lineage, if present.
func RecoverLongFormSectionPlanRepair(events []ledger.Event, missionID, pendingEventID, planEventID string, original SectionalReportPlan) (LongFormSectionPlanRepairResult, bool, error) {
	lineage, err := longFormPendingLineage(events, pendingEventID)
	if err != nil {
		return LongFormSectionPlanRepairResult{}, false, err
	}
	matches, err := matchingSectionPlanRepairs(events, planEventID, lineage)
	if err != nil {
		return LongFormSectionPlanRepairResult{}, false, err
	}
	if len(matches) > 1 {
		return LongFormSectionPlanRepairResult{}, false, fmt.Errorf("%w: multiple Section plan repairs match one report plan", producterror.ErrConflict)
	}
	if len(matches) == 0 {
		return LongFormSectionPlanRepairResult{}, false, nil
	}
	parent, err := sectionPlanRepairParent(events, missionID, planEventID, original)
	if err != nil {
		return LongFormSectionPlanRepairResult{}, false, err
	}
	result, err := decodeSectionPlanRepair(events, matches[0], parent, lineage)
	return result, err == nil, err
}
