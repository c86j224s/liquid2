package app

import "github.com/c86j224s/liquid2/plasma/internal/reportexecution"

import (
	"context"
	"fmt"
	"strings"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
)

type reportILBundleStore interface {
	CommitReportILBundleConditionally(context.Context, string, string, []artifactcontract.Raw, ledger.Event, func([]ledger.Event) (ledger.Event, bool, error)) ([]artifactcontract.Raw, ledger.Event, bool, error)
}

// ReportILBundleRequest contains all durable IL artifacts and the canonical
// Markdown completion event. StoreCompleted is committed with both atomically.
type ReportILBundleRequest struct {
	MissionID      string
	PendingID      string
	Artifacts      []artifactcontract.CreateRequest
	StoreCompleted ledger.AppendRequest
	Terminal       ledger.AppendRequest
}

func (s *Service) CreateReportILBundleIfOpen(ctx context.Context, req ReportILBundleRequest) ([]artifactcontract.Raw, ledger.Event, bool, error) {
	missionID := strings.TrimSpace(req.MissionID)
	pendingID := strings.TrimSpace(req.PendingID)
	if err := validateID("mis_", missionID); err != nil {
		return nil, ledger.Event{}, false, err
	}
	if err := validateID("evt_", pendingID); err != nil {
		return nil, ledger.Event{}, false, fmt.Errorf("%w: pending event id is invalid", ErrInvalidInput)
	}
	if len(req.Artifacts) < 6 || len(req.Artifacts) > 10 {
		return nil, ledger.Event{}, false, fmt.Errorf("%w: report IL bundle artifact count is outside the closed current, legacy, and optional-image range", ErrInvalidInput)
	}
	if req.Terminal.EventType != "report.artifact.created" || req.StoreCompleted.EventType != "report.il_store.completed" {
		return nil, ledger.Event{}, false, fmt.Errorf("%w: report IL bundle terminal events are invalid", ErrInvalidInput)
	}
	if err := reportexecution.ValidateReportILEventBindings(req.StoreCompleted, missionID, pendingID); err != nil {
		return nil, ledger.Event{}, false, err
	}
	if err := reportexecution.ValidateReportILEventBindings(req.Terminal, missionID, pendingID); err != nil {
		return nil, ledger.Event{}, false, err
	}
	if strings.TrimSpace(req.StoreCompleted.EventID) == strings.TrimSpace(req.Terminal.EventID) ||
		strings.TrimSpace(req.StoreCompleted.EventID) == pendingID || strings.TrimSpace(req.Terminal.EventID) == pendingID {
		return nil, ledger.Event{}, false, fmt.Errorf("%w: report IL event ids must be distinct", ErrInvalidInput)
	}
	if err := reportexecution.ValidateReportILStoreCompletedPayload(req.StoreCompleted.Payload, pendingID); err != nil {
		return nil, ledger.Event{}, false, err
	}
	store, ok := s.store.(reportILBundleStore)
	if !ok {
		return nil, ledger.Event{}, false, fmt.Errorf("%w: report IL bundle store is required", ErrInvalidInput)
	}
	artifacts := make([]artifactcontract.Raw, 0, len(req.Artifacts))
	seenArtifactIDs := make(map[string]struct{}, len(req.Artifacts))
	for _, artifactReq := range req.Artifacts {
		artifact, err := artifactcontract.Build(artifactReq)
		if err != nil {
			return nil, ledger.Event{}, false, err
		}
		if artifact.MissionID != missionID {
			return nil, ledger.Event{}, false, fmt.Errorf("%w: report IL artifact mission mismatch", ErrInvalidInput)
		}
		if _, exists := seenArtifactIDs[artifact.ArtifactID]; exists {
			return nil, ledger.Event{}, false, fmt.Errorf("%w: duplicate report IL artifact id", ErrInvalidInput)
		}
		seenArtifactIDs[artifact.ArtifactID] = struct{}{}
		artifacts = append(artifacts, artifact)
	}
	if err := reportexecution.ValidateReportILTerminalLineage(req.Terminal, pendingID, artifacts); err != nil {
		return nil, ledger.Event{}, false, err
	}
	storeCompleted, err := buildLedgerEvent(req.StoreCompleted)
	if err != nil {
		return nil, ledger.Event{}, false, err
	}
	return store.CommitReportILBundleConditionally(ctx, missionID, pendingID, artifacts, storeCompleted, func(events []ledger.Event) (ledger.Event, bool, error) {
		pending, found := reportexecution.ReportILPendingEvent(events, missionID, pendingID)
		if !found {
			return ledger.Event{}, false, fmt.Errorf("%w: report IL experimental pending event is not open", ErrInvalidInput)
		}
		if err := reportexecution.ValidateReportILPendingFamily(pending); err != nil {
			return ledger.Event{}, false, err
		}
		if reportPendingWasClosed(events, pendingID) {
			return ledger.Event{}, false, nil
		}
		terminal, err := buildLedgerEvent(req.Terminal)
		if err != nil {
			return ledger.Event{}, false, err
		}
		if terminal.EventID == pendingID || terminal.EventID == storeCompleted.EventID {
			return ledger.Event{}, false, fmt.Errorf("%w: report IL event ids must be distinct", ErrInvalidInput)
		}
		valid, err := reportexecution.ValidateReportTerminalAppend(pending, terminal)
		if err != nil {
			return ledger.Event{}, false, err
		}
		if !valid {
			return ledger.Event{}, false, fmt.Errorf("%w: report IL terminal event is invalid", ErrInvalidInput)
		}
		if err := ValidateAgentExecutorAppend(events, []ledger.Event{terminal}); err != nil {
			return ledger.Event{}, false, fmt.Errorf("report IL terminal executor validation failed: %w", err)
		}
		return terminal, true, nil
	})
}

func reportPendingWasClosed(events []ledger.Event, pendingID string) bool {
	completed := ledgerstate.CompletedReportPendingEventIDs(ledgerStateEventsFromApp(events))
	_, closed := completed[pendingID]
	return closed
}

var _ ledger.Event = ledger.Event{}
