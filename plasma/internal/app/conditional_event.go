package app

import (
	"context"
	"fmt"
	"strings"
)

// AppendEventConditionally atomically appends one event or returns the
// canonical event selected by the caller's replay check.
func (s *Service) AppendEventConditionally(
	ctx context.Context,
	missionID string,
	build func([]LedgerEvent) (AppendEventRequest, LedgerEvent, bool, error),
) (LedgerEvent, bool, error) {
	missionID = strings.TrimSpace(missionID)
	if err := validateID("mis_", missionID); err != nil {
		return LedgerEvent{}, false, err
	}
	if build == nil {
		return LedgerEvent{}, false, fmt.Errorf("%w: conditional event builder is required", ErrInvalidInput)
	}
	store, ok := s.store.(ConditionalLedgerStore)
	if !ok {
		return LedgerEvent{}, false, fmt.Errorf("%w: conditional ledger store is required", ErrInvalidInput)
	}
	var replay LedgerEvent
	appended, err := store.AppendLedgerEventsConditionally(ctx, missionID, func(events []LedgerEvent) ([]LedgerEvent, error) {
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
		return []LedgerEvent{event}, nil
	})
	if err != nil {
		return LedgerEvent{}, false, err
	}
	if replay.EventID != "" {
		return replay, false, nil
	}
	if len(appended) != 1 {
		return LedgerEvent{}, false, fmt.Errorf("%w: conditional event was not appended", ErrConflict)
	}
	return appended[0], true, nil
}
