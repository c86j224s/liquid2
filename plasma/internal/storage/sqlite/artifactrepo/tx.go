package artifactrepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	artifactmodel "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	sourcemodel "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// InsertRawArtifactTx inserts a raw artifact inside a caller-owned transaction.
func InsertRawArtifactTx(ctx context.Context, tx *sql.Tx, artifact artifactmodel.Raw) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO plasma_raw_artifacts (
  artifact_id, mission_id, media_type, byte_size, sha256, storage_uri, filename,
  producer_type, producer_id, created_at, content_blob
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rawArtifactArgs(artifact)...)
	return err
}

// GetRawArtifactTx reads a raw artifact inside a caller-owned transaction or queryer.
func GetRawArtifactTx(ctx context.Context, tx interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, artifactID string) (artifactmodel.Raw, error) {
	var artifact artifactmodel.Raw
	var createdAt string
	err := tx.QueryRowContext(ctx, `
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
		return artifactmodel.Raw{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return artifactmodel.Raw{}, err
	}
	artifact.CreatedAt = parsed
	artifact.Content = append([]byte(nil), artifact.Content...)
	return artifact, nil
}

// InsertSourceSnapshotTx inserts a source snapshot and artifact links in a caller-owned transaction.
func InsertSourceSnapshotTx(ctx context.Context, tx *sql.Tx, snapshot sourcemodel.Snapshot) error {
	locatorsJSON := string(snapshot.Locators)
	if locatorsJSON == "" {
		locatorsJSON = "[]"
	}
	if !json.Valid([]byte(locatorsJSON)) {
		return producterror.ErrInvalidInput
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
		sqlitevalue.FormatTime(snapshot.CapturedAt),
		sqlitevalue.FormatOptionalTime(snapshot.ExternalUpdatedAt),
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
	return nil
}

// GetRawArtifactByMissionSHA reads an existing raw artifact for idempotent root workflows.
func GetRawArtifactByMissionSHA(ctx context.Context, tx *sql.Tx, missionID, sha string) (artifactmodel.Raw, bool, error) {
	var artifactID string
	err := tx.QueryRowContext(ctx, `
SELECT artifact_id
FROM plasma_raw_artifacts
WHERE mission_id = ? AND sha256 = ?`, missionID, sha).Scan(&artifactID)
	if err == sql.ErrNoRows {
		return artifactmodel.Raw{}, false, nil
	}
	if err != nil {
		return artifactmodel.Raw{}, false, err
	}
	artifact, err := GetRawArtifactTx(ctx, tx, artifactID)
	return artifact, err == nil, err
}

func rawArtifactArgs(artifact artifactmodel.Raw) []any {
	return []any{
		artifact.ArtifactID,
		artifact.MissionID,
		artifact.MediaType,
		artifact.ByteSize,
		artifact.SHA256,
		artifact.StorageURI,
		artifact.Filename,
		artifact.Producer.Type,
		artifact.Producer.ID,
		sqlitevalue.FormatTime(artifact.CreatedAt),
		artifact.Content,
	}
}
