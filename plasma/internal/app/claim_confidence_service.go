package app

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
)

// UpdateClaimConfidence는 records builder가 검증한 append request만 장부에 추가한다.
func (s *Service) UpdateClaimConfidence(ctx context.Context, req researchrecords.UpdateClaimConfidenceRequest) (ledger.Event, error) {
	appendRequest, err := researchrecords.BuildClaimConfidenceUpdate(ctx, researchrecords.ConfidenceRequirements{
		GetClaimRecord:         s.store.GetClaimRecord,
		RequireEvidenceRecords: s.requireEvidenceRecords,
	}, req)
	if err != nil {
		return ledger.Event{}, err
	}
	return s.AppendEvent(ctx, appendRequest)
}
