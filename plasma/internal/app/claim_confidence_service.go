package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (s *Service) UpdateClaimConfidence(ctx context.Context, req UpdateClaimConfidenceRequest) (LedgerEvent, error) {
	eventID := strings.TrimSpace(req.EventID)
	missionID := strings.TrimSpace(req.MissionID)
	claimID := strings.TrimSpace(req.ClaimID)
	if err := validateID("evt_", eventID); err != nil {
		return LedgerEvent{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return LedgerEvent{}, err
	}
	if err := validateID("clm_", claimID); err != nil {
		return LedgerEvent{}, err
	}
	if err := validateProducer(req.Producer); err != nil {
		return LedgerEvent{}, err
	}

	claim, err := s.store.GetClaimRecord(ctx, claimID)
	if err != nil {
		return LedgerEvent{}, err
	}
	if claim.MissionID != missionID {
		return LedgerEvent{}, fmt.Errorf("%w: confidence claim belongs to another mission", ErrInvalidInput)
	}
	confidence, err := normalizeConfidence(req.Confidence)
	if err != nil {
		return LedgerEvent{}, err
	}
	if strings.TrimSpace(confidence.Rationale) == "" {
		return LedgerEvent{}, fmt.Errorf("%w: confidence rationale is required", ErrInvalidInput)
	}
	basisEvidenceIDs, err := normalizeIDList("evd_", req.BasisEvidenceIDs)
	if err != nil {
		return LedgerEvent{}, err
	}
	if err := s.requireEvidenceRecords(ctx, missionID, basisEvidenceIDs); err != nil {
		return LedgerEvent{}, err
	}
	producer := normalizeProducer(req.Producer)
	origin, err := normalizeClaimConfidenceOrigin(req.Origin, producer)
	if err != nil {
		return LedgerEvent{}, err
	}
	payload, err := json.Marshal(ClaimConfidenceUpdatePayload{
		ClaimID:          claimID,
		Confidence:       confidence,
		BasisEvidenceIDs: basisEvidenceIDs,
		Origin:           origin,
	})
	if err != nil {
		return LedgerEvent{}, err
	}
	return s.AppendEvent(ctx, AppendEventRequest{
		EventID:          eventID,
		MissionID:        missionID,
		EventType:        ClaimConfidenceUpdatedEvent,
		Producer:         producer,
		CausationEventID: strings.TrimSpace(req.CausationEventID),
		CorrelationID:    strings.TrimSpace(req.CorrelationID),
		Payload:          payload,
	})
}

func ClaimConfidenceUpdatesFromEvents(events []LedgerEvent) []ClaimConfidenceUpdate {
	updates := make([]ClaimConfidenceUpdate, 0)
	for _, event := range events {
		update, ok := ClaimConfidenceUpdateFromEvent(event)
		if ok {
			updates = append(updates, update)
		}
	}
	return updates
}

func ClaimConfidenceUpdateFromEvent(event LedgerEvent) (ClaimConfidenceUpdate, bool) {
	if event.EventType != ClaimConfidenceUpdatedEvent {
		return ClaimConfidenceUpdate{}, false
	}
	var payload ClaimConfidenceUpdatePayload
	raw := event.Payload
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ClaimConfidenceUpdate{}, false
	}
	confidence, err := normalizeConfidence(payload.Confidence)
	if err != nil {
		return ClaimConfidenceUpdate{}, false
	}
	claimID := strings.TrimSpace(payload.ClaimID)
	if err := validateID("clm_", claimID); err != nil {
		return ClaimConfidenceUpdate{}, false
	}
	basisEvidenceIDs, err := normalizeIDList("evd_", payload.BasisEvidenceIDs)
	if err != nil {
		return ClaimConfidenceUpdate{}, false
	}
	origin, err := normalizeClaimConfidenceOrigin(payload.Origin, event.Producer)
	if err != nil {
		return ClaimConfidenceUpdate{}, false
	}
	return ClaimConfidenceUpdate{
		EventID:          event.EventID,
		MissionID:        event.MissionID,
		Sequence:         event.Sequence,
		ClaimID:          claimID,
		Confidence:       confidence,
		BasisEvidenceIDs: basisEvidenceIDs,
		Origin:           origin,
		Producer:         event.Producer,
		CreatedAt:        event.CreatedAt,
	}, true
}

func normalizeClaimConfidenceOrigin(origin string, producer Producer) (string, error) {
	trimmed := strings.TrimSpace(origin)
	if trimmed == "" {
		switch strings.TrimSpace(producer.Type) {
		case "user", "steering_chat":
			trimmed = "user"
		case "agent", "agent_session":
			trimmed = "agent"
		case "autopilot":
			trimmed = "autopilot"
		default:
			trimmed = "system"
		}
	}
	switch trimmed {
	case "user", "agent", "autopilot", "system":
		return trimmed, nil
	default:
		return "", fmt.Errorf("%w: unsupported confidence update origin", ErrInvalidInput)
	}
}
