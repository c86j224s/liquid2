package sqlite

import (
	"context"

	artifactmodel "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/artifactrepo"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/missionrepo"
)

// CommitRawArtifactWithEventConditionally는 조건이 맞을 때 raw artifact와 이벤트를 한 트랜잭션으로 저장한다.
func (s *Store) CommitRawArtifactWithEventConditionally(
	ctx context.Context,
	artifact artifactmodel.Raw,
	build func([]ledger.Event) (ledger.Event, bool, error),
) (artifactmodel.Raw, ledger.Event, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	defer tx.Rollback()
	events, err := missionrepo.ListLedgerEventsTx(ctx, tx, artifact.MissionID)
	if err != nil {
		return artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	event, create, err := build(events)
	if err != nil {
		return artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	if create {
		event, err = missionrepo.AppendLedgerEventTx(ctx, tx, event)
		if err != nil {
			return artifactmodel.Raw{}, ledger.Event{}, false, err
		}
		if err := artifactrepo.InsertRawArtifactTx(ctx, tx, artifact); err != nil {
			return artifactmodel.Raw{}, ledger.Event{}, false, err
		}
	} else if artifact, err = artifactrepo.GetRawArtifactTx(ctx, tx, artifact.ArtifactID); err != nil {
		return artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	return artifact, event, create, nil
}
