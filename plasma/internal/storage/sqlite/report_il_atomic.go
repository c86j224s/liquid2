package sqlite

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/artifactrepo"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/missionrepo"
)

// CommitReportILBundleConditionally stores the six product artifacts and
// canonical terminal event in one SQLite transaction.
func (s *Store) CommitReportILBundleConditionally(ctx context.Context, missionID, pendingID string, artifacts []artifact.Raw, storeCompleted ledger.Event, build func([]ledger.Event) (ledger.Event, bool, error)) ([]artifact.Raw, ledger.Event, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ledger.Event{}, false, err
	}
	defer tx.Rollback()
	events, err := missionrepo.ListLedgerEventsTx(ctx, tx, missionID)
	if err != nil {
		return nil, ledger.Event{}, false, err
	}
	terminal, create, err := build(events)
	if err != nil || !create {
		return nil, ledger.Event{}, false, err
	}
	if terminal.EventID == storeCompleted.EventID || terminal.EventID == pendingID || storeCompleted.EventID == pendingID {
		return nil, ledger.Event{}, false, fmt.Errorf("report IL event ids must be distinct")
	}
	if _, err := missionrepo.AppendLedgerEventTx(ctx, tx, storeCompleted); err != nil {
		return nil, ledger.Event{}, false, err
	}
	committed, err := missionrepo.AppendLedgerEventTx(ctx, tx, terminal)
	if err != nil {
		return nil, ledger.Event{}, false, err
	}
	for _, current := range artifacts {
		if err := artifactrepo.InsertRawArtifactTx(ctx, tx, current); err != nil {
			return nil, ledger.Event{}, false, err
		}
	}
	if reportrun.IsReportEventType(committed.EventType) {
		if err := s.applyReportRunRegistrationTx(ctx, tx, missionID); err != nil {
			return nil, ledger.Event{}, false, fmt.Errorf("report IL run registration failed: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, ledger.Event{}, false, err
	}
	return artifacts, committed, true, nil
}
