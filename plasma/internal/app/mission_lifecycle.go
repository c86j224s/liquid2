package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	MissionLifecycleActive   = "active"
	MissionLifecycleArchived = "archived"

	MissionArchivedEvent = "mission.archived"
	MissionRestoredEvent = "mission.restored"
)

type MissionLifecycleChangeRequest struct {
	EventID   string
	MissionID string
	Producer  Producer
	Reason    string
}

type MissionLifecycleChangeResult struct {
	Event      *LedgerEvent      `json:"event,omitempty"`
	Projection MissionProjection `json:"projection"`
	Idempotent bool              `json:"idempotent,omitempty"`
}

func (s *Service) ArchiveMission(ctx context.Context, req MissionLifecycleChangeRequest) (MissionLifecycleChangeResult, error) {
	return s.changeMissionLifecycle(ctx, req, MissionLifecycleArchived, MissionArchivedEvent)
}

func (s *Service) RestoreMission(ctx context.Context, req MissionLifecycleChangeRequest) (MissionLifecycleChangeResult, error) {
	return s.changeMissionLifecycle(ctx, req, MissionLifecycleActive, MissionRestoredEvent)
}

func (s *Service) changeMissionLifecycle(ctx context.Context, req MissionLifecycleChangeRequest, targetState, eventType string) (MissionLifecycleChangeResult, error) {
	if err := validateID("evt_", req.EventID); err != nil {
		return MissionLifecycleChangeResult{}, err
	}
	if err := validateID("mis_", req.MissionID); err != nil {
		return MissionLifecycleChangeResult{}, err
	}
	if req.Producer.Type != "user" {
		return MissionLifecycleChangeResult{}, fmt.Errorf("%w: mission lifecycle updates require a user producer", ErrInvalidInput)
	}
	payload, err := json.Marshal(map[string]any{
		"lifecycle_state": targetState,
		"reason":          strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return MissionLifecycleChangeResult{}, err
	}
	var idempotent bool
	appended, err := s.appendLedgerEventsConditionally(ctx, req.MissionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		if len(events) == 0 {
			return nil, fmt.Errorf("%w: mission does not exist", ErrInvalidInput)
		}
		projection, err := BuildProjection(req.MissionID, events)
		if err != nil {
			return nil, err
		}
		if normalizeMissionLifecycleState(projection.LifecycleState) == targetState {
			idempotent = true
			return nil, nil
		}
		if err := validateNoActiveAgentWork(events); err != nil {
			return nil, err
		}
		event, err := buildLedgerEvent(AppendEventRequest{
			EventID:   req.EventID,
			MissionID: req.MissionID,
			EventType: eventType,
			Producer:  req.Producer,
			Payload:   payload,
		})
		if err != nil {
			return nil, err
		}
		return []LedgerEvent{event}, nil
	})
	if err != nil {
		return MissionLifecycleChangeResult{}, err
	}
	projection, err := s.RebuildProjection(ctx, req.MissionID)
	if err != nil {
		return MissionLifecycleChangeResult{}, err
	}
	result := MissionLifecycleChangeResult{Projection: projection, Idempotent: idempotent}
	if len(appended) > 0 {
		result.Event = &appended[0]
	}
	return result, nil
}

func normalizeMissionLifecycleState(value string) string {
	switch strings.TrimSpace(value) {
	case MissionLifecycleArchived:
		return MissionLifecycleArchived
	default:
		return MissionLifecycleActive
	}
}
