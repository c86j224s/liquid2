package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/ledgerstate"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

type reportILBundleStore interface {
	CommitReportILBundleConditionally(context.Context, string, string, []RawArtifact, LedgerEvent, func([]LedgerEvent) (LedgerEvent, bool, error)) ([]RawArtifact, LedgerEvent, bool, error)
}

// ReportILBundleRequest contains all durable IL artifacts and the canonical
// Markdown completion event. StoreCompleted is committed with both atomically.
type ReportILBundleRequest struct {
	MissionID      string
	PendingID      string
	Artifacts      []CreateRawArtifactRequest
	StoreCompleted AppendEventRequest
	Terminal       AppendEventRequest
}

func (s *Service) CreateReportILBundleIfOpen(ctx context.Context, req ReportILBundleRequest) ([]RawArtifact, LedgerEvent, bool, error) {
	missionID := strings.TrimSpace(req.MissionID)
	pendingID := strings.TrimSpace(req.PendingID)
	if err := validateID("mis_", missionID); err != nil {
		return nil, LedgerEvent{}, false, err
	}
	if err := validateID("evt_", pendingID); err != nil {
		return nil, LedgerEvent{}, false, fmt.Errorf("%w: pending event id is invalid", ErrInvalidInput)
	}
	if len(req.Artifacts) < 6 || len(req.Artifacts) > 10 {
		return nil, LedgerEvent{}, false, fmt.Errorf("%w: report IL bundle artifact count is outside the closed current, legacy, and optional-image range", ErrInvalidInput)
	}
	if req.Terminal.EventType != "report.artifact.created" || req.StoreCompleted.EventType != "report.il_store.completed" {
		return nil, LedgerEvent{}, false, fmt.Errorf("%w: report IL bundle terminal events are invalid", ErrInvalidInput)
	}
	if err := validateReportILEventBindings(req.StoreCompleted, missionID, pendingID); err != nil {
		return nil, LedgerEvent{}, false, err
	}
	if err := validateReportILEventBindings(req.Terminal, missionID, pendingID); err != nil {
		return nil, LedgerEvent{}, false, err
	}
	if strings.TrimSpace(req.StoreCompleted.EventID) == strings.TrimSpace(req.Terminal.EventID) ||
		strings.TrimSpace(req.StoreCompleted.EventID) == pendingID || strings.TrimSpace(req.Terminal.EventID) == pendingID {
		return nil, LedgerEvent{}, false, fmt.Errorf("%w: report IL event ids must be distinct", ErrInvalidInput)
	}
	if err := validateReportILStoreCompletedPayload(req.StoreCompleted.Payload, pendingID); err != nil {
		return nil, LedgerEvent{}, false, err
	}
	store, ok := s.store.(reportILBundleStore)
	if !ok {
		return nil, LedgerEvent{}, false, fmt.Errorf("%w: report IL bundle store is required", ErrInvalidInput)
	}
	artifacts := make([]RawArtifact, 0, len(req.Artifacts))
	seenArtifactIDs := make(map[string]struct{}, len(req.Artifacts))
	for _, artifactReq := range req.Artifacts {
		artifact, err := buildRawArtifact(artifactReq)
		if err != nil {
			return nil, LedgerEvent{}, false, err
		}
		if artifact.MissionID != missionID {
			return nil, LedgerEvent{}, false, fmt.Errorf("%w: report IL artifact mission mismatch", ErrInvalidInput)
		}
		if _, exists := seenArtifactIDs[artifact.ArtifactID]; exists {
			return nil, LedgerEvent{}, false, fmt.Errorf("%w: duplicate report IL artifact id", ErrInvalidInput)
		}
		seenArtifactIDs[artifact.ArtifactID] = struct{}{}
		artifacts = append(artifacts, artifact)
	}
	if err := validateReportILTerminalLineage(req.Terminal, pendingID, artifacts); err != nil {
		return nil, LedgerEvent{}, false, err
	}
	storeCompleted, err := buildLedgerEvent(req.StoreCompleted)
	if err != nil {
		return nil, LedgerEvent{}, false, err
	}
	return store.CommitReportILBundleConditionally(ctx, missionID, pendingID, artifacts, storeCompleted, func(events []LedgerEvent) (LedgerEvent, bool, error) {
		pending, found := reportILPendingEvent(events, missionID, pendingID)
		if !found {
			return LedgerEvent{}, false, fmt.Errorf("%w: report IL experimental pending event is not open", ErrInvalidInput)
		}
		if err := validateReportILPendingFamily(pending); err != nil {
			return LedgerEvent{}, false, err
		}
		if reportPendingWasClosed(events, pendingID) {
			return LedgerEvent{}, false, nil
		}
		terminal, err := buildLedgerEvent(req.Terminal)
		if err != nil {
			return LedgerEvent{}, false, err
		}
		if terminal.EventID == pendingID || terminal.EventID == storeCompleted.EventID {
			return LedgerEvent{}, false, fmt.Errorf("%w: report IL event ids must be distinct", ErrInvalidInput)
		}
		valid, err := validateReportTerminalAppend(pending, terminal)
		if err != nil {
			return LedgerEvent{}, false, err
		}
		if !valid {
			return LedgerEvent{}, false, fmt.Errorf("%w: report IL terminal event is invalid", ErrInvalidInput)
		}
		if err := ValidateAgentExecutorAppend(events, []LedgerEvent{terminal}); err != nil {
			return LedgerEvent{}, false, fmt.Errorf("report IL terminal executor validation failed: %w", err)
		}
		return terminal, true, nil
	})
}

