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

var allowedClaimTypes = map[string]bool{
	"descriptive": true, "evaluative": true, "recommendation": true, "risk": true, "decision": true,
}

var allowedClaimLifecycleStates = map[string]bool{
	"draft": true, "proposed": true, "needs_review": true, "approved": true,
	"rejected": true, "superseded": true, "archived": true,
}

var allowedClaimProposedLifecycleStates = map[string]bool{
	"draft": true, "proposed": true, "needs_review": true,
}

var allowedClaimApprovalStates = map[string]bool{
	"not_required": true, "pending": true, "approved": true, "rejected": true,
}

// BuildClaimRecord validates and normalizes a claim in the historical validation order.
// Evidence and question callbacks are required once the builder reaches their lookup phases,
// including when the corresponding ID slices are empty; the mission-event callback is required
// on assertion and approval paths. Missing callbacks are programmer misuse and are not guarded.
func BuildClaimRecord(ctx context.Context, requirements ClaimRequirements, req CreateClaimRecordRequest, createdEvent ledger.Event) (ClaimRecord, error) {
	claimID := strings.TrimSpace(req.ClaimID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateClaimID("clm_", claimID); err != nil {
		return ClaimRecord{}, err
	}
	if err := validateClaimID("mis_", missionID); err != nil {
		return ClaimRecord{}, err
	}
	if strings.TrimSpace(req.Text) == "" {
		return ClaimRecord{}, fmt.Errorf("%w: claim text is required", producterror.ErrInvalidInput)
	}
	if createdEvent.MissionID != missionID || strings.TrimSpace(createdEvent.EventID) != strings.TrimSpace(req.CreatedEventID) {
		return ClaimRecord{}, fmt.Errorf("%w: claim creation event mismatch", producterror.ErrInvalidInput)
	}
	state, err := normalizeClaimLifecycleState(req.State, "proposed")
	if err != nil {
		return ClaimRecord{}, err
	}
	if state != "approved" && !allowedClaimProposedLifecycleStates[state] {
		return ClaimRecord{}, fmt.Errorf("%w: claim terminal state requires a transition event", producterror.ErrInvalidInput)
	}
	claimProposalID, err := claimProposalIDFromCreatedEvent(createdEvent, claimID)
	if err != nil {
		return ClaimRecord{}, err
	}
	claimType := strings.TrimSpace(req.ClaimType)
	if claimType == "" {
		claimType = "descriptive"
	}
	if !allowedClaimTypes[claimType] {
		return ClaimRecord{}, fmt.Errorf("%w: unsupported claim type", producterror.ErrInvalidInput)
	}

	supportingIDs, err := normalizeClaimIDList("evd_", req.SupportingEvidenceIDs)
	if err != nil {
		return ClaimRecord{}, err
	}
	opposingIDs, err := normalizeClaimIDList("evd_", req.OpposingEvidenceIDs)
	if err != nil {
		return ClaimRecord{}, err
	}
	if err := requirements.RequireEvidenceRecords(ctx, missionID, append(append([]string{}, supportingIDs...), opposingIDs...)); err != nil {
		return ClaimRecord{}, err
	}
	questionIDs, err := normalizeClaimIDList("qst_", req.DependsOnQuestionIDs)
	if err != nil {
		return ClaimRecord{}, err
	}
	if err := requirements.RequireQuestionRecords(ctx, missionID, questionIDs); err != nil {
		return ClaimRecord{}, err
	}

	userAssertionEventID := strings.TrimSpace(req.UserAssertionEventID)
	if len(supportingIDs)+len(opposingIDs) == 0 && userAssertionEventID == "" {
		return ClaimRecord{}, fmt.Errorf("%w: claim requires evidence ids or a user assertion event", producterror.ErrInvalidInput)
	}
	if userAssertionEventID != "" {
		event, err := requirements.RequireMissionEvent(ctx, missionID, userAssertionEventID)
		if err != nil {
			return ClaimRecord{}, err
		}
		if !isClaimApprovalProducer(event.Producer) {
			return ClaimRecord{}, fmt.Errorf("%w: user assertion claim requires a user or steering_chat event", producterror.ErrInvalidInput)
		}
	}

	confidence, err := NormalizeConfidence(req.Confidence)
	if err != nil {
		return ClaimRecord{}, err
	}
	approval, err := normalizeClaimApproval(req.Approval)
	if err != nil {
		return ClaimRecord{}, err
	}
	if state == "approved" || approval.State == "approved" {
		event, err := requirements.RequireMissionEvent(ctx, missionID, approval.ApprovalEventID)
		if err != nil {
			return ClaimRecord{}, err
		}
		if err := requireClaimApprovalEvent(event, claimID, claimProposalID); err != nil {
			return ClaimRecord{}, err
		}
		state = "approved"
		approval.State = "approved"
		if approval.ApprovedAt.IsZero() {
			approval.ApprovedAt = event.CreatedAt
		}
	}

	return ClaimRecord{
		SchemaVersion: ClaimRecordSchemaVersion, ObjectKind: ClaimRecordObjectKind,
		ClaimID: claimID, MissionID: missionID, State: state,
		Text: strings.TrimSpace(req.Text), ClaimType: claimType,
		SupportingEvidenceIDs: supportingIDs, OpposingEvidenceIDs: opposingIDs,
		DependsOnQuestionIDs: questionIDs, UserAssertionEventID: userAssertionEventID,
		Confidence: confidence, Approval: approval,
		CreatedEventID: strings.TrimSpace(req.CreatedEventID), CreatedAt: time.Now().UTC(),
	}, nil
}

func normalizeClaimApproval(approval ClaimApproval) (ClaimApproval, error) {
	state := strings.TrimSpace(approval.State)
	if state == "" {
		state = "pending"
	}
	if !allowedClaimApprovalStates[state] {
		return ClaimApproval{}, fmt.Errorf("%w: unsupported approval state", producterror.ErrInvalidInput)
	}
	return ClaimApproval{State: state, Required: true, ApprovalEventID: strings.TrimSpace(approval.ApprovalEventID), ApprovedAt: approval.ApprovedAt}, nil
}

func claimProposalIDFromCreatedEvent(event ledger.Event, claimID string) (string, error) {
	if event.EventType != "claim.proposed" {
		return "", fmt.Errorf("%w: claim requires a claim.proposed creation event", producterror.ErrInvalidInput)
	}
	var payload struct {
		ClaimID    string `json:"claim_id"`
		ProposalID string `json:"proposal_id"`
	}
	if err := unmarshalClaimEventPayload(event, &payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.ClaimID) != claimID {
		return "", fmt.Errorf("%w: claim.proposed event does not reference claim", producterror.ErrInvalidInput)
	}
	return strings.TrimSpace(payload.ProposalID), nil
}

func requireClaimApprovalEvent(event ledger.Event, claimID, proposalID string) error {
	if !isClaimApprovalProducer(event.Producer) {
		return fmt.Errorf("%w: approved claim requires a user or steering_chat event", producterror.ErrInvalidInput)
	}
	switch event.EventType {
	case "claim.approved":
		var payload struct {
			ClaimID string `json:"claim_id"`
		}
		if err := unmarshalClaimEventPayload(event, &payload); err != nil {
			return err
		}
		if strings.TrimSpace(payload.ClaimID) != claimID {
			return fmt.Errorf("%w: claim.approved event does not reference claim", producterror.ErrInvalidInput)
		}
		return nil
	case "proposal.approved":
		var payload claimProposalDecisionPayload
		if err := unmarshalClaimEventPayload(event, &payload); err != nil {
			return err
		}
		if proposalID == "" || strings.TrimSpace(payload.ProposalID) != proposalID {
			return fmt.Errorf("%w: proposal.approved event does not reference claim proposal", producterror.ErrInvalidInput)
		}
		if !containsClaimString(trimClaimStringList(payload.ApprovedObjectIDs), claimID) || containsClaimString(trimClaimStringList(payload.RejectedObjectIDs), claimID) {
			return fmt.Errorf("%w: proposal.approved event does not approve claim", producterror.ErrInvalidInput)
		}
		return nil
	default:
		return fmt.Errorf("%w: approved claim requires a claim or proposal approval event", producterror.ErrInvalidInput)
	}
}

type claimProposalDecisionPayload struct {
	ProposalID        string   `json:"proposal_id"`
	ApprovedObjectIDs []string `json:"approved_object_ids"`
	RejectedObjectIDs []string `json:"rejected_object_ids"`
}

func unmarshalClaimEventPayload(event ledger.Event, target any) error {
	payload := event.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("%w: invalid ledger event payload", producterror.ErrInvalidInput)
	}
	return nil
}

func normalizeClaimLifecycleState(state, defaultState string) (string, error) {
	trimmed := strings.TrimSpace(state)
	if trimmed == "" {
		trimmed = defaultState
	}
	if !allowedClaimLifecycleStates[trimmed] {
		return "", fmt.Errorf("%w: unsupported lifecycle state", producterror.ErrInvalidInput)
	}
	return trimmed, nil
}

func normalizeClaimIDList(prefix string, ids []string) ([]string, error) {
	normalized := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if err := validateClaimID(prefix, trimmed); err != nil {
			return nil, err
		}
		if _, ok := seen[trimmed]; ok {
			return nil, fmt.Errorf("%w: duplicate id", producterror.ErrInvalidInput)
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized, nil
}

func validateClaimID(prefix, id string) error {
	if !strings.HasPrefix(strings.TrimSpace(id), prefix) || len(strings.TrimSpace(id)) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}

func isClaimApprovalProducer(producer ledger.Producer) bool {
	return producer.Type == "user" || producer.Type == "steering_chat"
}

func trimClaimStringList(values []string) []string {
	trimmed := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		candidate := strings.TrimSpace(value)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		trimmed = append(trimmed, candidate)
	}
	return trimmed
}

func containsClaimString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
