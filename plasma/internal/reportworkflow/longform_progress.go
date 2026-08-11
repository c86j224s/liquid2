package reportworkflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partassembly"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partedit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partplan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
)

type sectionIndex struct {
	part    int
	section int
}

type longFormProgress struct {
	currentSessionID string
	partPlans        map[int]partplan.Output
	sections         map[sectionIndex]sectiondraft.Draft
	sectionGaps      map[sectionIndex]sectiondraft.EvidenceGap
	parts            map[int]partassembly.PartDraft
	editedParts      map[int]partedit.PartDraft
}

func (progress longFormProgress) hasValidatedDownstream() bool {
	return len(progress.sections) > 0 || len(progress.parts) > 0 || len(progress.editedParts) > 0
}

// loadLongFormProgress는 retry lineage를 순회하며 stage-owned Recover 결과만
// prefix 진행 상태로 집계한다. root는 payload와 artifact를 직접 해석하지 않는다.
func (runner Runner) loadLongFormProgress(ctx context.Context, input DraftInput, planOut plan.LongFormOutput, repairOut plan.LongFormSectionRepairOutput) (longFormProgress, error) {
	events, err := runner.service.ListEvents(ctx, input.MissionID)
	if err != nil {
		return longFormProgress{}, err
	}
	progress := longFormProgress{
		currentSessionID: planOut.ReportPlanSessionID,
		partPlans:        map[int]partplan.Output{},
		sections:         map[sectionIndex]sectiondraft.Draft{},
		sectionGaps:      map[sectionIndex]sectiondraft.EvidenceGap{},
		parts:            map[int]partassembly.PartDraft{},
		editedParts:      map[int]partedit.PartDraft{},
	}
	lineage, err := plan.RecoveryLineage(events, input.PendingEventID)
	if err != nil {
		return longFormProgress{}, err
	}
	repairPosition, repairedCoordinates, err := sectionPlanRepairPosition(events, repairOut)
	if err != nil {
		return longFormProgress{}, err
	}
	for _, attemptID := range lineage {
		for eventPosition, event := range events {
			switch event.EventType {
			case reporting.PartPlanCreatedEventType:
				if planOut.PartPlanningEnabled {
					if err := applyPartPlan(attemptID, event, planOut, &progress); err != nil {
						return longFormProgress{}, err
					}
				}
			case sectiondraft.CreatedEventType:
				if err := runner.applySection(ctx, attemptID, event, planOut, &progress); err != nil {
					return longFormProgress{}, err
				}
			case sectiondraft.EvidenceGapEventType:
				if attemptID == input.PendingEventID {
					if evidenceGapPrecedesRepair(event, eventPosition, repairPosition, repairedCoordinates) {
						continue
					}
					if err := applySectionGap(input.PendingEventID, event, planOut, &progress); err != nil {
						return longFormProgress{}, err
					}
				}
			case partassembly.CreatedEventType:
				if err := runner.applyPart(ctx, attemptID, event, planOut, &progress); err != nil {
					return longFormProgress{}, err
				}
			case reporting.PartEditedEventType:
				if err := runner.applyPartEdit(ctx, attemptID, events, event, planOut, &progress); err != nil {
					return longFormProgress{}, err
				}
			}
		}
	}
	return progress, nil
}

func sectionPlanRepairPosition(events []ledger.Event, repairOut plan.LongFormSectionRepairOutput) (int, map[sectionIndex]bool, error) {
	coordinates := map[sectionIndex]bool{}
	if repairOut.Event.EventID == "" {
		return -1, coordinates, nil
	}
	position := -1
	for index, event := range events {
		if event.EventID == repairOut.Event.EventID {
			position = index
			break
		}
	}
	if position < 0 {
		return -1, nil, fmt.Errorf("%w: recovered Section plan repair event is missing", producterror.ErrConflict)
	}
	for _, replacement := range repairOut.Replacements {
		coordinates[sectionIndex{part: replacement.PartIndex - 1, section: replacement.SectionIndex - 1}] = true
	}
	return position, coordinates, nil
}

