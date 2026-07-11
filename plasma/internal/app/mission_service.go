package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

var ErrInvalidInput = errors.New("invalid input")
var ErrConflict = errors.New("conflict")

type MissionStore interface {
	CreateMission(context.Context, Mission) error
	AppendLedgerEvent(context.Context, LedgerEvent) (LedgerEvent, error)
	ListLedgerEvents(context.Context, string) ([]LedgerEvent, error)
}

type MissionListStore interface {
	ListMissions(context.Context) ([]Mission, error)
}

type ConditionalLedgerStore interface {
	AppendLedgerEventsConditionally(context.Context, string, func([]LedgerEvent) ([]LedgerEvent, error)) ([]LedgerEvent, error)
}

func (s *Service) CreateMission(ctx context.Context, req CreateMissionRequest) (Mission, error) {
	if err := validateID("mis_", req.MissionID); err != nil {
		return Mission{}, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return Mission{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	now := time.Now().UTC()
	mission := Mission{
		MissionID: req.MissionID,
		Title:     strings.TrimSpace(req.Title),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.CreateMission(ctx, mission); err != nil {
		return Mission{}, err
	}
	return mission, nil
}

func BuildMissionCreatedAppendRequest(req MissionCreatedEventRequest) AppendEventRequest {
	return AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "mission.created",
		Producer:  req.Producer,
		Payload: mustJSON(map[string]any{
			"title":     req.Title,
			"objective": req.Objective,
			"scope":     req.Scope,
		}),
	}
}

func (s *Service) AppendEvent(ctx context.Context, req AppendEventRequest) (LedgerEvent, error) {
	event, err := buildLedgerEvent(req)
	if err != nil {
		return LedgerEvent{}, err
	}
	if EventLocksAgentExecutor(event.EventType) {
		appended, err := s.appendLedgerEventsConditionally(ctx, event.MissionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
			if err := ValidateAgentExecutorAppend(events, []LedgerEvent{event}); err != nil {
				return nil, err
			}
			return []LedgerEvent{event}, nil
		})
		if err != nil {
			return LedgerEvent{}, err
		}
		if len(appended) != 1 {
			return LedgerEvent{}, fmt.Errorf("%w: expected one appended event", ErrInvalidInput)
		}
		return appended[0], nil
	}
	return s.store.AppendLedgerEvent(ctx, event)
}

func (s *Service) AppendEvents(ctx context.Context, missionID string, reqs []AppendEventRequest) ([]LedgerEvent, error) {
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("%w: at least one event is required", ErrInvalidInput)
	}
	return s.appendLedgerEventsConditionally(ctx, missionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		built := make([]LedgerEvent, 0, len(reqs))
		for _, req := range reqs {
			if strings.TrimSpace(req.MissionID) != missionID {
				return nil, fmt.Errorf("%w: event mission_id must match %s", ErrInvalidInput, missionID)
			}
			event, err := buildLedgerEvent(req)
			if err != nil {
				return nil, err
			}
			built = append(built, event)
		}
		if err := ValidateAgentExecutorAppend(events, built); err != nil {
			return nil, err
		}
		return built, nil
	})
}

func (s *Service) AppendEventsIfNoActiveAgentWork(ctx context.Context, missionID string, reqs []AppendEventRequest) ([]LedgerEvent, error) {
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("%w: at least one event is required", ErrInvalidInput)
	}
	return s.appendLedgerEventsConditionally(ctx, missionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
		if err := validateNoActiveAgentWork(events); err != nil {
			return nil, err
		}
		built := make([]LedgerEvent, 0, len(reqs))
		for _, req := range reqs {
			if strings.TrimSpace(req.MissionID) != missionID {
				return nil, fmt.Errorf("%w: event mission_id must match %s", ErrInvalidInput, missionID)
			}
			event, err := buildLedgerEvent(req)
			if err != nil {
				return nil, err
			}
			built = append(built, event)
		}
		if err := ValidateAgentExecutorAppend(events, built); err != nil {
			return nil, err
		}
		return built, nil
	})
}

