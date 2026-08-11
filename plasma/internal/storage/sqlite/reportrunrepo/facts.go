package reportrunrepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"mime"
	"sort"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
)

// DeleteFactsForArtifactTx loads the transaction snapshot needed to preview or
// execute deletion for the report run owning artifactID.
func DeleteFactsForArtifactTx(ctx context.Context, tx *sql.Tx, missionID string, artifactID string) (reportrun.DeleteFacts, error) {
	run, err := runForArtifactTx(ctx, tx, missionID, artifactID)
	if err != nil {
		return reportrun.DeleteFacts{}, err
	}
	inconsistent, err := hasRunMissionInconsistentMembershipTx(ctx, tx, run.RunID)
	if err != nil {
		return reportrun.DeleteFacts{}, err
	}
	if inconsistent {
		run.LifecycleState = reportrun.LifecycleAmbiguous
	}
	events, err := memberEventsTx(ctx, tx, run.RunID)
	if err != nil {
		return reportrun.DeleteFacts{}, err
	}
	artifacts, err := memberArtifactsTx(ctx, tx, run.RunID)
	if err != nil {
		return reportrun.DeleteFacts{}, err
	}
	shared, malformed, err := sharedArtifactsTx(ctx, tx, run, events, artifacts)
	if err != nil {
		return reportrun.DeleteFacts{}, err
	}
	return reportrun.DeleteFacts{Run: run, Events: events, Artifacts: artifacts, SharedArtifacts: shared, MalformedReference: malformed}, nil
}

func runForArtifactTx(ctx context.Context, tx *sql.Tx, missionID string, artifactID string) (reportrun.Run, error) {
	runID, err := finalMarkdownRunForArtifactTx(ctx, tx, missionID, artifactID)
	if err != nil {
		return reportrun.Run{}, err
	}
	row := tx.QueryRowContext(ctx, runSelectSQL()+`
WHERE plasma_report_runs.run_id = ?`, runID)
	return scanRun(row)
}

func finalMarkdownRunForArtifactTx(ctx context.Context, tx *sql.Tx, missionID string, artifactID string) (string, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT a.run_id, a.artifact_role, raw.media_type
FROM plasma_report_run_artifacts a
JOIN plasma_report_runs r ON r.run_id = a.run_id
JOIN plasma_raw_artifacts raw ON raw.artifact_id = a.artifact_id
WHERE r.mission_id = ?
  AND a.mission_id = r.mission_id
  AND raw.mission_id = r.mission_id
  AND a.artifact_id = ?
  AND a.ownership = 'created'
  AND r.lifecycle_state <> 'purged'
ORDER BY r.updated_at DESC, r.run_id`, missionID, artifactID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var runIDs []string
	for rows.Next() {
		var runID string
		var role string
		var mediaType string
		if err := rows.Scan(&runID, &role, &mediaType); err != nil {
			return "", err
		}
		if role != reportrun.ArtifactRoleFinal || !isCanonicalMarkdownMediaType(mediaType) {
			continue
		}
		runIDs = append(runIDs, runID)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(runIDs) == 0 {
		return "", sql.ErrNoRows
	}
	if len(runIDs) != 1 {
		return "", sql.ErrNoRows
	}
	return runIDs[0], nil
}

func memberEventsTx(ctx context.Context, tx *sql.Tx, runID string) ([]reportrun.MemberEvent, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT m.run_id, m.event_id, m.mission_id, m.event_role, m.attempt_event_id, m.created_at,
       e.event_id, e.mission_id, e.sequence, e.event_type, e.payload_json, e.created_at
FROM plasma_report_run_events m
JOIN plasma_report_runs r ON r.run_id = m.run_id AND r.mission_id = m.mission_id
JOIN plasma_ledger_events e ON e.event_id = m.event_id AND e.mission_id = r.mission_id
WHERE m.run_id = ?
ORDER BY e.sequence`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []reportrun.MemberEvent
	for rows.Next() {
		var member reportrun.MemberEvent
		var membershipCreatedAt string
		var eventCreatedAt string
		if err := rows.Scan(
			&member.Membership.RunID, &member.Membership.EventID,
			&member.Membership.MissionID, &member.Membership.EventRole,
			&member.Membership.AttemptEventID, &membershipCreatedAt,
			&member.Event.EventID, &member.Event.MissionID, &member.Event.Sequence,
			&member.Event.EventType, &member.Event.Payload, &eventCreatedAt); err != nil {
			return nil, err
		}
		created, err := parseTime(membershipCreatedAt)
		if err != nil {
			return nil, err
		}
		eventCreated, err := parseTime(eventCreatedAt)
		if err != nil {
			return nil, err
		}
		member.Membership.CreatedAt = created
		member.Event.CreatedAt = eventCreated
		events = append(events, member)
	}
	return events, rows.Err()
}

