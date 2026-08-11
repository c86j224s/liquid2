package reporting

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func decodeSectionPlanRepair(events []ledger.Event, event ledger.Event, parent sectionPlanRepairParentState, lineage map[string]bool) (LongFormSectionPlanRepairResult, error) {
	payload, err := sectionPlanRepairPayload(event)
	if err != nil {
		return LongFormSectionPlanRepairResult{}, err
	}
	base := MarkdownReportEventBase{
		MissionID: event.MissionID, PendingEventID: payload.PendingEventID,
		AgentExecutor: payload.AgentExecutor, AgentModel: payload.AgentModel,
		AgentReasoningEffort: payload.AgentReasoningEffort, AgentSelectionSource: payload.AgentSelectionSource,
		AgentSessionID: payload.AgentSessionID, PreviousAgentSessionID: payload.PreviousAgentSessionID,
		ReturnedAgentSessionID: payload.ReturnedAgentSessionID, ToolSessionID: payload.ToolSessionID,
		ReportMode: payload.ReportMode, ReportSessionPolicy: payload.ReportSessionPolicy,
		ReportSessionPolicySelection: payload.ReportSessionPolicySelection,
		GenerationGuidanceProfile:    payload.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     payload.GenerationGuidanceSHA256,
		SessionChainKind:             payload.SessionChainKind, ReportPlanSessionID: payload.ReportPlanSessionID,
		ReportSessionID: payload.ReportSessionID, Producer: event.Producer,
	}
	req := LongFormSectionPlanRepairEventRequest{
		MarkdownReportEventBase: base, PlanEventID: payload.PlanEventID,
		Coordinates: payload.Coordinates, Replacements: payload.Replacements,
		Unrepairable: payload.Outcome == sectionPlanRepairOutcomeUnrepairable,
	}
	if payload.Kind != longFormSectionPlanRepairKind || payload.RepairRound != 1 || !lineage[strings.TrimSpace(payload.PendingEventID)] {
		return LongFormSectionPlanRepairResult{}, fmt.Errorf("%w: Section plan repair binding is invalid", producterror.ErrConflict)
	}
	if err := validateSectionPlanRepairRequest(req, parent); err != nil {
		return LongFormSectionPlanRepairResult{}, err
	}
	coordinates, err := normalizeSectionPlanRepairCoordinates(parent.plan, payload.Coordinates)
	if err != nil {
		return LongFormSectionPlanRepairResult{}, err
	}
	if err := validateRepairTargetHistory(events, payload.PendingEventID, payload.PlanEventID, coordinates, event.EventID); err != nil {
		return LongFormSectionPlanRepairResult{}, err
	}
	switch payload.Outcome {
	case sectionPlanRepairOutcomeApplied:
		plan, replacements, err := applySectionPlanReplacements(parent.plan, payload.Replacements)
		if err != nil || !sameRepairCoordinates(coordinatesForReplacements(replacements), coordinates) {
			return LongFormSectionPlanRepairResult{}, fmt.Errorf("%w: Section plan repair replacements differ from their coordinates", producterror.ErrConflict)
		}
		return LongFormSectionPlanRepairResult{Event: event, Plan: plan, Coordinates: coordinates, Replacements: replacements}, nil
	case sectionPlanRepairOutcomeUnrepairable:
		if len(payload.Replacements) != 0 {
			return LongFormSectionPlanRepairResult{}, fmt.Errorf("%w: unrepairable Section plan outcome has replacements", producterror.ErrConflict)
		}
		return LongFormSectionPlanRepairResult{Event: event, Plan: parent.plan, Coordinates: coordinates, Unrepairable: true}, nil
	default:
		return LongFormSectionPlanRepairResult{}, fmt.Errorf("%w: Section plan repair outcome is invalid", producterror.ErrConflict)
	}
}

func sectionPlanRepairPayload(event ledger.Event) (sectionPlanRepairEventPayload, error) {
	var payload sectionPlanRepairEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return payload, fmt.Errorf("%w: Section plan repair payload is invalid", producterror.ErrConflict)
	}
	return payload, nil
}
