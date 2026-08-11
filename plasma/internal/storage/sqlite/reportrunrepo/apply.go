package reportrunrepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// ApplyRegistrationTx upserts a mission report-run projection inside a
// caller-owned transaction. Existing purged tombstones are never rehydrated.
func ApplyRegistrationTx(ctx context.Context, tx *sql.Tx, registration reportrun.Registration) error {
	return applyRegistrationTx(ctx, tx, registration, false)
}

// ApplyNativeRegistrationTx applies registration for a current write. Any
// mission mismatch between run membership and underlying rows rejects the
// caller-owned transaction.
func ApplyNativeRegistrationTx(ctx context.Context, tx *sql.Tx, registration reportrun.Registration) error {
	return applyRegistrationTx(ctx, tx, registration, true)
}

func applyRegistrationTx(ctx context.Context, tx *sql.Tx, registration reportrun.Registration, native bool) error {
	checked, err := enforceRegistrationMissionInvariantTx(ctx, tx, registration, native)
	if err != nil {
		return err
	}
	registration = checked
	insertedRuns := map[string]bool{}
	acceptedRuns := map[string]bool{}
	touchedRuns := map[string]bool{}
	for _, run := range registration.Runs {
		inserted, accepted, err := upsertRunTx(ctx, tx, run, native)
		if err != nil {
			return err
		}
		if !accepted {
			continue
		}
		insertedRuns[run.RunID] = inserted
		acceptedRuns[run.RunID] = true
	}
	for _, event := range registration.Events {
		if !acceptedRuns[event.RunID] {
			if native {
				return fmt.Errorf("%w: report run is purged", producterror.ErrConflict)
			}
			continue
		}
		inserted, err := insertEventMembershipTx(ctx, tx, event, native)
		if err != nil {
			return err
		}
		if inserted && !insertedRuns[event.RunID] {
			touchedRuns[event.RunID] = true
		}
	}
	for _, artifact := range registration.Artifacts {
		if !acceptedRuns[artifact.RunID] {
			if native {
				return fmt.Errorf("%w: report run is purged", producterror.ErrConflict)
			}
			continue
		}
		inserted, err := insertArtifactMembershipTx(ctx, tx, artifact, native)
		if err != nil {
			return err
		}
		if inserted && !insertedRuns[artifact.RunID] {
			touchedRuns[artifact.RunID] = true
		}
	}
	for _, run := range registration.Runs {
		if acceptedRuns[run.RunID] && touchedRuns[run.RunID] {
			if err := bumpRunRevisionTx(ctx, tx, run.RunID, run.UpdatedAt); err != nil {
				return err
			}
		}
	}
	return nil
}

func enforceRegistrationMissionInvariantTx(ctx context.Context, tx *sql.Tx, registration reportrun.Registration, native bool) (reportrun.Registration, error) {
	runMission := map[string]string{}
	for _, run := range registration.Runs {
		runMission[run.RunID] = run.MissionID
	}
	ambiguousRuns := map[string]bool{}
	validEvents := make([]reportrun.EventMembership, 0, len(registration.Events))
	for _, event := range registration.Events {
		if !sameMission(runMission[event.RunID], event.MissionID) {
			if native {
				return reportrun.Registration{}, fmt.Errorf("report-run event membership mission mismatch for %s", event.EventID)
			}
			ambiguousRuns[event.RunID] = true
			continue
		}
		ok, err := ledgerEventMissionMatchesTx(ctx, tx, event.EventID, event.MissionID)
		if err != nil {
			return reportrun.Registration{}, err
		}
		if !ok {
			if native {
				return reportrun.Registration{}, fmt.Errorf("report-run event membership missing or cross-mission for %s", event.EventID)
			}
			ambiguousRuns[event.RunID] = true
			continue
		}
		validEvents = append(validEvents, event)
	}
	validArtifacts := make([]reportrun.ArtifactMembership, 0, len(registration.Artifacts))
	for _, artifact := range registration.Artifacts {
		if !sameMission(runMission[artifact.RunID], artifact.MissionID) {
			if native {
				return reportrun.Registration{}, fmt.Errorf("report-run artifact membership mission mismatch for %s", artifact.ArtifactID)
			}
			ambiguousRuns[artifact.RunID] = true
			continue
		}
		ok, err := rawArtifactMissionMatchesTx(ctx, tx, artifact.ArtifactID, artifact.MissionID)
		if err != nil {
			return reportrun.Registration{}, err
		}
		if !ok {
			if native {
				return reportrun.Registration{}, fmt.Errorf("report-run artifact membership missing or cross-mission for %s", artifact.ArtifactID)
			}
			ambiguousRuns[artifact.RunID] = true
			continue
		}
		validArtifacts = append(validArtifacts, artifact)
	}
	if len(ambiguousRuns) > 0 {
		for index := range registration.Runs {
			if ambiguousRuns[registration.Runs[index].RunID] {
				registration.Runs[index].LifecycleState = reportrun.LifecycleAmbiguous
			}
		}
	}
	registration.Events = validEvents
	registration.Artifacts = validArtifacts
	return registration, nil
}

