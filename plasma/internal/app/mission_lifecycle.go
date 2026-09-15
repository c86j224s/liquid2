package app

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
)

// ArchiveMission applies the archive transition through the mission policy and
// appends/rebuilds it through the application storage boundary.
func (s *Service) ArchiveMission(ctx context.Context, req mission.MissionLifecycleChangeRequest) (mission.MissionLifecycleChangeResult, error) {
	return s.changeMissionLifecycle(ctx, req, mission.LifecycleArchived, mission.ArchivedEvent)
}

// RestoreMission applies the restore transition through the mission policy and
// appends/rebuilds it through the application storage boundary.
func (s *Service) RestoreMission(ctx context.Context, req mission.MissionLifecycleChangeRequest) (mission.MissionLifecycleChangeResult, error) {
	return s.changeMissionLifecycle(ctx, req, mission.LifecycleActive, mission.RestoredEvent)
}

func (s *Service) changeMissionLifecycle(ctx context.Context, req mission.MissionLifecycleChangeRequest, targetState, eventType string) (mission.MissionLifecycleChangeResult, error) {
	appendReq, err := mission.BuildLifecycleChange(req, targetState, eventType)
	if err != nil {
		return mission.MissionLifecycleChangeResult{}, err
	}
	var idempotent bool
	appended, err := s.appendLedgerEventsConditionally(ctx, req.MissionID, func(events []ledger.Event) ([]ledger.Event, error) {
		needed, err := mission.LifecycleChangeNeeded(req.MissionID, events, targetState)
		if err != nil {
			return nil, err
		}
		if !needed {
			idempotent = true
			return nil, nil
		}
		if err := validateNoActiveAgentWork(events); err != nil {
			return nil, err
		}
		event, err := buildLedgerEvent(appendReq)
		if err != nil {
			return nil, err
		}
		return []ledger.Event{event}, nil
	})
	if err != nil {
		return mission.MissionLifecycleChangeResult{}, err
	}
	projection, err := s.RebuildProjection(ctx, req.MissionID)
	if err != nil {
		return mission.MissionLifecycleChangeResult{}, err
	}
	result := mission.MissionLifecycleChangeResult{Projection: projection, Idempotent: idempotent}
	if len(appended) > 0 {
		result.Event = &appended[0]
	}
	return result, nil
}