func evidenceGapPrecedesRepair(event ledger.Event, eventPosition, repairPosition int, coordinates map[sectionIndex]bool) bool {
	if repairPosition < 0 || eventPosition >= repairPosition {
		return false
	}
	var payload struct {
		PartIndex    int `json:"part_index"`
		SectionIndex int `json:"section_index"`
	}
	if json.Unmarshal(event.Payload, &payload) != nil {
		return false
	}
	return coordinates[sectionIndex{part: payload.PartIndex - 1, section: payload.SectionIndex - 1}]
}

// applyPartPlan은 Part-plan stage가 검증한 typed output만 plan 좌표에 병합한다.
func applyPartPlan(pendingEventID string, event ledger.Event, planOut plan.LongFormOutput, progress *longFormProgress) error {
	out, ok, err := partplan.Recover(partplan.RecoverInput{
		Event: event, MissionID: planOut.Event.MissionID, PendingEventID: pendingEventID,
		PlanEventID: planOut.Event.EventID, PartCount: len(planOut.Plan.Parts),
		AgentExecutor: planOut.AgentExecutor, AgentModel: planOut.AgentModel,
		AgentReasoningEffort: planOut.AgentReasoningEffort, AgentSelectionSource: planOut.AgentSelectionSource,
		ReportSessionPolicy: planOut.ReportSessionPolicy, ReportSessionPolicySelection: planOut.ReportSessionPolicySelection,
		GenerationGuidanceProfile: planOut.GenerationGuidanceProfile, GenerationGuidanceSHA256: planOut.GenerationGuidanceSHA256,
		SessionChainKind: planOut.SessionChainKind, ReportPlanSessionID: planOut.ReportPlanSessionID,
	})
	if err != nil || !ok {
		return err
	}
	if _, exists := progress.partPlans[out.PartIndex]; exists {
		return fmt.Errorf("%w: multiple recovered Part plans match one Part", producterror.ErrConflict)
	}
	progress.partPlans[out.PartIndex] = out
	return nil
}

// applySection은 Section stage Recover가 검증한 Markdown draft만 집계한다.
func (runner Runner) applySection(ctx context.Context, pendingEventID string, event ledger.Event, planOut plan.LongFormOutput, progress *longFormProgress) error {
	out, ok, err := sectiondraft.Recover(ctx, sectiondraft.RecoverInput{
		Service: runner.service, Event: event, MissionID: planOut.Event.MissionID,
		PendingEventID: pendingEventID, PlanEventID: planOut.Event.EventID, Plan: planOut.Plan,
	})
	if err != nil || !ok {
		return err
	}
	key := sectionIndex{part: out.PartIndex, section: out.SectionIndex}
	if _, exists := progress.sections[key]; exists {
		return fmt.Errorf("%w: multiple recovered Sections match one plan coordinate", producterror.ErrConflict)
	}
	progress.sections[key] = out.Draft
	delete(progress.sectionGaps, key)
	if out.Draft.SessionID != "" {
		progress.currentSessionID = out.Draft.SessionID
	}
	return nil
}

