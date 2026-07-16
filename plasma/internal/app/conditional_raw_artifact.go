package app

import (
	"context"
	"fmt"
)

type conditionalRawArtifactStore interface {
	CommitRawArtifactWithEventConditionally(
		context.Context,
		RawArtifact,
		func([]LedgerEvent) (LedgerEvent, bool, error),
	) (RawArtifact, LedgerEvent, bool, error)
}

func (s *Service) CreateRawArtifactWithEventConditionally(
	ctx context.Context,
	artifactReq CreateRawArtifactRequest,
	eventReqForEvents func([]LedgerEvent, RawArtifact) (AppendEventRequest, LedgerEvent, bool, error),
) (RawArtifact, LedgerEvent, bool, error) {
	if eventReqForEvents == nil {
		return RawArtifact{}, LedgerEvent{}, false, fmt.Errorf("%w: conditional event builder is required", ErrInvalidInput)
	}
	store, ok := s.store.(conditionalRawArtifactStore)
	if !ok {
		return RawArtifact{}, LedgerEvent{}, false, fmt.Errorf("%w: conditional raw artifact store is required", ErrInvalidInput)
	}
	artifact, err := buildRawArtifact(artifactReq)
	if err != nil {
		return RawArtifact{}, LedgerEvent{}, false, err
	}
	return store.CommitRawArtifactWithEventConditionally(ctx, artifact, func(events []LedgerEvent) (LedgerEvent, bool, error) {
		req, existing, create, err := eventReqForEvents(events, artifact)
		if err != nil {
			return LedgerEvent{}, false, err
		}
		if !create {
			return existing, false, nil
		}
		event, err := buildLedgerEvent(req)
		return event, err == nil, err
	})
}
