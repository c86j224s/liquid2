package app

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/missionusage"
)

// MissionUsage returns the corrected token usage projection for one mission.
// It is a read-only view over the durable ledger and does not persist aggregates.
func (s *Service) MissionUsage(ctx context.Context, missionID string) (missionusage.Summary, error) {
	if err := validateID("mis_", missionID); err != nil {
		return missionusage.Summary{}, err
	}
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return missionusage.Summary{}, err
	}
	return missionusage.Project(events), nil
}