func sameMission(runMission string, membershipMission string) bool {
	return runMission != "" && runMission == membershipMission
}

func ledgerEventMissionMatchesTx(ctx context.Context, tx *sql.Tx, eventID string, missionID string) (bool, error) {
	var found string
	err := tx.QueryRowContext(ctx, `SELECT mission_id FROM plasma_ledger_events WHERE event_id = ?`, eventID).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return found == missionID, nil
}

func rawArtifactMissionMatchesTx(ctx context.Context, tx *sql.Tx, artifactID string, missionID string) (bool, error) {
	var found string
	err := tx.QueryRowContext(ctx, `SELECT mission_id FROM plasma_raw_artifacts WHERE artifact_id = ?`, artifactID).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return found == missionID, nil
}

func upsertRunTx(ctx context.Context, tx *sql.Tx, run reportrun.Run, native bool) (bool, bool, error) {
	if run.Usage.AggregationVersion == "" {
		run.Usage.AggregationVersion = reportrun.UsageAggregationVersion
	}
	var current reportrun.Run
	err := tx.QueryRowContext(ctx, `
SELECT run_id, mission_id, lifecycle_state, revision, title, final_artifact_id, registration_status
FROM plasma_report_runs
WHERE run_id = ?`, run.RunID).Scan(
		&current.RunID, &current.MissionID, &current.LifecycleState, &current.Revision,
		&current.Title, &current.FinalArtifactID, &current.RegistrationStatus)
	if err == sql.ErrNoRows {
		_, err = tx.ExecContext(ctx, `
INSERT INTO plasma_report_runs (
  run_id, mission_id, root_pending_event_id, lifecycle_state, revision, title,
  final_artifact_id, registration_status, aggregation_version, created_at, updated_at
) VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?)`,
			run.RunID, run.MissionID, run.RootPendingEventID, run.LifecycleState,
			run.Title, run.FinalArtifactID, run.RegistrationStatus,
			run.Usage.AggregationVersion, sqlitevalue.FormatTime(run.CreatedAt),
			sqlitevalue.FormatTime(run.UpdatedAt))
		return true, true, err
	}
	if err != nil {
		return false, false, err
	}
	if current.MissionID != run.MissionID {
		return false, false, fmt.Errorf("report-run mission mismatch for %s", run.RunID)
	}
	if current.LifecycleState == reportrun.LifecyclePurged {
		if native {
			return false, false, fmt.Errorf("%w: report run is purged", producterror.ErrConflict)
		}
		return false, false, nil
	}
	if current.RegistrationStatus == reportrun.RegistrationNative && run.RegistrationStatus == reportrun.RegistrationBackfilled {
		run.RegistrationStatus = current.RegistrationStatus
	}
	changed := current.LifecycleState != run.LifecycleState ||
		current.Title != run.Title ||
		current.FinalArtifactID != run.FinalArtifactID ||
		current.RegistrationStatus != run.RegistrationStatus
	if !changed {
		return false, true, nil
	}
	result, err := tx.ExecContext(ctx, `
UPDATE plasma_report_runs
SET lifecycle_state = ?, revision = revision + 1, title = ?, final_artifact_id = ?,
    registration_status = ?, updated_at = ?
WHERE run_id = ? AND lifecycle_state <> 'purged'`,
		run.LifecycleState, run.Title, run.FinalArtifactID, run.RegistrationStatus,
		sqlitevalue.FormatTime(run.UpdatedAt), run.RunID)
	if err != nil {
		return false, true, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, true, err
	}
	if affected == 0 {
		if native {
			return false, false, fmt.Errorf("%w: report run is purged", producterror.ErrConflict)
		}
		return false, false, nil
	}
	return false, true, nil
}

