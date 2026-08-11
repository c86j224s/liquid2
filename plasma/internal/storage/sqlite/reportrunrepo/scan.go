package reportrunrepo

import (
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRun(scanner rowScanner) (reportrun.Run, error) {
	var run reportrun.Run
	var purgedAt string
	var createdAt string
	var updatedAt string
	var usagePartial int
	err := scanner.Scan(
		&run.RunID, &run.MissionID, &run.RootPendingEventID, &run.LifecycleState,
		&run.Revision, &run.Title, &run.FinalArtifactID, &run.RegistrationStatus,
		&purgedAt, &run.PurgedByType, &run.PurgedByID,
		&run.Usage.UsageRecordCount, &run.Usage.UsageAvailableCount,
		&run.Usage.UsageUnavailableCount, &run.Usage.InputTokens,
		&run.Usage.CachedInputTokens, &run.Usage.UncachedInputTokens,
		&run.Usage.OutputTokens, &run.Usage.ReasoningOutputTokens,
		&run.Usage.TotalTokens, &usagePartial, &run.Usage.AggregationVersion,
		&createdAt, &updatedAt)
	if err != nil {
		return reportrun.Run{}, err
	}
	if purgedAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, purgedAt)
		if err != nil {
			return reportrun.Run{}, err
		}
		run.PurgedAt = parsed
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return reportrun.Run{}, err
	}
	parsedUpdatedAt, err := time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return reportrun.Run{}, err
	}
	run.CreatedAt = parsedCreatedAt
	run.UpdatedAt = parsedUpdatedAt
	run.Usage.UsagePartial = usagePartial == 1
	return run, nil
}

func runSelectSQL() string {
	return `
SELECT plasma_report_runs.run_id, plasma_report_runs.mission_id,
       plasma_report_runs.root_pending_event_id, plasma_report_runs.lifecycle_state,
       plasma_report_runs.revision, plasma_report_runs.title,
       plasma_report_runs.final_artifact_id, plasma_report_runs.registration_status,
       plasma_report_runs.purged_at, plasma_report_runs.purged_by_type,
       plasma_report_runs.purged_by_id, plasma_report_runs.usage_record_count,
       plasma_report_runs.usage_available_count, plasma_report_runs.usage_unavailable_count,
       plasma_report_runs.input_tokens, plasma_report_runs.cached_input_tokens,
       plasma_report_runs.uncached_input_tokens, plasma_report_runs.output_tokens,
       plasma_report_runs.reasoning_output_tokens, plasma_report_runs.total_tokens,
       plasma_report_runs.usage_partial, plasma_report_runs.aggregation_version,
       plasma_report_runs.created_at, plasma_report_runs.updated_at
FROM plasma_report_runs`
}
