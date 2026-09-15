package reportdocument

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
)

// ScopeReaders supplies record access without moving persistence into document policy.
// Callbacks preserve caller-specific validation and are invoked in reference order.
type ScopeReaders struct {
	GetEvidenceRecord     func(context.Context, string) (researchrecords.EvidenceRecord, error)
	GetClaimRecord        func(context.Context, string) (researchrecords.ClaimRecord, error)
	GetQuestionRecord     func(context.Context, string) (researchrecords.QuestionRecord, error)
	GetOptionRecord       func(context.Context, string) (researchrecords.OptionRecord, error)
	RequireApprovedObject func(context.Context, string, string, string) error
}

// ResolveScope preserves approval ordering and first-seen evidence ordering.
func ResolveScope(ctx context.Context, readers ScopeReaders, missionID string, scope ReportEvidenceScope) (ScopeRecords, error) {
	records := ScopeRecords{}
	evidenceByID := map[string]researchrecords.EvidenceRecord{}
	addEvidence := func(evidenceID string, direct bool) error {
		if _, ok := evidenceByID[evidenceID]; ok {
			return nil
		}
		evidence, err := readers.GetEvidenceRecord(ctx, evidenceID)
		if err != nil {
			return err
		}
		if evidence.MissionID != missionID {
			return fmt.Errorf("%w: report evidence belongs to another mission", producterror.ErrInvalidInput)
		}
		if evidence.State == "rejected" || evidence.State == "superseded" || evidence.State == "archived" {
			return fmt.Errorf("%w: report evidence is not active", producterror.ErrInvalidInput)
		}
		if scope.AcceptedOnly {
			if err := requireApprovedEvidence(ctx, readers, evidence); err != nil {
				return err
			}
		}
		evidenceByID[evidenceID] = evidence
		records.Evidence = append(records.Evidence, evidence)
		return nil
	}

	for _, evidenceID := range scope.EvidenceIDs {
		if err := addEvidence(evidenceID, true); err != nil {
			return ScopeRecords{}, err
		}
	}
	for _, claimID := range scope.ClaimIDs {
		claim, err := readers.GetClaimRecord(ctx, claimID)
		if err != nil {
			return ScopeRecords{}, err
		}
		if claim.MissionID != missionID {
			return ScopeRecords{}, fmt.Errorf("%w: report claim belongs to another mission", producterror.ErrInvalidInput)
		}
		if scope.AcceptedOnly && claim.State != "approved" {
			if err := readers.RequireApprovedObject(ctx, missionID, claim.CreatedEventID, claim.ClaimID); err != nil {
				return ScopeRecords{}, fmt.Errorf("%w: accepted-only report claim must be approved", err)
			}
		}
		if !scope.AcceptedOnly && !scope.IncludeProposed && claim.State != "approved" {
			if err := readers.RequireApprovedObject(ctx, missionID, claim.CreatedEventID, claim.ClaimID); err != nil {
				return ScopeRecords{}, fmt.Errorf("%w: report scope excludes proposed claims", err)
			}
		}
		if claim.State == "rejected" || claim.State == "superseded" || claim.State == "archived" {
			return ScopeRecords{}, fmt.Errorf("%w: report claim is not active", producterror.ErrInvalidInput)
		}
		records.Claims = append(records.Claims, claim)
		for _, evidenceID := range append(append([]string{}, claim.SupportingEvidenceIDs...), claim.OpposingEvidenceIDs...) {
			if err := addEvidence(evidenceID, false); err != nil {
				return ScopeRecords{}, err
			}
		}
	}
	for _, questionID := range scope.QuestionIDs {
		question, err := readers.GetQuestionRecord(ctx, questionID)
		if err != nil {
			return ScopeRecords{}, err
		}
		if question.MissionID != missionID {
			return ScopeRecords{}, fmt.Errorf("%w: report question belongs to another mission", producterror.ErrInvalidInput)
		}
		if question.State == "rejected" || question.State == "superseded" {
			return ScopeRecords{}, fmt.Errorf("%w: report question is not active", producterror.ErrInvalidInput)
		}
		records.Questions = append(records.Questions, question)
	}
	for _, optionID := range scope.OptionIDs {
		option, err := readers.GetOptionRecord(ctx, optionID)
		if err != nil {
			return ScopeRecords{}, err
		}
		if option.MissionID != missionID {
			return ScopeRecords{}, fmt.Errorf("%w: report option belongs to another mission", producterror.ErrInvalidInput)
		}
		if option.State == "rejected" || option.State == "superseded" || option.State == "archived" {
			return ScopeRecords{}, fmt.Errorf("%w: report option is not active", producterror.ErrInvalidInput)
		}
		records.Options = append(records.Options, option)
	}
	return records, nil
}

func requireApprovedEvidence(ctx context.Context, readers ScopeReaders, evidence researchrecords.EvidenceRecord) error {
	if evidence.State == "approved" {
		return nil
	}
	if err := readers.RequireApprovedObject(ctx, evidence.MissionID, evidence.CreatedEventID, evidence.EvidenceID); err != nil {
		return fmt.Errorf("%w: report evidence must be approved", err)
	}
	return nil
}
