package app

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"strings"
)

// AppendEventsConditionally builds and appends a batch against one latest ledger snapshot.
// The conditional store is required so callers never fall back to sequential writes.
func (s *Service) AppendEventsConditionally(
	ctx context.Context,
	missionID string,
	build func([]ledger.Event) ([]ledger.AppendRequest, error),
) ([]ledger.Event, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return nil, err
	}
	if build == nil {
		return nil, fmt.Errorf("%w: conditional event batch builder is required", ErrInvalidInput)
	}
	store, ok := s.store.(ConditionalLedgerStore)
	if !ok {
		return nil, fmt.Errorf("%w: conditional ledger store is required", ErrInvalidInput)
	}
	return store.AppendLedgerEventsConditionally(ctx, missionID, func(events []ledger.Event) ([]ledger.Event, error) {
		reqs, err := build(events)
		if err != nil {
			return nil, err
		}
		built := make([]ledger.Event, 0, len(reqs))
		for _, req := range reqs {
			if strings.TrimSpace(req.MissionID) != missionID {
				return nil, fmt.Errorf("%w: conditional event mission differs", ErrInvalidInput)
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

// AppendEventConditionally는 이벤트 하나를 조건부로 append하거나, 호출자의 replay
// 검사에서 선택한 canonical 이벤트를 반환하는 원자적 경계다.
func (s *Service) AppendEventConditionally(
	ctx context.Context,
	missionID string,
	build func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error),
) (ledger.Event, bool, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return ledger.Event{}, false, err
	}
	if build == nil {
		return ledger.Event{}, false, fmt.Errorf("%w: conditional event builder is required", ErrInvalidInput)
	}
	store, ok := s.store.(ConditionalLedgerStore)
	if !ok {
		return ledger.Event{}, false, fmt.Errorf("%w: conditional ledger store is required", ErrInvalidInput)
	}
	var replay ledger.Event
	appended, err := store.AppendLedgerEventsConditionally(ctx, missionID, func(events []ledger.Event) ([]ledger.Event, error) {
		req, existing, create, err := build(events)
		if err != nil {
			return nil, err
		}
		if !create {
			replay = existing
			return nil, nil
		}
		if strings.TrimSpace(req.MissionID) != missionID {
			return nil, fmt.Errorf("%w: conditional event mission differs", ErrInvalidInput)
		}
		event, err := buildLedgerEvent(req)
		if err != nil {
			return nil, err
		}
		return []ledger.Event{event}, nil
	})
	if err != nil {
		return ledger.Event{}, false, err
	}
	if replay.EventID != "" {
		return replay, false, nil
	}
	if len(appended) != 1 {
		return ledger.Event{}, false, fmt.Errorf("%w: conditional event was not appended", ErrConflict)
	}
	return appended[0], true, nil
}
