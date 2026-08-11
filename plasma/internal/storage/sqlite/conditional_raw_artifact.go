package sqlite

import (
	"context"
	"fmt"

	artifactmodel "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
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
		if reportrun.IsReportEventType(event.EventType) {
			if err := s.applyReportRunRegistrationTx(ctx, tx, artifact.MissionID); err != nil {
				return artifactmodel.Raw{}, ledger.Event{}, false, err
			}
		}
	} else if artifact, err = artifactrepo.GetRawArtifactTx(ctx, tx, artifact.ArtifactID); err != nil {
		return artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	return artifact, event, create, nil
}

// CommitDesignedReportHTMLExportConditionally commits the designed HTML
// content model artifact, HTML artifact, terminal event, and report-run
// registration in one transaction. It deliberately supports only this
// two-artifact report completion path.
func (s *Store) CommitDesignedReportHTMLExportConditionally(
	ctx context.Context,
	missionID string,
	contentModel artifactmodel.Raw,
	html artifactmodel.Raw,
	build func([]ledger.Event) ([]ledger.Event, bool, error),
) (artifactmodel.Raw, artifactmodel.Raw, ledger.Event, bool, error) {
	if contentModel.MissionID != missionID || html.MissionID != missionID {
		return artifactmodel.Raw{}, artifactmodel.Raw{}, ledger.Event{}, false, fmt.Errorf("designed HTML artifact mission mismatch")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return artifactmodel.Raw{}, artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	defer tx.Rollback()
	events, err := missionrepo.ListLedgerEventsTx(ctx, tx, missionID)
	if err != nil {
		return artifactmodel.Raw{}, artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	toAppend, create, err := build(events)
	if err != nil {
		return artifactmodel.Raw{}, artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	if !create {
		return artifactmodel.Raw{}, artifactmodel.Raw{}, ledger.Event{}, false, nil
	}
	if len(toAppend) != 1 {
		return artifactmodel.Raw{}, artifactmodel.Raw{}, ledger.Event{}, false, fmt.Errorf("designed HTML export requires exactly one terminal event")
	}
	committed, err := missionrepo.AppendLedgerEventTx(ctx, tx, toAppend[0])
	if err != nil {
		return artifactmodel.Raw{}, artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	if err := artifactrepo.InsertRawArtifactTx(ctx, tx, contentModel); err != nil {
		return artifactmodel.Raw{}, artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	if err := artifactrepo.InsertRawArtifactTx(ctx, tx, html); err != nil {
		return artifactmodel.Raw{}, artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	if reportrun.IsReportEventType(committed.EventType) {
		if err := s.applyReportRunRegistrationTx(ctx, tx, missionID); err != nil {
			return artifactmodel.Raw{}, artifactmodel.Raw{}, ledger.Event{}, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return artifactmodel.Raw{}, artifactmodel.Raw{}, ledger.Event{}, false, err
	}
	return contentModel, html, committed, true, nil
}
