package researchproposal

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

// ApprovalReaders supplies mission-bound event access. Reads occur lazily so the
// approval-producer fast path and malformed-event precedence remain unchanged.
type ApprovalReaders struct {
	RequireMissionEvent func(context.Context, string, string) (ledger.Event, error)
	ListMissionEvents   func(context.Context, string) ([]ledger.Event, error)
}

// RequireApprovedObject validates historical approval without persisting state.
func RequireApprovedObject(ctx context.Context, readers ApprovalReaders, missionID, createdEventID, objectID string) error {
	createdEvent, err := readers.RequireMissionEvent(ctx, missionID, createdEventID)
	if err != nil {
		return err
	}
	if isApprovalProducer(createdEvent.Producer) {
		return nil
	}
	proposalID, err := proposalIDFromCreatedEvent(createdEvent, objectID)
	if err != nil {
		return err
	}
	events, err := readers.ListMissionEvents(ctx, missionID)
	if err != nil {
		return err
	}
	for _, event := range events {
		if event.EventType != "proposal.approved" && event.EventType != "proposal.partially_approved" {
			continue
		}
		if !isApprovalProducer(event.Producer) {
			continue
		}
		var payload ProposalDecisionPayload
		if err := unmarshalEventPayload(event, &payload); err != nil {
			return err
		}
		if strings.TrimSpace(payload.ProposalID) == proposalID && containsString(trimStringList(payload.ApprovedObjectIDs), objectID) {
			return nil
		}
	}
	return fmt.Errorf("%w: proposal object is not approved", producterror.ErrInvalidInput)
}

func proposalIDFromCreatedEvent(event ledger.Event, objectID string) (string, error) {
	var payload struct {
		ProposalID string `json:"proposal_id"`
		EvidenceID string `json:"evidence_id"`
		ClaimID    string `json:"claim_id"`
		QuestionID string `json:"question_id"`
		OptionID   string `json:"option_id"`
	}
	if err := unmarshalEventPayload(event, &payload); err != nil {
		return "", err
	}
	if payload.EvidenceID != "" && strings.TrimSpace(payload.EvidenceID) != objectID {
		return "", fmt.Errorf("%w: event does not reference report object", producterror.ErrInvalidInput)
	}
	if payload.ClaimID != "" && strings.TrimSpace(payload.ClaimID) != objectID {
		return "", fmt.Errorf("%w: event does not reference report object", producterror.ErrInvalidInput)
	}
	if payload.QuestionID != "" && strings.TrimSpace(payload.QuestionID) != objectID {
		return "", fmt.Errorf("%w: event does not reference report object", producterror.ErrInvalidInput)
	}
	if payload.OptionID != "" && strings.TrimSpace(payload.OptionID) != objectID {
		return "", fmt.Errorf("%w: event does not reference report object", producterror.ErrInvalidInput)
	}
	proposalID := strings.TrimSpace(payload.ProposalID)
	if proposalID == "" {
		return "", fmt.Errorf("%w: created event does not reference a proposal", producterror.ErrInvalidInput)
	}
	return proposalID, nil
}