func memberArtifactsTx(ctx context.Context, tx *sql.Tx, runID string) ([]reportrun.MemberArtifact, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT m.run_id, m.artifact_id, m.mission_id, m.artifact_role, m.ownership,
       m.attempt_event_id, m.source_event_id, m.created_at,
       a.artifact_id, a.mission_id, a.media_type, a.byte_size
FROM plasma_report_run_artifacts m
JOIN plasma_report_runs r ON r.run_id = m.run_id AND r.mission_id = m.mission_id
JOIN plasma_raw_artifacts a ON a.artifact_id = m.artifact_id AND a.mission_id = r.mission_id
WHERE m.run_id = ?
ORDER BY m.created_at, m.artifact_id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var artifacts []reportrun.MemberArtifact
	for rows.Next() {
		var member reportrun.MemberArtifact
		var createdAt string
		if err := rows.Scan(
			&member.Membership.RunID, &member.Membership.ArtifactID,
			&member.Membership.MissionID, &member.Membership.ArtifactRole,
			&member.Membership.Ownership, &member.Membership.AttemptEventID,
			&member.Membership.SourceEventID, &createdAt,
			&member.Artifact.ArtifactID, &member.Artifact.MissionID,
			&member.Artifact.MediaType, &member.Artifact.ByteSize); err != nil {
			return nil, err
		}
		created, err := parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		member.Membership.CreatedAt = created
		artifacts = append(artifacts, member)
	}
	return artifacts, rows.Err()
}

func hasRunMissionInconsistentMembershipTx(ctx context.Context, tx *sql.Tx, runID string) (bool, error) {
	var count int64
	err := tx.QueryRowContext(ctx, `
SELECT (
  SELECT COUNT(*)
  FROM plasma_report_run_events m
  JOIN plasma_report_runs r ON r.run_id = m.run_id
  LEFT JOIN plasma_ledger_events e ON e.event_id = m.event_id
  WHERE m.run_id = ?
    AND (m.mission_id <> r.mission_id OR e.event_id IS NULL OR e.mission_id <> r.mission_id)
) + (
  SELECT COUNT(*)
  FROM plasma_report_run_artifacts m
  JOIN plasma_report_runs r ON r.run_id = m.run_id
  LEFT JOIN plasma_raw_artifacts a ON a.artifact_id = m.artifact_id
  WHERE m.run_id = ?
    AND (m.mission_id <> r.mission_id OR a.artifact_id IS NULL OR a.mission_id <> r.mission_id)
)`, runID, runID).Scan(&count)
	return count > 0, err
}

