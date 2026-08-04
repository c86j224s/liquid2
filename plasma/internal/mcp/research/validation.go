package research

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

var (
	toolEvidenceTypes = map[string]bool{"quote": true, "fact": true, "table_row": true, "statistic": true, "observation": true, "interpretation": true, "reaction": true, "rumor": true, "controversy": true, "market_signal": true, "code": true, "formula": true, "benchmark": true, "open_question": true}
	toolClaimTypes    = map[string]bool{"descriptive": true, "evaluative": true, "recommendation": true, "risk": true, "decision": true}
	toolPriorities    = map[string]bool{"low": true, "medium": true, "high": true}
	toolConfidence    = map[string]bool{"low": true, "medium": true, "high": true, "unknown": true}
)

func validateEvidenceProposeInput(input evidenceProposeInput) error {
	if err := validateProposalInputs(input.ProposalID, input.EventID, input.ProposalEventID); err != nil {
		return err
	}
	if err := validateID("evd_", input.EvidenceID); err != nil {
		return err
	}
	if strings.TrimSpace(input.Summary) == "" {
		return fmt.Errorf("%w: evidence summary is required", app.ErrInvalidInput)
	}
	evidenceType := strings.TrimSpace(input.EvidenceType)
	if evidenceType == "" {
		return fmt.Errorf("%w: evidence_type is required", app.ErrInvalidInput)
	}
	if !toolEvidenceTypes[evidenceType] {
		return fmt.Errorf("%w: unsupported evidence type", app.ErrInvalidInput)
	}
	if len(input.SnapshotRefs) == 0 {
		return fmt.Errorf("%w: evidence proposal requires snapshot refs", app.ErrInvalidInput)
	}
	if err := validateSnapshotRefs(input.SnapshotRefs); err != nil {
		return err
	}
	return validateConfidence(input.Confidence)
}

func validateQuestionsProposeInput(input questionsProposeInput) error {
	if err := validateProposalInputs(input.ProposalID, input.EventID, input.ProposalEventID); err != nil {
		return err
	}
	if err := validateID("qst_", input.QuestionID); err != nil {
		return err
	}
	if strings.TrimSpace(input.Text) == "" {
		return fmt.Errorf("%w: question text is required", app.ErrInvalidInput)
	}
	priority := strings.TrimSpace(input.Priority)
	if priority != "" && !toolPriorities[priority] {
		return fmt.Errorf("%w: unsupported question priority", app.ErrInvalidInput)
	}
	if err := validateIDList("evd_", input.RelatedEvidenceIDs); err != nil {
		return err
	}
	return validateIDList("clm_", input.RelatedClaimIDs)
}

func validateClaimsProposeInput(input claimsProposeInput) error {
	if err := validateProposalInputs(input.ProposalID, input.EventID, input.ProposalEventID); err != nil {
		return err
	}
	if err := validateID("clm_", input.ClaimID); err != nil {
		return err
	}
	if strings.TrimSpace(input.Text) == "" {
		return fmt.Errorf("%w: claim text is required", app.ErrInvalidInput)
	}
	claimType := strings.TrimSpace(input.ClaimType)
	if claimType != "" && !toolClaimTypes[claimType] {
		return fmt.Errorf("%w: unsupported claim type", app.ErrInvalidInput)
	}
	if err := validateIDList("evd_", input.SupportingEvidenceIDs); err != nil {
		return err
	}
	if err := validateIDList("evd_", input.OpposingEvidenceIDs); err != nil {
		return err
	}
	if err := validateIDList("qst_", input.DependsOnQuestionIDs); err != nil {
		return err
	}
	userAssertionEventID := strings.TrimSpace(input.UserAssertionEventID)
	if len(input.SupportingEvidenceIDs)+len(input.OpposingEvidenceIDs) == 0 && userAssertionEventID == "" {
		return fmt.Errorf("%w: claim proposal requires evidence ids or a user assertion event", app.ErrInvalidInput)
	}
	if userAssertionEventID != "" {
		if err := validateID("evt_", userAssertionEventID); err != nil {
			return err
		}
	}
	return validateConfidence(input.Confidence)
}

func validateClaimConfidenceInput(input claimConfidenceInput) error {
	if err := validateID("clm_", input.ClaimID); err != nil {
		return err
	}
	if err := validateID("evt_", input.EventID); err != nil {
		return err
	}
	if strings.TrimSpace(input.CausationEventID) != "" {
		if err := validateID("evt_", input.CausationEventID); err != nil {
			return err
		}
	}
	if err := validateIDList("evd_", input.BasisEvidenceIDs); err != nil {
		return err
	}
	if err := validateConfidence(input.Confidence); err != nil {
		return err
	}
	if strings.TrimSpace(input.Confidence.Rationale) == "" {
		return fmt.Errorf("%w: confidence rationale is required", app.ErrInvalidInput)
	}
	return nil
}

func validateProposalsSubmitInput(input proposalsSubmitInput) error {
	if err := validateID("prp_", input.ProposalID); err != nil {
		return err
	}
	if err := validateID("evt_", input.EventID); err != nil {
		return err
	}
	if len(input.ObjectRefs) == 0 {
		return fmt.Errorf("%w: proposal submit requires object refs", app.ErrInvalidInput)
	}
	for _, ref := range input.ObjectRefs {
		if err := validateObjectRef(ref); err != nil {
			return err
		}
	}
	return nil
}
