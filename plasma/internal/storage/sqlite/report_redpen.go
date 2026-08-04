package sqlite

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/artifactrepo"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/missionrepo"
)

// CommitReportRedpenRevision는 redpen revision artifact와 이벤트를 함께 저장한다.
func (s *Store) CommitReportRedpenRevision(
	ctx context.Context,
	candidate app.RawArtifact,
	build func([]app.LedgerEvent, app.RawArtifact) (app.LedgerEvent, bool, error),
) (app.RawArtifact, app.LedgerEvent, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	defer tx.Rollback()

	events, err := missionrepo.ListLedgerEventsTx(ctx, tx, candidate.MissionID)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	target, exists, err := artifactrepo.GetRawArtifactByMissionSHA(ctx, tx, candidate.MissionID, candidate.SHA256)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	if !exists {
		target = candidate
	}
	event, appendEvent, err := build(events, target)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	if !appendEvent {
		if !exists {
			return app.RawArtifact{}, app.LedgerEvent{}, false, fmt.Errorf("redpen no-op target is not stored")
		}
		if err := tx.Commit(); err != nil {
			return app.RawArtifact{}, app.LedgerEvent{}, false, err
		}
		return target, event, false, nil
	}
	committed, err := missionrepo.AppendLedgerEventTx(ctx, tx, event)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	if !exists {
		if err := artifactrepo.InsertRawArtifactTx(ctx, tx, candidate); err != nil {
			return app.RawArtifact{}, app.LedgerEvent{}, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	return target, committed, true, nil
}
