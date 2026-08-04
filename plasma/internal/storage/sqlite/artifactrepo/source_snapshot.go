package artifactrepo

import (
	"context"
	"time"

	sourcemodel "github.com/c86j224s/liquid2/plasma/internal/source"
)

// CreateSourceSnapshot stores a source snapshot and its ordered artifact links.
func (r *Repository) CreateSourceSnapshot(ctx context.Context, snapshot sourcemodel.Snapshot) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := InsertSourceSnapshotTx(ctx, tx, snapshot); err != nil {
		return err
	}
	return tx.Commit()
}

// GetSourceSnapshot reads a source snapshot by stable snapshot ID.
func (r *Repository) GetSourceSnapshot(ctx context.Context, snapshotID string) (sourcemodel.Snapshot, error) {
	var snapshot sourcemodel.Snapshot
	var capturedAt string
	var externalUpdatedAt string
	var locatorsJSON string
	err := r.db.QueryRowContext(ctx, `
SELECT snapshot_id, mission_id, connector_id, connector_type, external_source_id,
       external_uri, external_version, connector_version, title, captured_at,
       external_updated_at, content_hash_algorithm, content_hash_value,
       locators_json, access_visibility, access_license, retrieval_policy
FROM plasma_source_snapshots
WHERE snapshot_id = ?`, snapshotID).Scan(
		&snapshot.SnapshotID,
		&snapshot.MissionID,
		&snapshot.Connector.ConnectorID,
		&snapshot.Connector.ConnectorType,
		&snapshot.Connector.ExternalSourceID,
		&snapshot.Connector.ExternalURI,
		&snapshot.Connector.ExternalVersion,
		&snapshot.Connector.ConnectorVersion,
		&snapshot.Title,
		&capturedAt,
		&externalUpdatedAt,
		&snapshot.ContentHash.Algorithm,
		&snapshot.ContentHash.Value,
		&locatorsJSON,
		&snapshot.Access.Visibility,
		&snapshot.Access.License,
		&snapshot.Access.RetrievalPolicy)
	if err != nil {
		return sourcemodel.Snapshot{}, err
	}

	parsed, err := time.Parse(time.RFC3339Nano, capturedAt)
	if err != nil {
		return sourcemodel.Snapshot{}, err
	}
	snapshot.CapturedAt = parsed
	if externalUpdatedAt != "" {
		parsedExternal, err := time.Parse(time.RFC3339Nano, externalUpdatedAt)
		if err != nil {
			return sourcemodel.Snapshot{}, err
		}
		snapshot.ExternalUpdatedAt = parsedExternal
	}
	snapshot.Locators = append([]byte(nil), locatorsJSON...)

	artifactIDs, err := r.snapshotArtifactIDs(ctx, snapshotID)
	if err != nil {
		return sourcemodel.Snapshot{}, err
	}
	snapshot.ArtifactIDs = artifactIDs
	return snapshot, nil
}

// ListSourceSnapshots reads mission source snapshots ordered by capture time descending.
func (r *Repository) ListSourceSnapshots(ctx context.Context, missionID string) ([]sourcemodel.Snapshot, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT snapshot_id
FROM plasma_source_snapshots
WHERE mission_id = ?
ORDER BY captured_at DESC, snapshot_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []sourcemodel.Snapshot
	for rows.Next() {
		var snapshotID string
		if err := rows.Scan(&snapshotID); err != nil {
			return nil, err
		}
		snapshot, err := r.GetSourceSnapshot(ctx, snapshotID)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (r *Repository) snapshotArtifactIDs(ctx context.Context, snapshotID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT artifact_id
FROM plasma_source_snapshot_artifacts
WHERE snapshot_id = ?
ORDER BY ordinal`, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifactIDs []string
	for rows.Next() {
		var artifactID string
		if err := rows.Scan(&artifactID); err != nil {
			return nil, err
		}
		artifactIDs = append(artifactIDs, artifactID)
	}
	return artifactIDs, rows.Err()
}
