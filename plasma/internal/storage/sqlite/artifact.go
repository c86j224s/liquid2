package sqlite

import (
	"context"
	"encoding/json"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (s *Store) CreateRawArtifact(ctx context.Context, artifact app.RawArtifact) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO plasma_raw_artifacts (
  artifact_id, mission_id, media_type, byte_size, sha256, storage_uri, filename,
  producer_type, producer_id, created_at, content_blob
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		artifact.ArtifactID,
		artifact.MissionID,
		artifact.MediaType,
		artifact.ByteSize,
		artifact.SHA256,
		artifact.StorageURI,
		artifact.Filename,
		artifact.Producer.Type,
		artifact.Producer.ID,
		formatTime(artifact.CreatedAt),
		artifact.Content)
	return err
}

func (s *Store) GetRawArtifact(ctx context.Context, artifactID string) (app.RawArtifact, error) {
	var artifact app.RawArtifact
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
SELECT artifact_id, mission_id, media_type, byte_size, sha256, storage_uri,
       filename, producer_type, producer_id, created_at, content_blob
FROM plasma_raw_artifacts
WHERE artifact_id = ?`, artifactID).Scan(
		&artifact.ArtifactID,
		&artifact.MissionID,
		&artifact.MediaType,
		&artifact.ByteSize,
		&artifact.SHA256,
		&artifact.StorageURI,
		&artifact.Filename,
		&artifact.Producer.Type,
		&artifact.Producer.ID,
		&createdAt,
		&artifact.Content)
	if err != nil {
		return app.RawArtifact{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return app.RawArtifact{}, err
	}
	artifact.CreatedAt = parsed
	artifact.Content = append([]byte(nil), artifact.Content...)
	return artifact, nil
}

func (s *Store) ListRawArtifacts(ctx context.Context, missionID string) ([]app.RawArtifact, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT artifact_id
FROM plasma_raw_artifacts
WHERE mission_id = ?
ORDER BY created_at DESC, artifact_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []app.RawArtifact
	for rows.Next() {
		var artifactID string
		if err := rows.Scan(&artifactID); err != nil {
			return nil, err
		}
		artifact, err := s.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, rows.Err()
}

func (s *Store) CreateSourceSnapshot(ctx context.Context, snapshot app.SourceSnapshot) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	locatorsJSON := string(snapshot.Locators)
	if locatorsJSON == "" {
		locatorsJSON = "[]"
	}
	if !json.Valid([]byte(locatorsJSON)) {
		return app.ErrInvalidInput
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO plasma_source_snapshots (
  snapshot_id, mission_id, connector_id, connector_type, external_source_id,
  external_uri, external_version, connector_version, title, captured_at,
  external_updated_at, content_hash_algorithm, content_hash_value,
  locators_json, access_visibility, access_license, retrieval_policy
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snapshot.SnapshotID,
		snapshot.MissionID,
		snapshot.Connector.ConnectorID,
		snapshot.Connector.ConnectorType,
		snapshot.Connector.ExternalSourceID,
		snapshot.Connector.ExternalURI,
		snapshot.Connector.ExternalVersion,
		snapshot.Connector.ConnectorVersion,
		snapshot.Title,
		formatTime(snapshot.CapturedAt),
		formatOptionalTime(snapshot.ExternalUpdatedAt),
		snapshot.ContentHash.Algorithm,
		snapshot.ContentHash.Value,
		locatorsJSON,
		snapshot.Access.Visibility,
		snapshot.Access.License,
		snapshot.Access.RetrievalPolicy); err != nil {
		return err
	}

	for i, artifactID := range snapshot.ArtifactIDs {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO plasma_source_snapshot_artifacts (snapshot_id, artifact_id, ordinal)
VALUES (?, ?, ?)`,
			snapshot.SnapshotID,
			artifactID,
			i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetSourceSnapshot(ctx context.Context, snapshotID string) (app.SourceSnapshot, error) {
	var snapshot app.SourceSnapshot
	var capturedAt string
	var externalUpdatedAt string
	var locatorsJSON string
	err := s.db.QueryRowContext(ctx, `
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
		return app.SourceSnapshot{}, err
	}

	parsed, err := time.Parse(time.RFC3339Nano, capturedAt)
	if err != nil {
		return app.SourceSnapshot{}, err
	}
	snapshot.CapturedAt = parsed
	if externalUpdatedAt != "" {
		parsedExternal, err := time.Parse(time.RFC3339Nano, externalUpdatedAt)
		if err != nil {
			return app.SourceSnapshot{}, err
		}
		snapshot.ExternalUpdatedAt = parsedExternal
	}
	snapshot.Locators = append([]byte(nil), locatorsJSON...)

	artifactIDs, err := s.snapshotArtifactIDs(ctx, snapshotID)
	if err != nil {
		return app.SourceSnapshot{}, err
	}
	snapshot.ArtifactIDs = artifactIDs
	return snapshot, nil
}

func (s *Store) ListSourceSnapshots(ctx context.Context, missionID string) ([]app.SourceSnapshot, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT snapshot_id
FROM plasma_source_snapshots
WHERE mission_id = ?
ORDER BY captured_at DESC, snapshot_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []app.SourceSnapshot
	for rows.Next() {
		var snapshotID string
		if err := rows.Scan(&snapshotID); err != nil {
			return nil, err
		}
		snapshot, err := s.GetSourceSnapshot(ctx, snapshotID)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (s *Store) snapshotArtifactIDs(ctx context.Context, snapshotID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
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

func formatOptionalTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
