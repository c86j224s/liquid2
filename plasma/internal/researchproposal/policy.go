package researchproposal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
)

var allowedProposalStates = map[string]bool{
	"pending_review":     true,
	"approved":           true,
	"partially_approved": true,
	"rejected":           true,
	"withdrawn":          true,
}

var allowedRequestedDecisions = map[string]bool{
	"approve": true,
	"reject":  true,
	"revise":  true,
	"split":   true,
}

// BuildProposalBundle validates and assembles a pending proposal bundle.
//
// requireRef owns storage lookup and mission membership checks. pendingRefs are
// the records assembled in the same atomic write; they bypass only that lookup,
// never kind, ID, or duplicate validation.
func BuildProposalBundle(
	ctx context.Context,
	requireRef func(context.Context, string, string, string) error,
	req CreateProposalBundleRequest,
	createdEvent ledger.Event,
	pendingRefs []researchcatalog.ObjectRef,
) (ProposalBundle, error) {
	proposalID := strings.TrimSpace(req.ProposalID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("prp_", proposalID); err != nil {
		return ProposalBundle{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return ProposalBundle{}, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return ProposalBundle{}, fmt.Errorf("%w: proposal title is required", producterror.ErrInvalidInput)
	}
	if createdEvent.MissionID != missionID || strings.TrimSpace(createdEvent.EventID) != strings.TrimSpace(req.CreatedEventID) {
		return ProposalBundle{}, fmt.Errorf("%w: proposal creation event mismatch", producterror.ErrInvalidInput)
	}
	if err := requireProposalSubmittedEvent(createdEvent, proposalID); err != nil {
		return ProposalBundle{}, err
	}
	state := strings.TrimSpace(req.State)
	if state == "" {
		state = "pending_review"
	}
	if state != "pending_review" {
		return ProposalBundle{}, fmt.Errorf("%w: proposal bundle must start pending_review", producterror.ErrInvalidInput)
	}
	requestedDecision := strings.TrimSpace(req.RequestedDecision)
	if !allowedRequestedDecisions[requestedDecision] {
		return ProposalBundle{}, fmt.Errorf("%w: unsupported requested decision", producterror.ErrInvalidInput)
	}
	objectRefs, err := normalizeObjectRefs(ctx, requireRef, missionID, req.ObjectRefs, pendingRefs)
	if err != nil {
		return ProposalBundle{}, err
	}
	if len(objectRefs) == 0 {
		return ProposalBundle{}, fmt.Errorf("%w: proposal bundle requires object refs", producterror.ErrInvalidInput)
	}

	now := time.Now().UTC()
	return ProposalBundle{
		SchemaVersion:     ProposalBundleSchemaVersion,
		ObjectKind:        ProposalBundleObjectKind,
		ProposalID:        proposalID,
		MissionID:         missionID,
		State:             state,
		Title:             strings.TrimSpace(req.Title),
		ObjectRefs:        objectRefs,
		RequestedDecision: requestedDecision,
		CreatedEventID:    strings.TrimSpace(req.CreatedEventID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

func normalizeObjectRefs(
	ctx context.Context,
	requireRef func(context.Context, string, string, string) error,
	missionID string,
	refs []researchcatalog.ObjectRef,
	pendingRefs []researchcatalog.ObjectRef,
) ([]researchcatalog.ObjectRef, error) {
	normalized := make([]researchcatalog.ObjectRef, 0, len(refs))
	seen := map[string]struct{}{}
	pending := map[string]struct{}{}
	for _, ref := range pendingRefs {
		objectKind := strings.TrimSpace(ref.ObjectKind)
		objectID := strings.TrimSpace(ref.ObjectID)
		if objectKind != "" && objectID != "" {
			pending[objectKind+"\x00"+objectID] = struct{}{}
		}
	}
	for _, ref := range refs {
		objectKind := strings.TrimSpace(ref.ObjectKind)
		objectID := strings.TrimSpace(ref.ObjectID)
		key := objectKind + "\x00" + objectID
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("%w: duplicate proposal object ref", producterror.ErrInvalidInput)
		}
		seen[key] = struct{}{}
		if err := validateObjectRefID(objectKind, objectID); err != nil {
			return nil, err
		}
		if _, ok := pending[key]; !ok {
			if requireRef == nil {
				return nil, fmt.Errorf("%w: proposal object ref lookup is required", producterror.ErrInvalidInput)
			}
			if err := requireRef(ctx, missionID, objectKind, objectID); err != nil {
				return nil, err
			}
		}
		normalized = append(normalized, researchcatalog.ObjectRef{ObjectKind: objectKind, ObjectID: objectID})
	}
	return normalized, nil
}

func validateObjectRefID(objectKind, objectID string) error {
	switch objectKind {
	case researchrecords.EvidenceRecordObjectKind:
		return validateID("evd_", objectID)
	case researchrecords.ClaimRecordObjectKind:
		return validateID("clm_", objectID)
	case researchrecords.QuestionRecordObjectKind:
		return validateID("qst_", objectID)
	case researchrecords.OptionRecordObjectKind:
		return validateID("opt_", objectID)
	default:
		return fmt.Errorf("%w: unsupported proposal object kind", producterror.ErrInvalidInput)
	}
}

func requireProposalSubmittedEvent(event ledger.Event, proposalID string) error {
	if event.EventType != "proposal.submitted" {
		return fmt.Errorf("%w: proposal bundle requires a proposal.submitted creation event", producterror.ErrInvalidInput)
	}
	var payload struct {
		ProposalID string `json:"proposal_id"`
	}
	if err := unmarshalEventPayload(event, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ProposalID) != proposalID {
		return fmt.Errorf("%w: proposal.submitted event does not reference proposal", producterror.ErrInvalidInput)
	}
	return nil
}

// ValidateStateChangeRequest validates request fields before any proposal read.
func ValidateStateChangeRequest(req UpdateProposalBundleStateRequest) (proposalID, nextState string, err error) {
	proposalID = strings.TrimSpace(req.ProposalID)
	if err := validateID("prp_", proposalID); err != nil {
		return "", "", err
	}
	nextState = strings.TrimSpace(req.State)
	if !allowedProposalStates[nextState] || nextState == "pending_review" {
		return "", "", fmt.Errorf("%w: unsupported proposal state transition", producterror.ErrInvalidInput)
	}
	return proposalID, nextState, nil
}

// ValidateTransition enforces the terminal proposal transition rule.
func ValidateTransition(current ProposalBundle, nextState string) error {
	if current.State != "pending_review" ||
		(nextState != "approved" && nextState != "partially_approved" && nextState != "rejected" && nextState != "withdrawn") {
		return fmt.Errorf("%w: invalid proposal state transition", producterror.ErrInvalidInput)
	}
	return nil
}

type ProposalDecisionPayload struct {
	ProposalID        string   `json:"proposal_id"`
	ApprovedObjectIDs []string `json:"approved_object_ids"`
	RejectedObjectIDs []string `json:"rejected_object_ids"`
}

// BuildStateUpdate validates the decision event and creates the storage update.
func BuildStateUpdate(
	req UpdateProposalBundleStateRequest,
	current ProposalBundle,
	event ledger.Event,
	nextState string,
) (ProposalBundleStateUpdate, error) {
	if !isApprovalProducer(event.Producer) {
		return ProposalBundleStateUpdate{}, fmt.Errorf("%w: proposal decision requires a user or steering_chat event", producterror.ErrInvalidInput)
	}
	if err := requireProposalDecisionEvent(event, current, nextState); err != nil {
		return ProposalBundleStateUpdate{}, err
	}

	now := time.Now().UTC()
	update := ProposalBundleStateUpdate{
		ProposalID:      strings.TrimSpace(req.ProposalID),
		FromState:       current.State,
		ToState:         nextState,
		DecisionEventID: strings.TrimSpace(req.DecisionEventID),
		DecidedAt:       event.CreatedAt,
		UpdatedAt:       now,
	}
	if update.DecidedAt.IsZero() {
		update.DecidedAt = now
	}
	return update, nil
}

func requireProposalDecisionEvent(event ledger.Event, bundle ProposalBundle, targetState string) error {
	if !proposalDecisionEventTypeMatches(event, targetState) {
		return fmt.Errorf("%w: proposal decision event does not match target state", producterror.ErrInvalidInput)
	}
	if targetState == "withdrawn" {
		return requireProposalIDPayload(event, bundle.ProposalID)
	}
	var payload ProposalDecisionPayload
	if err := unmarshalEventPayload(event, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ProposalID) != bundle.ProposalID {
		return fmt.Errorf("%w: proposal decision event does not reference proposal", producterror.ErrInvalidInput)
	}
	approvedIDs := trimStringList(payload.ApprovedObjectIDs)
	rejectedIDs := trimStringList(payload.RejectedObjectIDs)
	refIDs := objectRefIDs(bundle.ObjectRefs)
	if err := requireDecisionIDsInRefs(approvedIDs, refIDs); err != nil {
		return err
	}
	if err := requireDecisionIDsInRefs(rejectedIDs, refIDs); err != nil {
		return err
	}
	if overlaps(approvedIDs, rejectedIDs) {
		return fmt.Errorf("%w: proposal decision ids overlap", producterror.ErrInvalidInput)
	}
	switch targetState {
	case "approved":
		if !sameStringSet(approvedIDs, refIDs) || len(rejectedIDs) != 0 {
			return fmt.Errorf("%w: proposal.approved payload must approve all refs", producterror.ErrInvalidInput)
		}
	case "rejected":
		if !sameStringSet(rejectedIDs, refIDs) || len(approvedIDs) != 0 {
			return fmt.Errorf("%w: proposal.rejected payload must reject all refs", producterror.ErrInvalidInput)
		}
	case "partially_approved":
		if len(approvedIDs) == 0 || len(rejectedIDs) == 0 {
			return fmt.Errorf("%w: proposal.partially_approved payload needs approved and rejected ids", producterror.ErrInvalidInput)
		}
		if !sameStringSet(append(append([]string{}, approvedIDs...), rejectedIDs...), refIDs) {
			return fmt.Errorf("%w: proposal.partially_approved payload must cover all refs", producterror.ErrInvalidInput)
		}
	}
	return nil
}

func proposalDecisionEventTypeMatches(event ledger.Event, targetState string) bool {
	switch targetState {
	case "approved":
		return event.EventType == "proposal.approved"
	case "partially_approved":
		return event.EventType == "proposal.partially_approved"
	case "rejected":
		return event.EventType == "proposal.rejected"
	case "withdrawn":
		return event.EventType == "proposal.withdrawn" || event.EventType == "proposal.rejected"
	default:
		return false
	}
}

func requireProposalIDPayload(event ledger.Event, proposalID string) error {
	var payload struct {
		ProposalID string `json:"proposal_id"`
	}
	if err := unmarshalEventPayload(event, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ProposalID) != proposalID {
		return fmt.Errorf("%w: proposal event does not reference proposal", producterror.ErrInvalidInput)
	}
	return nil
}

func unmarshalEventPayload(event ledger.Event, target any) error {
	payload := event.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("%w: invalid ledger event payload", producterror.ErrInvalidInput)
	}
	return nil
}

func trimStringList(values []string) []string {
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

func requireDecisionIDsInRefs(decisionIDs, refIDs []string) error {
	for _, id := range decisionIDs {
		if !containsString(refIDs, id) {
			return fmt.Errorf("%w: proposal decision references unknown object", producterror.ErrInvalidInput)
		}
	}
	return nil
}

func sameStringSet(left, right []string) bool {
	left = trimStringList(left)
	right = trimStringList(right)
	if len(left) != len(right) {
		return false
	}
	for _, value := range left {
		if !containsString(right, value) {
			return false
		}
	}
	return true
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func overlaps(left, right []string) bool {
	for _, value := range left {
		if containsString(right, value) {
			return true
		}
	}
	return false
}

func isApprovalProducer(producer ledger.Producer) bool {
	return producer.Type == "user" || producer.Type == "steering_chat"
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}
