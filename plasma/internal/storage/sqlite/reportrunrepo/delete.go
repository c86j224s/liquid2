package reportrunrepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// DeleteRunTx removes member events and run-owned unshared artifacts, then
// leaves a purged usage tombstone for the report run.
func DeleteRunTx(ctx context.Context, tx *sql.Tx, decision reportrun.DeleteDecision) error {
	if len(decision.Preview.Blockers) > 0 {
		return fmt.Errorf("blocked report run delete")
	}
	runID := decision.Preview.RunID
	var missionID string
	if err := tx.QueryRowContext(ctx, `SELECT mission_id FROM plasma_report_runs WHERE run_id = ?`, runID).Scan(&missionID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM plasma_report_run_events WHERE run_id = ? AND mission_id = ?`, runID, missionID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM plasma_report_run_artifacts WHERE run_id = ? AND mission_id = ?`, runID, missionID); err != nil {
		return err
	}
	if err := deleteIDsForMissionTx(ctx, tx, "plasma_ledger_events", "event_id", missionID, decision.DeleteEventIDs); err != nil {
		return err
	}
	if err := deleteIDsForMissionTx(ctx, tx, "plasma_raw_artifacts", "artifact_id", missionID, decision.DeleteArtifactIDs); err != nil {
		return err
	}
	usage := decision.RetainedUsage
	if usage.AggregationVersion == "" {
		usage.AggregationVersion = reportrun.UsageAggregationVersion
	}
	_, err := tx.ExecContext(ctx, `
UPDATE plasma_report_runs
SET lifecycle_state = 'purged',
    revision = revision + 1,
    title = '',
    final_artifact_id = '',
    purged_at = ?,
    purged_by_type = ?,
    purged_by_id = ?,
    usage_record_count = ?,
    usage_available_count = ?,
    usage_unavailable_count = ?,
    input_tokens = ?,
    cached_input_tokens = ?,
    uncached_input_tokens = ?,
    output_tokens = ?,
    reasoning_output_tokens = ?,
    total_tokens = ?,
    usage_partial = ?,
    aggregation_version = ?,
    updated_at = ?
WHERE run_id = ? AND mission_id = ?`,
		sqlitevalue.FormatTime(decision.PurgedAt), decision.PurgedByType,
		decision.PurgedByID, usage.UsageRecordCount, usage.UsageAvailableCount,
		usage.UsageUnavailableCount, usage.InputTokens, usage.CachedInputTokens,
		usage.UncachedInputTokens, usage.OutputTokens, usage.ReasoningOutputTokens,
		usage.TotalTokens, boolInt(usage.UsagePartial), usage.AggregationVersion,
		sqlitevalue.FormatTime(decision.PurgedAt), runID, missionID)
	return err
}

func deleteIDsForMissionTx(ctx context.Context, tx *sql.Tx, table string, column string, missionID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	args = append([]any{missionID}, args...)
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE mission_id = ? AND %s IN (%s)`, table, column, placeholders), args...)
	return err
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}
