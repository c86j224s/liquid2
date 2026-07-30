package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type finalEditPlannedPart struct {
	SectionCount int
}

func longFormPlanPipeline(events []app.LedgerEvent, pendingEventID string, planEventID string) (FinalEditPipelinePlanState, bool, error) {
	pendingEventID = strings.TrimSpace(pendingEventID)
	planEventID = strings.TrimSpace(planEventID)
	acceptedPending, err := longFormPendingLineage(events, pendingEventID)
	if err != nil {
		return FinalEditPipelinePlanState{}, false, err
	}
	for _, event := range events {
		if event.EventType != "report.plan.created" || strings.TrimSpace(event.EventID) != planEventID {
			continue
		}
		state, ok, err := FinalEditPipelineFromPlanEvent(event)
		if err != nil {
			return state, false, err
		}
		if !ok {
			continue
		}
		if !acceptedPending[state.PendingEventID] {
			return FinalEditPipelinePlanState{}, false, fmt.Errorf("%w: bound long-form plan pending differs", app.ErrConflict)
		}
		return state, true, nil
	}
	return FinalEditPipelinePlanState{}, false, nil
}

func validateLongFormFinalPipeline(events []app.LedgerEvent, binding LongFormFinalizeBinding, allowReaderStyleGate bool) error {
	plan, ok, err := longFormPlanPipeline(events, binding.PendingEventID, binding.PlanEventID)
	if err != nil {
		return err
	}
	if !ok || plan.Pipeline == "" {
		return nil
	}
	if !isSupportedFinalEditPipeline(plan.Pipeline) {
		return fmt.Errorf("%w: unsupported final edit pipeline", app.ErrConflict)
	}
	if !allowReaderStyleGate {
		return fmt.Errorf("%w: final edit pipeline canonical finalization requires corrective gate submission", app.ErrConflict)
	}
	if binding.CompositionStrategy != LongFormCompositionNarrativeEdit {
		return fmt.Errorf("%w: final edit pipeline finalization requires submitted manuscript Markdown", app.ErrConflict)
	}
	if binding.PostReportHumanize != plan.PostReportHumanize {
		return fmt.Errorf("%w: final edit pipeline finalization humanize setting differs from plan", app.ErrConflict)
	}
	return nil
}

func validateFinalEditReaderPlannedSections(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, binding FinalEditStageBinding, acceptedPending map[string]bool, planParts []finalEditPlannedPart) error {
	sectionIDs := make(map[[2]int]string)
	for _, event := range events {
		if event.EventType != "report.section.created" {
			continue
		}
		payload := eventPayload(event)
		if !acceptedPending[payloadString(payload, "pending_event_id")] {
			continue
		}
		if payloadString(payload, "plan_event_id") != binding.PlanEventID {
			return fmt.Errorf("%w: final edit reader source section plan lineage differs from binding", app.ErrConflict)
		}
		partIndex, sectionIndex := jsonInt(payload["part_index"]), jsonInt(payload["section_index"])
		if partIndex < 1 || partIndex > len(planParts) || sectionIndex < 1 || sectionIndex > planParts[partIndex-1].SectionCount {
			return fmt.Errorf("%w: final edit reader source section lineage is out of range", app.ErrConflict)
		}
		key := [2]int{partIndex, sectionIndex}
		if sectionIDs[key] != "" {
			return fmt.Errorf("%w: final edit reader source section lineage has duplicates", app.ErrConflict)
		}
		sectionIDs[key] = payloadString(payload, "artifact_id")
	}
	for partIndex, part := range planParts {
		for sectionIndex := 1; sectionIndex <= part.SectionCount; sectionIndex++ {
			artifactID := sectionIDs[[2]int{partIndex + 1, sectionIndex}]
			if artifactID == "" {
				return fmt.Errorf("%w: final edit reader source sections are incomplete", app.ErrConflict)
			}
			artifact, err := store.GetRawArtifact(ctx, artifactID)
			if err != nil {
				return err
			}
			if artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" || artifact.SHA256 != contentSHA256(artifact.Content) {
				return fmt.Errorf("%w: final edit reader source section artifact is foreign or not Markdown", app.ErrConflict)
			}
		}
	}
	return nil
}

func validateFinalEditReaderSourceArtifact(artifact app.RawArtifact, request app.CreateRawArtifactRequest) error {
	expectedSHA := contentSHA256(request.Content)
	if artifact.ArtifactID != request.ArtifactID ||
		artifact.MissionID != request.MissionID ||
		artifact.MediaType != request.MediaType ||
		artifact.Filename != request.Filename ||
		artifact.Producer != request.Producer ||
		artifact.SHA256 != expectedSHA {
		return fmt.Errorf("%w: existing reader source artifact differs from deterministic contract", app.ErrConflict)
	}
	return nil
}