func (s *Service) ListMissions(ctx context.Context) ([]Mission, error) {
	store, ok := s.store.(MissionListStore)
	if !ok {
		return nil, fmt.Errorf("%w: mission list store is required", ErrInvalidInput)
	}
	return store.ListMissions(ctx)
}

func buildLedgerEvent(req AppendEventRequest) (LedgerEvent, error) {
	if err := validateID("evt_", req.EventID); err != nil {
		return LedgerEvent{}, err
	}
	if err := validateID("mis_", req.MissionID); err != nil {
		return LedgerEvent{}, err
	}
	if strings.TrimSpace(req.EventType) == "" {
		return LedgerEvent{}, fmt.Errorf("%w: event type is required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Producer.Type) == "" || strings.TrimSpace(req.Producer.ID) == "" {
		return LedgerEvent{}, fmt.Errorf("%w: producer type and id are required", ErrInvalidInput)
	}
	payload := req.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if !json.Valid(payload) {
		return LedgerEvent{}, fmt.Errorf("%w: payload must be valid JSON", ErrInvalidInput)
	}
	if err := validateWorkflowEventPayload(strings.TrimSpace(req.EventType), strings.TrimSpace(req.MissionID), payload); err != nil {
		return LedgerEvent{}, err
	}
	if err := validateSourceStateEventPayload(strings.TrimSpace(req.EventType), payload); err != nil {
		return LedgerEvent{}, err
	}

	event := LedgerEvent{
		EventID:          req.EventID,
		MissionID:        req.MissionID,
		EventType:        strings.TrimSpace(req.EventType),
		Producer:         Producer{Type: strings.TrimSpace(req.Producer.Type), ID: strings.TrimSpace(req.Producer.ID)},
		CausationEventID: strings.TrimSpace(req.CausationEventID),
		CorrelationID:    strings.TrimSpace(req.CorrelationID),
		Payload:          append(json.RawMessage(nil), payload...),
		CreatedAt:        time.Now().UTC(),
	}
	return event, nil
}

func (s *Service) ListEvents(ctx context.Context, missionID string) ([]LedgerEvent, error) {
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	return s.store.ListLedgerEvents(ctx, missionID)
}

func (s *Service) appendLedgerEventsConditionally(ctx context.Context, missionID string, build func([]LedgerEvent) ([]LedgerEvent, error)) ([]LedgerEvent, error) {
	if store, ok := s.store.(ConditionalLedgerStore); ok {
		return store.AppendLedgerEventsConditionally(ctx, missionID, build)
	}
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return nil, err
	}
	toAppend, err := build(events)
	if err != nil {
		return nil, err
	}
	appended := make([]LedgerEvent, 0, len(toAppend))
	for _, event := range toAppend {
		committed, err := s.store.AppendLedgerEvent(ctx, event)
		if err != nil {
			return nil, err
		}
		appended = append(appended, committed)
	}
	return appended, nil
}

func validateNoActiveAgentWork(events []LedgerEvent) error {
	if workflowHasOpenAgentPending(events) {
		return fmt.Errorf("%w: agent turn is already running for this mission", ErrInvalidInput)
	}
	if workflowHasOpenReportDraftPending(events) {
		return fmt.Errorf("%w: report draft is already running for this mission", ErrInvalidInput)
	}
	for _, run := range projectWorkflowRuns(events) {
		if !workflowstate.TerminalStatus(run.Status) {
			return fmt.Errorf("%w: workflow %s is %s for this mission", ErrInvalidInput, run.WorkflowRunID, run.Status)
		}
	}
	return nil
}

func workflowHasOpenReportDraftPending(events []LedgerEvent) bool {
	return ledgerstate.HasOpenReportPending(ledgerStateEventsFromApp(events))
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", ErrInvalidInput, prefix)
	}
	return nil
}
