package app

import (
	"context"
	"fmt"
	"strings"
)

type conditionalRawArtifactStore interface {
	CommitRawArtifactWithEventConditionally(
		context.Context,
		RawArtifact,
		func([]LedgerEvent) (LedgerEvent, bool, error),
	) (RawArtifact, LedgerEvent, bool, error)
}

type conditionalDesignedReportHTMLExportStore interface {
	CommitDesignedReportHTMLExportConditionally(
		context.Context,
		string,
		RawArtifact,
		RawArtifact,
		func([]LedgerEvent) ([]LedgerEvent, bool, error),
	) (RawArtifact, RawArtifact, LedgerEvent, bool, error)
}

// CreateRawArtifactWithEventConditionally는 조건이 맞을 때 raw artifact와 이벤트를 함께 기록한다.
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

// CreateDesignedReportHTMLExportIfOpen stores the content model artifact, HTML
// artifact, and report.artifact.exported terminal event under the same pending
// open check. It is intentionally narrow to the designed HTML two-artifact
// completion path.
func (s *Service) CreateDesignedReportHTMLExportIfOpen(
	ctx context.Context,
	missionID string,
	pendingEventID string,
	contentModelReq CreateRawArtifactRequest,
	htmlReq CreateRawArtifactRequest,
	eventReqForArtifacts func(RawArtifact, RawArtifact) AppendEventRequest,
) (RawArtifact, RawArtifact, LedgerEvent, bool, error) {
	if eventReqForArtifacts == nil {
		return RawArtifact{}, RawArtifact{}, LedgerEvent{}, false, fmt.Errorf("%w: designed HTML terminal event builder is required", ErrInvalidInput)
	}
	if err := validateID("mis_", missionID); err != nil {
		return RawArtifact{}, RawArtifact{}, LedgerEvent{}, false, err
	}
	pendingEventID = strings.TrimSpace(pendingEventID)
	if pendingEventID == "" {
		return RawArtifact{}, RawArtifact{}, LedgerEvent{}, false, fmt.Errorf("%w: pending event is required", ErrInvalidInput)
	}
	store, ok := s.store.(conditionalDesignedReportHTMLExportStore)
	if !ok {
		return RawArtifact{}, RawArtifact{}, LedgerEvent{}, false, fmt.Errorf("%w: designed HTML conditional store is required", ErrInvalidInput)
	}
	contentModel, err := buildRawArtifact(contentModelReq)
	if err != nil {
		return RawArtifact{}, RawArtifact{}, LedgerEvent{}, false, err
	}
	html, err := buildRawArtifact(htmlReq)
	if err != nil {
		return RawArtifact{}, RawArtifact{}, LedgerEvent{}, false, err
	}
	if contentModel.MissionID != missionID || html.MissionID != missionID {
		return RawArtifact{}, RawArtifact{}, LedgerEvent{}, false, fmt.Errorf("%w: artifact mission_id must match %s", ErrInvalidInput, missionID)
	}
	return store.CommitDesignedReportHTMLExportConditionally(ctx, missionID, contentModel, html, func(events []LedgerEvent) ([]LedgerEvent, bool, error) {
		terminalReq := eventReqForArtifacts(contentModel, html)
		return buildReportTerminalEventsIfOpen(events, missionID, pendingEventID, []AppendEventRequest{terminalReq})
	})
}