func sharedArtifactsTx(ctx context.Context, tx *sql.Tx, run reportrun.Run, events []reportrun.MemberEvent, artifacts []reportrun.MemberArtifact) ([]reportrun.SharedArtifact, bool, error) {
	candidates := map[string]int64{}
	for _, artifact := range artifacts {
		if artifact.Membership.Ownership == reportrun.OwnershipCreated && artifact.Membership.ArtifactRole != reportrun.ArtifactRoleInput {
			candidates[artifact.Artifact.ArtifactID] = artifact.Artifact.ByteSize
		}
	}
	if len(candidates) == 0 {
		return nil, false, nil
	}
	reasons := map[string]map[string]bool{}
	addReason := func(artifactID, reason string) {
		if reasons[artifactID] == nil {
			reasons[artifactID] = map[string]bool{}
		}
		reasons[artifactID][reason] = true
	}
	for artifactID := range candidates {
		if linked, err := hasSourceSnapshotLinkTx(ctx, tx, artifactID); err != nil {
			return nil, false, err
		} else if linked {
			addReason(artifactID, "source_snapshot_link")
		}
		if linked, err := hasOtherRunMembershipTx(ctx, tx, run.RunID, artifactID); err != nil {
			return nil, false, err
		} else if linked {
			addReason(artifactID, "other_report_run")
		}
	}
	referenced, malformed, err := outsideLedgerReferencesTx(ctx, tx, memberEventIDs(events), candidates)
	if err != nil {
		return nil, false, err
	}
	for artifactID := range referenced {
		addReason(artifactID, "ledger_reference")
	}
	var shared []reportrun.SharedArtifact
	for artifactID, reasonSet := range reasons {
		var itemReasons []string
		for reason := range reasonSet {
			itemReasons = append(itemReasons, reason)
		}
		sort.Strings(itemReasons)
		shared = append(shared, reportrun.SharedArtifact{ArtifactID: artifactID, ByteSize: candidates[artifactID], Reasons: itemReasons})
	}
	sort.Slice(shared, func(i, j int) bool { return shared[i].ArtifactID < shared[j].ArtifactID })
	return shared, malformed, nil
}

func hasSourceSnapshotLinkTx(ctx context.Context, tx *sql.Tx, artifactID string) (bool, error) {
	var count int64
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM plasma_source_snapshot_artifacts WHERE artifact_id = ?`, artifactID).Scan(&count)
	return count > 0, err
}

func hasOtherRunMembershipTx(ctx context.Context, tx *sql.Tx, runID string, artifactID string) (bool, error) {
	var count int64
	err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM plasma_report_run_artifacts a
JOIN plasma_report_runs r ON r.run_id = a.run_id
WHERE a.artifact_id = ? AND a.run_id <> ? AND r.lifecycle_state <> 'purged'`, artifactID, runID).Scan(&count)
	return count > 0, err
}

func outsideLedgerReferencesTx(ctx context.Context, tx *sql.Tx, memberIDs map[string]bool, candidates map[string]int64) (map[string]bool, bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT event_id, payload_json FROM plasma_ledger_events`)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	referenced := map[string]bool{}
	malformed := false
	for rows.Next() {
		var eventID string
		var payloadJSON string
		if err := rows.Scan(&eventID, &payloadJSON); err != nil {
			return nil, false, err
		}
		if memberIDs[eventID] {
			continue
		}
		var payload any
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			malformed = true
			continue
		}
		findArtifactReferences(payload, candidates, referenced)
	}
	return referenced, malformed, rows.Err()
}

func isCanonicalMarkdownMediaType(mediaType string) bool {
	base, _, err := mime.ParseMediaType(strings.TrimSpace(mediaType))
	if err != nil {
		base = mediaType
	}
	base = strings.ToLower(strings.TrimSpace(base))
	return base == "text/markdown" || base == "text/x-markdown"
}

func findArtifactReferences(value any, candidates map[string]int64, found map[string]bool) {
	switch typed := value.(type) {
	case string:
		if _, ok := candidates[typed]; ok {
			found[typed] = true
		}
	case []any:
		for _, item := range typed {
			findArtifactReferences(item, candidates, found)
		}
	case map[string]any:
		for _, item := range typed {
			findArtifactReferences(item, candidates, found)
		}
	}
}

func memberEventIDs(events []reportrun.MemberEvent) map[string]bool {
	out := map[string]bool{}
	for _, event := range events {
		out[event.Event.EventID] = true
	}
	return out
}