// applySectionGap records only the latest valid gap attempt for the current
// pending event. A later created Section removes the gap in applySection.
func applySectionGap(pendingEventID string, event ledger.Event, planOut plan.LongFormOutput, progress *longFormProgress) error {
	out, ok, err := sectiondraft.RecoverEvidenceGap(sectiondraft.RecoverEvidenceGapInput{
		Event: event, MissionID: planOut.Event.MissionID, PendingEventID: pendingEventID,
		PlanEventID: planOut.Event.EventID, Plan: planOut.Plan,
		AgentExecutor:       planOut.AgentExecutor,
		SessionChainKind:    planOut.SessionChainKind,
		ReportPlanSessionID: planOut.ReportPlanSessionID,
	})
	if err != nil || !ok {
		return err
	}
	key := sectionIndex{part: out.PartIndex - 1, section: out.SectionIndex - 1}
	if _, complete := progress.sections[key]; complete {
		return nil
	}
	current, exists := progress.sectionGaps[key]
	if !exists {
		if out.Attempt != 1 {
			return fmt.Errorf("%w: recovered section evidence gap attempt %d without attempt 1", producterror.ErrConflict, out.Attempt)
		}
		progress.sectionGaps[key] = out
		if out.SessionID != "" {
			progress.currentSessionID = out.SessionID
		}
		return nil
	}
	if out.Attempt != current.Attempt+1 {
		return fmt.Errorf("%w: recovered section evidence gap attempt transition %d to %d is invalid", producterror.ErrConflict, current.Attempt, out.Attempt)
	}
	if strings.TrimSpace(out.SessionID) != strings.TrimSpace(current.SessionID) ||
		strings.TrimSpace(out.ToolSessionID) != strings.TrimSpace(current.ToolSessionID) {
		return fmt.Errorf("%w: recovered section evidence gap changed retry session binding", producterror.ErrConflict)
	}
	if out.SourceSessionID != current.SourceSessionID {
		return fmt.Errorf("%w: recovered section evidence gap changed source session lineage", producterror.ErrConflict)
	}
	progress.sectionGaps[key] = out
	if out.SessionID != "" {
		progress.currentSessionID = out.SessionID
	}
	return nil
}

// applyPart는 Part assembly stage Recover가 검증한 Part draft만 집계한다.
func (runner Runner) applyPart(ctx context.Context, pendingEventID string, event ledger.Event, planOut plan.LongFormOutput, progress *longFormProgress) error {
	out, ok, err := partassembly.Recover(ctx, partassembly.RecoverInput{
		Service: runner.service, Event: event, MissionID: planOut.Event.MissionID,
		PendingEventID: pendingEventID, PlanEventID: planOut.Event.EventID, Plan: planOut.Plan,
	})
	if err != nil || !ok {
		return err
	}
	if _, exists := progress.parts[out.PartIndex]; exists {
		return fmt.Errorf("%w: multiple recovered Parts match one plan coordinate", producterror.ErrConflict)
	}
	progress.parts[out.PartIndex] = out.Draft
	if out.Draft.SessionID != "" {
		progress.currentSessionID = out.Draft.SessionID
	}
	return nil
}

// applyPartEdit은 Part-edit stage Recover가 검증한 outcome만 source Part 자리에 병합한다.
func (runner Runner) applyPartEdit(ctx context.Context, pendingEventID string, events []ledger.Event, event ledger.Event, planOut plan.LongFormOutput, progress *longFormProgress) error {
	out, ok, err := partedit.Recover(ctx, partedit.RecoverInput{
		Service: runner.service, Event: event, Events: events, MissionID: planOut.Event.MissionID,
		PendingEventID: pendingEventID, PlanEventID: planOut.Event.EventID, Plan: planOut.Plan,
		Sources:       partEditSources(progress.parts),
		AgentExecutor: planOut.AgentExecutor, AgentModel: planOut.AgentModel,
		AgentReasoningEffort: planOut.AgentReasoningEffort, AgentSelectionSource: planOut.AgentSelectionSource,
		MCPMode: planOut.MCPMode, ReportSessionPolicy: planOut.ReportSessionPolicy,
		ReportSessionPolicySelection: planOut.ReportSessionPolicySelection,
		GenerationGuidanceProfile:    planOut.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     planOut.GenerationGuidanceSHA256,
		SessionChainKind:             planOut.SessionChainKind,
		ReportPlanSessionID:          planOut.ReportPlanSessionID,
	})
	if err != nil || !ok {
		return err
	}
	if _, exists := progress.editedParts[out.PartIndex]; exists {
		return fmt.Errorf("%w: multiple recovered Part edits match one Part", producterror.ErrConflict)
	}
	progress.editedParts[out.PartIndex] = out.Draft
	return nil
}

func partEditSources(parts map[int]partassembly.PartDraft) map[int]partedit.PartDraft {
	sources := make(map[int]partedit.PartDraft, len(parts))
	for index, draft := range parts {
		sources[index] = partedit.PartDraft{
			Title: draft.Title, Markdown: draft.Markdown, ArtifactID: draft.ArtifactID, WordCount: draft.WordCount,
		}
	}
	return sources
}
