package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/missionrepo"
)

// PreviewMissionHardDelete는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) PreviewMissionHardDelete(ctx context.Context, missionID string) (app.MissionHardDeleteImpact, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	defer tx.Rollback()
	return missionHardDeleteImpactTx(ctx, tx, missionID)
}

// HardDeleteMission는 SQLite 저장소 어댑터의 명시적 상태 전이를 수행한다. 결과는 SQLite 기록으로 확인한다.
func (s *Store) HardDeleteMission(ctx context.Context, missionID string, validate func([]app.LedgerEvent) error) (app.MissionHardDeleteImpact, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	defer tx.Rollback()

	events, err := missionrepo.ListLedgerEventsTx(ctx, tx, missionID)
	if err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if validate != nil {
		if err := validate(events); err != nil {
			return app.MissionHardDeleteImpact{}, err
		}
	}

	impact, err := missionHardDeleteImpactTx(ctx, tx, missionID)
	if err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	for _, statement := range missionHardDeleteStatements {
		args := []any{missionID}
		if statement.MissionIDArgs == 2 {
			args = append(args, missionID)
		}
		if _, err := tx.ExecContext(ctx, statement.SQL, args...); err != nil {
			return app.MissionHardDeleteImpact{}, err
		}
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM plasma_missions WHERE mission_id = ?`, missionID)
	if err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if affected != 1 {
		return app.MissionHardDeleteImpact{}, fmt.Errorf("%w: mission does not exist", app.ErrInvalidInput)
	}
	if err := tx.Commit(); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	return impact, nil
}

type missionHardDeleteStatement struct {
	SQL           string
	MissionIDArgs int
}

var missionHardDeleteStatements = []missionHardDeleteStatement{
	{SQL: `DELETE FROM plasma_report_blocks WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_report_versions WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_reports WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_report_run_artifacts WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_report_run_events WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_report_runs WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_evidence_records WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_claim_records WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_question_records WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_option_records WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_proposal_bundles WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_source_snapshot_artifacts
	  WHERE snapshot_id IN (SELECT snapshot_id FROM plasma_source_snapshots WHERE mission_id = ?)
	     OR artifact_id IN (SELECT artifact_id FROM plasma_raw_artifacts WHERE mission_id = ?)`, MissionIDArgs: 2},
	{SQL: `DELETE FROM plasma_source_snapshots WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_raw_artifacts WHERE mission_id = ?`, MissionIDArgs: 1},
	{SQL: `DELETE FROM plasma_ledger_events WHERE mission_id = ?`, MissionIDArgs: 1},
}

func missionHardDeleteImpactTx(ctx context.Context, tx *sql.Tx, missionID string) (app.MissionHardDeleteImpact, error) {
	var exists int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM plasma_missions WHERE mission_id = ?`, missionID).Scan(&exists); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if exists != 1 {
		return app.MissionHardDeleteImpact{}, fmt.Errorf("%w: mission does not exist", app.ErrInvalidInput)
	}

	impact := app.MissionHardDeleteImpact{}
	var err error
	if impact.LedgerEvents, err = countMissionRowsTx(ctx, tx, "plasma_ledger_events", missionID); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if impact.RawArtifacts, err = countMissionRowsTx(ctx, tx, "plasma_raw_artifacts", missionID); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(byte_size), 0) FROM plasma_raw_artifacts WHERE mission_id = ?`, missionID).Scan(&impact.RawArtifactBytes); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if impact.SourceSnapshots, err = countMissionRowsTx(ctx, tx, "plasma_source_snapshots", missionID); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM plasma_source_snapshot_artifacts
WHERE snapshot_id IN (SELECT snapshot_id FROM plasma_source_snapshots WHERE mission_id = ?)
   OR artifact_id IN (SELECT artifact_id FROM plasma_raw_artifacts WHERE mission_id = ?)`,
		missionID, missionID).Scan(&impact.SourceSnapshotArtifactLinks); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if impact.EvidenceRecords, err = countMissionRowsTx(ctx, tx, "plasma_evidence_records", missionID); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if impact.ClaimRecords, err = countMissionRowsTx(ctx, tx, "plasma_claim_records", missionID); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if impact.QuestionRecords, err = countMissionRowsTx(ctx, tx, "plasma_question_records", missionID); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if impact.OptionRecords, err = countMissionRowsTx(ctx, tx, "plasma_option_records", missionID); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if impact.ProposalBundles, err = countMissionRowsTx(ctx, tx, "plasma_proposal_bundles", missionID); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if impact.Reports, err = countMissionRowsTx(ctx, tx, "plasma_reports", missionID); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if impact.ReportVersions, err = countMissionRowsTx(ctx, tx, "plasma_report_versions", missionID); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	if impact.ReportBlocks, err = countMissionRowsTx(ctx, tx, "plasma_report_blocks", missionID); err != nil {
		return app.MissionHardDeleteImpact{}, err
	}
	return impact, nil
}

func countMissionRowsTx(ctx context.Context, tx *sql.Tx, table string, missionID string) (int64, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE mission_id = ?`, table)
	var count int64
	err := tx.QueryRowContext(ctx, query, missionID).Scan(&count)
	return count, err
}