func validateReportILEventBindings(req AppendEventRequest, missionID, pendingID string) error {
	if err := validateID("evt_", strings.TrimSpace(req.EventID)); err != nil {
		return fmt.Errorf("%w: report IL event id is invalid", ErrInvalidInput)
	}
	if strings.TrimSpace(req.MissionID) != missionID || strings.TrimSpace(req.CausationEventID) != pendingID || strings.TrimSpace(req.CorrelationID) != pendingID {
		return fmt.Errorf("%w: report IL event lineage binding is invalid", ErrInvalidInput)
	}
	return nil
}

func validateReportILStoreCompletedPayload(raw []byte, pendingID string) error {
	var payload struct {
		Kind           string `json:"kind"`
		PendingEventID string `json:"pending_event_id"`
		PipelineFamily string `json:"pipeline_family"`
		Stage          string `json:"stage"`
		Status         string `json:"status"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return fmt.Errorf("%w: invalid report IL store receipt", ErrInvalidInput)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("%w: invalid report IL store receipt", ErrInvalidInput)
	}
	if payload.Kind != "report_il_stage_progress" || payload.PendingEventID != pendingID || payload.PipelineFamily != reportilcontract.PipelineFamily || payload.Stage != "il_store" || payload.Status != "completed" {
		return fmt.Errorf("%w: invalid report IL store receipt", ErrInvalidInput)
	}
	return nil
}

func validateReportILTerminalLineage(event AppendEventRequest, pendingID string, artifacts []RawArtifact) error {
	payload, err := reportilcontract.DecodeTerminalPayload(event.Payload)
	if err != nil {
		return fmt.Errorf("%w: invalid report IL terminal lineage: %v", ErrInvalidInput, err)
	}
	if payload.PendingEventID != pendingID {
		return fmt.Errorf("%w: report IL terminal pending binding is invalid", ErrInvalidInput)
	}
	actual := make([]reportilcontract.Artifact, 0, len(artifacts))
	for _, item := range artifacts {
		actual = append(actual, reportilcontract.Artifact{ArtifactID: item.ArtifactID, MissionID: item.MissionID, MediaType: item.MediaType, Filename: item.Filename, ByteSize: item.ByteSize, SHA256: item.SHA256, Content: item.Content})
	}
	if err := reportilcontract.ValidateActual(payload.Bundle, actual); err != nil {
		return fmt.Errorf("%w: invalid report IL terminal lineage: %v", ErrInvalidInput, err)
	}
	return nil
}

func reportILPendingEvent(events []LedgerEvent, missionID, pendingID string) (LedgerEvent, bool) {
	for _, event := range events {
		if event.EventID == pendingID && event.MissionID == missionID && event.EventType == "report.draft.pending" {
			return event, true
		}
	}
	return LedgerEvent{}, false
}

func validateReportILPendingFamily(event LedgerEvent) error {
	var payload struct {
		PipelineFamily string `json:"pipeline_family"`
	}
	decoder := json.NewDecoder(bytes.NewReader(event.Payload))
	if err := decoder.Decode(&payload); err != nil || payload.PipelineFamily != reportilcontract.PipelineFamily {
		return fmt.Errorf("%w: report IL pending event is not experimental", ErrInvalidInput)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("%w: report IL pending event is malformed", ErrInvalidInput)
	}
	return nil
}

func reportPendingWasClosed(events []LedgerEvent, pendingID string) bool {
	completed := ledgerstate.CompletedReportPendingEventIDs(ledgerStateEventsFromApp(events))
	_, closed := completed[pendingID]
	return closed
}

var _ ledger.Event = ledger.Event{}
