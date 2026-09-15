package researchrecords

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const ClaimConfidenceUpdatedEvent = "claim.confidence.updated"

// ClaimConfidenceUpdatePayload는 claim confidence 변경 event payload다.
type ClaimConfidenceUpdatePayload struct {
	ClaimID          string     `json:"claim_id"`
	Confidence       Confidence `json:"confidence"`
	BasisEvidenceIDs []string   `json:"basis_evidence_ids,omitempty"`
	Origin           string     `json:"origin"`
}

// ClaimConfidenceUpdate는 장부 event에서 읽은 claim confidence 변경 projection이다.
type ClaimConfidenceUpdate struct {
	EventID          string          `json:"event_id"`
	MissionID        string          `json:"mission_id"`
	Sequence         int64           `json:"sequence"`
	ClaimID          string          `json:"claim_id"`
	Confidence       Confidence      `json:"confidence"`
	BasisEvidenceIDs []string        `json:"basis_evidence_ids,omitempty"`
	Origin           string          `json:"origin"`
	Producer         ledger.Producer `json:"producer"`
	CreatedAt        time.Time       `json:"created_at"`
}

// UpdateClaimConfidenceRequest는 claim confidence 변경 요청이다.
type UpdateClaimConfidenceRequest struct {
	EventID          string
	MissionID        string
	ClaimID          string
	Confidence       Confidence
	BasisEvidenceIDs []string
	Origin           string
	Producer         ledger.Producer
	CausationEventID string
	CorrelationID    string
}

// ConfidenceRequirements supplies application-owned record and evidence lookups.
// The evidence callback is required when its lookup phase is reached, including for
// an empty evidence ID list.
type ConfidenceRequirements struct {
	GetClaimRecord         func(context.Context, string) (ClaimRecord, error)
	RequireEvidenceRecords func(context.Context, string, []string) error
}

// BuildClaimConfidenceUpdate validates and normalizes a confidence event request in
// the historical validation order, without appending it to the ledger.
func BuildClaimConfidenceUpdate(ctx context.Context, requirements ConfidenceRequirements, req UpdateClaimConfidenceRequest) (ledger.AppendRequest, error) {
	eventID := strings.TrimSpace(req.EventID)
	missionID := strings.TrimSpace(req.MissionID)
	claimID := strings.TrimSpace(req.ClaimID)
	if err := validateID("evt_", eventID); err != nil {
		return ledger.AppendRequest{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return ledger.AppendRequest{}, err
	}
	if err := validateID("clm_", claimID); err != nil {
		return ledger.AppendRequest{}, err
	}
	if err := validateProducer(req.Producer); err != nil {
		return ledger.AppendRequest{}, err
	}

	claim, err := requirements.GetClaimRecord(ctx, claimID)
	if err != nil {
		return ledger.AppendRequest{}, err
	}
	if claim.MissionID != missionID {
		return ledger.AppendRequest{}, fmt.Errorf("%w: confidence claim belongs to another mission", producterror.ErrInvalidInput)
	}
	confidence, err := NormalizeConfidence(req.Confidence)
	if err != nil {
		return ledger.AppendRequest{}, err
	}
	if strings.TrimSpace(confidence.Rationale) == "" {
		return ledger.AppendRequest{}, fmt.Errorf("%w: confidence rationale is required", producterror.ErrInvalidInput)
	}
	basisEvidenceIDs, err := normalizeIDList("evd_", req.BasisEvidenceIDs)
	if err != nil {
		return ledger.AppendRequest{}, err
	}
	if err := requirements.RequireEvidenceRecords(ctx, missionID, basisEvidenceIDs); err != nil {
		return ledger.AppendRequest{}, err
	}
	producer := normalizeProducer(req.Producer)
	origin, err := normalizeClaimConfidenceOrigin(req.Origin, producer)
	if err != nil {
		return ledger.AppendRequest{}, err
	}
	payload, err := json.Marshal(ClaimConfidenceUpdatePayload{
		ClaimID:          claimID,
		Confidence:       confidence,
		BasisEvidenceIDs: basisEvidenceIDs,
		Origin:           origin,
	})
	if err != nil {
		return ledger.AppendRequest{}, err
	}
	return ledger.AppendRequest{
		EventID:          eventID,
		MissionID:        missionID,
		EventType:        ClaimConfidenceUpdatedEvent,
		Producer:         producer,
		CausationEventID: strings.TrimSpace(req.CausationEventID),
		CorrelationID:    strings.TrimSpace(req.CorrelationID),
		Payload:          payload,
	}, nil
}

// ClaimConfidenceUpdatesFromEvents는 장부 이벤트에서 claim confidence 변경 이력을 추출한다.
func ClaimConfidenceUpdatesFromEvents(events []ledger.Event) []ClaimConfidenceUpdate {
	updates := make([]ClaimConfidenceUpdate, 0)
	for _, event := range events {
		update, ok := ClaimConfidenceUpdateFromEvent(event)
		if ok {
			updates = append(updates, update)
		}
	}
	return updates
}

// ClaimConfidenceUpdateFromEvent는 단일 장부 이벤트를 claim confidence 변경으로 해석한다.
func ClaimConfidenceUpdateFromEvent(event ledger.Event) (ClaimConfidenceUpdate, bool) {
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
	confidence, err := NormalizeConfidence(payload.Confidence)
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

func normalizeClaimConfidenceOrigin(origin string, producer ledger.Producer) (string, error) {
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
		return "", fmt.Errorf("%w: unsupported confidence update origin", producterror.ErrInvalidInput)
	}
}
