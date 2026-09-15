package sqlite

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/artifactrepo"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/missionrepo"
)

// CommitReportRedpenRevision는 redpen revision artifact와 이벤트를 함께 저장한다.
func (s *Store) CommitReportRedpenRevision(
	ctx context.Context,
	candidate artifactcontract.Raw,
	build func([]ledger.Event, artifactcontract.Raw, string) (ledger.Event, bool, error),
) (artifactcontract.Raw, ledger.Event, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	defer tx.Rollback()

	events, err := missionrepo.ListLedgerEventsTx(ctx, tx, candidate.MissionID)
	if err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	target, exists, err := artifactrepo.GetRawArtifactByMissionSHA(ctx, tx, candidate.MissionID, candidate.SHA256)
	if err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	if !exists {
		target = candidate
	}
	ownership := app.ReportRedpenArtifactOwnershipReferenced
	if !exists {
		ownership = app.ReportRedpenArtifactOwnershipCreated
	}
	event, appendEvent, err := build(events, target, ownership)
	if err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	if !appendEvent {
		if !exists {
			return artifactcontract.Raw{}, ledger.Event{}, false, fmt.Errorf("redpen no-op target is not stored")
		}
		if err := tx.Commit(); err != nil {
			return artifactcontract.Raw{}, ledger.Event{}, false, err
		}
		return target, event, false, nil
	}
	committed, err := missionrepo.AppendLedgerEventTx(ctx, tx, event)
	if err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	if !exists {
		if err := artifactrepo.InsertRawArtifactTx(ctx, tx, candidate); err != nil {
			return artifactcontract.Raw{}, ledger.Event{}, false, err
		}
	}
	if reportrun.IsReportEventType(committed.EventType) {
		if err := s.applyReportRunRegistrationTx(ctx, tx, candidate.MissionID); err != nil {
			return artifactcontract.Raw{}, ledger.Event{}, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, false, err
	}
	return target, committed, true, nil
}
