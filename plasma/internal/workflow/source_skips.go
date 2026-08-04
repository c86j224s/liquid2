package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// appendRemovedSourceSkips records sources removed after the workflow began so
// later steps do not silently reuse them.
func (runner Runner) appendRemovedSourceSkips(ctx context.Context, view workflowstate.WorkflowRunView, stepID string, stepIndex int) error {
	sources, err := runner.Service.ListSourceSnapshotsWithState(ctx, source.ListRequest{MissionID: view.MissionID, IncludeRemoved: true})
	if err != nil {
		return err
	}
	events, err := runner.Service.ListEvents(ctx, view.MissionID)
	if err != nil {
		return err
	}
	byID := eventsByID(events)
	boundarySequence := workflowRunStartBoundarySequence(view, byID)
	alreadySkipped := workflowSourceSkipKeys(events, view.WorkflowRunID)
	for _, source := range sources {
		if !source.State.Removed {
			continue
		}
		removedEventID := strings.TrimSpace(source.State.RemovedEventID)
		if removedEventID == "" {
			continue
		}
		removedEvent, ok := byID[removedEventID]
		if ok && boundarySequence > 0 && removedEvent.Sequence <= boundarySequence {
			continue
		}
		key := source.SnapshotID + "|" + removedEventID
		if _, ok := alreadySkipped[key]; ok {
			continue
		}
		if _, err := runner.appendWorkflowEvent(ctx, view.MissionID, workflowstate.WorkflowSourceSkippedEvent, workflowstate.WorkflowSourceSkippedPayload{
			WorkflowRunID:   view.WorkflowRunID,
			MissionID:       view.MissionID,
			WorkflowStepID:  stepID,
			StepIndex:       stepIndex,
			SnapshotID:      source.SnapshotID,
			Reason:          "source_removed",
			RemovedEventID:  removedEventID,
			SkippedAt:       runner.now().Format(time.RFC3339Nano),
			RetrievalPolicy: source.Access.RetrievalPolicy,
			ConnectorType:   source.Connector.ConnectorType,
		}, ledger.Producer{Type: "workflow", ID: view.WorkflowRunID}); err != nil {
			return err
		}
		alreadySkipped[key] = struct{}{}
	}
	return nil
}

func eventsByID(events []ledger.Event) map[string]ledger.Event {
	byID := make(map[string]ledger.Event, len(events))
	for _, event := range events {
		byID[event.EventID] = event
	}
	return byID
}

func workflowRunStartBoundarySequence(view workflowstate.WorkflowRunView, byID map[string]ledger.Event) int64 {
	for _, eventID := range []string{view.StartedEventID, view.RequestedEventID} {
		event, ok := byID[strings.TrimSpace(eventID)]
		if ok {
			return event.Sequence
		}
	}
	return 0
}

func workflowSourceSkipKeys(events []ledger.Event, workflowRunID string) map[string]struct{} {
	keys := map[string]struct{}{}
	for _, event := range events {
		if event.EventType != workflowstate.WorkflowSourceSkippedEvent {
			continue
		}
		var payload workflowstate.WorkflowSourceSkippedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.WorkflowRunID) != strings.TrimSpace(workflowRunID) {
			continue
		}
		snapshotID := strings.TrimSpace(payload.SnapshotID)
		removedEventID := strings.TrimSpace(payload.RemovedEventID)
		if snapshotID == "" || removedEventID == "" {
			continue
		}
		keys[snapshotID+"|"+removedEventID] = struct{}{}
	}
	return keys
}