func insertEventMembershipTx(ctx context.Context, tx *sql.Tx, event reportrun.EventMembership, native bool) (bool, error) {
	result, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO plasma_report_run_events (
  run_id, event_id, mission_id, event_role, attempt_event_id, created_at
) SELECT ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM plasma_report_runs WHERE run_id = ? AND mission_id = ? AND lifecycle_state <> 'purged')
  AND EXISTS (SELECT 1 FROM plasma_ledger_events WHERE event_id = ? AND mission_id = ?)`,
		event.RunID, event.EventID, event.MissionID, event.EventRole,
		event.AttemptEventID, sqlitevalue.FormatTime(event.CreatedAt),
		event.RunID, event.MissionID, event.EventID, event.MissionID)
	if err != nil {
		return false, err
	}
	inserted, err := rowInserted(result)
	if err != nil {
		return false, err
	}
	if !inserted {
		if err := conflictIfRunPurgedTx(ctx, tx, event.RunID, native); err != nil {
			return false, err
		}
	}
	return inserted, nil
}

func insertArtifactMembershipTx(ctx context.Context, tx *sql.Tx, artifact reportrun.ArtifactMembership, native bool) (bool, error) {
	var existing reportrun.ArtifactMembership
	err := tx.QueryRowContext(ctx, `
SELECT run_id, artifact_id, mission_id, artifact_role, ownership, attempt_event_id, source_event_id
FROM plasma_report_run_artifacts
WHERE run_id = ? AND artifact_id = ?`, artifact.RunID, artifact.ArtifactID).Scan(
		&existing.RunID, &existing.ArtifactID, &existing.MissionID,
		&existing.ArtifactRole, &existing.Ownership, &existing.AttemptEventID,
		&existing.SourceEventID)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	if err == nil {
		merged, changed := reportrun.MergeArtifactMembership(existing, artifact)
		if !changed {
			return false, nil
		}
		result, err := tx.ExecContext(ctx, `
UPDATE plasma_report_run_artifacts
SET artifact_role = ?, ownership = ?, attempt_event_id = ?, source_event_id = ?
WHERE run_id = ? AND artifact_id = ?
  AND EXISTS (
    SELECT 1 FROM plasma_report_runs
    WHERE run_id = ? AND mission_id = plasma_report_run_artifacts.mission_id
      AND lifecycle_state <> 'purged'
  )
  AND EXISTS (SELECT 1 FROM plasma_raw_artifacts WHERE artifact_id = ? AND mission_id = plasma_report_run_artifacts.mission_id)`,
			merged.ArtifactRole, merged.Ownership, merged.AttemptEventID,
			merged.SourceEventID, artifact.RunID, artifact.ArtifactID,
			artifact.RunID, artifact.ArtifactID)
		if err != nil {
			return false, err
		}
		inserted, err := rowInserted(result)
		if err != nil {
			return false, err
		}
		if !inserted {
			if err := conflictIfRunPurgedTx(ctx, tx, artifact.RunID, native); err != nil {
				return false, err
			}
		}
		return inserted, nil
	}
	result, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO plasma_report_run_artifacts (
  run_id, artifact_id, mission_id, artifact_role, ownership,
  attempt_event_id, source_event_id, created_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM plasma_report_runs WHERE run_id = ? AND mission_id = ? AND lifecycle_state <> 'purged')
  AND EXISTS (SELECT 1 FROM plasma_raw_artifacts WHERE artifact_id = ? AND mission_id = ?)`,
		artifact.RunID, artifact.ArtifactID, artifact.MissionID, artifact.ArtifactRole,
		artifact.Ownership, artifact.AttemptEventID, artifact.SourceEventID,
		sqlitevalue.FormatTime(artifact.CreatedAt), artifact.RunID, artifact.MissionID,
		artifact.ArtifactID, artifact.MissionID)
	if err != nil {
		return false, err
	}
	inserted, err := rowInserted(result)
	if err != nil {
		return false, err
	}
	if !inserted {
		if err := conflictIfRunPurgedTx(ctx, tx, artifact.RunID, native); err != nil {
			return false, err
		}
	}
	return inserted, nil
}

func bumpRunRevisionTx(ctx context.Context, tx *sql.Tx, runID string, updatedAt time.Time) error {
	_, err := tx.ExecContext(ctx, `
UPDATE plasma_report_runs
SET revision = revision + 1, updated_at = ?
WHERE run_id = ? AND lifecycle_state <> 'purged'`,
		sqlitevalue.FormatTime(updatedAt), runID)
	return err
}

func rowInserted(result sql.Result) (bool, error) {
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func conflictIfRunPurgedTx(ctx context.Context, tx *sql.Tx, runID string, native bool) error {
	if !native {
		return nil
	}
	var lifecycleState string
	err := tx.QueryRowContext(ctx, `SELECT lifecycle_state FROM plasma_report_runs WHERE run_id = ?`, runID).Scan(&lifecycleState)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if lifecycleState == reportrun.LifecyclePurged {
		return fmt.Errorf("%w: report run is purged", producterror.ErrConflict)
	}
	return nil
}
