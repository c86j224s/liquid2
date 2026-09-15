package app

import (
	"context"
	"fmt"
	"strings"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

type conditionalRawArtifactStore interface {
	CommitRawArtifactWithEventConditionally(
		context.Context,
		artifactcontract.Raw,
		func([]ledger.Event) (ledger.Event, bool, error),
	) (artifactcontract.Raw, ledger.Event, bool, error)
}

type conditionalDesignedReportHTMLExportStore interface {
	CommitDesignedReportHTMLExportConditionally(
		context.Context,
		string,
		artifactcontract.Raw,
		artifactcontract.Raw,
		func([]ledger.Event) ([]ledger.Event, bool, error),
	) (artifactcontract.Raw, artifactcontract.Raw, ledger.Event, bool, error)
}

// CreateRawArtifactWithEventConditionally는 조건이 맞을 때 raw artifact와 이벤트를 함께 기록한다.
func (s *Service) CreateRawArtifactWithEventConditionally(
	ctx context.Context,
	artifactReq artifactcontract.CreateRequest,
	eventReqForEvents func([]ledger.Event, artifactcontract.Raw) (ledger.AppendRequest, ledger.Event, bool, error),
) (artifactcontract.Raw, ledger.Event, bool, error) {
	if eventReqForEvents == nil {
		return artifactcontract.Raw{}, ledger.Event{}, false, fmt.Errorf("%w: conditional event builder is required", ErrInvalidInput)
	}
	store, ok := s.store.(conditionalRawArtifactStore)
	if !ok {
		return artifactcontract.Raw{}, ledger.Event{}, false, fmt.Errorf("%w: conditional raw artifact store is required", ErrInvalidInput)
	}
	artifact, err := artifactcontract.Build(artifactReq)
	if err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	return store.CommitRawArtifactWithEventConditionally(ctx, artifact, func(events []ledger.Event) (ledger.Event, bool, error) {
		req, existing, create, err := eventReqForEvents(events, artifact)
		if err != nil {
			return ledger.Event{}, false, err
		}
		if !create {
			return existing, false, nil
		}
		event, err := buildLedgerEvent(req)
		return event, err == nil, err
	})
}

// CreateMarkdownReportArtifactIfOpen stores one Markdown artifact and closes an
// open report pending event in the same transaction.
func (s *Service) CreateMarkdownReportArtifactIfOpen(
	ctx context.Context,
	missionID string,
	pendingEventID string,
	artifactReq artifactcontract.CreateRequest,
	eventReqForArtifact func(artifactcontract.Raw) ledger.AppendRequest,
) (artifactcontract.Raw, ledger.Event, bool, error) {
	if eventReqForArtifact == nil {
		return artifactcontract.Raw{}, ledger.Event{}, false, fmt.Errorf("%w: Markdown report terminal event builder is required", ErrInvalidInput)
	}
	if err := validateID("mis_", missionID); err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	pendingEventID = strings.TrimSpace(pendingEventID)
	if pendingEventID == "" {
		return artifactcontract.Raw{}, ledger.Event{}, false, fmt.Errorf("%w: pending event is required", ErrInvalidInput)
	}
	store, ok := s.store.(conditionalRawArtifactStore)
	if !ok {
		return artifactcontract.Raw{}, ledger.Event{}, false, fmt.Errorf("%w: conditional raw artifact store is required", ErrInvalidInput)
	}
	artifact, err := artifactcontract.Build(artifactReq)
	if err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	if artifact.MissionID != missionID {
		return artifactcontract.Raw{}, ledger.Event{}, false, fmt.Errorf("%w: artifact mission_id must match %s", ErrInvalidInput, missionID)
	}
	return store.CommitRawArtifactWithEventConditionally(ctx, artifact, func(events []ledger.Event) (ledger.Event, bool, error) {
		built, open, err := buildReportTerminalEventsIfOpen(events, missionID, pendingEventID, []ledger.AppendRequest{eventReqForArtifact(artifact)})
		if err != nil || !open {
			return ledger.Event{}, false, err
		}
		if len(built) != 1 {
			return ledger.Event{}, false, fmt.Errorf("%w: Markdown report closure requires one terminal event", ErrInvalidInput)
		}
		return built[0], true, nil
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
	contentModelReq artifactcontract.CreateRequest,
	htmlReq artifactcontract.CreateRequest,
	eventReqForArtifacts func(artifactcontract.Raw, artifactcontract.Raw) ledger.AppendRequest,
) (artifactcontract.Raw, artifactcontract.Raw, ledger.Event, bool, error) {
	if eventReqForArtifacts == nil {
		return artifactcontract.Raw{}, artifactcontract.Raw{}, ledger.Event{}, false, fmt.Errorf("%w: designed HTML terminal event builder is required", ErrInvalidInput)
	}
	if err := validateID("mis_", missionID); err != nil {
		return artifactcontract.Raw{}, artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	pendingEventID = strings.TrimSpace(pendingEventID)
	if pendingEventID == "" {
		return artifactcontract.Raw{}, artifactcontract.Raw{}, ledger.Event{}, false, fmt.Errorf("%w: pending event is required", ErrInvalidInput)
	}
	store, ok := s.store.(conditionalDesignedReportHTMLExportStore)
	if !ok {
		return artifactcontract.Raw{}, artifactcontract.Raw{}, ledger.Event{}, false, fmt.Errorf("%w: designed HTML conditional store is required", ErrInvalidInput)
	}
	contentModel, err := artifactcontract.Build(contentModelReq)
	if err != nil {
		return artifactcontract.Raw{}, artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	html, err := artifactcontract.Build(htmlReq)
	if err != nil {
		return artifactcontract.Raw{}, artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	if contentModel.MissionID != missionID || html.MissionID != missionID {
		return artifactcontract.Raw{}, artifactcontract.Raw{}, ledger.Event{}, false, fmt.Errorf("%w: artifact mission_id must match %s", ErrInvalidInput, missionID)
	}
	return store.CommitDesignedReportHTMLExportConditionally(ctx, missionID, contentModel, html, func(events []ledger.Event) ([]ledger.Event, bool, error) {
		terminalReq := eventReqForArtifacts(contentModel, html)
		return buildReportTerminalEventsIfOpen(events, missionID, pendingEventID, []ledger.AppendRequest{terminalReq})
	})
}
