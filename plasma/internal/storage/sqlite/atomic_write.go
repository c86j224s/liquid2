package sqlite

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/artifactrepo"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/missionrepo"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/reportrepo"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/researchrepo"
)

// CommitAtomicWrite는 artifact와 장부 이벤트를 하나의 SQLite 트랜잭션으로 기록한다.
func (s *Store) CommitAtomicWrite(ctx context.Context, write app.AtomicWrite) (app.AtomicWriteResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.AtomicWriteResult{}, err
	}
	defer tx.Rollback()

	events := make([]ledger.Event, 0, len(write.Events))
	for _, event := range write.Events {
		committed, err := missionrepo.AppendLedgerEventTx(ctx, tx, event)
		if err != nil {
			return app.AtomicWriteResult{}, err
		}
		events = append(events, committed)
	}
	for _, artifact := range write.RawArtifacts {
		if err := artifactrepo.InsertRawArtifactTx(ctx, tx, artifact); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, snapshot := range write.SourceSnapshots {
		if err := artifactrepo.InsertSourceSnapshotTx(ctx, tx, snapshot); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, record := range write.EvidenceRecords {
		if err := researchrepo.InsertEvidenceRecordTx(ctx, tx, record); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, record := range write.ClaimRecords {
		if err := researchrepo.InsertClaimRecordTx(ctx, tx, record); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, record := range write.QuestionRecords {
		if err := researchrepo.InsertQuestionRecordTx(ctx, tx, record); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, bundle := range write.ProposalBundles {
		if err := researchrepo.InsertProposalBundleTx(ctx, tx, bundle); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, report := range write.Reports {
		if err := reportrepo.InsertReportTx(ctx, tx, report); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, version := range write.ReportVersions {
		if err := reportrepo.InsertReportVersionTx(ctx, tx, version); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	for _, block := range write.ReportBlocks {
		if err := reportrepo.InsertReportBlockTx(ctx, tx, block); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	missionIDs := map[string]bool{}
	for _, event := range events {
		if reportrun.IsReportEventType(event.EventType) {
			missionIDs[event.MissionID] = true
		}
	}
	for missionID := range missionIDs {
		if err := s.applyReportRunRegistrationTx(ctx, tx, missionID); err != nil {
			return app.AtomicWriteResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return app.AtomicWriteResult{}, err
	}
	return app.AtomicWriteResult{Events: events}, nil
}
