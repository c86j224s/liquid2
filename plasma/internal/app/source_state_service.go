package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	SourceRemovedEvent  = "source.removed"
	SourceRestoredEvent = "source.restored"
)

type sourceStateEventPayload struct {
	SnapshotID string `json:"snapshot_id"`
	Reason     string `json:"reason,omitempty"`
}

type sourceUpdatedEventPayload struct {
	OldSnapshotID string `json:"old_snapshot_id"`
	NewSnapshotID string `json:"new_snapshot_id"`
}

func (s *Service) ListSourceSnapshotsWithState(ctx context.Context, req ListSourceSnapshotsRequest) ([]SourceSnapshot, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	store, ok := s.store.(SourceSnapshotListStore)
	if !ok {
		return nil, fmt.Errorf("%w: source snapshot list store is required", ErrInvalidInput)
	}
	snapshots, err := store.ListSourceSnapshots(ctx, missionID)
	if err != nil {
		return nil, err
	}
	states, err := s.sourceStateMap(ctx, missionID)
	if err != nil {
		return nil, err
	}
	filtered := make([]SourceSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		state := states[snapshot.SnapshotID]
		if state.State == "" {
			state.State = SourceStateActive
		}
		snapshot.State = state
		if snapshot.State.Removed && !req.IncludeRemoved {
			continue
		}
		if snapshot.State.Superseded && !req.IncludeSuperseded {
			continue
		}
		filtered = append(filtered, snapshot)
	}
	return filtered, nil
}

func (s *Service) sourceState(ctx context.Context, missionID string, snapshotID string) (SourceState, error) {
	states, err := s.sourceStateMap(ctx, missionID)
	if err != nil {
		return SourceState{}, err
	}
	state := states[strings.TrimSpace(snapshotID)]
	if state.State == "" {
		state.State = SourceStateActive
	}
	return state, nil
}

func (s *Service) sourceStateMap(ctx context.Context, missionID string) (map[string]SourceState, error) {
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return nil, err
	}
	states := map[string]SourceState{}
	for _, event := range events {
		switch event.EventType {
		case SourceRemovedEvent:
			payload, ok := decodeSourceStatePayload(event)
			if !ok {
				continue
			}
			previous := states[payload.SnapshotID]
			states[payload.SnapshotID] = preserveSourceSupersededState(SourceState{
				State:          SourceStateRemoved,
				Removed:        true,
				RemovedAt:      eventTime(event),
				RemovedEventID: event.EventID,
				RemovedReason:  strings.TrimSpace(payload.Reason),
			}, previous)
		case SourceRestoredEvent:
			payload, ok := decodeSourceStatePayload(event)
			if !ok {
				continue
			}
			previous := states[payload.SnapshotID]
			states[payload.SnapshotID] = preserveSourceSupersededState(SourceState{
				State:           SourceStateActive,
				Removed:         false,
				RestoredAt:      eventTime(event),
				RestoredEventID: event.EventID,
				RemovedAt:       previous.RemovedAt,
				RemovedEventID:  previous.RemovedEventID,
				RemovedReason:   previous.RemovedReason,
			}, previous)
		case ConfluenceUpdatedEvent:
			payload, ok := decodeSourceUpdatedPayload(event)
			if !ok {
				continue
			}
			previous := states[payload.OldSnapshotID]
			if previous.State == "" {
				previous.State = SourceStateActive
			}
			previous.Superseded = true
			previous.SupersededAt = eventTime(event)
			previous.SupersededBy = payload.NewSnapshotID
			previous.SupersededEventID = event.EventID
			states[payload.OldSnapshotID] = previous
		case ConfluenceUpdateCurrentEvent, ConfluenceUpdateAvailableEvent, ConfluenceUpdateFailedEvent:
			applyConfluenceUpdateState(states, event)
		}
	}
	return states, nil
}

func preserveSourceSupersededState(next SourceState, previous SourceState) SourceState {
	next.Superseded = previous.Superseded
	next.SupersededAt = previous.SupersededAt
	next.SupersededBy = previous.SupersededBy
	next.SupersededEventID = previous.SupersededEventID
	next.ConfluenceUpdate = previous.ConfluenceUpdate
	return next
}

func decodeSourceStatePayload(event LedgerEvent) (sourceStateEventPayload, bool) {
	var payload sourceStateEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return sourceStateEventPayload{}, false
	}
	payload.SnapshotID = strings.TrimSpace(payload.SnapshotID)
	return payload, payload.SnapshotID != ""
}

func decodeSourceUpdatedPayload(event LedgerEvent) (sourceUpdatedEventPayload, bool) {
	var payload sourceUpdatedEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return sourceUpdatedEventPayload{}, false
	}
	payload.OldSnapshotID = strings.TrimSpace(payload.OldSnapshotID)
	payload.NewSnapshotID = strings.TrimSpace(payload.NewSnapshotID)
	return payload, payload.OldSnapshotID != "" && payload.NewSnapshotID != ""
}

func validateSourceStateEventPayload(eventType string, payload json.RawMessage) error {
	switch eventType {
	case SourceRemovedEvent, SourceRestoredEvent:
		var typed sourceStateEventPayload
		if err := json.Unmarshal(payload, &typed); err != nil {
			return fmt.Errorf("%w: invalid source state payload", ErrInvalidInput)
		}
		if err := validateID("src_", strings.TrimSpace(typed.SnapshotID)); err != nil {
			return err
		}
		return nil
	case ConfluenceUpdatedEvent:
		var typed sourceUpdatedEventPayload
		if err := json.Unmarshal(payload, &typed); err != nil {
			return fmt.Errorf("%w: invalid source update payload", ErrInvalidInput)
		}
		if err := validateID("src_", strings.TrimSpace(typed.OldSnapshotID)); err != nil {
			return err
		}
		if err := validateID("src_", strings.TrimSpace(typed.NewSnapshotID)); err != nil {
			return err
		}
		return nil
	case ConfluenceUpdateFailedEvent:
		return validateConfluenceUpdateStateEventPayload(payload)
	default:
		return nil
	}
}

func eventTime(event LedgerEvent) time.Time {
	if event.CreatedAt.IsZero() {
		return time.Now().UTC()
	}
	return event.CreatedAt
}
