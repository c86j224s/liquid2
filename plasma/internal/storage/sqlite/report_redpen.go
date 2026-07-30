package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

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

	events, err := listLedgerEventsTx(ctx, tx, candidate.MissionID)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	target, exists, err := getRawArtifactByMissionSHATx(ctx, tx, candidate.MissionID, candidate.SHA256)
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
	committed, err := appendLedgerEventTx(ctx, tx, event)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	if !exists {
		if err := insertRawArtifactTx(ctx, tx, candidate); err != nil {
			return app.RawArtifact{}, app.LedgerEvent{}, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	return target, committed, true, nil
}

func getRawArtifactByMissionSHATx(ctx context.Context, tx *sql.Tx, missionID, sha string) (app.RawArtifact, bool, error) {
	var artifactID string
	err := tx.QueryRowContext(ctx, `
SELECT artifact_id
FROM plasma_raw_artifacts
WHERE mission_id = ? AND sha256 = ?`, missionID, sha).Scan(&artifactID)
	if err == sql.ErrNoRows {
		return app.RawArtifact{}, false, nil
	}
	if err != nil {
		return app.RawArtifact{}, false, err
	}
	artifact, err := getRawArtifactTx(ctx, tx, artifactID)
	return artifact, err == nil, err
}
